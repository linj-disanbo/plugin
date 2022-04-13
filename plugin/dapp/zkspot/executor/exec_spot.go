package executor

import (
	"github.com/33cn/chain33/types"
	exchangetypes "github.com/33cn/plugin/plugin/dapp/exchange/types"
	spot "github.com/33cn/plugin/plugin/dapp/zkspot/executor/spot"
)

/*
 * 实现交易的链上执行接口
 * 关键数据上链（statedb）并生成交易回执（log）
 */

// 限价交易
func (e *zksync) Exec_LimitOrder(payload *exchangetypes.LimitOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := spot.NewAction(e, tx, index)
	return action.LimitOrder(payload, "")
}

//市价交易
func (e *zksync) Exec_MarketOrder(payload *exchangetypes.MarketOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	//TODO marketOrder
	return nil, types.ErrActionNotSupport
}

// 撤单
func (e *zksync) Exec_RevokeOrder(payload *exchangetypes.RevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := spot.NewAction(e, tx, index)
	return action.RevokeOrder(payload)
}

// 绑定委托交易地址
func (e *zksync) Exec_ExchangeBind(payload *exchangetypes.ExchangeBind, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := spot.NewAction(e, tx, index)
	return actiondb.ExchangeBind(payload)
}

// 委托交易
func (e *zksync) Exec_EntrustOrder(payload *exchangetypes.EntrustOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := spot.NewAction(e, tx, index)
	return action.EntrustOrder(payload)
}

// 委托撤单
func (e *zksync) Exec_EntrustRevokeOrder(payload *exchangetypes.EntrustRevokeOrder, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := spot.NewAction(e, tx, index)
	return action.EntrustRevokeOrder(payload)
}
