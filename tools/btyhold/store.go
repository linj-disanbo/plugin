// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"path/filepath"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var errReadOnly = errors.New("read-only db: write not allowed")

// openStoreRO 以只读方式打开 chain33 store 数据库(目录内含 store.db)。
// 注意: 运行中的节点对数据库持有排它 flock, 因此本工具要么先停节点,
// 要么把整个 datadir 目录拷贝一份再扫。
func openStoreRO(storeDir string) (*leveldb.DB, error) {
	path := filepath.Join(storeDir, "store.db")
	db, err := leveldb.OpenFile(path, &opt.Options{
		ReadOnly:               true,
		OpenFilesCacheCapacity: 16,
		BlockCacheCapacity:     128 * opt.MiB,
		Filter:                 filter.NewBloomFilter(10),
	})
	if err != nil {
		return nil, fmt.Errorf("open %s read-only: %w (请先停止 chain33 节点, 或拷贝一份 datadir 再扫描)", path, err)
	}
	return db, nil
}

// hasPrefix 判断 DB 中是否存在指定前缀的键
func hasPrefix(db *leveldb.DB, prefix []byte) bool {
	it := db.NewIterator(util.BytesPrefix(prefix), nil)
	defer it.Release()
	return it.Next()
}

// roDB 把只读 goleveldb 包装成 chain33 的 dbm.DB 接口(供 mavl 树读取)。
type roDB struct {
	db *leveldb.DB
}

var _ dbm.DB = (*roDB)(nil)

func (r *roDB) Get(key []byte) ([]byte, error) {
	val, err := r.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return val, nil
}
func (r *roDB) Set(key, value []byte) error        { return errReadOnly }
func (r *roDB) SetSync(key, value []byte) error    { return errReadOnly }
func (r *roDB) Delete(key []byte) error            { return errReadOnly }
func (r *roDB) DeleteSync(key []byte) error        { return errReadOnly }
func (r *roDB) Begin()                             {}
func (r *roDB) Commit() error                      { return errReadOnly }
func (r *roDB) Rollback()                          {}
func (r *roDB) Close()                             { _ = r.db.Close() }
func (r *roDB) NewBatch(sync bool) dbm.Batch       { return &roBatch{} }
func (r *roDB) BeginTx() (dbm.TxKV, error)         { return nil, errReadOnly }
func (r *roDB) CompactRange(start, limit []byte) error { return errReadOnly }
func (r *roDB) Print()                             {}
func (r *roDB) Stats() map[string]string           { return nil }
func (r *roDB) SetCacheSize(size int)              {}
func (r *roDB) GetCache() *lru.ARCCache            { return nil }
func (r *roDB) Iterator(start, end []byte, reverse bool) dbm.Iterator {
	it := r.db.NewIterator(&util.Range{Start: start, Limit: end}, nil)
	return &roIter{Iterator: it, reverse: reverse}
}

type roBatch struct{}

func (b *roBatch) Set(key, value []byte)        {}
func (b *roBatch) Delete(key []byte)            {}
func (b *roBatch) Write() error                 { return errReadOnly }
func (b *roBatch) ValueSize() int               { return 0 }
func (b *roBatch) ValueLen() int                { return 0 }
func (b *roBatch) Reset()                       {}
func (b *roBatch) UpdateWriteSync(sync bool)    {}

type roIter struct {
	iterator.Iterator
	reverse bool
}

func (it *roIter) Rewind() bool     { it.Iterator.First(); return it.Iterator.Valid() }
func (it *roIter) Seek(key []byte) bool { return it.Iterator.Seek(key) }
func (it *roIter) Next() bool       { it.Iterator.Next(); return it.Iterator.Valid() }
func (it *roIter) Valid() bool      { return it.Iterator.Valid() }
func (it *roIter) Key() []byte      { return it.Iterator.Key() }
func (it *roIter) Value() []byte    { return it.Iterator.Value() }
func (it *roIter) ValueCopy() []byte {
	return append([]byte(nil), it.Iterator.Value()...)
}
func (it *roIter) Error() error     { return it.Iterator.Error() }
func (it *roIter) Prefix() []byte   { return nil }
func (it *roIter) IsReverse() bool  { return it.reverse }
func (it *roIter) Close()           { it.Iterator.Release() }

// incPrefix 返回把 prefix 最后一个字节 +1 的字节串(用于生成 [start, end) 的 end 边界)
func incPrefix(prefix []byte) []byte {
	out := append([]byte(nil), prefix...)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i] < 0xff {
			out[i]++
			return out[:i+1]
		}
	}
	return nil
}
