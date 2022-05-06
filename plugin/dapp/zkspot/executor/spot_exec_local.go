package executor

import (
	"fmt"

	"github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/types"
	ety "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

/*
* Local execution of transaction related data, data is not on the chain
* Non-critical data, local storage (localDB), used for auxiliary query, high efficiency
 */

func (e *zkspot) ExecLocal_LimitOrder(payload *ety.SpotLimitOrder, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return e.interExecLocal(tx, receiptData, index)
}

func (e *zkspot) ExecLocal_MarketOrder(payload *ety.SpotMarketOrder, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return e.interExecLocal(tx, receiptData, index)
}

func (e *zkspot) ExecLocal_RevokeOrder(payload *ety.SpotRevokeOrder, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return e.interExecLocal(tx, receiptData, index)
}

func (e *zkspot) ExecLocal_EntrustOrder(payload *ety.SpotLimitOrder, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return e.interExecLocal(tx, receiptData, index)
}

func (e *zkspot) ExecLocal_EntrustRevokeOrder(payload *ety.SpotMarketOrder, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return e.interExecLocal(tx, receiptData, index)
}

func (e *zkspot) interExecLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	historyTable := NewHistoryOrderTable(e.GetLocalDB())
	marketTable := NewMarketDepthTable(e.GetLocalDB())
	orderTable := NewMarketOrderTable(e.GetLocalDB())
	if receiptData.Ty == types.ExecOk {
		for _, log := range receiptData.Logs {
			switch log.Ty {
			case ety.TyMarketOrderLog, ety.TyRevokeOrderLog, ety.TyLimitOrderLog:
				receipt := &ety.ReceiptSpotMatch{}
				if err := types.Decode(log.Log, receipt); err != nil {
					elog.Error("updateIndex", "log.type.decode", err)
					return nil, err
				}
				e.updateIndex(marketTable, orderTable, historyTable, receipt)
			}
		}
	}

	var kvs []*types.KeyValue
	kv, err := marketTable.Save()
	if err != nil {
		elog.Error("updateIndex", "marketTable.Save", err.Error())
		return nil, nil
	}
	kvs = append(kvs, kv...)

	kv, err = orderTable.Save()
	if err != nil {
		elog.Error("updateIndex", "orderTable.Save", err.Error())
		return nil, nil
	}
	kvs = append(kvs, kv...)

	kv, err = historyTable.Save()
	if err != nil {
		elog.Error("updateIndex", "historyTable.Save", err.Error())
		return nil, nil
	}
	kvs = append(kvs, kv...)
	dbSet.KV = append(dbSet.KV, kvs...)
	dbSet = e.addAutoRollBack(tx, dbSet.KV)
	localDB := e.GetLocalDB()
	for _, kv1 := range dbSet.KV {
		//elog.Info("updateIndex", "localDB.Set", string(kv1.Key))
		err := localDB.Set(kv1.Key, kv1.Value)
		if err != nil {
			elog.Error("updateIndex", "localDB.Set", err.Error())
			return dbSet, err
		}
	}
	return dbSet, nil
}

// Set automatic rollback
func (e *zkspot) addAutoRollBack(tx *types.Transaction, kv []*types.KeyValue) *types.LocalDBSet {
	dbSet := &types.LocalDBSet{}
	dbSet.KV = e.AddRollbackKV(tx, tx.Execer, kv)
	return dbSet
}

