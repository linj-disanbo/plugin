// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kvmvccmavl kvmvcc+mavl接口
package kvmvccmavl

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/store"
	"github.com/33cn/chain33/types"
	"github.com/hashicorp/golang-lru"
)

var (
	kmlog = log.New("module", "kvmvccMavl")
	// ErrStateHashLost ...
	ErrStateHashLost        = errors.New("ErrStateHashLost")
	kvmvccMavlFork    int64 = 200 * 10000
	isDelMavlData           = false
	delMavlDataHeight       = kvmvccMavlFork + 10000
	delMavlDataState  int32
	wg                sync.WaitGroup
	quit              bool
)

const (
	canceSize         = 2048 //可以缓存2048个roothash, height对
	batchDataSize     = 1024 * 1024 * 1
	delMavlStateStart = 1
	delMavlStateEnd   = 0
	mvccPrefix        = ".-mvcc-." //同common/db中的mvccPrefix一致
)

// SetLogLevel set log level
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog disable log output
func DisableLog() {
	kmlog.SetHandler(log.DiscardHandler())
}

func init() {
	drivers.Reg("kvmvccMavl", New)
}

// KVmMavlStore provide kvmvcc and mavl store interface implementation
type KVmMavlStore struct {
	*drivers.BaseStore
	*KVMVCCStore
	*MavlStore
	cance *lru.Cache
}

type subKVMVCCConfig struct {
	EnableMVCCIter bool `json:"enableMVCCIter"`
}

type subMavlConfig struct {
	EnableMavlPrefix bool  `json:"enableMavlPrefix"`
	EnableMVCC       bool  `json:"enableMVCC"`
	EnableMavlPrune  bool  `json:"enableMavlPrune"`
	PruneHeight      int32 `json:"pruneHeight"`
}

type subConfig struct {
	EnableMVCCIter   bool  `json:"enableMVCCIter"`
	EnableMavlPrefix bool  `json:"enableMavlPrefix"`
	EnableMVCC       bool  `json:"enableMVCC"`
	EnableMavlPrune  bool  `json:"enableMavlPrune"`
	PruneHeight      int32 `json:"pruneHeight"`
}

// New construct KVMVCCStore module
func New(cfg *types.Store, sub []byte) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	var kvms *KVmMavlStore
	var subcfg subConfig
	var subKVMVCCcfg subKVMVCCConfig
	var subMavlcfg subMavlConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
		subKVMVCCcfg.EnableMVCCIter = subcfg.EnableMVCCIter
		subMavlcfg.EnableMavlPrefix = subcfg.EnableMavlPrefix
		subMavlcfg.EnableMVCC = subcfg.EnableMVCC
		subMavlcfg.EnableMavlPrune = subcfg.EnableMavlPrune
		subMavlcfg.PruneHeight = subcfg.PruneHeight
	}
	cance, err := lru.New(canceSize)
	if err != nil {
		panic("new KVmMavlStore fail")
	}

	kvms = &KVmMavlStore{bs, NewKVMVCC(&subKVMVCCcfg, bs.GetDB()),
		NewMavl(&subMavlcfg, bs.GetDB()), cance}
	// 查询是否已经删除mavl
	_, err = bs.GetDB().Get(genDelMavlKey(mvccPrefix))
	if err == nil {
		isDelMavlData = true
	}
	bs.SetChild(kvms)
	return kvms
}

// Close the KVmMavlStore module
func (kvmMavls *KVmMavlStore) Close() {
	quit = true
	wg.Wait()
	kvmMavls.BaseStore.Close()
	kvmMavls.KVMVCCStore.Close()
	kvmMavls.MavlStore.Close()
	kmlog.Info("store kvmMavls closed")
}

// Set kvs with statehash to KVmMavlStore
func (kvmMavls *KVmMavlStore) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	if datas.Height < kvmvccMavlFork {
		hash, err := kvmMavls.MavlStore.Set(datas, sync)
		if err != nil {
			return hash, err
		}
		_, err = kvmMavls.KVMVCCStore.Set(datas, hash, sync)
		if err != nil {
			return hash, err
		}
		if err == nil {
			kvmMavls.cance.Add(string(hash), datas.Height)
		}
		return hash, err
	}
	// 仅仅做kvmvcc
	hash, err := kvmMavls.KVMVCCStore.Set(datas, nil, sync)
	if err == nil {
		kvmMavls.cance.Add(string(hash), datas.Height)
	}
	// 删除Mavl数据
	if datas.Height > delMavlDataHeight && !isDelMavlData && !isDelMavling() {
		wg.Add(1)
		go DelMavl(kvmMavls.GetDB())
	}
	return hash, err
}

// Get kvs with statehash from KVmMavlStore
func (kvmMavls *KVmMavlStore) Get(datas *types.StoreGet) [][]byte {
	if value, ok := kvmMavls.cance.Get(string(datas.StateHash)); ok {
		if value.(int64) < kvmvccMavlFork {
			return kvmMavls.MavlStore.Get(datas)
		}
		return kvmMavls.KVMVCCStore.Get(datas)
	}
	return kvmMavls.KVMVCCStore.Get(datas)
}

