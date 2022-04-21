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
	acc     *dexAccount
	order   *et.Order
	feeRate int32
	cfg     *types.Chain33Config
	// external infos
	blocktime int64
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

func (s *spotTaker) FrozenTokenForLimitOrder() ([]*types.ReceiptLog, []*types.KeyValue, error) {
	// TODO
	precision := int64(1e8) // cfg.GetCoinPrecision()
	or := s.order.GetLimitOrder()
	if or.GetOp() == et.OpBuy {
		amount := SafeMul(or.GetAmount(), or.GetPrice(), precision)
		fee := calcMtfFee(amount, int32(getFeeRate(s.acc)))
		total := SafeAdd(amount, int64(fee))

		err := s.acc.Frozen(or.RightAsset, uint64(total))
		if err != nil {
			elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
			return nil, nil, et.ErrAssetBalance
		}
	} else {
		/* if payload.GetOp() == et.OpSell */
		amount := or.GetAmount()
		err := s.acc.Frozen(or.LeftAsset, uint64(or.GetAmount()))
		if err != nil {
			elog.Error("limit check left balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
			return nil, nil, et.ErrAssetBalance
		}
	}
	return nil, nil, nil
}

func (s *spotTaker) Trade(maker *spotMaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(maker.order)
	matchDetail := s.calcTradeInfo(maker, balance)

	receipt3, kvs3, err := maker.orderTraded(matchDetail)
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
	info.feeTaker = SafeMul(info.rightBalance, int64(s.feeRate), s.cfg.GetCoinPrecision())
	info.feeMater = SafeMul(info.rightBalance, int64(maker.feeRate), s.cfg.GetCoinPrecision())
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

	s.order.UpdateTime = s.blocktime

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

func (m *spotMaker) orderTraded(matchDetail matchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.matched

	// fee and AVGPrice
	m.order.DigestedFee += matchDetail.feeMater
	m.order.AVGPrice = caclAVGPrice(m.order, m.order.GetLimitOrder().Price, matched)

	m.order.UpdateTime = m.blocktime

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

func (a *SpotAction) matchModel2(payload *et.LimitOrder, matchorder *et.Order, or *et.Order, re *et.ReceiptExchange, takerFee int32, taker *spotTaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var matched int64

	// cfg := a.api.GetConfig()
	matched = taker.calcTradeBalance(matchorder)
	elog.Info("try match", "activeId", or.OrderID, "passiveId", matchorder.OrderID, "activeAddr", or.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", payload.Price)

	accMatch, err := LoadSpotAccount(matchorder.Addr, matchorder.GetLimitOrder().Order.AccountID, a.statedb)
	if err != nil {
		return nil, nil, err
	}
	maker := spotMaker{
		spotTrader: spotTrader{
			acc:     accMatch,
			order:   matchorder,
			feeRate: matchorder.GetRate(),
			cfg:     a.api.GetConfig(),
		},
	}

	logs, kvs, err = taker.Trade(&maker)
	re = taker.re
	return logs, kvs, nil
}
