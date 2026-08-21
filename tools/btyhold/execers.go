// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/33cn/chain33/common/address"
)

// defaultExecers 常见执行器(合约)名, 用于把合约地址映射成可读名字。
// 覆盖 chain33 系统执行器 + plugin 仓库主要 dapp; 自定义执行器可用 --execers-file 追加。
var defaultExecers = []string{
	// chain33 系统
	"coins", "manage", "none",
	// plugin dapp(主链常见)
	"ticket", "token", "trade", "paracross", "hashlock", "multisig", "evm",
	"cert", "privacy", "autonomy", "blackwhite", "collateralize", "coinsx",
	"dpos", "game", "guess", "issuance", "lottery", "oracle", "pokerbull",
	"qbftNode", "retrieve", "unfreeze", "valnode", "relay", "wasm",
	"exchange", "loan", "mix", "rollup", "vote", "storage", "lightclient",
	"echo", "accountmanager", "norm", "dex", "zksync", "bridgevmxgo",
	"x2ethereum", "evmxgo", "rgbx",
}

// loadExecers 返回 合约地址 -> 执行器名 的映射
func loadExecers(extraFile string) (map[string]string, error) {
	m := make(map[string]string)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		addr := address.ExecAddress(name)
		if _, ok := m[addr]; !ok {
			m[addr] = name
		}
	}
	for _, n := range defaultExecers {
		add(n)
	}
	if extraFile != "" {
		f, err := os.Open(extraFile)
		if err != nil {
			return nil, fmt.Errorf("open execers file: %w", err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			add(sc.Text())
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// execName 根据合约地址返回执行器名(未知则返回空)
func execName(em map[string]string, execAddr string) string {
	return em[execAddr]
}

// resolveExec 把 --exec 参数解析为合约地址; 支持直接传执行器名或合约地址
func resolveExec(em map[string]string, arg string) (string, error) {
	// 已知执行器名 -> 地址
	for addr, name := range em {
		if name == arg {
			return addr, nil
		}
	}
	// 合法地址(btc/eth 格式) -> 原样
	if err := address.CheckAddress(arg, -1); err == nil {
		return arg, nil
	}
	// 其它情况当作执行器名处理
	addr := address.ExecAddress(arg)
	if addr == "" {
		return "", fmt.Errorf("无法解析 --exec 参数: %q", arg)
	}
	return addr, nil
}
