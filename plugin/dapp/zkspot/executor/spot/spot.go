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
	accountdb *accountRepo
	orderdb   *orderSRepo
	matcher1  *matcher
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

	accX, err := a.accountdb.LoadAccount(order.order.Addr, uint64(token))
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

func (a *Spot) getFeeRate(fromaddr string, left, right uint64) (int32, int32, error) {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())
	if err != nil {
		elog.Error("getFeeRate ParseConfig", "err", err)
		return 0, 0, err
	}
	tradeFee := tCfg.GetTrade(left, right)

	// Taker/Maker fee may relate to user (fromaddr) level in dex
	return tradeFee.Taker, tradeFee.Maker, nil
}

func (a *Spot) GetSpotFee(fromaddr string, left, right uint64) (*spotFee, error) {
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
	acc, err := a.accountdb.LoadSpotAccount(fromaddr, accountID)
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

func (a *Spot) LoadNftTrader(fromaddr string, accountID uint64) (*NftTrader, error) {
	acc, err := a.accountdb.LoadSpotAccount(fromaddr, accountID)
	if err != nil {
		elog.Error("executor/exchangedb LoadSpotAccount load taker account", "err", err)
		return nil, err
	}

	return &NftTrader{
		acc: acc,
		cfg: a.env.GetAPI().GetConfig(),
	}, nil
}

func createNftOrder(payload *et.SpotNftOrder, entrustAddr string, inits []orderInit) *et.SpotOrder {
	or := &et.SpotOrder{
		Value:       &et.SpotOrder_NftOrder{NftOrder: payload},
		Ty:          et.TyLimitOrderAction,
		EntrustAddr: entrustAddr,
		Executed:    0,
		AVGPrice:    0,
		Balance:     payload.GetAmount(),
		Status:      et.Ordered,
	}
	for _, initFun := range inits {
		or = initFun(or)
	}
	return or
}

func (a *Spot) CreateNftOrder(fromaddr string, trader *NftTrader, payload *et.SpotNftOrder, entrustAddr string) (*types.Receipt, error) {
	fees, err := a.GetSpotFee(fromaddr, payload.LeftAsset, payload.RightAsset)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	trader.fee = fees

	order := createNftOrder(payload, entrustAddr,
		[]orderInit{a.initLimitOrder(), fees.initLimitOrder()})
	trader.order = newSpotOrder(order, a.orderdb)

	tid, amount := trader.order.NeedToken(a.env.GetAPI().GetConfig().GetCoinPrecision())
	err = trader.CheckTokenAmountForLimitOrder(tid, amount)
	if err != nil {
		return nil, err
	}
	trader.matches = &et.ReceiptSpotMatch{
		Order: trader.order.order,
		Index: a.GetIndex(),
	}

	receipt1, err := a.NftOrderReceipt(trader)
	if err != nil {
		return nil, err
	}
	receipt3, err := trader.FrozenForNftOrder()
	if err != nil {
		return nil, err
	}
	receipt1 = et.MergeReceipt(receipt1, receipt3)

	return receipt1, nil
}

func (a *Spot) NftOrderReceipt(taker *NftTrader) (*types.Receipt, error) {
	kvs := taker.order.repo.GetOrderKvSet(taker.order.order)
	receiptlog := &types.ReceiptLog{Ty: et.TyNftOrderLog, Log: types.Encode(taker.matches)}
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: []*types.ReceiptLog{receiptlog}}
	return receipts, nil
}

func createNftTakerOrder(payload *et.SpotNftTakerOrder, entrustAddr string, order2 *spotOrder, inits []orderInit) *et.SpotOrder {
	or := &et.SpotOrder{
		Value:       &et.SpotOrder_NftTakerOrder{NftTakerOrder: payload},
		Ty:          et.TyLimitOrderAction,
		EntrustAddr: entrustAddr,
		Executed:    0,
		AVGPrice:    0,
		Balance:     order2.order.Balance,
		Status:      et.Ordered,
	}
	for _, initFun := range inits {
		or = initFun(or)
	}
	return or
}

func (a *Spot) TradeNft(fromaddr string, taker *NftTrader, payload *et.SpotNftTakerOrder, entrustAddr string) (*types.Receipt, error) {
	order2, err := a.orderdb.findOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("CreateNftTakerOrder findOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if spotOrder2.isActiveOrder() {
		return nil, et.ErrOrderID
	}
	maker, err := a.LoadNftTrader(order2.Addr, order2.GetNftOrder().Order.AccountID)
	if err != nil {
		return nil, err
	}
	maker.order = spotOrder2

	order, err := a.CreateNftTakerOrder(fromaddr, taker, payload, entrustAddr)
	if err != nil {
		return nil, err
	}
	_ = order

	log, kv, err := taker.matchModel(maker, order2, a.orderdb.statedb)
	return &types.Receipt{KV: kv, Logs: log}, nil
}

func (a *Spot) CreateNftTakerOrder(fromaddr string, acc *NftTrader, payload *et.SpotNftTakerOrder, entrustAddr string) (*et.SpotOrder, error) {
	order2, err := a.orderdb.findOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("CreateNftTakerOrder findOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if spotOrder2.isActiveOrder() {
		return nil, et.ErrOrderID
	}

	fees, err := a.GetSpotFee(fromaddr, order2.GetNftOrder().LeftAsset, order2.GetNftOrder().RightAsset)
	if err != nil {
		elog.Error("CreateNftTakerOrder getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order1 := createNftTakerOrder(payload, entrustAddr, spotOrder2,
		[]orderInit{a.initLimitOrder(), fees.initLimitOrder()})
	acc.order = newSpotOrder(order1, a.orderdb)

	tid, amount := acc.order.nftTakerOrderNeedToken(spotOrder2, a.env.GetAPI().GetConfig().GetCoinPrecision())
	err = acc.CheckTokenAmountForLimitOrder(tid, amount)
	if err != nil {
		return nil, err
	}
	acc.matches = &et.ReceiptSpotMatch{
		Order: acc.order.order,
		Index: a.GetIndex(),
	}

	return order1, nil
}