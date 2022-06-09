package spot

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

//GetIndex get index
func GetIndex(height int64, index int64) int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (height*types.MaxTxsPerBlock + int64(index)) * 1e4
}

func (a *Spot) MatchLimitOrder(payload *et.SpotLimitOrder, taker *SpotTrader) (*types.Receipt, error) {
	matcher1 := newMatcher(a.env.GetStateDB(), a.env.GetLocalDB(), a.env.GetAPI())
	elog.Info("LimitOrder", "height", a.env.GetHeight(), "order-price", payload.GetPrice(), "op", OpSwap(payload.Op), "index", taker.order.GetOrderID())
	return matcher1.MatchLimitOrder(payload, taker)
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