// MemSet set kvs to the mem of KVmMavlStore module and return the StateHash
func (kvmMavls *KVmMavlStore) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	if datas.Height < kvmvccMavlFork {
		hash, err := kvmMavls.MavlStore.MemSet(datas, sync)
		if err != nil {
			return hash, err
		}
		_, err = kvmMavls.KVMVCCStore.MemSet(datas, hash, sync)
		if err != nil {
			return hash, err
		}
		if err == nil {
			kvmMavls.cance.Add(string(hash), datas.Height)
		}
		return hash, err
	}
	// 仅仅做kvmvcc
	hash, err := kvmMavls.KVMVCCStore.MemSet(datas, nil, sync)
	if err == nil {
		kvmMavls.cance.Add(string(hash), datas.Height)
	}
	// 删除Mavl数据
	if datas.Height > delMavlDataHeight && !isDelMavlData && !isDelMavling() {
		wg.Add(1)
		go DelMavl(kvmMavls.GetDB())
	}
	return hash, err
}

// Commit kvs in the mem of KVmMavlStore module to state db and return the StateHash
func (kvmMavls *KVmMavlStore) Commit(req *types.ReqHash) ([]byte, error) {
	if value, ok := kvmMavls.cance.Get(string(req.Hash)); ok {
		if value.(int64) < kvmvccMavlFork {
			hash, err := kvmMavls.MavlStore.Commit(req)
			if err != nil {
				return hash, err
			}
			_, err = kvmMavls.KVMVCCStore.Commit(req)
			return hash, err
		}
		return kvmMavls.KVMVCCStore.Commit(req)
	}
	return kvmMavls.KVMVCCStore.Commit(req)
}

// Rollback kvs in the mem of KVmMavlStore module and return the StateHash
func (kvmMavls *KVmMavlStore) Rollback(req *types.ReqHash) ([]byte, error) {
	if value, ok := kvmMavls.cance.Get(string(req.Hash)); ok {
		if value.(int64) < kvmvccMavlFork {
			hash, err := kvmMavls.MavlStore.Rollback(req)
			if err != nil {
				return hash, err
			}
			_, err = kvmMavls.KVMVCCStore.Rollback(req)
			return hash, err
		}
		return kvmMavls.KVMVCCStore.Rollback(req)
	}
	return kvmMavls.KVMVCCStore.Rollback(req)
}

// IterateRangeByStateHash travel with Prefix by StateHash  to get the latest version kvs.
func (kvmMavls *KVmMavlStore) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	if value, ok := kvmMavls.cance.Get(string(statehash)); ok {
		if value.(int64) < kvmvccMavlFork {
			kvmMavls.MavlStore.IterateRangeByStateHash(statehash, start, end, ascending, fn)
			return
		}
		kvmMavls.KVMVCCStore.IterateRangeByStateHash(statehash, start, end, ascending, fn)
		return
	}
	kvmMavls.KVMVCCStore.IterateRangeByStateHash(statehash, start, end, ascending, fn)
}

// ProcEvent handles supported events
func (kvmMavls *KVmMavlStore) ProcEvent(msg queue.Message) {
	msg.ReplyErr("KVmMavlStore", types.ErrActionNotSupport)
}

// Del set kvs to nil with StateHash
func (kvmMavls *KVmMavlStore) Del(req *types.StoreDel) ([]byte, error) {
	if req.Height < kvmvccMavlFork {
		hash, err := kvmMavls.MavlStore.Del(req)
		if err != nil {
			return hash, err
		}
		_, err = kvmMavls.KVMVCCStore.Del(req)
		if err != nil {
			return hash, err
		}
		if err == nil {
			kvmMavls.cance.Remove(string(req.StateHash))
		}
		return hash, err
	}
	// 仅仅做kvmvcc
	hash, err := kvmMavls.KVMVCCStore.Del(req)
	if err == nil {
		kvmMavls.cance.Remove(string(req.StateHash))
	}
	return hash, err
}

// DelMavl 数据库中mavl的清除
// 达到kvmvccMavlFork + 100000 后触发清除
func DelMavl(db dbm.DB) {
	defer wg.Done()
	setDelMavl(delMavlStateStart)
	defer setDelMavl(delMavlStateEnd)
	isDel := delMavlData(db)
	if isDel {
		isDelMavlData = true
		kmlog.Info("DelMavl success")
	}
}

func delMavlData(db dbm.DB) bool {
	it := db.Iterator(nil, nil, true)
	batch := db.NewBatch(true)
	for it.Rewind(); it.Valid(); it.Next() {
		if quit {
			return false
		}
		if !bytes.HasPrefix(it.Key(), []byte(mvccPrefix)) { // 将非mvcc的mavl数据全部删除
			batch.Delete(it.Key())
			if batch.ValueSize() > batchDataSize {
				batch.Write()
				batch.Reset()
			}
		}
	}
	batch.Set(genDelMavlKey(mvccPrefix), []byte(""))
	batch.Write()
	return true
}

func genDelMavlKey(prefix string) []byte {
	delMavl := "--delMavlData--"
	return []byte(fmt.Sprintf("%s%s", prefix, delMavl))
}

func isDelMavling() bool {
	return atomic.LoadInt32(&delMavlDataState) == 1
}

func setDelMavl(state int32) {
	atomic.StoreInt32(&delMavlDataState, state)
}
