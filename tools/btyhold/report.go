// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// contractStat 合约(执行器)账户汇总
type contractStat struct {
	ExecAddr string
	Name     string
	Balance  int64
	Frozen   int64
	Count    int
}

func (c *contractStat) Total() int64 { return c.Balance + c.Frozen }

// formatCoin 把 satoshi 转成 8 位小数币值
func formatCoin(v int64) string {
	return strconv.FormatFloat(float64(v)/1e8, 'f', 8, 64)
}

func printSummary(res *scanResult, mainTotal, mainFrozen, execTotal int64, holderCount, zeroCount, contractCount int) {
	fmt.Printf("== 存储布局: %s\n", res.Layout)
	fmt.Printf("主账户总数: %d (持有者 %d, 零余额 %d)\n", len(res.Main), holderCount, zeroCount)
	fmt.Printf("主账户余额合计: %s (冻结 %s)\n", formatCoin(mainTotal), formatCoin(mainFrozen))
	fmt.Printf("合约账户: %d 个合约, %d 条记录, 余额合计 %s (冻结 %s)\n",
		contractCount, len(res.Exec), formatCoin(execTotal), formatCoin(execFrozenTotal(res)))
	fmt.Printf("主账户+合约账户总资产: %s\n", formatCoin(mainTotal+mainFrozen+execTotal))
}

func execFrozenTotal(res *scanResult) int64 {
	var t int64
	for _, e := range res.Exec {
		t += e.Frozen
	}
	return t
}

func sortedMain(res *scanResult, min int64, all bool) []mainAccount {
	var out []mainAccount
	for _, a := range res.Main {
		if !all && a.Balance+a.Frozen <= 0 {
			continue
		}
		if a.Balance+a.Frozen < min {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].Balance+out[i].Frozen, out[j].Balance+out[j].Frozen
		if ti != tj {
			return ti > tj
		}
		return out[i].Addr < out[j].Addr
	})
	return out
}

func buildContracts(res *scanResult, em map[string]string, min int64) []contractStat {
	m := make(map[string]*contractStat)
	for _, e := range res.Exec {
		c, ok := m[e.ExecAddr]
		if !ok {
			c = &contractStat{ExecAddr: e.ExecAddr, Name: execName(em, e.ExecAddr)}
			m[e.ExecAddr] = c
		}
		c.Balance += e.Balance
		c.Frozen += e.Frozen
		c.Count++
	}
	var out []contractStat
	for _, c := range m {
		if c.Total() >= min {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total() != out[j].Total() {
			return out[i].Total() > out[j].Total()
		}
		return out[i].ExecAddr < out[j].ExecAddr
	})
	return out
}

