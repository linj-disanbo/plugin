package spot

import (
	"fmt"

	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
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

//GetIndex get index
func GetIndex(height int64, index int64) int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (height*types.MaxTxsPerBlock + int64(index)) * 1e4
}

//GetKVSet get kv set
func (a *SpotAction) GetKVSet(order *et.SpotOrder) (kvset []*types.KeyValue) {
	return GetOrderKvSet(order)
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

// set the transaction logic method
// rules:
//1. The purchase price is higher than the market price, and the price is matched from low to high.
//2. Sell orders are matched at prices lower than market prices.
//3. Match the same prices on a first-in, first-out basis
func (a *Spot) MatchLimitOrder(payload *et.SpotLimitOrder, entrustAddr string, taker *SpotTrader) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	// A single transaction can match up to {MaxCount} orders, the maximum depth can be matched, the system has to protect itself
	// TODO next-price, next-order-list
	matcher1 := newMatcher(a.env.GetStateDB(), a.env.GetLocalDB(), a.env.GetAPI())
	taker.matches = &et.ReceiptSpotMatch{
		Order: taker.order,
		Index: a.GetIndex(),
	}
	for {
		if matcher1.isDone() {
			break
		}

		//Obtain price information of existing market listing
		marketDepthList, _ := matcher1.QueryMarketDepth(payload)
		if marketDepthList == nil || len(marketDepthList.List) == 0 {
			break
		}
		for _, marketDepth := range marketDepthList.List {
			elog.Info("LimitOrder debug find depth", "height", a.env.GetHeight(), "amount", marketDepth.Amount, "price", marketDepth.Price, "order-price", payload.GetPrice(), "op", OpSwap(payload.Op), "index", a.GetIndex())
			if matcher1.isDone() || matcher1.priceDone(payload, marketDepth) {
				break
			}

			for {
				if matcher1.isDone() {
					break
				}

				orderList, err := matcher1.findOrderIDListByPrice(payload, marketDepth)
				if err != nil || orderList == nil || len(orderList.List) == 0 {
					break
				}
				// got orderlist to trade
				for _, matchorder := range orderList.List {
					if matcher1.isDone() {
						break
					}
					// Check the order status
					order, err := findOrderByOrderID(a.env.GetStateDB(), a.env.GetLocalDB(), matchorder.GetOrderID())
					if err != nil || order.Status != et.Ordered {
						continue
					}
					log, kv, err := matcher1.matchModel(order, taker)
					if err != nil {
						elog.Error("matchModel", "height", a.env.GetHeight(), "orderID", order.GetOrderID(), "payloadID", taker.order.GetOrderID(), "error", err)
						return nil, err
					}
					logs = append(logs, log...)
					kvs = append(kvs, kv...)
					if taker.order.Status == et.Completed {
						matcher1.done = true
						break
					}
					// match depth count
					matcher1.recordMatchCount()
				}
				if matcher1.isEndOrderList(marketDepth.Price) {
					break
				}
			}
		}
	}

	kvs = append(kvs, GetOrderKvSet(taker.order)...)
	receiptlog := &types.ReceiptLog{Ty: et.TyLimitOrderLog, Log: types.Encode(taker.matches)}
	logs = append(logs, receiptlog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

// Query the status database according to the order number
// Localdb deletion sequence: delete the cache in real time first, and modify the DB uniformly during block generation.
// The cache data will be deleted. However, if the cache query fails, the deleted data can still be queried in the DB
/*
func findOrderByOrderID(statedb dbm.KV, localdb dbm.KV, orderID int64) (*et.SpotOrder, error) {
	data, err := statedb.Get(calcOrderKey(orderID))
	if err != nil {
		elog.Error("findOrderByOrderID.Get", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	var order et.SpotOrder
	err = types.Decode(data, &order)
	if err != nil {
		elog.Error("findOrderByOrderID.Decode", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	order.Executed = order.GetLimitOrder().Amount - order.Balance
	return &order, nil
}
*/
//QueryHistoryOrderList Only the order information is returned
func QueryHistoryOrderList(localdb dbm.KV, left, right uint32, primaryKey string, count, direction int32) (types.Message, error) {
	var todo et.DBprefix
	table := NewHistoryOrderTable(localdb, todo)
	prefix := []byte(fmt.Sprintf("%08d:%08d", left, right))
	indexName := "name"
	if count == 0 {
		count = et.Count
	}
	var rows []*tab.Row
	var err error
	var orderList et.SpotOrderList
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
		order := row.Data.(*et.SpotOrder)
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
	var todo et.DBprefix
	var table *tab.Table
	if status == et.Completed || status == et.Revoked {
		table = NewHistoryOrderTable(localdb, todo)
	} else {
		table = NewMarketOrderTable(localdb, todo)
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
	var orderList et.SpotOrderList
	for _, row := range rows {
		order := row.Data.(*et.SpotOrder)
		order.Executed = order.GetLimitOrder().Amount - order.Balance
		orderList.List = append(orderList.List, order)
	}
	if len(rows) == int(count) {
		orderList.PrimaryKey = string(rows[len(rows)-1].Primary)
	}
	return &orderList, nil
}

func QueryMarketDepth1(marketTable *tab.Table, left, right uint32, op int32, price int64) (*et.SpotMarketDepth, error) {
	return queryMarketDepth(marketTable, left, right, op, price)
}

func queryMarketDepth(marketTable *tab.Table, left, right uint32, op int32, price int64) (*et.SpotMarketDepth, error) {
	primaryKey := []byte(fmt.Sprintf("%08d:%08d:%d:%016d", left, right, op, price))
	row, err := marketTable.GetData(primaryKey)
	if err != nil {
		// In localDB, delete is set to nil first and deleted last
		if err == types.ErrDecode && row == nil {
			err = types.ErrNotFound
		}
		return nil, err
	}
	return row.Data.(*et.SpotMarketDepth), nil
}

func (a *SpotAction) LoadDexAccount(chain33addr string, accountID uint64) (*dexAccount, error) {
	return LoadSpotAccount(chain33addr, accountID, a.statedb)
}

func (a *SpotAction) CalcMaxActive(accountID uint64, token uint32, amount string) (uint64, error) {
	acc, err := LoadSpotAccount(a.fromaddr, accountID, a.statedb)
	if err != nil {
		return 0, err
	}
	idx := acc.findTokenIndex(token)
	if idx < 0 {
		return 0, nil
	}
	return acc.acc.Balance[idx].Balance, nil
}
