// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/33cn/chain33/types"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// chain33 store 数据库的 key 布局(不同 store 驱动):
//   1. kvmvccmavl + enableMVCCIter=true  : 每个 key 的最新值放在 ".-mvcc-.l.<stateKey>"
//   2. kvmvccmavl + enableMVCCIter=false : 全历史版本放在 ".-mvcc-.d.<stateKey>.<20位版本号>"
//   3. 纯 mavl 树                        : 节点按哈希存储, 需要按最新 stateHash 迭代状态树
// 其中 stateKey 对 bty 主币为:
//   - 主账户: "mavl-<execer>-<symbol>-<addr>"
//   - 合约账户: "mavl-<execer>-<symbol>-exec-<合约地址>:<用户地址>"
const (
	mvccLast = ".-mvcc-.l."
	mvccData = ".-mvcc-.d."
)

// mainAccount 主账户余额
type mainAccount struct {
	Addr    string
	Balance int64
	Frozen  int64
}

// execAccount 合约(执行器)内的账户余额
type execAccount struct {
	ExecAddr string // 合约地址 = address.ExecAddress(执行器名)
	Addr     string // 用户地址
	Balance  int64
	Frozen   int64
}

// scanResult 一次扫描的结果
type scanResult struct {
	Layout string // 检测到的存储布局
	Main   []mainAccount
	Exec   []execAccount
}

func statePrefix(execer, symbol string) string {
	return "mavl-" + execer + "-" + symbol + "-"
}

func execPrefix(execer, symbol string) string {
	return statePrefix(execer, symbol) + "exec-"
}

// scanStore 打开 store 数据库并扫描
// stateHash: 纯 mavl 布局时可选, 指定最新状态哈希(0x 开头); 为空时尝试从库内记录获取
// layout: auto|mvcciter|mvccdata|mavl, 为空表示自动识别
func scanStore(storeDir, execer, symbol, stateHash, layout string) (*scanResult, error) {
	db, err := openStoreRO(storeDir)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	mp := statePrefix(execer, symbol)
	ep := execPrefix(execer, symbol)

	res := &scanResult{}
	switch layout {
	case "mvcciter":
		res.Layout = "kvmvccmavl(mvccIter, 最新值, 强制)"
		if err := scanMVCCLast(db, mp, ep, res); err != nil {
			return nil, err
		}
		return res, nil
	case "mvccdata":
		res.Layout = "kvmvccmavl(simpleMVCC, 版本去重, 强制)"
		if err := scanMVCCData(db, mp, ep, res); err != nil {
			return nil, err
		}
		return res, nil
	case "mavl":
		res.Layout = "mavl(状态树迭代, 强制)"
		if err := scanMavlTree(db, mp, ep, res, stateHash); err != nil {
			return nil, err
		}
		return res, nil
	}

	// 自动识别: 优先用 MVCC 全量数据(.-mvcc-.d. 所有历史版本, 按 key 取最大版本),
	// 该路径在任何 kvmvccmavl 节点上都完整可靠。last-key 快路径(.-mvcc-.l.)只包含
	// enableMVCCIter 启用之后被写过的 key —— 若节点中途才开启该配置, 数据不完整,
	// 因此只在用户显式指定 --layout mvcciter 时才使用。
	switch {
	case hasPrefix(db, []byte(mvccData+mp)):
		res.Layout = "kvmvccmavl(MVCC全量数据, 版本去重)"
		if err := scanMVCCData(db, mp, ep, res); err != nil {
			return nil, err
		}
	case hasPrefix(db, []byte(mvccLast+mp)):
		res.Layout = "kvmvccmavl(mvccIter last-key)"
		if err := scanMVCCLast(db, mp, ep, res); err != nil {
			return nil, err
		}
	case hasPrefix(db, []byte("_mrhp_")):
		res.Layout = "mavl(状态树迭代)"
		if err := scanMavlTree(db, mp, ep, res, stateHash); err != nil {
			return nil, err
		}
	case hasPrefix(db, []byte("mavl-")):
		// 可能是纯 mavl 但未开 prune(库内没有根哈希记录), 必须显式指定 --layout mavl 与 --statehash
		return nil, fmt.Errorf("检测到纯 mavl 布局但没有根哈希记录, 请先用节点 RPC(chain33.GetLastHeader) 获取最新 stateHash, 再以 --layout mavl --statehash 0x... 重新扫描")
	default:
		return nil, fmt.Errorf("无法识别 store 数据库布局(未找到 bty 账户数据前缀), 请确认 --db 指向 mavltree 目录且 --execer/--symbol 正确; 如确为纯 mavl 库请用 --layout mavl --statehash 0x... 强制指定")
	}
	return res, nil
}

