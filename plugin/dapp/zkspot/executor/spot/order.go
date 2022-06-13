package spot

import (
	"encoding/hex"
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	tab "github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

type spotOrder struct {
	statedb  dbm.KV
	localdb  dbm.KV
	dbprefix et.DBprefix
	repo     *orderSRepo
}

// type spotOrder et.SpotOrder

func newSpotOrder(statedb dbm.KV, localdb dbm.KV, dbprefix et.DBprefix) *spotOrder {
	return &spotOrder{
		repo:     newOrderSRepo(statedb, dbprefix),
		statedb:  statedb,
		localdb:  localdb,
		dbprefix: dbprefix,
	}
}

func (o *spotOrder) find(id int64) (*et.SpotOrder, error) {
	return o.repo.findOrderBy(id)
}

func (o *spotOrder) checkRevoke(fromaddr string, order *et.SpotOrder) error {
	if order.Addr != fromaddr {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return et.ErrAddr
	}
	if order.Status == et.Completed || order.Status == et.Revoked {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", order.Addr, "order.status", order.Status)
		return et.ErrOrderSatus
	}
	return nil
}

func (o *spotOrder) calcFrozenToken(order *et.SpotOrder, precision int64) (uint32, uint64) {
	price := order.GetLimitOrder().GetPrice()
	balance := order.GetBalance()

	if order.GetLimitOrder().GetOp() == et.OpBuy {
		amount := CalcActualCost(et.OpBuy, balance, price, precision)
		amount += SafeMul(balance, int64(order.Rate), precision)
		return order.GetLimitOrder().RightAsset, uint64(amount)
	}
	return order.GetLimitOrder().LeftAsset, uint64(balance)
}

// buy 按最大量判断余额是否够
// 因为在吃单时, 价格是变动的, 所以实际锁定的量是会浮动的
// 实现上, 按最大量判断余额是否够, 在成交时, 按实际需要量扣除. 最后变成挂单时, 进行锁定
func (o *spotOrder) NeedToken(order *et.SpotOrder, precision int64) (uint32, int64) {
	or := order.GetLimitOrder()
	if or.GetOp() == et.OpBuy {
		amount := SafeMul(or.GetAmount(), or.GetPrice(), precision)
		fee := calcMtfFee(amount, int32(order.TakerRate))
		total := SafeAdd(amount, int64(fee))
		return or.LeftAsset, total
	}

	/* if payload.GetOp() == et.OpSell */
	return or.LeftAsset, or.GetAmount()
}

func (o *spotOrder) Revoke(order *et.SpotOrder, blockTime int64, txhash []byte, txindex int) (*types.Receipt, error) {
	order.Status = et.Revoked
	order.UpdateTime = blockTime
	order.RevokeHash = hex.EncodeToString(txhash)
	kvs := o.repo.GetOrderKvSet(order)

	re := &et.ReceiptSpotMatch{
		Order: order,
		Index: int64(txindex),
	}
	receiptlog := &types.ReceiptLog{Ty: et.TyRevokeOrderLog, Log: types.Encode(re)}
	return &types.Receipt{KV: kvs, Logs: []*types.ReceiptLog{receiptlog}}, nil
}

func (o *spotOrder) isActiveOrder(order *et.SpotOrder) bool {
	return order.Status == et.Ordered
}

// statedb: order, account
// localdb: market-depth, market-orders, history-orders

func calcOrderKey(prefix string, orderID int64) []byte {
	return []byte(fmt.Sprintf("%s"+orderKeyFmt, prefix, orderID))
}

func FindOrderByOrderID(statedb dbm.KV, localdb dbm.KV, dbprefix et.DBprefix, orderID int64) (*et.SpotOrder, error) {
	return newOrderSRepo(statedb, dbprefix).findOrderBy(orderID)
}

// orderSRepo statedb repo
type orderSRepo struct {
	statedb  dbm.KV
	dbprefix et.DBprefix
}

func newOrderSRepo(statedb dbm.KV, dbprefix et.DBprefix) *orderSRepo {
	return &orderSRepo{
		statedb:  statedb,
		dbprefix: dbprefix,
	}
}

func (repo *orderSRepo) orderKey(orderID int64) []byte {
	return calcOrderKey(repo.dbprefix.GetStatedbPrefix(), orderID)
}

func (repo *orderSRepo) findOrderBy(orderID int64) (*et.SpotOrder, error) {
	key := repo.orderKey(orderID)
	data, err := repo.statedb.Get(key)
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

func (repo *orderSRepo) GetOrderKvSet(order *et.SpotOrder) (kvset []*types.KeyValue) {
	kvset = append(kvset, &types.KeyValue{Key: repo.orderKey(order.OrderID), Value: types.Encode(order)})
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
func QueryMarketDepth(localdb dbm.KV, dbprefix et.DBprefix, in *et.SpotQueryMarketDepth) (*et.SpotMarketDepthList, error) {
	left, right, op := in.LeftAsset, in.RightAsset, in.Op
	count, primaryKey := in.Count, in.PrimaryKey
	marketTable := NewMarketDepthTable(localdb, dbprefix)

	return queryMarketDepthList(marketTable, left, right, op, primaryKey, count)
}

func queryMarketDepthList(table *tab.Table, left, right uint32, op int32, primaryKey string, count int32) (*et.SpotMarketDepthList, error) {
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
