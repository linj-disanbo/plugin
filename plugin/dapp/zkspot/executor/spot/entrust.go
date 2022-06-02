package spot

import (
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

var (
	exchangeBindKeyPrefix = []byte("ebind-")
)

func getBindKeyPrefix() []byte {
	return exchangeBindKeyPrefix
}

type Entrust struct {
	fromaddr string
	height   int64
	statedb  dbm.KV
	prefix   et.DBprefix
}

func NewEntrust(s string, h int64, d dbm.KV) *Entrust {
	return &Entrust{
		fromaddr: s,
		height:   h,
		statedb:  d,
	}
}

func (a *Entrust) SetDB(d dbm.KV, prefix et.DBprefix) {
	a.statedb = d
	a.prefix = prefix
}

// dummy
func (a *Entrust) LimitOrder(payload *et.SpotLimitOrder, s string) (*types.Receipt, error) {
	return nil, nil
}

func (a *Entrust) RevokeOrder(payload *et.SpotRevokeOrder) (*types.Receipt, error) {
	return nil, nil
}

func (a *Entrust) Bind(payload *et.SpotExchangeBind) (*types.Receipt, error) {
	if a.fromaddr != payload.GetExchangeAddress() {
		return nil, types.ErrFromAddr
	}
	// If the value is null, the binding is unbound. If the value is not null, the address format is verified
	if len(payload.GetEntrustAddress()) > 0 {
		if err := address.CheckAddress(payload.GetEntrustAddress(), a.height); err != nil {
			return nil, err
		}
	}

	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	oldbind := a.getBind(payload.GetExchangeAddress())
	log := getBindLog(payload, oldbind)
	logs = append(logs, log)

	saveBind(a.statedb, payload)
	kv := getBindKV(payload)
	kvs = append(kvs, kv...)

	kvs = append(kvs, kv...)
	receipt := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipt, nil
}

func (a *Entrust) CheckBind(addr string) error {
	entrustAddr, addr := a.fromaddr, addr

	if !a.checkBind(entrustAddr, addr) {
		return et.ErrBindAddr
	}

	return nil
}

func (a *Entrust) checkBind(entrustAddr, addr string) bool {
	return a.getBind(addr) == entrustAddr
}

func (a *Entrust) getBind(addr string) string {
	value, err := a.statedb.Get(bindKey(addr))
	if err != nil || value == nil {
		return ""
	}
	var bind et.SpotExchangeBind
	err = types.Decode(value, &bind)
	if err != nil {
		panic(err)
	}
	return bind.GetEntrustAddress()
}

func bindKey(id string) (key []byte) {
	key = append(key, getBindKeyPrefix()...)
	key = append(key, []byte(id)...)
	return key
}

func getBindLog(payload *et.SpotExchangeBind, old string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = et.TyExchangeBindLog
	r := &et.ReceiptDexBind{}
	r.ExchangeAddress = payload.ExchangeAddress
	r.OldEntrustAddress = old
	r.NewEntrustAddress = payload.EntrustAddress
	log.Log = types.Encode(r)
	return log
}

func saveBind(db dbm.KV, payload *et.SpotExchangeBind) {
	set := getBindKV(payload)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func getBindKV(payload *et.SpotExchangeBind) (kvset []*types.KeyValue) {
	value := types.Encode(payload)
	kvset = append(kvset, &types.KeyValue{Key: bindKey(payload.ExchangeAddress), Value: value})
	return kvset
}
