package spot

import (
	"encoding/hex"

	log "github.com/33cn/chain33/common/log/log15"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

var (
	elog = log.New("module", logName)
)

type Spot struct {
	// block info
	env *drivers.DriverBase
	tx  *et.TxInfo
}

func NewSpot(e *drivers.DriverBase, tx *et.TxInfo) *Spot {
	return &Spot{
		env: e,
		tx:  tx,
	}
}

func (a *Spot) RevokeOrder(fromaddr string, payload *et.SpotRevokeOrder) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	order, err := findOrderByOrderID(a.env.GetStateDB(), a.env.GetLocalDB(), payload.GetOrderID())
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

	price := order.GetLimitOrder().GetPrice()
	balance := order.GetBalance()
	cfg := a.env.GetAPI().GetConfig()

	if order.GetLimitOrder().GetOp() == et.OpBuy {
		accX, err := LoadSpotAccount(order.Addr, uint64(order.GetLimitOrder().RightAsset), a.env.GetStateDB())
		if err != nil {
			return nil, err
		}
		amount := CalcActualCost(et.OpBuy, balance, price, cfg.GetCoinPrecision())
		amount += SafeMul(balance, int64(order.Rate), cfg.GetCoinPrecision())

		receipt, err := accX.Active(order.GetLimitOrder().RightAsset, uint64(amount))
		if err != nil {
			elog.Error("RevokeOrder.ExecActive", "addr", fromaddr, "amount", amount, "err", err.Error())
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kvs = append(kvs, receipt.KV...)
	}
	if order.GetLimitOrder().GetOp() == et.OpSell {
		accX, err := LoadSpotAccount(order.Addr, uint64(order.GetLimitOrder().RightAsset), a.env.GetStateDB())
		if err != nil {
			return nil, err
		}

		receipt, err := accX.Active(order.GetLimitOrder().RightAsset, uint64(balance))
		if err != nil {
			elog.Error("RevokeOrder.ExecActive", "addr", fromaddr, "amount", balance, "err", err.Error())
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kvs = append(kvs, receipt.KV...)
	}

	order.Status = et.Revoked
	order.UpdateTime = a.env.GetBlockTime()
	order.RevokeHash = hex.EncodeToString([]byte(a.tx.Hash))
	kvs = append(kvs, GetOrderKvSet(order)...)
	re := &et.ReceiptSpotMatch{
		Order: order,
		Index: int64(a.tx.Index),
	}
	receiptlog := &types.ReceiptLog{Ty: et.TyRevokeOrderLog, Log: types.Encode(re)}
	logs = append(logs, receiptlog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}
