package executor

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"

	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// Action action struct
type SpotAction struct {
	statedb   dbm.KV
	txhash    []byte
	fromaddr  string
	toaddr    string
	blocktime int64
	height    int64
	execaddr  string
	localDB   dbm.KVDB
	index     int
	api       client.QueueProtocolAPI
}

//NewAction ...
func NewSpotAction(e *exchange, tx *types.Transaction, index int) *SpotAction {
	hash := tx.Hash()
	fromaddr := tx.From()
	toaddr := tx.GetTo()
	return &SpotAction{
		statedb:   e.GetStateDB(),
		txhash:    hash,
		fromaddr:  fromaddr,
		toaddr:    toaddr,
		blocktime: e.GetBlockTime(),
		height:    e.GetHeight(),
		execaddr:  dapp.ExecAddress(string(tx.Execer)),
		localDB:   e.GetLocalDB(),
		index:     index,
		api:       e.GetAPI(),
	}
}

//NewAction ...
func NewSpotAction2(e *zkspot, tx *types.Transaction, index int) *SpotAction {
	hash := tx.Hash()
	fromaddr := tx.From()
	toaddr := tx.GetTo()
	return &SpotAction{
		statedb:   e.GetStateDB(),
		txhash:    hash,
		fromaddr:  fromaddr,
		toaddr:    toaddr,
		blocktime: e.GetBlockTime(),
		height:    e.GetHeight(),
		execaddr:  dapp.ExecAddress(string(tx.Execer)),
		localDB:   e.GetLocalDB(),
		index:     index,
		api:       e.GetAPI(),
	}
}

//GetIndex get index
func (a *SpotAction) GetIndex() int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (a.height*types.MaxTxsPerBlock + int64(a.index)) * 1e4
}

//GetKVSet get kv set
func (a *SpotAction) GetKVSet(order *et.Order) (kvset []*types.KeyValue) {
	return GetOrderKvSet(order)
}

func GetOrderKvSet(order *et.Order) (kvset []*types.KeyValue) {
	kvset = append(kvset, &types.KeyValue{Key: calcOrderKey(order.OrderID), Value: types.Encode(order)})
	return kvset
}

//OpSwap reverse
func (a *SpotAction) OpSwap(op int32) int32 {
	if op == et.OpBuy {
		return et.OpSell
	}
	return et.OpBuy
}

//CalcActualCost Calculate actual cost
func CalcActualCost(op int32, amount int64, price, coinPrecision int64) int64 {
	if op == et.OpBuy {
		return SafeMul(amount, price, coinPrecision)
	}
	return amount
}

//CheckPrice price  1<=price<=1e16
func CheckPrice(price int64) bool {
	if price > 1e16 || price < 1 {
		return false
	}
	return true
}

//CheckOp ...
func CheckOp(op int32) bool {
	if op == et.OpBuy || op == et.OpSell {
		return true
	}
	return false
}

//CheckCount ...
func CheckCount(count int32) bool {
	return count <= 20 && count >= 0
}

//CheckAmount 最小交易 1coin
func CheckAmount(amount, coinPrecision int64) bool {
	if amount < 1 || amount >= types.MaxCoin*coinPrecision {
		return false
	}
	return true
}

//CheckDirection ...
func CheckDirection(direction int32) bool {
	if direction == et.ListASC || direction == et.ListDESC {
		return true
	}
	return false
}

//CheckStatus ...
func CheckStatus(status int32) bool {
	if status == et.Ordered || status == et.Completed || status == et.Revoked {
		return true
	}
	return false
}

//CheckExchangeAsset
func CheckExchangeAsset(coinExec string, left, right uint32) bool {
	if left == right {
		return false
	}
	return true
}

//  千分之一的手续费  实际数值是  1e8 * 0.1% = 1e5
// 4 / 100000
func getFeeRate(acc *dexAccount) uint64 {
	return 1e5
}

