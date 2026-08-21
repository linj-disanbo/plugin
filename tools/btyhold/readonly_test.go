// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// snapshotDir 记录目录下所有文件的 (相对路径, 大小, mtime, sha256)
func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		out[rel] = fmt.Sprintf("size=%d mtime=%d sha256=%x", info.Size(), info.ModTime().UnixNano(), h.Sum(nil))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertDirUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("DB 目录文件数量变化: before=%d after=%d\nbefore=%v\nafter=%v", len(before), len(after), before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("DB 文件被修改: %s\n before: %s\n after:  %s", k, v, after[k])
		}
	}
}

// 三种布局的库, 扫描前后文件必须完全不变(证明工具只读)
func TestScanDoesNotModifyDB(t *testing.T) {
	type gen func(*testing.T, string)
	cases := []struct {
		name string
		g    gen
	}{
		{"mvcc-last", genMVCCLastDB},
		{"mvcc-data", genMVCCDataDB},
		{"mavl-tree", genMavlDB},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := mkDir(t)
			c.g(t, dir)
			before := snapshotDir(t, dir)

			res, err := scanStore(dir, "coins", "bty", "", "")
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Main) == 0 && len(res.Exec) == 0 {
				t.Fatal("scan 结果为空, 测试无效")
			}
			// 再走一次 contracts 报告路径, 确认也没有写
			if err := reportContracts(res, nil, 0, ""); err != nil {
				t.Fatal(err)
			}
			after := snapshotDir(t, dir)
			assertDirUnchanged(t, before, after)
		})
	}
}

// roDB 的写方法必须全部返回 errReadOnly
func TestRODBRejectsWrites(t *testing.T) {
	dir := mkDir(t)
	genMVCCLastDB(t, dir)
	db, err := openStoreRO(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ro := &roDB{db: db}

	k := []byte("mavl-coins-bty-test")
	if err := ro.Set(k, []byte("x")); err != errReadOnly {
		t.Fatalf("Set 应返回 errReadOnly, got %v", err)
	}
	if err := ro.SetSync(k, []byte("x")); err != errReadOnly {
		t.Fatalf("SetSync 应返回 errReadOnly, got %v", err)
	}
	if err := ro.Delete(k); err != errReadOnly {
		t.Fatalf("Delete 应返回 errReadOnly, got %v", err)
	}
	if err := ro.DeleteSync(k); err != errReadOnly {
		t.Fatalf("DeleteSync 应返回 errReadOnly, got %v", err)
	}
	if err := ro.Commit(); err != errReadOnly {
		t.Fatalf("Commit 应返回 errReadOnly, got %v", err)
	}
	if _, err := ro.BeginTx(); err != errReadOnly {
		t.Fatalf("BeginTx 应返回 errReadOnly, got %v", err)
	}
	if err := ro.CompactRange(k, nil); err != errReadOnly {
		t.Fatalf("CompactRange 应返回 errReadOnly, got %v", err)
	}
	if err := ro.NewBatch(true).Write(); err != errReadOnly {
		t.Fatalf("batch.Write 应返回 errReadOnly, got %v", err)
	}
}

// 打开方式必须是 ReadOnly
func TestOpenIsReadOnly(t *testing.T) {
	dir := mkDir(t)
	genMVCCLastDB(t, dir)
	db, err := openStoreRO(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// 读取能正常工作
	val, err := db.Get([]byte(".-mvcc-.l.mavl-coins-bty-"+testMainAddr1), nil)
	if err != nil {
		t.Fatalf("read should work: %v", err)
	}
	if len(val) == 0 {
		t.Fatal("read returned empty value")
	}
}

