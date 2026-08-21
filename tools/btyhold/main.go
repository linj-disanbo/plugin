// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main 提供 bty 主币余额扫描工具:
//   - 离线模式(直接读 store 数据库): 列出所有持有 bty 的地址、把每个合约当成一个账户列出其总余额,
//     以及查询某个合约内部不同账户的余额;
//   - 在线模式(走节点公开 JSON-RPC): 列出各合约(执行器)账户的总余额(无内部地址明细)。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "scan", "holders", "contracts", "contract":
		if err := runOffline(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "rpc-contracts", "rpc-totals":
		if err := runRPC(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`btyscan - 遍历 chain33 store 数据库 / 节点 RPC, 统计 bty 主币持有地址与合约账户

用法:
  btyscan <command> [flags]

离线命令(直接读 store 数据库, 需要先停止节点, 或拷贝一份 datadir):
  scan        扫描并输出: 汇总 + bty 持有地址列表 + 合约账户列表
  holders     只输出 bty 持有地址(主账户 balance+frozen>0)
  contracts   只输出合约账户(把每个合约当成一个账户, 汇总其总余额/冻结/内部账户数)
  contract    查询某个合约内部不同账户的余额(钻取), 用 --exec 指定合约

在线命令(连节点 JSON-RPC, 不需要停节点):
  rpc-contracts  通过公开 RPC 列出所有合约账户的总余额(无内部地址明细)
  rpc-totals     通过公开 RPC 输出主链 bty 账户总数与总量

公共参数:
  --db <dir>            store 数据库目录(含 store.db), 默认 datadir/mavltree
  --execer <name>       币所在执行器, 默认 coins
  --symbol <symbol>     币符号, 默认 bty
  --exec <addr|name>    合约地址或执行器名(contract 钻取用)
  --min <n>             只显示余额(含冻结)>= n satoshi 的项, 默认 0
  --all                 连 0 余额的主账户也列出(默认只列 balance+frozen>0)
  --csv <file>          明细写入 CSV; scan 模式拆两个文件: <file> 存持有地址,
                        <去扩展名>-contracts<扩展名> 存合约账户; 其它命令写单个文件
  --execers-file <file> 每行一个执行器名, 用于把合约地址映射成名字(可扩展)
  --url <url>           JSON-RPC 地址, 默认 http://127.0.0.1:9671
  --count <n>           RPC 分页每页条数, 默认 10000
  --statehash <hash>    最新状态哈希(0x 开头); 纯 mavl 库(未开 prune)需要显式指定,
                        可在节点运行时用 chain33.GetLastHeader RPC 获取
  --layout <layout>     store 布局: auto|mvcciter|mvccdata|mavl, 默认 auto(自动识别)
  -h, --help            帮助

示例:
  # 先停节点(或拷贝 datadir), 再扫描
  btyscan scan --db datadir/mavltree --symbol bty
  # scan --csv 会生成 holders.csv 与 holders-contracts.csv 两个文件
  btyscan scan --db datadir/mavltree --csv holders.csv
  # 只导出持有地址 / 只导出合约账户
  btyscan holders --db datadir/mavltree --csv holders.csv
  btyscan contracts --db datadir/mavltree --csv contracts.csv
  btyscan contracts --db datadir/mavltree --min 100000000
  # ticket 合约(执行器)内部各账户余额
  btyscan contract --db datadir/mavltree --exec ticket
  btyscan contract --db datadir/mavltree --exec 16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp
  # 节点在线时, 用公开 RPC 看合约总余额
  btyscan rpc-contracts --url http://127.0.0.1:9671
  btyscan rpc-totals --url http://127.0.0.1:9671
`)
}

// options 所有子命令的公共参数
type options struct {
	storeDir    string
	execer      string
	symbol      string
	exec        string
	csvFile     string
	execersFile string
	min         int64
	all         bool
	url         string
	count       int64
	stateHash   string
	layout      string
}

func parseFlags(cmd string, args []string) (*options, error) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	opts := &options{}
	fs.StringVar(&opts.storeDir, "db", "datadir/mavltree", "store db dir")
	fs.StringVar(&opts.execer, "execer", "coins", "coin execer")
	fs.StringVar(&opts.symbol, "symbol", "bty", "coin symbol")
	fs.StringVar(&opts.exec, "exec", "", "contract address or execer name")
	fs.StringVar(&opts.csvFile, "csv", "", "csv output file")
	fs.StringVar(&opts.execersFile, "execers-file", "", "file with extra execer names")
	fs.Int64Var(&opts.min, "min", 0, "minimum balance (satoshi)")
	fs.BoolVar(&opts.all, "all", false, "include zero-balance main accounts")
	fs.StringVar(&opts.url, "url", "http://127.0.0.1:9671", "json-rpc url")
	fs.Int64Var(&opts.count, "count", 10000, "rpc page size")
	fs.StringVar(&opts.stateHash, "statehash", "", "latest state hash (hex, 纯 mavl 布局需要)")
	fs.StringVar(&opts.layout, "layout", "auto", "store layout: auto|mvcciter|mvccdata|mavl")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	return opts, nil
}

func runOffline(cmd string, args []string) error {
	opts, err := parseFlags(cmd, args)
	if err != nil {
		return err
	}
	em, err := loadExecers(opts.execersFile)
	if err != nil {
		return err
	}
	res, err := scanStore(opts.storeDir, opts.execer, opts.symbol, opts.stateHash, opts.layout)
	if err != nil {
		return err
	}
	switch cmd {
	case "scan":
		return reportScan(res, em, opts.min, opts.all, opts.csvFile)
	case "holders":
		return reportHolders(res, opts.min, opts.all, opts.csvFile)
	case "contracts":
		return reportContracts(res, em, opts.min, opts.csvFile)
	case "contract":
		if opts.exec == "" {
			return fmt.Errorf("contract 子命令需要 --exec <合约地址或执行器名>")
		}
		return reportContract(res, em, opts.exec, opts.csvFile)
	}
	return fmt.Errorf("unknown offline command %q", cmd)
}
