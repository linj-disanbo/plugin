package spot

import (
	"encoding/hex"
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

type orderInit func(*et.SpotOrder) *et.SpotOrder

func createLimitOrder(payload *et.SpotLimitOrder, entrustAddr string, inits []orderInit) *et.SpotOrder {
	or := &et.SpotOrder{
		Value:       &et.SpotOrder_LimitOrder{LimitOrder: payload},
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

func createAssetLimitOrder(payload *et.AssetLimitOrder, entrustAddr string, inits []orderInit) *et.SpotOrder {
	or := &et.SpotOrder{
		Value:       &et.SpotOrder_AssetLimitOrder{AssetLimitOrder: payload},
		Ty:          et.TyAssetLimitOrderAction,
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

type spotOrder struct {
	order *et.SpotOrder

	repo *orderSRepo
}

func newSpotOrder(order *et.SpotOrder, orderdb *orderSRepo) *spotOrder {
	return &spotOrder{
		repo:  orderdb,
		order: order,
	}
}

func (o *spotOrder) checkRevoke(fromaddr string) error {
	if o.order.Addr != fromaddr {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", o.order.Addr, "order.status", o.order.Status)
		return et.ErrAddr
	}
	if o.order.Status == et.Completed || o.order.Status == et.Revoked {
		elog.Error("RevokeOrder.OrderCheck", "addr", fromaddr, "order.addr", o.order.Addr, "order.status", o.order.Status)
		return et.ErrOrderSatus
	}
	return nil
}

func (o *spotOrder) calcFrozenToken(rightPrecision int64) (*et.Asset, uint64) {
	order := o.order
	price := o.GetPrice()
	balance := order.GetBalance()

	left, right := o.GetAsset()
	if o.GetOp() == et.OpBuy {
		amount := CalcActualCost(et.OpBuy, balance, price, rightPrecision)
		amount += SafeMul(amount, int64(order.Rate), rightPrecision)
		return right, uint64(amount)
	}
	return left, uint64(balance)
}

// buy 按最大量判断余额是否够
// 因为在吃单时, 价格是变动的, 所以实际锁定的量是会浮动的
// 实现上, 按最大量判断余额是否够, 在成交时, 按实际需要量扣除. 最后变成挂单时, 进行锁定
func (o *spotOrder) NeedToken(precision int64) (uint64, int64) {
	or := o.order.GetLimitOrder()
	if or.GetOp() == et.OpBuy {
		amount := SafeMul(or.GetAmount(), or.GetPrice(), precision)
		fee := calcMtfFee(amount, int32(o.order.TakerRate), precision)
		total := SafeAdd(amount, int64(fee))
		return or.RightAsset, total
	}

	/* if payload.GetOp() == et.OpSell */
	return or.LeftAsset, or.GetAmount()
}

func (o *spotOrder) nftTakerOrderNeedToken(o2 *spotOrder, precision int64) (uint64, int64) {
	or := o2.order
	amount := SafeMul(or.GetBalance(), or.GetNftOrder().Price, precision)
	fee := calcMtfFee(amount, int32(o.order.TakerRate), precision)
	total := SafeAdd(amount, int64(fee))
	return or.GetNftOrder().RightAsset, total
}

func (o *spotOrder) Revoke(blockTime int64, txhash []byte, txindex int) (*types.Receipt, error) {
	order := o.order
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

func (o *spotOrder) isActiveOrder() bool {
	return o.order.Status == et.Ordered
}

func (o *spotOrder) orderUpdate(matchDetail *et.MatchInfo) {
	matched := matchDetail.Matched

	// fee and AVGPrice
	o.order.DigestedFee += matchDetail.FeeTaker
	o.order.AVGPrice = matchDetail.Price

	// status
	if matched == o.order.GetBalance() {
		o.order.Status = et.Completed
	} else {
		o.order.Status = et.Ordered
	}

	// order matched
	o.order.Executed = matched
	o.order.Balance -= matched
}

func (o *spotOrder) Traded(matchDetail *et.MatchInfo, blocktime int64) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	o.orderUpdate(matchDetail)
	o.order.UpdateTime = blocktime
	kvs := o.repo.GetOrderKvSet(o.order)
	return []*types.ReceiptLog{}, kvs, nil
}

func (o *spotOrder) GetOp() int32 {
	switch o.order.Ty {
	case et.TyLimitOrderAction:
		return o.order.GetLimitOrder().GetOp()
	case et.TyAssetLimitOrderAction:
		return o.order.GetAssetLimitOrder().GetOp()
	case et.TyNftOrderAction:
		return o.order.GetNftOrder().GetOp()
	}
	panic("Not support op")
}

func (o *spotOrder) GetPrice() int64 {
	switch o.order.Ty {
	case et.TyLimitOrderAction:
		return o.order.GetLimitOrder().GetPrice()
	case et.TyAssetLimitOrderAction:
		return o.order.GetAssetLimitOrder().GetPrice()
	case et.TyNftOrderAction:
		return o.order.GetNftOrder().GetPrice()
	}
	panic("Not support price")
}

func (o *spotOrder) GetAsset() (*et.Asset, *et.Asset) {
	switch o.order.Ty {
	case et.TyLimitOrderAction:
		return newAsset1(o.order.GetLimitOrder().LeftAsset), newAsset1(o.order.GetLimitOrder().RightAsset)
	case et.TyAssetLimitOrderAction:
		return o.order.GetAssetLimitOrder().LeftAsset, o.order.GetAssetLimitOrder().RightAsset
	case et.TyNftOrderAction:
		return newNftAsset1(o.order.GetNftOrder().LeftAsset), newAsset1(o.order.GetNftOrder().RightAsset)
	}
	panic("Not support GetAsset")
}

// statedb: order, account
// localdb: market-depth, market-orders, history-orders

func calcOrderKey(prefix string, orderID int64) []byte {
	return []byte(fmt.Sprintf("%s"+orderKeyFmt, prefix, orderID))
}

func FindOrderByOrderID(statedb dbm.KV, localdb dbm.KV, dbprefix et.DBprefix, orderID int64) (*et.SpotOrder, error) {
	return newOrderSRepo(statedb, dbprefix).findOrderBy(orderID)
}

func FindOrderByOrderNftID(statedb dbm.KV, localdb dbm.KV, dbprefix et.DBprefix, orderID int64) (*et.SpotOrder, error) {
	return newOrderSRepo(statedb, dbprefix).findNftOrderBy(orderID)
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

func (repo *orderSRepo) findNftOrderBy(orderID int64) (*et.SpotOrder, error) {
	key := repo.orderKey(orderID)
	data, err := repo.statedb.Get(key)
	if err != nil {
		elog.Error("findNftOrderBy.Get", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	var order et.SpotOrder
	err = types.Decode(data, &order)
	if err != nil {
		elog.Error("findNftOrderBy.Decode", "orderID", orderID, "err", err.Error())
		return nil, err
	}
	if order.GetNftOrder() == nil {
		elog.Error("findNftOrderBy", "order", "nil")
		return nil, err
	}
	order.Executed = order.GetNftOrder().Amount - order.Balance
	return &order, nil
}

func (repo *orderSRepo) GetOrderKvSet(order *et.SpotOrder) (kvset []*types.KeyValue) {
	kvset = append(kvset, &types.KeyValue{Key: repo.orderKey(order.OrderID), Value: types.Encode(order)})
	return kvset
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
