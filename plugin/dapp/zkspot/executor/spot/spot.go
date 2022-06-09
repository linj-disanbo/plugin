package spot

import (
	"encoding/hex"
	"fmt"

	"github.com/33cn/chain33/common/db/table"
	log "github.com/33cn/chain33/common/log/log15"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

var (
	elog = log.New("module", logName)
)

type Spot struct {
	env      *drivers.DriverBase
	tx       *et.TxInfo
	dbprefix et.DBprefix
	feeAcc   *SpotTrader
	feeAcc2  *DexAccount
}

type GetFeeAccount func() (*DexAccount, error)

func NewSpot(e *drivers.DriverBase, tx *et.TxInfo, dbprefix et.DBprefix) (*Spot, error) {
	spot := &Spot{
		env:      e,
		tx:       tx,
		dbprefix: dbprefix,
	}
	return spot, nil
}

func (a *Spot) SetFeeAcc(funcGetFeeAccount GetFeeAccount) error {
	feeAcc, err := funcGetFeeAccount()
	if err != nil {
		return err
	}
	a.feeAcc2 = feeAcc
	return nil
}

func (a *Spot) RevokeOrder(fromaddr string, payload *et.SpotRevokeOrder) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	orderdb := newOrderSRepo(a.env.GetStateDB(), a.dbprefix)
	order, err := orderdb.findOrderBy(payload.GetOrderID())
	if err != nil {
		return nil, err
	}
	if order.Addr != fromaddr {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return nil, et.ErrAddr
	}
	if order.Status == et.Completed || order.Status == et.Revoked {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return nil, et.ErrOrderSatus
	}

	cfg := a.env.GetAPI().GetConfig()
	token, amount := orderFrozenToken(order, cfg.GetCoinPrecision())

	accX, err := LoadSpotAccount(order.Addr, uint64(token), a.env.GetStateDB())
	receipt, err := accX.Active(token, uint64(amount))
	if err != nil {
		elog.Error("RevokeOrder.ExecActive", "addr", fromaddr, "amount", amount, "err", err.Error())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kvs = append(kvs, receipt.KV...)

	order.Status = et.Revoked
	order.UpdateTime = a.env.GetBlockTime()
	order.RevokeHash = hex.EncodeToString([]byte(a.tx.Hash))
	kvs = append(kvs, orderdb.GetOrderKvSet(order)...)
	re := &et.ReceiptSpotMatch{
		Order: order,
		Index: int64(a.tx.Index),
	}
	receiptlog := &types.ReceiptLog{Ty: et.TyRevokeOrderLog, Log: types.Encode(re)}
	logs = append(logs, receiptlog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

func (a *Spot) getFeeRate(fromaddr string, left, right uint32) (int32, int32, error) {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())
	if err != nil {
		elog.Error("getFeeRate ParseConfig", "err", err)
		return 0, 0, err
	}
	tradeFee := tCfg.GetTrade(left, right)

	// Taker/Maker fee may relate to user (fromaddr) level in dex
	return tradeFee.Taker, tradeFee.Maker, nil
}

func (a *Spot) GetSpotFee(fromaddr string, left, right uint32) (*spotFee, error) {
	takerFee, makerFee, err := a.getFeeRate(fromaddr, left, right)
	if err != nil {
		return nil, err
	}
	return &spotFee{
		addr:  a.feeAcc2.acc.Addr,
		id:    a.feeAcc2.acc.Id,
		taker: takerFee,
		maker: makerFee,
	}, nil
}

type spotFee struct {
	addr  string
	id    uint64
	taker int32
	maker int32
}

func (f *spotFee) initLimitOrder() func(*et.SpotOrder) *et.SpotOrder {
	return func(order *et.SpotOrder) *et.SpotOrder {
		order.Rate = f.maker
		order.TakerRate = f.taker
		return order
	}
}

// execLocal ...
func (a *Spot) ExecLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	historyTable := NewHistoryOrderTable(a.env.GetLocalDB(), a.dbprefix)
	marketTable := NewMarketDepthTable(a.env.GetLocalDB(), a.dbprefix)
	orderTable := NewMarketOrderTable(a.env.GetLocalDB(), a.dbprefix)
	if receiptData.Ty == types.ExecOk {
		for _, log := range receiptData.Logs {
			switch log.Ty {
			case et.TyMarketOrderLog, et.TyRevokeOrderLog, et.TyLimitOrderLog:
				receipt := &et.ReceiptSpotMatch{}
				if err := types.Decode(log.Log, receipt); err != nil {
					elog.Error("updateIndex", "log.type.decode", err)
					return nil, err
				}
				a.updateIndex(marketTable, orderTable, historyTable, receipt)
			}
		}
	}

	var kvs []*types.KeyValue
	kv, err := marketTable.Save()
	if err != nil {
		elog.Error("updateIndex", "marketTable.Save", err.Error())
		return nil, err
	}
	kvs = append(kvs, kv...)

	kv, err = orderTable.Save()
	if err != nil {
		elog.Error("updateIndex", "orderTable.Save", err.Error())
		return nil, err
	}
	kvs = append(kvs, kv...)

	kv, err = historyTable.Save()
	if err != nil {
		elog.Error("updateIndex", "historyTable.Save", err.Error())
		return nil, err
	}
	kvs = append(kvs, kv...)
	dbSet.KV = append(dbSet.KV, kvs...)
	return dbSet, nil
}

func (a *Spot) updateIndex(marketTable, orderTable, historyTable *table.Table, receipt *et.ReceiptSpotMatch) (kvs []*types.KeyValue) {
	elog.Info("updateIndex", "order.status", receipt.Order.Status)
	switch receipt.Order.Status {
	case et.Ordered:
		err := a.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
		err = a.updateMatchOrders(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetMatchOrders(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	case et.Completed:
		err := a.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
		err = a.updateMatchOrders(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetMatchOrders(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	case et.Revoked:
		err := a.updateOrder(marketTable, orderTable, historyTable, receipt.GetOrder(), receipt.GetIndex())
		if err != nil {
			return nil
		}
	}

	return
}

func (a *Spot) updateOrder(marketTable, orderTable, historyTable *table.Table, order *et.SpotOrder, index int64) error {
	left := order.GetLimitOrder().GetLeftAsset()
	right := order.GetLimitOrder().GetRightAsset()
	op := order.GetLimitOrder().GetOp()
	price := order.GetLimitOrder().GetPrice()
	switch order.Status {
	case et.Ordered:
		var markDepth et.SpotMarketDepth
		depth, err := getMarketDepth(marketTable, left, right, op, price)
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

	case et.Completed:
		err := historyTable.Replace(order)
		if err != nil {
			elog.Error("updateIndex", "historyTable.Replace", err.Error())
		}
	case et.Revoked:
		var marketDepth et.SpotMarketDepth
		depth, err := getMarketDepth(marketTable, left, right, op, price)
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
		order.Status = et.Revoked
		order.Index = index
		err = historyTable.Replace(order)
		if err != nil {
			elog.Error("updateIndex", "historyTable.Replace", err.Error())
		}
	}
	return nil
}
func (a *Spot) updateMatchOrders(marketTable, orderTable, historyTable *table.Table, order *et.SpotOrder, matchOrders []*et.SpotOrder, index int64) error {
	left := order.GetLimitOrder().GetLeftAsset()
	right := order.GetLimitOrder().GetRightAsset()
	op := order.GetLimitOrder().GetOp()
	if len(matchOrders) > 0 {
		cache := make(map[int64]int64)
		for i, matchOrder := range matchOrders {
			if matchOrder.Balance == 0 && matchOrder.Executed == 0 {
				var matchDepth et.SpotMarketDepth
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
			if matchOrder.Status == et.Completed {
				err := orderTable.DelRow(matchOrder)
				if err != nil {
					elog.Error("updateIndex", "orderTable.DelRow", err.Error())
				}
				matchOrder.Index = index + int64(i+1)
				err = historyTable.Replace(matchOrder)
				if err != nil {
					elog.Error("updateIndex", "historyTable.Replace", err.Error())
				}
			} else if matchOrder.Status == et.Ordered {
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
			var matchDepth et.SpotMarketDepth
			depth, err := getMarketDepth(marketTable, left, right, OpSwap(op), pr)
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
