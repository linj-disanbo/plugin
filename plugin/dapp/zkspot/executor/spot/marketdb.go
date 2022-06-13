package spot

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

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

//QueryHistoryOrderList Only the order information is returned
func QueryHistoryOrderList(localdb dbm.KV, dbprefix et.DBprefix, in *et.SpotQueryHistoryOrderList) (types.Message, error) {
	left, right, primaryKey, count, direction := in.LeftAsset, in.RightAsset, in.PrimaryKey, in.Count, in.Direction

	table := NewHistoryOrderTable(localdb, dbprefix)
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
func QueryOrderList(localdb dbm.KV, dbprefix et.DBprefix, in *et.SpotQueryOrderList) (types.Message, error) {
	var table *tab.Table
	if in.Status == et.Completed || in.Status == et.Revoked {
		table = NewHistoryOrderTable(localdb, dbprefix)
	} else {
		table = NewMarketOrderTable(localdb, dbprefix)
	}
	prefix := []byte(fmt.Sprintf("%s:%d", in.Address, in.Status))
	indexName := "addr_status"
	count := in.Count
	if count == 0 {
		count = et.Count
	}
	var rows []*tab.Row
	var err error
	if in.PrimaryKey == "" {
		rows, err = table.ListIndex(indexName, prefix, nil, count, in.Direction)
	} else {
		rows, err = table.ListIndex(indexName, prefix, []byte(in.PrimaryKey), count, in.Direction)
	}
	if err != nil {
		elog.Error("QueryOrderList.", "addr", in.Address, "err", err.Error())
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

func getMarketDepth(marketTable *tab.Table, left, right uint32, op int32, price int64) (*et.SpotMarketDepth, error) {
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