//LimitOrder ...
func (a *SpotAction) LimitOrder(payload *et.LimitOrder, entrustAddr string) (*types.Receipt, error) {
	leftAsset := payload.GetLeftAsset()
	rightAsset := payload.GetRightAsset()
	cfg := a.api.GetConfig()
	if !CheckExchangeAsset(cfg.GetCoinExec(), leftAsset, rightAsset) {
		return nil, et.ErrAsset
	}
	if !CheckAmount(payload.GetAmount(), cfg.GetCoinPrecision()) {
		return nil, et.ErrAssetAmount
	}
	if !CheckPrice(payload.GetPrice()) {
		return nil, et.ErrAssetPrice
	}
	if !CheckOp(payload.GetOp()) {
		return nil, et.ErrAssetOp
	}

	acc, err := LoadSpotAccount(a.fromaddr, payload.Order.AccountID, a.statedb)
	if err != nil {
		return nil, err
	}

	//Check your account balance first
	if payload.GetOp() == et.OpBuy {
		amount := SafeMul(payload.GetAmount(), payload.GetPrice(), cfg.GetCoinPrecision())
		fee := calcMtfFee(amount, int32(getFeeRate(acc)))
		total := SafeAdd(amount, int64(fee))
		err = acc.Frozen(payload.RightAsset, uint64(total))
		if err != nil {
			elog.Error("limit check right balance", "addr", a.fromaddr, "avail", acc.acc.Balance, "need", amount)
			return nil, et.ErrAssetBalance
		}
		return a.matchLimitOrder(payload, acc, acc, entrustAddr)
	}
	if payload.GetOp() == et.OpSell {
		amount := payload.GetAmount()
		err = acc.Frozen(payload.LeftAsset, payload.Order.GetAmount())
		if err != nil {
			elog.Error("limit check left balance", "addr", a.fromaddr, "avail", acc.acc.Balance, "need", amount)
			return nil, et.ErrAssetBalance
		}
		return a.matchLimitOrder(payload, acc, acc, entrustAddr)
	}
	return nil, fmt.Errorf("unknow op")
}

