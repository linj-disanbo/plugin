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
	env     *drivers.DriverBase
	tx      *et.TxInfo
	feeAcc  *SpotTrader
	feeAcc2 *DexAccount
}

type GetFeeAccount func() (*DexAccount, error)

func NewSpot(e *drivers.DriverBase, tx *et.TxInfo) (*Spot, error) {
	spot := &Spot{
		env: e,
		tx:  tx,
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

// fee
func (a *Spot) GetFeeAcc() error {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())
	if err != nil {
		elog.Error("executor/spot ParseConfig", "err", err)
		return err
	}
	// Taker/Maker fee may relate to user (fromaddr) level in dex

	feeAcc, err := a.LoadUser(tCfg.GetFeeAddr(), tCfg.GetFeeAddrID())
	if err != nil {
		elog.Error("executor/spot LoadUser", "err", err)
		return err
	}
	a.feeAcc = feeAcc
	return nil
}

func (a *Spot) getFeeRate(fromaddr string, left, right uint32) (int32, int32, error) {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())
	if err != nil {
		elog.Error("executor/exchangedb ParseConfig", "err", err)
		return 0, 0, err
	}
	trade := tCfg.GetTrade(left, right)

	// Taker/Maker fee may relate to user (fromaddr) level in dex
	return trade.Taker, trade.Maker, nil
}

func (a *Spot) LoadFee(trader *SpotTrader, left, right uint32) error {
	t, m, err := a.getFeeRate(trader.acc.acc.Addr, left, right)
	if err != nil {
		return err
	}

	trader.takerFee = t
	trader.makerFee = m
	return nil
}

func (a *Spot) GetFees(fromaddr string, left, right uint32) (*feeDetail, error) {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())

	if err != nil {
		elog.Error("executor/exchangedb ParseConfig", "err", err)
		return nil, err

	}
	trade := tCfg.GetTrade(left, right)

	// Taker/Maker fee may relate to user (fromaddr) level in dex

	return &feeDetail{
		addr:  a.feeAcc2.acc.Addr,
		id:    1, //
		taker: trade.Taker,
		maker: trade.Maker,
	}, nil
}