func (e *zkspot) updateIndex(marketTable, orderTable, historyTable *table.Table, receipt *ety.ReceiptSpotMatch) (kvs []*types.KeyValue) {
	elog.Info("updateIndex", "order.status", receipt.Order.Status)
	switch receipt.Order.Status {
	case ety.Ordered:
		err := e.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
		err = e.updateMatchOrders(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetMatchOrders(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	case ety.Completed:
		err := e.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
		err = e.updateMatchOrders(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetMatchOrders(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	case ety.Revoked:
		err := e.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	}

	return
}

func (e *zkspot) updateOrder(marketTable, orderTable, historyTable *table.Table, order *ety.SpotOrder, index int64) error {
	left := order.GetLimitOrder().GetLeftAsset()
	right := order.GetLimitOrder().GetRightAsset()
	op := order.GetLimitOrder().GetOp()
	price := order.GetLimitOrder().GetPrice()
	switch order.Status {
	case ety.Ordered:
		var markDepth ety.SpotMarketDepth
		depth, err := queryMarketDepth(marketTable, left, right, op, price)
		if err == types.ErrNotFound {
			markDepth.Price = price
			markDepth.LeftAsset = left
			markDepth.RightAsset = right
			markDepth.Op = op
			markDepth.Amount = order.Balance
		} else {
			markDepth.Price = price
			markDepth.LeftAsset = left
			markDepth.RightAsset = right
			markDepth.Op = op
			markDepth.Amount = depth.Amount + order.Balance
		}
		err = marketTable.Replace(&markDepth)
		if err != nil {
			elog.Error("updateIndex", "marketTable.Replace", err.Error())
		}
		err = orderTable.Replace(order)
		if err != nil {
			elog.Error("updateIndex", "orderTable.Replace", err.Error())
		}

	case ety.Completed:
		err := historyTable.Replace(order)
		if err != nil {
			elog.Error("updateIndex", "historyTable.Replace", err.Error())
		}
	case ety.Revoked:
		var marketDepth ety.SpotMarketDepth
		depth, err := queryMarketDepth(marketTable, left, right, op, price)
		if err == nil {
			marketDepth.Price = price
			marketDepth.LeftAsset = left
			marketDepth.RightAsset = right
			marketDepth.Op = op
			marketDepth.Amount = depth.Amount - order.Balance

			if marketDepth.Amount > 0 {
				err = marketTable.Replace(&marketDepth)
				if err != nil {
					elog.Error("updateIndex", "marketTable.Replace", err.Error())
				}
			}
			if marketDepth.Amount <= 0 {
				err = marketTable.DelRow(&marketDepth)
				if err != nil {
					elog.Error("updateIndex", "marketTable.DelRow", err.Error())
				}
			}
		}

		primaryKey := []byte(fmt.Sprintf("%022d", order.OrderID))
		err = orderTable.Del(primaryKey)
		if err != nil {
			elog.Error("updateIndex", "orderTable.Del", err.Error())
		}
		order.Status = ety.Revoked
		order.Index = index
		err = historyTable.Replace(order)
		if err != nil {
			elog.Error("updateIndex", "historyTable.Replace", err.Error())
		}
	}
	return nil
}
func (e *zkspot) updateMatchOrders(marketTable, orderTable, historyTable *table.Table, order *ety.SpotOrder, matchOrders []*ety.SpotOrder, index int64) error {
	left := order.GetLimitOrder().GetLeftAsset()
	right := order.GetLimitOrder().GetRightAsset()
	op := order.GetLimitOrder().GetOp()
	if len(matchOrders) > 0 {
		cache := make(map[int64]int64)
		for i, matchOrder := range matchOrders {
			if matchOrder.Balance == 0 && matchOrder.Executed == 0 {
				var matchDepth ety.SpotMarketDepth
				matchDepth.Price = matchOrder.AVGPrice
				matchDepth.LeftAsset = left
				matchDepth.RightAsset = right
				matchDepth.Op = OpSwap(op)
				matchDepth.Amount = 0
				err := marketTable.DelRow(&matchDepth)
				if err != nil && err != types.ErrNotFound {
					elog.Error("updateIndex", "marketTable.DelRow", err.Error())
				}
				continue
			}
			if matchOrder.Status == ety.Completed {
				err := orderTable.DelRow(matchOrder)
				if err != nil {
					elog.Error("updateIndex", "orderTable.DelRow", err.Error())
				}
				matchOrder.Index = index + int64(i+1)
				err = historyTable.Replace(matchOrder)
				if err != nil {
					elog.Error("updateIndex", "historyTable.Replace", err.Error())
				}
			} else if matchOrder.Status == ety.Ordered {
				err := orderTable.Replace(matchOrder)
				if err != nil {
					elog.Error("updateIndex", "orderTable.Replace", err.Error())
				}
			}
			executed := cache[matchOrder.GetLimitOrder().Price]
			executed = executed + matchOrder.Executed
			cache[matchOrder.GetLimitOrder().Price] = executed
		}

		for pr, executed := range cache {
			var matchDepth ety.SpotMarketDepth
			depth, err := queryMarketDepth(marketTable, left, right, OpSwap(op), pr)
			if err != nil {
				continue
			} else {
				matchDepth.Price = pr
				matchDepth.LeftAsset = left
				matchDepth.RightAsset = right
				matchDepth.Op = OpSwap(op)
				matchDepth.Amount = depth.Amount - executed
			}
			if matchDepth.Amount > 0 {
				err = marketTable.Replace(&matchDepth)
				if err != nil {
					elog.Error("updateIndex", "marketTable.Replace", err.Error())
				}
			}
			if matchDepth.Amount <= 0 {
				err = marketTable.DelRow(&matchDepth)
				if err != nil {
					elog.Error("updateIndex", "marketTable.DelRow", err.Error())
				}
			}
		}
	}
	return nil
}

//OpSwap ...
func OpSwap(op int32) int32 {
	if op == ety.OpBuy {
		return ety.OpSell
	}
	return ety.OpBuy
}
