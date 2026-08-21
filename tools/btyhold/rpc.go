// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/33cn/chain33/types"
)

// 在线模式: 只使用节点公开 JSON-RPC, 不读数据库。
// 说明: 公开 RPC 只能拿到"合约账户汇总"(GetExecBalance 的返回项不含内部地址),
// 以及主链账户总数/总量(GetTotalCoins 不含地址明细)。要枚举持有地址或合约内部
// 账户余额, 请使用离线模式(scan/holders/contract)。

type jrpcClient struct {
	url string
	hc  *http.Client
	id  uint64
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

func newJRPC(url string) *jrpcClient {
	return &jrpcClient{
		url: url,
		hc:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *jrpcClient) call(method string, params any, out any) error {
	c.id++
	reqBody, err := json.Marshal(&rpcRequest{JSONRPC: "2.0", ID: c.id, Method: method, Params: []any{params}})
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return fmt.Errorf("rpc %s: %w", c.url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var rr rpcResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return fmt.Errorf("rpc %s: 非法响应 %q", method, string(body))
	}
	if len(rr.Error) > 0 && string(rr.Error) != "null" {
		return fmt.Errorf("rpc %s: %s", method, string(rr.Error))
	}
	if out != nil && len(rr.Result) > 0 {
		return json.Unmarshal(rr.Result, out)
	}
	return nil
}

type rpcHeader struct {
	Height    int64  `json:"height"`
	StateHash string `json:"stateHash"` // 0x 开头十六进制
}

// rpcGetStateHash 取最新 stateHash(用于 GetTotalCoins)
func rpcGetStateHash(cli *jrpcClient) (string, error) {
	var h rpcHeader
	if err := cli.call("Chain33.GetLastHeader", struct{}{}, &h); err != nil {
		return "", err
	}
	if h.StateHash == "" {
		return "", fmt.Errorf("GetLastHeader 未返回 stateHash")
	}
	return h.StateHash, nil
}

func base64ToKey(s string) string {
	if s == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(data)
}

func runRPC(cmd string, args []string) error {
	opts, err := parseFlags(cmd, args)
	if err != nil {
		return err
	}
	em, err := loadExecers(opts.execersFile)
	if err != nil {
		return err
	}
	cli := newJRPC(opts.url)

	switch cmd {
	case "rpc-totals":
		return rpcTotals(cli, opts)
	case "rpc-contracts":
		return rpcContracts(cli, em, opts)
	}
	return fmt.Errorf("unknown rpc command %q", cmd)
}

func rpcTotals(cli *jrpcClient, opts *options) error {
	stateHash, err := rpcGetStateHash(cli)
	if err != nil {
		return err
	}
	var totalNum, totalAmount int64
	var pages int
	startKey := ""
	for {
		var reply struct {
			Count   int64  `json:"count"`
			Num     int64  `json:"num"`
			Amount  int64  `json:"amount"`
			NextKey string `json:"nextKey"` // base64
		}
		params := map[string]any{
			"execer":   opts.execer,
			"symbol":   opts.symbol,
			"stateHash": hexToBase64(stateHash),
			"count":    opts.count,
			"startKey": base64.StdEncoding.EncodeToString([]byte(startKey)),
		}
		if err := cli.call("Chain33.GetTotalCoins", params, &reply); err != nil {
			return err
		}
		totalNum += reply.Num
		totalAmount += reply.Amount
		pages++
		next := base64ToKey(reply.NextKey)
		if next == "" || next == startKey {
			break
		}
		startKey = next
	}
	fmt.Printf("== RPC GetTotalCoins (execer=%s symbol=%s, stateHash=%s)\n", opts.execer, opts.symbol, stateHash)
	fmt.Printf("主链 bty 账户总数: %d (分页 %d 次)\n", totalNum, pages)
	fmt.Printf("主链 bty 总量: %s\n", formatCoin(totalAmount))
	fmt.Println("注意: GetTotalCoins 只统计主账户, 不含锁在合约里的 bty; 且不返回地址明细")
	if totalNum <= 1 {
		fmt.Println("警告: 返回的账户数异常偏少。节点公开的 GetTotalCoins 依赖 store 的迭代接口,")
		fmt.Println("      若节点 enableMVCCIter 启用较晚, 迭代只能看到近期写过的账户, 数据不完整。")
		fmt.Println("      完整数据请用离线模式: btyscan scan --db <mavltree目录>")
	}
	return nil
}

// hexToBase64 把 "0x..." 十六进制字符串转成 base64(chain33 JRPC 的 []byte 参数用 base64 编码)
func hexToBase64(s string) string {
	raw, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return s // 原样返回, 由服务端报错
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func rpcContracts(cli *jrpcClient, em map[string]string, opts *options) error {
	agg := make(map[string]*contractStat)
	startKey := ""
	stateHash, err := rpcGetStateHash(cli)
	if err != nil {
		return err
	}
	for {
		params := map[string]any{
			"execer":   opts.execer,
			"symbol":   opts.symbol,
			"count":    opts.count,
			"stateHash": hexToBase64(stateHash),
			"addr":     "",
			"nextKey":  base64.StdEncoding.EncodeToString([]byte(startKey)),
		}
		var raw string
		if err := cli.call("Chain33.GetExecBalance", params, &raw); err != nil {
			return err
		}
		data, err := hex.DecodeString(raw)
		if err != nil {
			return err
		}
		var reply types.ReplyGetExecBalance
		if err := types.Decode(data, &reply); err != nil {
			return err
		}
		for _, it := range reply.Items {
			addr := string(it.ExecAddr)
			c, ok := agg[addr]
			if !ok {
				c = &contractStat{ExecAddr: addr, Name: execName(em, addr)}
				agg[addr] = c
			}
			c.Balance += it.Active
			c.Frozen += it.Frozen
			c.Count++
		}
		next := string(reply.NextKey)
		if next == "" || next == startKey {
			break
		}
		startKey = next
	}

	var contracts []contractStat
	for _, c := range agg {
		if c.Total() >= opts.min {
			contracts = append(contracts, *c)
		}
	}
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].Total() != contracts[j].Total() {
			return contracts[i].Total() > contracts[j].Total()
		}
		return contracts[i].ExecAddr < contracts[j].ExecAddr
	})

	if opts.exec != "" {
		execAddr, err := resolveExec(em, opts.exec)
		if err != nil {
			return err
		}
		c, ok := agg[execAddr]
		if !ok {
			return fmt.Errorf("合约 %s 下没有 bty 余额", execAddr)
		}
		fmt.Printf("== 合约 %s (%s): 余额 %s, 冻结 %s, 合计 %s, 记录数 %d\n",
			execAddr, c.Name, formatCoin(c.Balance), formatCoin(c.Frozen), formatCoin(c.Total()), c.Count)
		fmt.Println("注意: 公开 RPC 的 GetExecBalance 不返回合约内部地址, 内部明细请用离线模式: btyscan contract --db ... --exec " + execAddr)
		return nil
	}

	fmt.Printf("== 合约账户(RPC, 把每个合约当成一个账户, %d): execAddr, name, balance, frozen, total, 记录数\n", len(contracts))
	fmt.Println("注意: 数据来自节点 GetExecBalance 迭代; 若节点 enableMVCCIter 启用较晚, 列表可能不完整;")
	fmt.Println("      且该 RPC 不返回合约内部地址, 权威/明细数据请用离线模式(btyscan scan --db ...)。")
	var rows [][]string
	for _, c := range contracts {
		name := c.Name
		if name == "" {
			name = "(未知执行器)"
		}
		fmt.Printf("%s  %-14s  %s  %s  %s  %d\n",
			c.ExecAddr, name, formatCoin(c.Balance), formatCoin(c.Frozen), formatCoin(c.Total()), c.Count)
		rows = append(rows, []string{c.ExecAddr, name, strconv.FormatInt(c.Balance, 10),
			strconv.FormatInt(c.Frozen, 10), strconv.FormatInt(c.Total(), 10), strconv.Itoa(c.Count)})
	}
	return writeCSV(opts.csvFile, []string{"execAddr", "name", "balance", "frozen", "total", "count"}, rows)
}
