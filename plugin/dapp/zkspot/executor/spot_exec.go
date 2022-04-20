package executor

import (
	"github.com/33cn/chain33/types"
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
func (e *zkspot) Exec_LimitOrder(payload *exchangetypes.LimitOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	// TODO
	err := checkZkSignature()
	if err != nil {
		return &types.Receipt{}, err
	}
	action := NewSpotAction2(e, tx, index)
	r, err := action.LimitOrder(payload, "")
	if err != nil {
		return r, err
	}
	// 构造 LimitOrder 的结算清单
	list := SampleSpotMatch( /* r *types.Receipt */ )
	action2 := NewAction(e, tx, index)
	r2, err := action2.SpotMatch(payload, &list)
	if err != nil {
		return r, err
	}
	return mergeReceipt(r, r2), nil
}

//市价交易
func (e *exchange) Exec_MarketOrder(payload *exchangetypes.MarketOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	//TODO marketOrder
	return nil, types.ErrActionNotSupport
}

// 撤单
func (e *exchange) Exec_RevokeOrder(payload *exchangetypes.RevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewSpotAction(e, tx, index)
	return action.RevokeOrder(payload)
}

// 绑定委托交易地址
func (e *exchange) Exec_ExchangeBind(payload *exchangetypes.ExchangeBind, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewSpotAction(e, tx, index)
	return actiondb.ExchangeBind(payload)
}

// 委托交易
func (e *exchange) Exec_EntrustOrder(payload *exchangetypes.EntrustOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewSpotAction(e, tx, index)
	return action.EntrustOrder(payload)
}

// 委托撤单
func (e *exchange) Exec_EntrustRevokeOrder(payload *exchangetypes.EntrustRevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewSpotAction(e, tx, index)
	return action.EntrustRevokeOrder(payload)
}
