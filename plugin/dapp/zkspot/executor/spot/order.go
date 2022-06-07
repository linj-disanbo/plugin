package spot

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// TODO

func calcOrderKey(orderID int64) []byte {
	return []byte("TODO")
}

func FindOrderByOrderID1(statedb dbm.KV, localdb dbm.KV, orderID int64) (*et.SpotOrder, error) {
	return findOrderByOrderID(statedb, localdb, orderID)
}

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

func GetOrderKvSet1(order *et.SpotOrder) (kvset []*types.KeyValue) {
	kvset = append(kvset, &types.KeyValue{Key: calcOrderKey(order.OrderID), Value: types.Encode(order)})
	return kvset
}

func findOrderIDListByPrice(localdb dbm.KV, left, right uint32, price int64, op, direction int32, primaryKey string) (*et.SpotOrderList, error) {
	var todo et.DBprefix
	table := NewMarketOrderTable(localdb, todo)
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
	var orderList et.SpotOrderList
	for _, row := range rows {
		order := row.Data.(*et.SpotOrder)
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

//OpSwap reverse
func OpSwap(op int32) int32 {
	if op == et.OpBuy {
		return et.OpSell
	}
	return et.OpBuy
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
func QueryMarketDepth(localdb dbm.KV, left, right uint32, op int32, primaryKey string, count int32) (*et.SpotMarketDepthList, error) {
	var todo et.DBprefix
	table := NewMarketDepthTable(localdb, todo)
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
		elog.Error("QueryMarketDepth.", "prefix", string(prefix), "left", left, "right", right, "err", err.Error())
		return nil, err
	}

	var list et.SpotMarketDepthList
	for _, row := range rows {
		list.List = append(list.List, row.Data.(*et.SpotMarketDepth))
	}
	if len(rows) == int(count) {
		list.PrimaryKey = string(rows[len(rows)-1].Primary)
	}
	return &list, nil
}