// classifyStateKey 把 (stateKey, value) 归类为主账户或合约账户
func classifyStateKey(mp, ep, stateKey string, value []byte, res *scanResult) {
	if len(value) == 0 {
		return // 空值(删除标记)跳过
	}
	var acc types.Account
	if err := types.Decode(value, &acc); err != nil {
		return // 无法解码的键(非账户数据)跳过
	}
	switch {
	case strings.HasPrefix(stateKey, ep):
		rest := stateKey[len(ep):]
		execAddr, addr := splitExecKey(rest)
		res.Exec = append(res.Exec, execAccount{
			ExecAddr: execAddr,
			Addr:     addr,
			Balance:  acc.Balance,
			Frozen:   acc.Frozen,
		})
	case strings.HasPrefix(stateKey, mp):
		addr := stateKey[len(mp):]
		res.Main = append(res.Main, mainAccount{
			Addr:    addr,
			Balance: acc.Balance,
			Frozen:  acc.Frozen,
		})
	}
}

// splitExecKey 拆分 "合约地址:用户地址"
func splitExecKey(rest string) (execAddr, addr string) {
	idx := strings.IndexByte(rest, ':')
	if idx < 0 {
		return rest, ""
	}
	return rest[:idx], rest[idx+1:]
}

// scanMVCCLast 扫描 ".-mvcc-.l." 最新值命名空间
func scanMVCCLast(db *leveldb.DB, mp, ep string, res *scanResult) error {
	it := db.NewIterator(util.BytesPrefix([]byte(mvccLast+mp)), nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if !bytes.HasPrefix(key, []byte(mvccLast)) {
			continue
		}
		classifyStateKey(mp, ep, string(key[len(mvccLast):]), it.Value(), res)
	}
	return it.Error()
}

// splitVersion 拆分 "stateKey.<20位版本号>"
func splitVersion(rest string) (stateKey string, ok bool) {
	idx := strings.LastIndexByte(rest, '.')
	if idx < 0 {
		return rest, false
	}
	s := rest[idx+1:]
	if len(s) != 20 {
		return rest, false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err != nil {
		return rest, false
	}
	return rest[:idx], true
}

// scanMVCCData 扫描 ".-mvcc-.d." 全历史版本, 每个 stateKey 取版本号最大的一条
func scanMVCCData(db *leveldb.DB, mp, ep string, res *scanResult) error {
	it := db.NewIterator(util.BytesPrefix([]byte(mvccData+mp)), nil)
	defer it.Release()

	var lastKey string
	var lastVal []byte
	flush := func() {
		if lastKey != "" {
			classifyStateKey(mp, ep, lastKey, lastVal, res)
		}
	}
	for it.Next() {
		key := it.Key()
		if !bytes.HasPrefix(key, []byte(mvccData)) {
			continue
		}
		stateKey, ok := splitVersion(string(key[len(mvccData):]))
		if !ok {
			continue
		}
		// 相同 stateKey 的记录在 key 顺序上连续, 升序迭代时后者版本更大
		if stateKey != lastKey {
			flush()
			lastKey = stateKey
			lastVal = append(lastVal[:0], it.Value()...)
		} else {
			lastVal = append(lastVal[:0], it.Value()...)
		}
	}
	flush()
	return it.Error()
}

// scanMavlTree 纯 mavl 布局: 用最新 stateHash 迭代状态树叶子区间
func scanMavlTree(db *leveldb.DB, mp, ep string, res *scanResult, stateHash string) error {
	var rootHash []byte
	if stateHash != "" {
		hash := strings.TrimPrefix(stateHash, "0x")
		raw, err := hex.DecodeString(hash)
		if err != nil || len(raw) != 32 {
			return fmt.Errorf("--statehash 格式错误: %q", stateHash)
		}
		rootHash = raw
		res.Layout = "mavl(状态树迭代, 指定 stateHash)"
	} else {
		// 从库内记录获取: 最大高度 + 对应根哈希(key = "_mrhp_<010d高度><32字节哈希>")
		maxHeight, err := readInt64(db, []byte("_..mcmbh.._"))
		if err != nil {
			return fmt.Errorf("读取最大区块高度失败: %w (纯 mavl 未开 prune 时需用 --statehash 指定最新状态哈希)", err)
		}
		prefix := []byte(fmt.Sprintf("_mrhp_%010d", maxHeight))
		it := db.NewIterator(&util.Range{Start: prefix, Limit: incPrefix(prefix)}, nil)
		defer it.Release()
		if !it.Next() {
			return fmt.Errorf("高度 %d 未找到根哈希记录", maxHeight)
		}
		rootHash = append([]byte(nil), it.Key()[len(prefix):]...)
		res.Layout = "mavl(状态树迭代, height=" + strconv.FormatInt(maxHeight, 10) + ")"
	}

	// 迭代 [mp, ep 结尾+1) 区间, 覆盖主账户与合约账户
	start := []byte(mp)
	end := incPrefix([]byte(ep))
	ro := &roDB{db: db}
	mavl.IterateRangeByStateHash(ro, rootHash, start, end, true, &mavl.TreeConfig{}, func(key, value []byte) bool {
		classifyStateKey(mp, ep, string(key), value, res)
		return false
	})
	return nil
}

// readInt64 读取 proto Int64 编码的值
func readInt64(db *leveldb.DB, key []byte) (int64, error) {
	val, err := db.Get(key, nil)
	if err != nil {
		return 0, err
	}
	var v types.Int64
	if err := types.Decode(val, &v); err != nil {
		return 0, err
	}
	return v.Data, nil
}
