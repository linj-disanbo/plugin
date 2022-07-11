package spot

import (
	"encoding/hex"
	"fmt"

	"github.com/33cn/chain33/account"
	dbm "github.com/33cn/chain33/common/db"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

type IGetSpotFee interface {
	GetSpotFee(fromaddr string, left, right uint64) (*spotFee, error)
}

type NftSpot struct {
	env      *drivers.DriverBase
	tx       *et.TxInfo
	dbprefix et.DBprefix
	feeAcc2  *DexAccount

	//
	leftAccDb *EvmxgoNftAccountRepo
	accountdb *accountRepo
	orderdb   *orderSRepo
	matcher1  *matcher
	// fee
	ExecAddr string
}

type EvmxgoNftAccountRepo struct {
	cfg     *types.Chain33Config
	statedb dbm.KV
	symbol  string

	accdb *account.DB
}

func newNftAccountRepo(db dbm.KV, cfg *types.Chain33Config) (*EvmxgoNftAccountRepo, error) {
	return &EvmxgoNftAccountRepo{
		statedb: db,
		cfg:     cfg}, nil
}

func (accdb *EvmxgoNftAccountRepo) NewAccount(addr string, accid uint64, nftid uint64) (*NftAccount, error) {
	var err error
	symbol := fmt.Sprintf("%d", nftid)
	if accdb.accdb == nil {
		accdb.accdb, err = account.NewAccountDB(accdb.cfg, "evmxgo", symbol, accdb.statedb)
		if err != nil {
			return nil, err
		}
	}
	return &NftAccount{accdb: accdb, address: addr, accid: accid, nftid: nftid, symbol: symbol}, nil
}

type NftAccount struct {
	accdb   *EvmxgoNftAccountRepo
	address string
	accid   uint64
	nftid   uint64
	symbol  string
}

func NewNftSpot(e *drivers.DriverBase, tx *et.TxInfo, dbprefix et.DBprefix) (*NftSpot, error) {
	leftAccDb, err := newNftAccountRepo(e.GetStateDB(), e.GetAPI().GetConfig())
	if err != nil {
		return nil, err
	}
	spot := &NftSpot{
		env:       e,
		tx:        tx,
		dbprefix:  dbprefix,
		leftAccDb: leftAccDb,
		accountdb: newAccountRepo(spotDexName, e.GetStateDB(), dbprefix),
		orderdb:   newOrderSRepo(e.GetStateDB(), dbprefix),
		matcher1:  newMatcher(e.GetStateDB(), e.GetLocalDB(), e.GetAPI(), dbprefix),
		ExecAddr:  tx.ExecAddr,
	}
	return spot, nil
}

func (a *NftSpot) SetFeeAcc(funcGetFeeAccount GetFeeAccount) error {
	feeAcc, err := funcGetFeeAccount()
	if err != nil {
		return err
	}
	a.feeAcc2 = feeAcc
	return nil
}

func (a *NftSpot) loadOrder(id int64) (*spotOrder, error) {
	order, err := a.orderdb.findOrderBy(id)
	if err != nil {
		return nil, err
	}

	orderx := newSpotOrder(order, a.orderdb)
	return orderx, nil
}

func (a *NftSpot) GetSpotFee(fromaddr string, left, right uint64) (*spotFee, error) {
	var takerFee, makerFee int32
	takerFee, makerFee = 10000, 10000

	return &spotFee{
		addr:  a.feeAcc2.acc.Addr,
		id:    a.feeAcc2.acc.Id,
		taker: takerFee,
		maker: makerFee,
	}, nil
}

// execLocal ...
func (a *NftSpot) ExecLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
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

//GetIndex get index
func (a *NftSpot) GetIndex() int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (a.env.GetHeight()*types.MaxTxsPerBlock + int64(a.tx.Index)) * 1e4
}

