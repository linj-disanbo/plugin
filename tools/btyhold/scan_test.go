// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	testMainAddr1 = "1CbEVT9RnM5oZhWMj4fxUrJX94VtRotzvs"
	testMainAddr2 = "1KcKqLWJ9nz6qX5B6MpF7v1n2qk3j4m5n6p7q8r9s"
	testMainAddr3 = "1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn"
	testExecAddr  = "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp" // ticket
)

var testTradeAddr = address.ExecAddress("trade")

// writeAccount 追加写入一个 proto Account 值
func writeAccount(batch *leveldb.Batch, key, addr string, balance, frozen int64) {
	acc := &types.Account{Addr: addr, Balance: balance, Frozen: frozen}
	batch.Put([]byte(key), types.Encode(acc))
}

func mkDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// genMVCCLastDB 构造 kvmvccmavl + mvccIter 布局(最新值在 .-mvcc-.l. 命名空间)
func genMVCCLastDB(t *testing.T, dir string) {
	t.Helper()
	db, err := leveldb.OpenFile(filepath.Join(dir, "store.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var batch leveldb.Batch
	// 主账户: 3 个
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-"+testMainAddr1, testMainAddr1, 500_0000_0000, 0)
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-"+testMainAddr2, testMainAddr2, 0, 100_0000_0000)
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-"+testMainAddr3, testMainAddr3, 0, 0)
	// 合约账户: ticket 合约下两个地址
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-exec-"+testExecAddr+":"+testMainAddr1, testMainAddr1, 800_0000_0000, 200_0000_0000)
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-exec-"+testExecAddr+":"+testMainAddr2, testMainAddr2, 0, 300_0000_0000)
	// trade 合约下一个地址
	writeAccount(&batch, ".-mvcc-.l.mavl-coins-bty-exec-"+testTradeAddr+":"+testMainAddr3, testMainAddr3, 10_0000_0000, 0)
	if err := db.Write(&batch, nil); err != nil {
		t.Fatal(err)
	}
}

// genMVCCDataDB 构造 kvmvccmavl + simpleMVCC 布局(全历史版本在 .-mvcc-.d. 命名空间)
func genMVCCDataDB(t *testing.T, dir string) {
	t.Helper()
	db, err := leveldb.OpenFile(filepath.Join(dir, "store.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var batch leveldb.Batch
	// 每个 key 写 2~3 个版本, 应取最大版本
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-"+testMainAddr1+".00000000000000000001", testMainAddr1, 100, 0)
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-"+testMainAddr1+".00000000000000000005", testMainAddr1, 500, 0)
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-"+testMainAddr2+".00000000000000000002", testMainAddr2, 0, 100)
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-"+testMainAddr3+".00000000000000000003", testMainAddr3, 0, 0)
	// 合约账户多版本
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-exec-"+testExecAddr+":"+testMainAddr1+".00000000000000000001", testMainAddr1, 100, 0)
	writeAccount(&batch, ".-mvcc-.d.mavl-coins-bty-exec-"+testExecAddr+":"+testMainAddr1+".00000000000000000009", testMainAddr1, 900, 100)
	if err := db.Write(&batch, nil); err != nil {
		t.Fatal(err)
	}
}

// genMavlDB 构造带 prune 记录的纯 mavl 布局(库内含 _mrhp_/_..mcmbh.._)
func genMavlDB(t *testing.T, dir string) {
	t.Helper()
	wdb, err := dbm.NewGoLevelDB("store", dir, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer wdb.Close()

	treeCfg := &mavl.TreeConfig{EnableMavlPrefix: true, EnableMavlPrune: true, PruneHeight: 1000}

	// 区块 1: 两个主账户 + 一个合约账户
	kvs1 := []*types.KeyValue{
		{Key: []byte("mavl-coins-bty-" + testMainAddr1), Value: types.Encode(&types.Account{Addr: testMainAddr1, Balance: 100})},
		{Key: []byte("mavl-coins-bty-" + testMainAddr2), Value: types.Encode(&types.Account{Addr: testMainAddr2, Balance: 200})},
		{Key: []byte("mavl-coins-bty-exec-" + testExecAddr + ":" + testMainAddr1), Value: types.Encode(&types.Account{Addr: testMainAddr1, Balance: 50})},
	}
	root1, err := mavl.SetKVPair(wdb, &types.StoreSet{Height: 1, KV: kvs1}, true, treeCfg)
	if err != nil {
		t.Fatal(err)
	}

	// 区块 2: 更新 addr1 余额, 新增 addr3 (必须串联前一区块 stateHash)
	kvs2 := []*types.KeyValue{
		{Key: []byte("mavl-coins-bty-" + testMainAddr1), Value: types.Encode(&types.Account{Addr: testMainAddr1, Balance: 300})},
		{Key: []byte("mavl-coins-bty-" + testMainAddr3), Value: types.Encode(&types.Account{Addr: testMainAddr3, Balance: 0})},
	}
	root2, err := mavl.SetKVPair(wdb, &types.StoreSet{StateHash: root1, Height: 2, KV: kvs2}, true, treeCfg)
	if err != nil {
		t.Fatal(err)
	}

	// 注意: mavl 包的全局 maxBlockHeight 会跨测试/跨库泄漏, 导致后续 SetKVPair
	// 可能不写 _..mcmbh.._ 记录; 这里显式写入扫描所需的记录, 保证测试确定性。
	// (真实链上这两条记录由节点在 EnableMavlPrune 时写入)
	if err := wdb.Set([]byte("_..mcmbh.._"), types.Encode(&types.Int64{Data: 2})); err != nil {
		t.Fatal(err)
	}
	if err := wdb.Set([]byte(fmt.Sprintf("_mrhp_%010d", 2)+string(root2)), types.Encode(&types.Int64{Data: 2})); err != nil {
		t.Fatal(err)
	}
}

// genMavlNoPruneDB 构造未开 prune 的纯 mavl 布局, 返回根哈希
func genMavlNoPruneDB(t *testing.T, dir string) []byte {
	t.Helper()
	wdb, err := dbm.NewGoLevelDB("store", dir, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer wdb.Close()

	treeCfg := &mavl.TreeConfig{}
	kvs1 := []*types.KeyValue{
		{Key: []byte("mavl-coins-bty-" + testMainAddr1), Value: types.Encode(&types.Account{Addr: testMainAddr1, Balance: 777})},
	}
	root, err := mavl.SetKVPair(wdb, &types.StoreSet{Height: 1, KV: kvs1}, true, treeCfg)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func checkScan(t *testing.T, res *scanResult, wantMain, wantExec int) {
	t.Helper()
	if len(res.Main) != wantMain {
		t.Fatalf("main accounts = %d, want %d (%+v)", len(res.Main), wantMain, res.Main)
	}
	if len(res.Exec) != wantExec {
		t.Fatalf("exec accounts = %d, want %d (%+v)", len(res.Exec), wantExec, res.Exec)
	}
}

func TestScanMVCCLast(t *testing.T) {
	dir := mkDir(t)
	genMVCCLastDB(t, dir)
	res, err := scanStore(dir, "coins", "bty", "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkScan(t, res, 3, 3)
	m1 := findMain(res, testMainAddr1)
	if m1 == nil || m1.Balance != 500_0000_0000 {
		t.Fatalf("addr1 balance wrong: %+v", m1)
	}
	m2 := findMain(res, testMainAddr2)
	if m2 == nil || m2.Frozen != 100_0000_0000 {
		t.Fatalf("addr2 frozen wrong: %+v", m2)
	}
	e1 := findExec(res, testExecAddr, testMainAddr1)
	if e1 == nil || e1.Balance != 800_0000_0000 || e1.Frozen != 200_0000_0000 {
		t.Fatalf("exec addr1 wrong: %+v", e1)
	}
}

func TestScanMVCCData(t *testing.T) {
	dir := mkDir(t)
	genMVCCDataDB(t, dir)
	res, err := scanStore(dir, "coins", "bty", "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkScan(t, res, 3, 1)
	m1 := findMain(res, testMainAddr1)
	if m1 == nil || m1.Balance != 500 {
		t.Fatalf("addr1 should take max version: %+v", m1)
	}
	e1 := findExec(res, testExecAddr, testMainAddr1)
	if e1 == nil || e1.Balance != 900 || e1.Frozen != 100 {
		t.Fatalf("exec addr1 should take max version: %+v", e1)
	}
}

func TestScanMavlTree(t *testing.T) {
	dir := mkDir(t)
	genMavlDB(t, dir)
	res, err := scanStore(dir, "coins", "bty", "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkScan(t, res, 3, 1)
	m1 := findMain(res, testMainAddr1)
	if m1 == nil || m1.Balance != 300 {
		t.Fatalf("addr1 should be latest version(300): %+v", m1)
	}
}

func TestScanMavlTreeWithStateHash(t *testing.T) {
	dir := mkDir(t)
	root := genMavlNoPruneDB(t, dir)
	if _, err := scanStore(dir, "coins", "bty", "", "auto"); err == nil {
		t.Fatal("expected error for mavl without prune records")
	}
	res, err := scanStore(dir, "coins", "bty", fmt.Sprintf("0x%x", root), "mavl")
	if err != nil {
		t.Fatal(err)
	}
	checkScan(t, res, 1, 0)
	m1 := findMain(res, testMainAddr1)
	if m1 == nil || m1.Balance != 777 {
		t.Fatalf("addr1 balance wrong: %+v", m1)
	}
}

func TestContractDrillDown(t *testing.T) {
	dir := mkDir(t)
	genMVCCLastDB(t, dir)
	res, err := scanStore(dir, "coins", "bty", "", "")
	if err != nil {
		t.Fatal(err)
	}
	// ticket 合约下应有 2 个内部账户, 合计 800+200+300 = 1300 bty
	var n int
	var total int64
	for _, e := range res.Exec {
		if e.ExecAddr == testExecAddr {
			n++
			total += e.Balance + e.Frozen
		}
	}
	if n != 2 || total != 1300_0000_0000 {
		t.Fatalf("ticket drill-down: n=%d total=%d", n, total)
	}
}

func TestContractNameMapping(t *testing.T) {
	em, err := loadExecers("")
	if err != nil {
		t.Fatal(err)
	}
	if name := execName(em, testExecAddr); name != "ticket" {
		t.Fatalf("execName(ticket addr) = %q, want ticket", name)
	}
	// resolveExec: 传名字或地址都行
	addr, err := resolveExec(em, "ticket")
	if err != nil || addr != testExecAddr {
		t.Fatalf("resolveExec(ticket) = %q, %v", addr, err)
	}
	addr2, err := resolveExec(em, testExecAddr)
	if err != nil || addr2 != testExecAddr {
		t.Fatalf("resolveExec(addr) = %q, %v", addr2, err)
	}
}

func findMain(res *scanResult, addr string) *mainAccount {
	for i := range res.Main {
		if res.Main[i].Addr == addr {
			return &res.Main[i]
		}
	}
	return nil
}

func findExec(res *scanResult, execAddr, addr string) *execAccount {
	for i := range res.Exec {
		if res.Exec[i].ExecAddr == execAddr && res.Exec[i].Addr == addr {
			return &res.Exec[i]
		}
	}
	return nil
}
