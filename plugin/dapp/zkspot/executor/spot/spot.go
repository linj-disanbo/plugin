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
	env      *drivers.DriverBase
	tx       *et.TxInfo
	dbprefix et.DBprefix
	feeAcc   *SpotTrader
	feeAcc2  *DexAccount

	//
	orderdb  *orderSRepo
	matcher1 *matcher
	// fee
}

type GetFeeAccount func() (*DexAccount, error)

func NewSpot(e *drivers.DriverBase, tx *et.TxInfo, dbprefix et.DBprefix) (*Spot, error) {
	spot := &Spot{
		env:      e,
		tx:       tx,
		dbprefix: dbprefix,
		orderdb:  newOrderSRepo(e.GetStateDB(), dbprefix),
		matcher1: newMatcher(e.GetStateDB(), e.GetLocalDB(), e.GetAPI(), dbprefix),
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

func (a *Spot) loadOrder(id int64) (*spotOrder, error) {
	order, err := a.orderdb.findOrderBy(id)
	if err != nil {
		return nil, err
	}

	orderx := newSpotOrder(order, a.orderdb)
	return orderx, nil
}

func (a *Spot) MatchLimitOrder(payload *et.SpotLimitOrder, taker *SpotTrader) (*types.Receipt, error) {
	matcher1 := newMatcher(a.env.GetStateDB(), a.env.GetLocalDB(), a.env.GetAPI(), a.dbprefix)
	elog.Info("LimitOrder", "height", a.env.GetHeight(), "order-price", payload.GetPrice(), "op", OpSwap(payload.Op), "index", taker.order.order.GetOrderID())
	receipt1, err := matcher1.MatchLimitOrder(payload, taker, a.orderdb)
	if err != nil {
		return nil, err
	}

	if taker.order.isActiveOrder() {
		receipt3, err := taker.FrozenForLimitOrder(taker.order)
		if err != nil {
			return nil, err
		}
		receipt1 = et.MergeReceipt(receipt1, receipt3)
	}

	return receipt1, nil
}

func (a *Spot) RevokeOrder(fromaddr string, payload *et.SpotRevokeOrder) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	order, err := a.loadOrder(payload.GetOrderID())
	if err != nil {
		return nil, err
	}

	err = order.checkRevoke(fromaddr)
	if err != nil {
		return nil, err
	}

	cfg := a.env.GetAPI().GetConfig()
	token, amount := order.calcFrozenToken(cfg.GetCoinPrecision())

	accX, err := LoadSpotAccount(order.order.Addr, uint64(token), a.env.GetStateDB())
	receipt, err := accX.Active(token, uint64(amount))
	if err != nil {
		elog.Error("RevokeOrder.ExecActive", "addr", fromaddr, "amount", amount, "err", err.Error())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kvs = append(kvs, receipt.KV...)

	r1, err := order.Revoke(a.env.GetBlockTime(), a.tx.Hash, a.tx.Index)
	if err != nil {
		elog.Error("RevokeOrder.Revoke", "addr", fromaddr, "amount", amount, "err", err.Error())
		return nil, err
	}

	kvs = append(kvs, r1.KV...)
	logs = append(logs, r1.Logs...)
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
				updateIndex(marketTable, orderTable, historyTable, receipt)
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

func (a *Spot) LoadUser(fromaddr string, accountID uint64) (*SpotTrader, error) {
	acc, err := LoadSpotAccount(fromaddr, accountID, a.env.GetStateDB())
	if err != nil {
		elog.Error("executor/exchangedb LoadSpotAccount load taker account", "err", err)
		return nil, err
	}

	return &SpotTrader{
		acc: acc,
		cfg: a.env.GetAPI().GetConfig(),
	}, nil
}

func (a *Spot) CreateLimitOrder(fromaddr string, acc *SpotTrader, payload *et.SpotLimitOrder, entrustAddr string) (*et.SpotOrder, error) {
	fees, err := a.GetSpotFee(fromaddr, payload.LeftAsset, payload.RightAsset)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order := createLimitOrder(payload, entrustAddr,
		[]orderInit{a.initLimitOrder(), fees.initLimitOrder()})
	acc.order = newSpotOrder(order, a.orderdb)

	tid, amount := acc.order.NeedToken(a.env.GetAPI().GetConfig().GetCoinPrecision())
	err = acc.CheckTokenAmountForLimitOrder(tid, amount)
	if err != nil {
		return nil, err
	}
	acc.matches = &et.ReceiptSpotMatch{
		Order: acc.order.order,
		Index: a.GetIndex(),
	}

	return order, nil
}

//GetIndex get index
func (a *Spot) GetIndex() int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (a.env.GetHeight()*types.MaxTxsPerBlock + int64(a.tx.Index)) * 1e4
}

func (a *Spot) initLimitOrder() func(*et.SpotOrder) *et.SpotOrder {
	return func(order *et.SpotOrder) *et.SpotOrder {
		order.OrderID = a.GetIndex()
		order.Index = a.GetIndex()
		order.CreateTime = a.env.GetBlockTime()
		order.UpdateTime = a.env.GetBlockTime()
		order.Hash = hex.EncodeToString(a.tx.Hash)
		order.Addr = a.tx.From
		return order
	}
}