func (a *NftSpot) initOrder() func(*et.SpotOrder) *et.SpotOrder {
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

func (a *NftSpot) LoadNftTrader(fromaddr string, accountID uint64) (*NftSpotTraderHelper, error) {
	acc, err := a.accountdb.LoadSpotAccount(fromaddr, accountID)
	if err != nil {
		elog.Error("executor/exchangedb LoadSpotAccount load taker account", "err", err)
		return nil, err
	}

	return &NftSpotTraderHelper{
		acc:      acc,
		cfg:      a.env.GetAPI().GetConfig(),
		accFee:   a.feeAcc2,
		execAddr: a.ExecAddr,
	}, nil
}

func (a *NftSpot) CreateNftOrder(fromaddr string, payload *et.SpotNftOrder, entrustAddr string) (*types.Receipt, error) {
	fees, err := a.GetSpotFee(fromaddr, payload.LeftAsset, payload.RightAsset)
	if err != nil {
		elog.Error("CreateNftOrder getFees", "err", err)
		return nil, err
	}

	order := createNftOrder(payload, et.TyNftOrder2Action, entrustAddr,
		[]orderInit{a.initOrder(), fees.initLimitOrder()})

	order2 := newSpotOrder(order, a.orderdb)

	trader, err := a.leftAccDb.NewAccount(fromaddr, payload.Order.AccountID, payload.LeftAsset)
	if err != nil {
		elog.Error("CreateNftOrder NewAccount", "err", err)
		return nil, err
	}

	amount := payload.Amount
	receipt3, err := trader.accdb.accdb.ExecFrozen(trader.address, a.ExecAddr, amount)
	if err != nil {
		elog.Error("CreateNftOrder ExecFrozen", "err", err)
		return nil, err
	}

	matches := &et.ReceiptSpotMatch{
		Order: order,
		Index: a.GetIndex(),
	}

	receipt1, err := a.NftOrderReceipt(order2, matches)
	if err != nil {
		elog.Error("CreateNftOrder NftOrderReceipt", "err", err)
		return nil, err
	}
	receipt1 = et.MergeReceipt(receipt1, receipt3)

	return receipt1, nil
}

func (a *NftSpot) NftOrderReceipt(order *spotOrder, matches *et.ReceiptSpotMatch) (*types.Receipt, error) {
	kvs := order.repo.GetOrderKvSet(order.order)
	receiptlog := &types.ReceiptLog{Ty: et.TyNftOrderLog, Log: types.Encode(matches)}
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: []*types.ReceiptLog{receiptlog}}
	return receipts, nil
}

func (a *NftSpot) TradeNft(fromaddr string, taker *NftSpotTraderHelper, payload *et.SpotNftTakerOrder, entrustAddr string) (*types.Receipt, error) {
	order2, err := a.orderdb.findNftOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("TradeNft findNftOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if !spotOrder2.isActiveOrder() {
		elog.Error("TradeNft findNftOrderBy", "err", et.ErrOrderID, "orderid", payload.OrderID)
		return nil, et.ErrOrderID
	}

	order, err := a.CreateNftTakerOrder(fromaddr, taker, payload, entrustAddr)
	if err != nil {
		elog.Error("TradeNft CreateNftTakerOrder", "err", et.ErrOrderID, "orderid", payload.OrderID)
		return nil, err
	}
	_ = order

	log, kv, err := taker.Trade(a, spotOrder2)
	if err != nil {
		elog.Error("TradeNft Trade", "err", err, "orderid", payload.OrderID)
		return nil, err
	}
	return &types.Receipt{KV: kv, Logs: log}, nil
}

func (a *NftSpot) CreateNftTakerOrder(fromaddr string, acc *NftSpotTraderHelper, payload *et.SpotNftTakerOrder, entrustAddr string) (*et.SpotOrder, error) {
	order2, err := a.orderdb.findNftOrderBy(payload.OrderID)
	if err != nil {
		elog.Error("CreateNftTakerOrder findOrderBy", "err", err, "orderid", payload.OrderID)
		return nil, err
	}

	spotOrder2 := newSpotOrder(order2, a.orderdb)
	if !spotOrder2.isActiveOrder() {
		return nil, et.ErrOrderID
	}

	fees, err := a.GetSpotFee(fromaddr, order2.GetNftOrder().LeftAsset, order2.GetNftOrder().RightAsset)
	if err != nil {
		elog.Error("CreateNftTakerOrder getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order1 := createNftTakerOrder2(payload, entrustAddr, spotOrder2,
		[]orderInit{a.initOrder(), fees.initLimitOrder()})
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

func createNftTakerOrder2(payload *et.SpotNftTakerOrder, entrustAddr string, order2 *spotOrder, inits []orderInit) *et.SpotOrder {
	or := &et.SpotOrder{
		Value:       &et.SpotOrder_NftTakerOrder{NftTakerOrder: payload},
		Ty:          et.TyNftOrder2Action,
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
