package executor

import (
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
type spotTaker struct {
	spotTrader
	re     *et.ReceiptExchange // TODO last need append to receipt
	accFee *dexAccount
}

type spotTrader struct {
	acc   *dexAccount
	order *et.Order
	cfg   *types.Chain33Config
}

type spotMaker struct {
	spotTrader
}

type matchInfo struct {
	matched      int64
	leftBalance  int64 // = trade balance
	rightBalance int64 // = * price
	feeTaker     int64 // use right token
	feeMater     int64 // use right token
}

func (s *spotTaker) FrozenTokenForLimitOrder() (*types.Receipt, error) {
	precision := s.cfg.GetCoinPrecision()
	or := s.order.GetLimitOrder()
	if or.GetOp() == et.OpBuy {
		amount := SafeMul(or.GetAmount(), or.GetPrice(), precision)
		fee := calcMtfFee(amount, int32(getFeeRate(s.acc)))
		total := SafeAdd(amount, int64(fee))

		receipt, err := s.acc.Frozen(or.RightAsset, uint64(total))
		if err != nil {
			elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
			return nil, et.ErrAssetBalance
		}
		return receipt, nil
	}

	/* if payload.GetOp() == et.OpSell */
	amount := or.GetAmount()
	receipt, err := s.acc.Frozen(or.LeftAsset, uint64(or.GetAmount()))
	if err != nil {
		elog.Error("limit check left balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
		return nil, et.ErrAssetBalance
	}
	return receipt, nil
}

func (s *spotTaker) UnFrozenFeeForLimitOrder() (*types.Receipt, error) {
	or := s.order.GetLimitOrder()
	if or.GetOp() != et.OpBuy {
		return nil, nil
	}
	precision := s.cfg.GetCoinPrecision()
	// takerFee - makerFee
	actvieFee := SafeMul(or.GetAmount(), int64(s.order.TakerRate-s.order.Rate), precision)
	receipt, err := s.acc.Active(or.RightAsset, uint64(actvieFee))
	if err != nil {
		elog.Error("UnFrozenFeeForLimitOrder", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", actvieFee)
		return nil, et.ErrAssetBalance
	}
	return receipt, nil
}

func (s *spotTaker) Trade(maker *spotMaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(maker.order)
	matchDetail := s.calcTradeInfo(maker, balance)

	receipt3, kvs3, err := maker.orderTraded(matchDetail, s.order)
	if err != nil {
		return receipt3, kvs3, err
	}

	receipt2, kvs2, err := s.orderTraded(matchDetail, maker.order)
	if err != nil {
		return receipt2, kvs2, err
	}

	receipt, kvs, err := s.settlement(maker, matchDetail)
	if err != nil {
		return receipt, kvs, err
	}

	kvs = append(kvs, kvs2...)
	kvs = append(kvs, kvs3...)
	receipt = append(receipt, receipt2...)
	receipt = append(receipt, receipt3...)

	return receipt, kvs, nil
}

func (s *spotTaker) calcTradeBalance(order *et.Order) int64 {
	if order.GetBalance() >= s.order.GetBalance() {
		return s.order.GetBalance()
	}
	return order.GetBalance()
}

func (s *spotTaker) calcTradeInfo(maker *spotMaker, balance int64) matchInfo {
	var info matchInfo
	info.matched = balance
	info.leftBalance = balance
	info.rightBalance = SafeMul(balance, maker.order.GetLimitOrder().Price, s.cfg.GetCoinPrecision())
	info.feeTaker = SafeMul(info.rightBalance, int64(s.order.TakerRate), s.cfg.GetCoinPrecision())
	info.feeMater = SafeMul(info.rightBalance, int64(maker.order.Rate), s.cfg.GetCoinPrecision())
	return info
}

// settlement
// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
func (s *spotTaker) settlement(maker *spotMaker, tradeBalance matchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	if s.acc.acc.Addr == maker.acc.acc.Addr {
		return s.selfSettlement(tradeBalance)
	}

	copyAcc := dupAccount(s.acc.acc)
	copyAccMaker := dupAccount(maker.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := uint32(1), uint32(2)
	if s.order.GetLimitOrder().Op == et.OpSell {
		s.acc.FrozenTranfer(maker.acc, leftToken, uint64(tradeBalance.leftBalance))
		maker.acc.FrozenTranfer(s.acc, rightToken, uint64(tradeBalance.rightBalance))
	} else {
		s.acc.FrozenTranfer(maker.acc, rightToken, uint64(tradeBalance.rightBalance))
		maker.acc.FrozenTranfer(s.acc, leftToken, uint64(tradeBalance.leftBalance))
	}

	s.acc.FrozenTranfer(s.accFee, rightToken, uint64(tradeBalance.feeTaker))
	maker.acc.FrozenTranfer(s.accFee, rightToken, uint64(tradeBalance.feeMater))

	kvs1 := s.acc.GetKVSet()
	kvs2 := maker.acc.GetKVSet()
	kvs3 := s.accFee.GetKVSet()

	kvs1 = append(kvs1, kvs2...)
	kvs1 = append(kvs1, kvs3...)

	re := et.ReceiptSpotTrade{
		Prev: &et.TradeAccounts{
			Taker: copyAcc,
			Maker: copyAccMaker,
			Fee:   copyFeeAcc,
		},
		Current: &et.TradeAccounts{
			Taker: s.acc.acc,
			Maker: maker.acc.acc,
			Fee:   s.accFee.acc,
		},
	}

	log1 := types.ReceiptLog{
		Ty:  et.TxSpotTradeLog,
		Log: types.Encode(&re),
	}
	return []*types.ReceiptLog{&log1}, kvs1, nil
}

// taker/maker the same user
func (s *spotTaker) selfSettlement(tradeBalance matchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	copyAcc := dupAccount(s.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := uint32(1), uint32(2)
	s.acc.Active(leftToken, uint64(tradeBalance.leftBalance))
	s.acc.Active(rightToken, uint64(tradeBalance.rightBalance))

	s.acc.FrozenTranfer(s.accFee, rightToken, uint64(tradeBalance.feeTaker+tradeBalance.feeMater))

	kvs1 := s.acc.GetKVSet()
	kvs3 := s.accFee.GetKVSet()
	kvs1 = append(kvs1, kvs3...)

	re := et.ReceiptSpotTrade{
		Prev: &et.TradeAccounts{
			Taker: copyAcc,
			Maker: copyAcc,
			Fee:   copyFeeAcc,
		},
		Current: &et.TradeAccounts{
			Taker: s.acc.acc,
			Maker: s.acc.acc,
			Fee:   s.accFee.acc,
		},
	}

	log1 := types.ReceiptLog{
		Ty:  et.TxSpotTradeLog,
		Log: types.Encode(&re),
	}
	return []*types.ReceiptLog{&log1}, kvs1, nil
}

func (s *spotTaker) orderTraded(matchDetail matchInfo, order *et.Order) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.matched

	// fee and AVGPrice
	s.order.DigestedFee += matchDetail.feeTaker
	s.order.AVGPrice = caclAVGPrice(s.order, s.order.GetLimitOrder().Price, matched)

	// status
	if matched == s.order.GetBalance() {
		s.order.Status = et.Completed
	} else {
		s.order.Status = et.Ordered
	}

	// order matched
	s.order.Executed = matched
	s.order.Balance -= matched

	s.re.Order = s.order
	s.re.MatchOrders = append(s.re.MatchOrders, order)
	// TODO n times trade, will gen n order-kvs
	kvs := GetOrderKvSet(s.order)
	return []*types.ReceiptLog{}, kvs, nil
}

func (m *spotMaker) orderTraded(matchDetail matchInfo, takerOrder *et.Order) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.matched

	// fee and AVGPrice
	m.order.DigestedFee += matchDetail.feeMater
	m.order.AVGPrice = caclAVGPrice(m.order, m.order.GetLimitOrder().Price, matched)

	m.order.UpdateTime = takerOrder.UpdateTime

	// status
	if matched == m.order.GetBalance() {
		m.order.Status = et.Completed
	} else {
		m.order.Status = et.Ordered
	}

	// order matched
	m.order.Executed = matched
	m.order.Balance -= matched
	kvs := GetOrderKvSet(m.order)
	return []*types.ReceiptLog{}, kvs, nil
}

func (m *matcher) matchModel(matchorder *et.Order, taker *spotTaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	matched := taker.calcTradeBalance(matchorder)
	elog.Info("try match", "activeId", taker.order.OrderID, "passiveId", matchorder.OrderID, "activeAddr", taker.order.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", taker.order.GetLimitOrder().Price)

	accMatch, err := LoadSpotAccount(matchorder.Addr, matchorder.GetLimitOrder().Order.AccountID, m.statedb)
	if err != nil {
		return nil, nil, err
	}
	maker := spotMaker{
		spotTrader: spotTrader{
			acc:   accMatch,
			order: matchorder,
			cfg:   m.api.GetConfig(),
		},
	}

	logs, kvs, err = taker.Trade(&maker)
	return logs, kvs, nil
}

type orderInit func(*et.Order) *et.Order

func createLimitOrder(payload *et.LimitOrder, entrustAddr string, inits []orderInit) *et.Order {
	or := &et.Order{
		Value:       &et.Order_LimitOrder{LimitOrder: payload},
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

type feeDetail struct {
	addr  string
	id    uint64
	taker int32
	maker int32
}

func (f *feeDetail) initLimitOrder() func(*et.Order) *et.Order {
	return func(order *et.Order) *et.Order {
		order.Rate = f.maker
		order.TakerRate = f.taker
		return order
	}
}
