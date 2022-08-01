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
	feeInfo  *SpotFee
	feeAccX  AssetAccount

	accountdb *accountRepos
	orderdb   *orderSRepo
	matcher1  *matcher
}

type GetFeeAccount func() (*SpotFee, error)

func NewSpot(e *drivers.DriverBase, tx *et.TxInfo, dbprefix et.DBprefix) (*Spot, error) {
	accRepos, err := newAccountRepo11(spotDexName, e.GetStateDB(), dbprefix, e.GetAPI().GetConfig(), "TODO")
	if err != nil {
		return nil, err
	}
	spot := &Spot{
		env:       e,
		tx:        tx,
		dbprefix:  dbprefix,
		accountdb: accRepos,
		orderdb:   newOrderSRepo(e.GetStateDB(), dbprefix),
		matcher1:  newMatcher(e.GetStateDB(), e.GetLocalDB(), e.GetAPI(), dbprefix),
	}
	return spot, nil
}

func (a *Spot) SetFeeAcc(funcGetFeeAccount GetFeeAccount) error {
	fee, err := funcGetFeeAccount()
	if err != nil {
		return err
	}
	acc, err := LoadSpotAccount(fee.Address, fee.AccID, a.env.GetStateDB(), a.dbprefix)
	if err != nil {
		elog.Error("LoadSpotAccount load taker account", "err", err)
		return err
	}
	a.feeAccX = &ZkAccount{acc: acc, asset: nil}
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
	receipt1, err := matcher1.MatchOrder(taker.order, taker, a.orderdb, a)
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

func (a *Spot) MatchAssetLimitOrder(payload1 *et.AssetLimitOrder, taker *SpotTrader) (*types.Receipt, error) {
	matcher1 := newMatcher(a.env.GetStateDB(), a.env.GetLocalDB(), a.env.GetAPI(), a.dbprefix)
	elog.Info("LimitOrder", "height", a.env.GetHeight(), "order-price", taker.order.GetPrice(), "op", OpSwap(taker.order.GetOp()), "index", taker.order.order.GetOrderID())
	receipt1, err := matcher1.MatchOrder(taker.order, taker, a.orderdb, a)
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

	_, right := order.GetAsset()

	// cfg := a.env.GetAPI().GetConfig()
	// cfg.GetCoinPrecision()
	token, amount := order.calcFrozenToken(GetCoinPrecision(int32(right.Ty)))

	accX, err := a.accountdb.LoadAccount(order.order.Addr, 1, token) // TODO
	if err != nil {
		elog.Error("RevokeOrder.LoadAccount", "addr", fromaddr, "amount", amount, "err", err.Error())
		return nil, err
	}
	receipt, err := accX.UnFrozen(int64(amount))
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

func BuySellAsset(op int32, left, right *et.Asset) (*et.Asset, *et.Asset) {
	buyAsset, sellAsset := left, right
	if op == et.OpSell {
		buyAsset, sellAsset = right, left
	}
	return buyAsset, sellAsset
}

func (a *Spot) LoadNewUser(fromaddr string, zkAccID uint64, payload *et.AssetLimitOrder) (*SpotTrader, error) {
	buyAsset, sellAsset := BuySellAsset(payload.Op, payload.LeftAsset, payload.RightAsset)
	return a.LoadTrader(fromaddr, zkAccID, buyAsset, sellAsset)
}

func (a *Spot) LoadTrader(fromaddr string, zkAccID uint64, buyAsset, sellAsset *et.Asset) (*SpotTrader, error) {
	accs, err := a.accountdb.LoadAccounts(fromaddr, zkAccID, buyAsset, sellAsset)
	if err != nil {
		return nil, err
	}

	return &SpotTrader{
		cfg:     a.env.GetAPI().GetConfig(),
		accFeeX: a.feeAccX,
		accX:    accs,
		AccID:   zkAccID,
	}, nil
}

func (a *Spot) CreateLimitOrder(fromaddr string, acc *SpotTrader, payload *et.SpotLimitOrder, entrustAddr string) (*et.SpotOrder, error) {
	left, right := NewZkAsset(payload.LeftAsset), NewZkAsset(payload.RightAsset)
	or := createLimitOrder(payload)
	fees, err := a.GetSpotFee(fromaddr, left, right)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order := createOrder(or, entrustAddr,
		[]orderInit{a.initOrder(), fees.initOrder()})
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

func (a *Spot) initOrder() func(*et.SpotOrder) *et.SpotOrder {
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

func (a *Spot) CreateNftOrder(fromaddr string, trader *SpotTrader, payload *et.SpotNftOrder, entrustAddr string) (*types.Receipt, error) {
	left, right := NewZkAsset(payload.LeftAsset), NewZkAsset(payload.RightAsset)
	fees, err := a.GetSpotFee(fromaddr, left, right)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	trader.fee = fees

	order := createNftOrder(payload, et.TyNftOrderAction) // , entrustAddr, 	[]orderInit{a.initOrder(), fees.initOrder()})
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

	receipt1, err := a.NftOrderReceipt(nil) // TODO
	if err != nil {
		return nil, err
	}
	receipt3, err := trader.FrozenForLimitOrder(trader.order)
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

func (a *Spot) TradeNft(fromaddr string, payload *et.SpotNftTakerOrder, entrustAddr string) (*types.Receipt, error) {
	order2, err := a.orderdb.findOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("CreateNftTakerOrder findOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if spotOrder2.isActiveOrder() {
		return nil, et.ErrOrderID
	}
	left, right := spotOrder2.GetAsset()
	maker, err := a.LoadTrader(order2.Addr, order2.GetNftOrder().Order.AccountID, right, left)
	if err != nil {
		return nil, err
	}
	maker.order = spotOrder2
	makerX := spotMaker{*maker}

	taker, err := a.LoadTrader(a.tx.From, payload.Order.AccountID, left, right)
	if err != nil {
		return nil, err
	}

	order, err := a.CreateNftTakerOrder(fromaddr, taker, payload, entrustAddr)
	if err != nil {
		return nil, err
	}
	_ = order

	logs, kvs, err := taker.Trade(&makerX)
	return &types.Receipt{KV: kvs, Logs: logs}, err
}

func (a *Spot) CreateNftTakerOrder(fromaddr string, acc *SpotTrader, payload *et.SpotNftTakerOrder, entrustAddr string) (*et.SpotOrder, error) {
	order2, err := a.orderdb.findOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("CreateNftTakerOrder findOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if spotOrder2.isActiveOrder() {
		return nil, et.ErrOrderID
	}

	left, right := NewZkAsset(order2.GetNftOrder().LeftAsset), NewZkAsset(order2.GetNftOrder().RightAsset)
	fees, err := a.GetSpotFee(fromaddr, left, right)
	if err != nil {
		elog.Error("CreateNftTakerOrder getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order1 := createNftTakerOrder(payload, spotOrder2) // ,	[]orderInit{a.initOrder(), fees.initOrder()})
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

func (a *Spot) CreateAssetLimitOrder(fromaddr string, acc *SpotTrader, payload *et.AssetLimitOrder, entrustAddr string) (*et.SpotOrder, error) {
	or := createAssetLimitOrder(payload)
	return a.CreateOrder(fromaddr, acc, or, payload.LeftAsset, payload.RightAsset, entrustAddr)
}

func (a *Spot) CreateOrder(fromaddr string, acc *SpotTrader,
	or *et.SpotOrder, left, right *et.Asset, entrustAddr string) (*et.SpotOrder, error) {

	fees, err := a.GetSpotFee(fromaddr, left, right)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order := createOrder(or, entrustAddr,
		[]orderInit{a.initOrder(), fees.initOrder()})
	acc.order = newSpotOrder(order, a.orderdb)

	_, amount := acc.order.NeedToken(acc.accX.sellAcc.GetCoinPrecision())
	err = acc.accX.sellAcc.CheckBalance(amount)
	if err != nil {
		return nil, err
	}
	acc.matches = &et.ReceiptSpotMatch{
		Order: acc.order.order,
		Index: a.GetIndex(),
	}

	return order, nil
}

func (a *Spot) getFeeRateFromCfg(fromaddr string, left, right string) (int32, int32, error) {
	tCfg, err := ParseConfig(a.env.GetAPI().GetConfig(), a.env.GetHeight())
	if err != nil {
		elog.Error("getFeeRate ParseConfig", "err", err)
		return 0, 0, err
	}
	tradeFee := tCfg.GetTrade(left, right)

	// Taker/Maker fee may relate to user (fromaddr) level in dex
	return tradeFee.Taker, tradeFee.Maker, nil
}

func (a *Spot) GetSpotFee(fromaddr string, left, right *et.Asset) (*SpotFee, error) {
	l, r := SymbolStr(left), SymbolStr(right)
	takerFee, makerFee, err := a.getFeeRateFromCfg(fromaddr, l, r)
	if err != nil {
		return nil, err
	}

	return &SpotFee{
		Address: a.feeInfo.Address,
		AccID:   a.feeInfo.AccID,
		taker:   takerFee,
		maker:   makerFee,
	}, nil
}

// Spot Fee Account
// Fee Rate
// Fee Rate for trader(User)
type SpotFee struct {
	Address string
	AccID   uint64
	taker   int32
	maker   int32
}

func (f *SpotFee) initOrder() func(*et.SpotOrder) *et.SpotOrder {
	return func(order *et.SpotOrder) *et.SpotOrder {
		order.Rate = f.maker
		order.TakerRate = f.taker
		return order
	}
}
