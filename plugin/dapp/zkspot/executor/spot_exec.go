package executor

import (
	"time"

	"github.com/33cn/chain33/types"
	"github.com/33cn/plugin/plugin/dapp/zkspot/executor/spot"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	exchangetypes "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

/*
 * 实现交易的链上执行接口
 * 关键数据上链（statedb）并生成交易回执（log）
 */

func checkZkSignature() error {
	return types.ErrAccountNotExist
}

// 限价交易
func (e *zkspot) Exec_LimitOrder(payload *exchangetypes.SpotLimitOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	// checkTx will check payload and zk Signature
	start := time.Now()
	action := NewSpotDex(e, tx, index)
	r, err := action.LimitOrder(payload, "")
	if err != nil {
		return r, err
	}
	// 构造 LimitOrder 的结算清单
	list := GetSpotMatch(r)
	end := time.Now()
	elog.Error("zkspot Exec_LimitOrder.LimitOrder", "cost", end.Sub(start))

	action2 := NewAction(e, tx, index)
	r2, err := action2.SpotMatch(payload, list)
	if err != nil {
		return r, err
	}
	end2 := time.Now()
	elog.Error("zkspot Exec_LimitOrder.SpotMatch", "cost", end2.Sub(start))

	return mergeReceipt(r, r2), nil
}

//市价交易
func (e *zkspot) Exec_MarketOrder(payload *exchangetypes.SpotMarketOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	//TODO marketOrder
	return nil, types.ErrActionNotSupport
}

// 撤单
func (e *zkspot) Exec_RevokeOrder(payload *exchangetypes.SpotRevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	txinfo := et.TxInfo{
		Hash:  string(tx.Hash()),
		Index: index,
	}
	spot := spot.NewSpot(&e.DriverBase, &txinfo)
	action := NewSpotDex(e, tx, index)
	return spot.RevokeOrder(action.fromaddr, payload)
}

// 绑定委托交易地址
func (e *zkspot) Exec_ExchangeBind(payload *exchangetypes.SpotExchangeBind, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewSpotDex(e, tx, index)
	return actiondb.ExchangeBind(payload)
}

// 委托交易
func (e *zkspot) Exec_EntrustOrder(payload *exchangetypes.SpotEntrustOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewSpotDex(e, tx, index)
	return action.EntrustOrder(payload)
}

// 委托撤单
func (e *zkspot) Exec_EntrustRevokeOrder(payload *exchangetypes.SpotEntrustRevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewSpotDex(e, tx, index)
	return action.EntrustRevokeOrder(payload)
}