func writeCSV(file string, header []string, rows [][]string) error {
	if file == "" {
		return nil
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		if err := w.Write(r); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	fmt.Printf("已写入 CSV: %s (%d 行)\n", file, len(rows))
	return nil
}

func reportScan(res *scanResult, em map[string]string, min int64, all bool, csvFile string) error {
	holdersFile := csvFile
	contractsFile := ""
	if csvFile != "" {
		// scan 同时输出持有地址与合约账户, 拆成两个文件, 避免相互覆盖
		ext := filepath.Ext(csvFile)
		contractsFile = strings.TrimSuffix(csvFile, ext) + "-contracts" + ext
	}
	if err := reportHolders(res, min, all, holdersFile); err != nil {
		return err
	}
	fmt.Println()
	return reportContracts(res, em, min, contractsFile)
}

func reportHolders(res *scanResult, min int64, all bool, csvFile string) error {
	holders := sortedMain(res, min, all)
	var mainTotal, mainFrozen int64
	var holderCount, zeroCount int
	for _, a := range res.Main {
		mainTotal += a.Balance
		mainFrozen += a.Frozen
		if a.Balance+a.Frozen > 0 {
			holderCount++
		} else {
			zeroCount++
		}
	}
	printSummary(res, mainTotal, mainFrozen, execTotal(res), holderCount, zeroCount, contractCount(res))

	fmt.Println()
	fmt.Printf("== bty 持有地址(%d): addr, balance, frozen, total\n", len(holders))
	var rows [][]string
	for _, a := range holders {
		line := fmt.Sprintf("%s  %s  %s  %s",
			a.Addr, formatCoin(a.Balance), formatCoin(a.Frozen), formatCoin(a.Balance+a.Frozen))
		fmt.Println(line)
		rows = append(rows, []string{a.Addr, strconv.FormatInt(a.Balance, 10),
			strconv.FormatInt(a.Frozen, 10), strconv.FormatInt(a.Balance+a.Frozen, 10),
			formatCoin(a.Balance), formatCoin(a.Frozen), formatCoin(a.Balance+a.Frozen)})
	}
	if err := writeCSV(csvFile, []string{"addr", "balance", "frozen", "total", "balance_coin", "frozen_coin", "total_coin"}, rows); err != nil {
		return err
	}
	return nil
}

func reportContracts(res *scanResult, em map[string]string, min int64, csvFile string) error {
	contracts := buildContracts(res, em, min)
	fmt.Printf("== 合约账户(把每个合约当成一个账户, %d): execAddr, name, balance, frozen, total, 内部账户数\n", len(contracts))
	var rows [][]string
	for _, c := range contracts {
		name := c.Name
		if name == "" {
			name = "(未知执行器)"
		}
		fmt.Printf("%s  %-14s  %s  %s  %s  %d\n",
			c.ExecAddr, name, formatCoin(c.Balance), formatCoin(c.Frozen), formatCoin(c.Total()), c.Count)
		rows = append(rows, []string{c.ExecAddr, name, strconv.FormatInt(c.Balance, 10),
			strconv.FormatInt(c.Frozen, 10), strconv.FormatInt(c.Total(), 10), strconv.Itoa(c.Count),
			formatCoin(c.Balance), formatCoin(c.Frozen), formatCoin(c.Total())})
	}
	return writeCSV(csvFile, []string{"execAddr", "name", "balance", "frozen", "total", "count", "balance_coin", "frozen_coin", "total_coin"}, rows)
}

func reportContract(res *scanResult, em map[string]string, execArg, csvFile string) error {
	execAddr, err := resolveExec(em, execArg)
	if err != nil {
		return err
	}
	name := execName(em, execAddr)
	if name == "" {
		name = "(未知执行器)"
	}
	var items []execAccount
	var bal, frozen int64
	for _, e := range res.Exec {
		if e.ExecAddr == execAddr {
			items = append(items, e)
			bal += e.Balance
			frozen += e.Frozen
		}
	}
	if len(items) == 0 {
		return fmt.Errorf("合约 %s (%s) 下没有 bty 余额记录", execAddr, name)
	}
	sort.Slice(items, func(i, j int) bool {
		ti, tj := items[i].Balance+items[i].Frozen, items[j].Balance+items[j].Frozen
		if ti != tj {
			return ti > tj
		}
		return items[i].Addr < items[j].Addr
	})
	fmt.Printf("== 合约 %s (%s) 内部账户: %d 个, 余额 %s, 冻结 %s, 合计 %s\n",
		execAddr, name, len(items), formatCoin(bal), formatCoin(frozen), formatCoin(bal+frozen))
	fmt.Println("addr, balance, frozen, total")
	var rows [][]string
	for _, e := range items {
		fmt.Printf("%s  %s  %s  %s\n", e.Addr, formatCoin(e.Balance), formatCoin(e.Frozen), formatCoin(e.Balance+e.Frozen))
		rows = append(rows, []string{e.Addr, strconv.FormatInt(e.Balance, 10), strconv.FormatInt(e.Frozen, 10),
			strconv.FormatInt(e.Balance+e.Frozen, 10), formatCoin(e.Balance), formatCoin(e.Frozen), formatCoin(e.Balance+e.Frozen)})
	}
	return writeCSV(csvFile, []string{"addr", "balance", "frozen", "total", "balance_coin", "frozen_coin", "total_coin"}, rows)
}

func execTotal(res *scanResult) int64 {
	var t int64
	for _, e := range res.Exec {
		t += e.Balance
	}
	return t
}

func contractCount(res *scanResult) int {
	m := make(map[string]struct{})
	for _, e := range res.Exec {
		m[e.ExecAddr] = struct{}{}
	}
	return len(m)
}