//RevokeOrder ...
func (a *SpotAction) RevokeOrder(payload *et.RevokeOrder) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	order, err := findOrderByOrderID(a.statedb, a.localDB, payload.GetOrderID())
	if err != nil {
		return nil, err
	}
	if order.Addr != a.fromaddr {
		elog.Error("RevokeOrder.OrderCheck", "addr", a.fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return nil, et.ErrAddr
	}
	if order.Status == et.Completed || order.Status == et.Revoked {
		elog.Error("RevokeOrder.OrderCheck", "addr", a.fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return nil, et.ErrOrderSatus
	}

	price := order.GetLimitOrder().GetPrice()
	balance := order.GetBalance()
	cfg := a.api.GetConfig()

	if order.GetLimitOrder().GetOp() == et.OpBuy {
		// TODO id
		accX, err := LoadSpotAccount(order.Addr, uint64(order.GetLimitOrder().RightAsset), a.statedb)
		if err != nil {
			return nil, err
		}
		amount := CalcActualCost(et.OpBuy, balance, price, cfg.GetCoinPrecision())
		amount += order.FeeBalance

		/*receipt,*/
		err = accX.Active(order.GetLimitOrder().RightAsset, uint64(amount))
		if err != nil {
			elog.Error("RevokeOrder.ExecActive", "addr", a.fromaddr, "amount", amount, "err", err.Error())
			return nil, err
		}
		//logs = append(logs, receipt.Logs...)
		//kvs = append(kvs, receipt.KV...)
	}
	if order.GetLimitOrder().GetOp() == et.OpSell {
		// TODO id
		accX, err := LoadSpotAccount(order.Addr, uint64(order.GetLimitOrder().RightAsset), a.statedb)
		if err != nil {
			return nil, err
		}

		/*receipt,*/
		err = accX.Active(order.GetLimitOrder().RightAsset, uint64(balance))
		if err != nil {
			elog.Error("RevokeOrder.ExecActive", "addr", a.fromaddr, "amount", balance, "err", err.Error())
			return nil, err
		}
		//logs = append(logs, receipt.Logs...)
		//kvs = append(kvs, receipt.KV...)
	}

	order.Status = et.Revoked
	order.UpdateTime = a.blocktime
	order.RevokeHash = hex.EncodeToString(a.txhash)
	kvs = append(kvs, a.GetKVSet(order)...)
	re := &et.ReceiptExchange{
		Order: order,
		Index: a.GetIndex(),
	}
	receiptlog := &types.ReceiptLog{Ty: et.TyRevokeOrderLog, Log: types.Encode(re)}
	logs = append(logs, receiptlog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

// set the transaction logic method
// rules:
//1. The purchase price is higher than the market price, and the price is matched from low to high.
//2. Sell orders are matched at prices lower than market prices.
//3. Match the same prices on a first-in, first-out basis
func (a *SpotAction) matchLimitOrder(payload *et.LimitOrder, leftAccountDB, rightAccountDB *dexAccount, entrustAddr string) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var priceKey string
	var count int

	//cfg := a.api.GetConfig()
	tCfg, err := ParseConfig(a.api.GetConfig(), a.height)
	if err != nil {
		elog.Error("executor/exchangedb matchLimitOrder.ParseConfig", "err", err)
		return nil, err
	}
	trade := tCfg.GetTrade(payload)

	or := &et.Order{
		OrderID:     a.GetIndex(),
		Value:       &et.Order_LimitOrder{LimitOrder: payload},
		Ty:          et.TyLimitOrderAction,
		Executed:    0,
		AVGPrice:    0,
		Balance:     payload.GetAmount(),
		Status:      et.Ordered,
		EntrustAddr: entrustAddr,
		Addr:        a.fromaddr,
		UpdateTime:  a.blocktime,
		Index:       a.GetIndex(),
		Rate:        trade.GetMaker(),
		MinFee:      trade.GetMinFee(),
		Hash:        hex.EncodeToString(a.txhash),
		CreateTime:  a.blocktime,
	}
	re := &et.ReceiptExchange{
		Order: or,
		Index: a.GetIndex(),
	}

	// A single transaction can match up to 100 historical orders, the maximum depth can be matched, the system has to protect itself
	// Iteration has listing price
	var done bool
	for {
		if count >= et.MaxMatchCount {
			break
		}
		if done {
			break
		}
		//Obtain price information of existing market listing
		marketDepthList, _ := QueryMarketDepth(a.localDB, payload.GetLeftAsset(), payload.GetRightAsset(), a.OpSwap(payload.Op), priceKey, et.Count)
		if marketDepthList == nil || len(marketDepthList.List) == 0 {
			break
		}
		for _, marketDepth := range marketDepthList.List {
			elog.Info("LimitOrder debug find depth", "height", a.height, "amount", marketDepth.Amount, "price", marketDepth.Price, "order-price", payload.GetPrice(), "op", a.OpSwap(payload.Op), "index", a.GetIndex())
			if count >= et.MaxMatchCount {
				done = true
				break
			}
			if payload.Op == et.OpBuy && marketDepth.Price > payload.GetPrice() {
				done = true
				break
			}
			if payload.Op == et.OpSell && marketDepth.Price < payload.GetPrice() {
				done = true
				break
			}

			var hasOrder = false
			var orderKey string
			for {
				if count >= et.MaxMatchCount {
					done = true
					break
				}
				orderList, err := findOrderIDListByPrice(a.localDB, payload.GetLeftAsset(), payload.GetRightAsset(), marketDepth.Price, a.OpSwap(payload.Op), et.ListASC, orderKey)
				if orderList != nil && !hasOrder {
					hasOrder = true
				}
				if err != nil {
					if err == types.ErrNotFound {
						break
					}
					elog.Error("findOrderIDListByPrice error", "height", a.height, "symbol", payload.GetLeftAsset(), "price", marketDepth.Price, "op", a.OpSwap(payload.Op), "error", err)
					return nil, err
				}
				// got orderlist to trade
				for _, matchorder := range orderList.List {
					if count >= et.MaxMatchCount {
						done = true
						break
					}
					// Check the order status
					order, err := findOrderByOrderID(a.statedb, a.localDB, matchorder.GetOrderID())
					if err != nil || order.Status != et.Ordered {
						if len(orderList.List) == 1 {
							hasOrder = true
						}
						continue
					}
					log, kv, err := a.matchModel2(leftAccountDB, rightAccountDB, payload, order, or, re, tCfg.GetFeeAddr(), trade.GetTaker()) // payload, or redundant
					if err != nil {
						if err == types.ErrNoBalance {
							elog.Warn("matchModel RevokeOrder", "height", a.height, "orderID", order.GetOrderID(), "payloadID", or.GetOrderID(), "error", err)
							continue
						}
						return nil, err
					}
					logs = append(logs, log...)
					kvs = append(kvs, kv...)
					if or.Status == et.Completed {
						receiptlog := &types.ReceiptLog{Ty: et.TyLimitOrderLog, Log: types.Encode(re)}
						logs = append(logs, receiptlog)
						receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
						return receipts, nil
					}
					// match depth count
					count = count + 1
				}
				if orderList.PrimaryKey == "" {
					break
				}
				orderKey = orderList.PrimaryKey
			}
			if !hasOrder {
				var matchorder et.Order
				matchorder.UpdateTime = a.blocktime
				matchorder.Status = et.Completed
				matchorder.Balance = 0
				matchorder.Executed = 0
				matchorder.AVGPrice = marketDepth.Price
				elog.Info("make empty match to del depth", "height", a.height, "price", marketDepth.Price, "amount", marketDepth.Amount)
				re.MatchOrders = append(re.MatchOrders, &matchorder)
			}
		}

		if marketDepthList.PrimaryKey == "" {
			break
		}
		priceKey = marketDepthList.PrimaryKey
	}

	kvs = append(kvs, a.GetKVSet(or)...)
	re.Order = or
	receiptlog := &types.ReceiptLog{Ty: et.TyLimitOrderLog, Log: types.Encode(re)}
	logs = append(logs, receiptlog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

/*
func (a *SpotAction) matchModel(leftAccountDB, rightAccountDB *dexAccount, payload *et.LimitOrder, matchorder *et.Order, or *et.Order, re *et.ReceiptExchange, feeAddr string, taker int32) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var matched int64

	if matchorder.GetBalance() >= or.GetBalance() {
		matched = or.GetBalance()
	} else {
		matched = matchorder.GetBalance()
	}

	elog.Info("try match", "activeId", or.OrderID, "passiveId", matchorder.OrderID, "activeAddr", or.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", payload.Price)

	cfg := a.api.GetConfig()
	var receipt *types.Receipt
	var err error

	accMatch, err := LoadSpotAccount(matchorder.Addr, matchorder.GetLimitOrder().Order.AccountID, a.statedb)
	if err != nil {
		return nil, nil, err
	}
	if payload.Op == et.OpSell {
		//Transfer of frozen assets
		amount := CalcActualCost(matchorder.GetLimitOrder().Op, matched, matchorder.GetLimitOrder().Price, cfg.GetCoinPrecision())
		if matchorder.Addr != a.fromaddr {
			/ *receipt,* / err = rightAccountDB.FrozenTranfer(accMatch, matchorder.GetLimitOrder().LeftAsset, uint64(amount))
		} else {
			/ *receipt,* / err = rightAccountDB.Active(matchorder.GetLimitOrder().LeftAsset, uint64(amount))
		}
		if err != nil {
			elog.Error("matchModel.ExecTransferFrozen", "from", matchorder.Addr, "to", a.fromaddr, "amount", amount, "err", err)
			return nil, nil, err
		}
		logs = append(logs, receipt.Logs...)
		kvs = append(kvs, receipt.KV...)

		// TODO if taker and maker is the same
		// not take seller fee
		//Charge fee
		activeFee := calcMtfFee(amount, taker) //Transaction fee of the active party
		if activeFee != 0 {
			receipt, err = rightAccountDB.FrozenTranfer(feeAddr, activeFee)
			if err != nil {
				elog.Error("matchModel.ExecTransfer sell", "from", a.fromaddr, "to", feeAddr,
					"amount", amount, "rate", taker, "activeFee", activeFee, "err", err.Error())
				return nil, nil, err
			}
			or.DigestedFee += activeFee
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		//The settlement of the corresponding assets for the transaction to be concluded
		amount = CalcActualCost(payload.Op, matched, matchorder.GetLimitOrder().Price, cfg.GetCoinPrecision())
		if a.fromaddr != matchorder.Addr {
			receipt, err = leftAccountDB.ExecTransfer(a.fromaddr, matchorder.Addr, a.execaddr, amount)
			if err != nil {
				elog.Error("matchModel.ExecTransfer", "from", a.fromaddr, "to", matchorder.Addr, "amount", amount, "err", err.Error())
				return nil, nil, err
			}
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		//Charge fee
		passiveFee := calcMtfFee(amount, matchorder.GetRate()) //Passive transaction fees
		if passiveFee != 0 {
			receipt, err = leftAccountDB.ExecTransfer(matchorder.Addr, feeAddr, a.execaddr, passiveFee)
			if err != nil {
				elog.Error("matchModel.ExecTransfer sell", "from", matchorder.Addr, "to", feeAddr,
					"amount", amount, "rate", matchorder.GetRate(), "passiveFee", passiveFee, "err", err.Error())
				return nil, nil, err
			}
			matchorder.DigestedFee += passiveFee
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		or.AVGPrice = caclAVGPrice(or, matchorder.GetLimitOrder().Price, matched)
		//Calculate the average transaction price
		matchorder.AVGPrice = caclAVGPrice(matchorder, matchorder.GetLimitOrder().Price, matched)
	}
	if payload.Op == et.OpBuy {
		amount := CalcActualCost(matchorder.GetLimitOrder().Op, matched, matchorder.GetLimitOrder().Price, cfg.GetCoinPrecision())
		if a.fromaddr != matchorder.Addr {
			receipt, err = leftAccountDB.ExecTransferFrozen(matchorder.Addr, a.fromaddr, a.execaddr, amount)
		} else {
			receipt, err = leftAccountDB.ExecActive(a.fromaddr, a.execaddr, amount)
		}
		if err != nil {
			elog.Error("matchModel.ExecTransferFrozen2", "from", matchorder.Addr, "to", a.fromaddr, "amount", amount, "err", err.Error())
			return nil, nil, err
		}
		logs = append(logs, receipt.Logs...)
		kvs = append(kvs, receipt.KV...)

		activeFee := calcMtfFee(amount, taker)
		if activeFee != 0 {
			receipt, err = leftAccountDB.ExecTransfer(a.fromaddr, feeAddr, a.execaddr, activeFee)
			if err != nil {
				elog.Error("matchModel.ExecTransfer buy", "from", a.fromaddr, "to", feeAddr,
					"amount", amount, "rate", taker, "activeFee", activeFee, "err", err.Error())
				return nil, nil, err
			}
			or.DigestedFee += activeFee
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		amount = CalcActualCost(payload.Op, matched, matchorder.GetLimitOrder().Price, cfg.GetCoinPrecision())
		if a.fromaddr != matchorder.Addr {
			receipt, err = rightAccountDB.ExecTransfer(a.fromaddr, matchorder.Addr, a.execaddr, amount)
			if err != nil {
				elog.Error("matchModel.ExecTransfer2", "from", a.fromaddr, "to", matchorder.Addr, "amount", amount, "err", err.Error())
				return nil, nil, err
			}
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		passiveFee := calcMtfFee(amount, matchorder.GetRate())
		if passiveFee != 0 {
			receipt, err = rightAccountDB.ExecTransfer(matchorder.Addr, feeAddr, a.execaddr, passiveFee)
			if err != nil {
				elog.Error("matchModel.ExecTransfer buy", "from", matchorder.Addr, "to", feeAddr,
					"amount", amount, "rate", matchorder.GetRate(), "passiveFee", passiveFee, "err", err.Error())
				return nil, nil, err
			}
			matchorder.DigestedFee += passiveFee
			logs = append(logs, receipt.Logs...)
			kvs = append(kvs, receipt.KV...)
		}

		or.AVGPrice = caclAVGPrice(or, matchorder.GetLimitOrder().Price, matched)
		matchorder.AVGPrice = caclAVGPrice(matchorder, matchorder.GetLimitOrder().Price, matched)
	}

	matchorder.UpdateTime = a.blocktime

	if matched == matchorder.GetBalance() {
		matchorder.Status = et.Completed
	} else {
		matchorder.Status = et.Ordered
	}

	if matched == or.GetBalance() {
		or.Status = et.Completed
	} else {
		or.Status = et.Ordered
	}

	if matched == or.GetBalance() {
		matchorder.Balance -= matched
		matchorder.Executed = matched
		kvs = append(kvs, a.GetKVSet(matchorder)...)

		or.Executed += matched
		or.Balance = 0
		kvs = append(kvs, a.GetKVSet(or)...) //or complete
	} else {
		or.Balance -= matched
		or.Executed += matched

		matchorder.Executed = matched
		matchorder.Balance = 0
		kvs = append(kvs, a.GetKVSet(matchorder)...) //matchorder complete
	}

	re.Order = or
	re.MatchOrders = append(re.MatchOrders, matchorder)
	return logs, kvs, nil
}
*/

// Query the status database according to the order number
// Localdb deletion sequence: delete the cache in real time first, and modify the DB uniformly during block generation.
// The cache data will be deleted. However, if the cache query fails, the deleted data can still be queried in the DB
func findOrderByOrderID(statedb dbm.KV, localdb dbm.KV, orderID int64) (*et.Order, error) {
	data, err := statedb.Get(calcOrderKey(orderID))
	if err != nil {
		elog.Error("findOrderByOrderID.Get", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	var order et.Order
	err = types.Decode(data, &order)
	if err != nil {
		elog.Error("findOrderByOrderID.Decode", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	order.Executed = order.GetLimitOrder().Amount - order.Balance
	return &order, nil
}

func findOrderIDListByPrice(localdb dbm.KV, left, right uint32, price int64, op, direction int32, primaryKey string) (*et.OrderList, error) {
	table := NewMarketOrderTable(localdb)
	prefix := []byte(fmt.Sprintf("%08d:%08d:%d:%016d", left, right, op, price))

	var rows []*tab.Row
	var err error
	if primaryKey == "" { // First query, the default display of the latest transaction record
		rows, err = table.ListIndex("market_order", prefix, nil, et.Count, direction)
	} else {
		rows, err = table.ListIndex("market_order", prefix, []byte(primaryKey), et.Count, direction)
	}
	if err != nil {
		if primaryKey == "" {
			elog.Error("findOrderIDListByPrice.", "left", left, "right", right, "price", price, "err", err.Error())
		}
		return nil, err
	}
	var orderList et.OrderList
	for _, row := range rows {
		order := row.Data.(*et.Order)
		// The replacement has been done
		order.Executed = order.GetLimitOrder().Amount - order.Balance
		orderList.List = append(orderList.List, order)
	}
	// Set the primary key index
	if len(rows) == int(et.Count) {
		orderList.PrimaryKey = string(rows[len(rows)-1].Primary)
	}
	return &orderList, nil
}

//Direction
//Buying depth is in reverse order by price, from high to low
func Direction(op int32) int32 {
	if op == et.OpBuy {
		return et.ListDESC
	}
	return et.ListASC
}

//QueryMarketDepth 这里primaryKey当作主键索引来用，
//The first query does not need to fill in the value, pay according to the price from high to low, selling orders according to the price from low to high query
func QueryMarketDepth(localdb dbm.KV, left, right uint32, op int32, primaryKey string, count int32) (*et.MarketDepthList, error) {
	table := NewMarketDepthTable(localdb)
	prefix := []byte(fmt.Sprintf("%08d:%08d:%d", left, right, op))
	if count == 0 {
		count = et.Count
	}
	var rows []*tab.Row
	var err error
	if primaryKey == "" { // First query, the default display of the latest transaction record
		rows, err = table.ListIndex("price", prefix, nil, count, Direction(op))
	} else {
		rows, err = table.ListIndex("price", prefix, []byte(primaryKey), count, Direction(op))
	}
	if err != nil {
		elog.Error("QueryMarketDepth.", "left", left, "right", right, "err", err.Error())
		return nil, err
	}

	var list et.MarketDepthList
	for _, row := range rows {
		list.List = append(list.List, row.Data.(*et.MarketDepth))
	}
	if len(rows) == int(count) {
		list.PrimaryKey = string(rows[len(rows)-1].Primary)
	}
	return &list, nil
}

//QueryHistoryOrderList Only the order information is returned
func QueryHistoryOrderList(localdb dbm.KV, left, right uint32, primaryKey string, count, direction int32) (types.Message, error) {
	table := NewHistoryOrderTable(localdb)
	prefix := []byte(fmt.Sprintf("%08d:%08d", left, right))
	indexName := "name"
	if count == 0 {
		count = et.Count
	}
	var rows []*tab.Row
	var err error
	var orderList et.OrderList
HERE:
	if primaryKey == "" { // First query, the default display of the latest transaction record
		rows, err = table.ListIndex(indexName, prefix, nil, count, direction)
	} else {
		rows, err = table.ListIndex(indexName, prefix, []byte(primaryKey), count, direction)
	}
	if err != nil && err != types.ErrNotFound {
		elog.Error("QueryCompletedOrderList.", "left", left, "right", right, "err", err.Error())
		return nil, err
	}
	if err == types.ErrNotFound {
		return &orderList, nil
	}
	for _, row := range rows {
		order := row.Data.(*et.Order)
		// This table contains orders completed,revoked so filtering is required
		if order.Status == et.Revoked {
			continue
		}
		// The replacement has been done
		order.Executed = order.GetLimitOrder().Amount - order.Balance
		orderList.List = append(orderList.List, order)
		if len(orderList.List) == int(count) {
			orderList.PrimaryKey = string(row.Primary)
			return &orderList, nil
		}
	}
	if len(orderList.List) != int(count) && len(rows) == int(count) {
		primaryKey = string(rows[len(rows)-1].Primary)
		goto HERE
	}
	return &orderList, nil
}

//QueryOrderList Displays the latest by default
func QueryOrderList(localdb dbm.KV, addr string, status, count, direction int32, primaryKey string) (types.Message, error) {
	var table *tab.Table
	if status == et.Completed || status == et.Revoked {
		table = NewHistoryOrderTable(localdb)
	} else {
		table = NewMarketOrderTable(localdb)
	}
	prefix := []byte(fmt.Sprintf("%s:%d", addr, status))
	indexName := "addr_status"
	if count == 0 {
		count = et.Count
	}
	var rows []*tab.Row
	var err error
	if primaryKey == "" {
		rows, err = table.ListIndex(indexName, prefix, nil, count, direction)
	} else {
		rows, err = table.ListIndex(indexName, prefix, []byte(primaryKey), count, direction)
	}
	if err != nil {
		elog.Error("QueryOrderList.", "addr", addr, "err", err.Error())
		return nil, err
	}
	var orderList et.OrderList
	for _, row := range rows {
		order := row.Data.(*et.Order)
		order.Executed = order.GetLimitOrder().Amount - order.Balance
		orderList.List = append(orderList.List, order)
	}
	if len(rows) == int(count) {
		orderList.PrimaryKey = string(rows[len(rows)-1].Primary)
	}
	return &orderList, nil
}

func queryMarketDepth(marketTable *tab.Table, left, right uint32, op int32, price int64) (*et.MarketDepth, error) {
	primaryKey := []byte(fmt.Sprintf("%08d:%08d:%d:%016d", left, right, op, price))
	row, err := marketTable.GetData(primaryKey)
	if err != nil {
		// In localDB, delete is set to nil first and deleted last
		if err == types.ErrDecode && row == nil {
			err = types.ErrNotFound
		}
		return nil, err
	}
	return row.Data.(*et.MarketDepth), nil
}

//SafeMul Safe multiplication of large numbers, prevent overflow
func SafeMul(x, y, coinPrecision int64) int64 {
	res := big.NewInt(0).Mul(big.NewInt(x), big.NewInt(y))
	res = big.NewInt(0).Div(res, big.NewInt(coinPrecision))
	return res.Int64()
}

//SafeAdd Safe add
func SafeAdd(x, y int64) int64 {
	res := big.NewInt(0).Add(big.NewInt(x), big.NewInt(y))
	return res.Int64()
}

//Calculate the average transaction price
func caclAVGPrice(order *et.Order, price int64, amount int64) int64 {
	x := big.NewInt(0).Mul(big.NewInt(order.AVGPrice), big.NewInt(order.GetLimitOrder().Amount-order.GetBalance()))
	y := big.NewInt(0).Mul(big.NewInt(price), big.NewInt(amount))
	total := big.NewInt(0).Add(x, y)
	div := big.NewInt(0).Add(big.NewInt(order.GetLimitOrder().Amount-order.GetBalance()), big.NewInt(amount))
	avg := big.NewInt(0).Div(total, div)
	return avg.Int64()
}

//计Calculation fee
func calcMtfFee(cost int64, rate int32) int64 {
	fee := big.NewInt(0).Mul(big.NewInt(cost), big.NewInt(int64(rate)))
	fee = big.NewInt(0).Div(fee, big.NewInt(types.DefaultCoinPrecision))
	return fee.Int64()
}

func ParseConfig(cfg *types.Chain33Config, height int64) (*et.Econfig, error) {
	banks, err := ParseStrings(cfg, "banks", height)
	if err != nil || len(banks) == 0 {
		return nil, err
	}
	coins, err := ParseCoins(cfg, "coins", height)
	if err != nil {
		return nil, err
	}
	exchanges, err := ParseSymbols(cfg, "exchanges", height)
	if err != nil {
		return nil, err
	}
	return &et.Econfig{
		Banks:     banks,
		Coins:     coins,
		Exchanges: exchanges,
	}, nil
}

func ParseStrings(cfg *types.Chain33Config, tradeKey string, height int64) (ret []string, err error) {
	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid val", "val", val, "key", tradeKey)
		return nil, et.ErrCfgFmt
	}

	for _, v := range datas {
		one, ok := v.(string)
		if !ok {
			elog.Error("invalid one", "one", one, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}
		ret = append(ret, one)
	}
	return
}

func ParseCoins(cfg *types.Chain33Config, tradeKey string, height int64) (coins []et.CoinCfg, err error) {
	coins = make([]et.CoinCfg, 0)

	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid coins", "val", val, "type", reflect.TypeOf(val))
		return nil, et.ErrCfgFmt
	}

	for _, e := range datas {
		v, ok := e.(map[string]interface{})
		if !ok {
			elog.Error("invalid coins one", "one", v, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}

		coin := et.CoinCfg{
			Coin:   v["coin"].(string),
			Execer: v["execer"].(string),
			Name:   v["name"].(string),
		}
		coins = append(coins, coin)
	}
	return
}

func ParseSymbols(cfg *types.Chain33Config, tradeKey string, height int64) (symbols map[string]*et.Trade, err error) {
	symbols = make(map[string]*et.Trade)

	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid Symbols", "val", val, "type", reflect.TypeOf(val))
		return nil, et.ErrCfgFmt
	}

	for _, e := range datas {
		v, ok := e.(map[string]interface{})
		if !ok {
			elog.Error("invalid Symbols one", "one", v, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}

		symbol := v["symbol"].(string)
		symbols[symbol] = &et.Trade{
			Symbol:       symbol,
			PriceDigits:  int32(formatInterface(v["priceDigits"])),
			AmountDigits: int32(formatInterface(v["amountDigits"])),
			Taker:        int32(formatInterface(v["taker"])),
			Maker:        int32(formatInterface(v["maker"])),
			MinFee:       formatInterface(v["minFee"]),
		}
	}
	return
}
func formatInterface(data interface{}) int64 {
	switch data.(type) {
	case int64:
		return data.(int64)
	case int32:
		return int64(data.(int32))
	case int:
		return int64(data.(int))
	default:
		return 0
	}
}

// 使用 chain33 地址为key
// 同样提供: account 基本和 token 级别的信息

// 现在为了实现简单: 只有一个交易所,
// 所以 资金帐号和现货交易所帐号是同一个

// 存款交易是系统代为存入的, 存到指定帐号上, 不是签名帐号中

// 用户帐号定义
// dex1 -> accountid -> tokenids 是一个对象
//  理论上, 对象越小越快, 但交易涉及两个资产. 如果一个资产是一个对象的. 要处理两个对象.
//  先实现再说
func (a *SpotAction) Deposit(payload *et.ZkDeposit) (*types.Receipt, error) {

	chain33Addr := payload.GetChain33Addr()
	amount := payload.GetAmount()

	// TODO tid 哪里定义, 里面不需要知道tid 是什么, 在合约里 id1 换 id2

	acc, err := a.LoadDexAccount(chain33Addr)
	if err != nil {
		return nil, err
	}
	amount2, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		return nil, et.ErrAssetBalance
	}
	// TODO 未处理 receipt kv log 等
	acc.Mint(uint32(payload.TokenId), amount2.Uint64())
	return nil, nil
}

func (a *SpotAction) LoadDexAccount(chain33addr string) (*dexAccount, error) {
	key := fmt.Sprintf("dexAccountKey%s", chain33addr)
	v, err := a.statedb.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	var val et.DexAccount
	err = types.Decode(v, &val)
	if err != nil {
		return nil, err
	}
	acc := GetDexAccount(&val)
	return acc, nil
}
