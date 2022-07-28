package spot

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
type spotTaker struct {
	SpotTrader
}

type SpotTrader struct {
	cfg   *types.Chain33Config
	acc   *DexAccount
	order *spotOrder
	fee   *spotFee
	//takerFee int32
	//makerFee int32

	//
	matches *et.ReceiptSpotMatch
	accFee  *DexAccount

	// TODO add
	tokenAcc AssetAccount
}

func (s *SpotTrader) GetOrder() *spotOrder {
	return s.order
}

func (s *SpotTrader) GetAccout() *DexAccount {
	return s.acc
}

type spotMaker struct {
	SpotTrader
}

func (s *SpotTrader) CheckTokenAmountForLimitOrder(tid uint64, total int64) error {
	if s.acc.getBalance(tid) < uint64(total) {
		elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "b", s.acc.getBalance(tid), "need", total)
		return et.ErrAssetBalance
	}
	return nil
}

func (s *SpotTrader) FrozenForLimitOrder(orderx *spotOrder) (*types.Receipt, error) {
	precision := s.cfg.GetCoinPrecision()
	asset, amount := orderx.calcFrozenToken(precision)

	receipt, err := s.acc.Frozen(asset.GetZkAssetid(), uint64(amount))
	if err != nil {
		elog.Error("FrozenForLimitOrder", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
		return nil, et.ErrAssetBalance
	}
	return receipt, nil
}

func (s *SpotTrader) Trade(maker *spotMaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(maker.order.order)
	matchDetail := s.calcTradeInfo(maker, balance)

	receipt3, kvs3, err := maker.orderTraded(&matchDetail, s.order.order, nil) // TODO
	if err != nil {
		elog.Error("maker.orderTraded", "err", err)
		return receipt3, kvs3, err
	}

	receipt2, kvs2, err := s.orderTraded(&matchDetail, maker.order.order)
	if err != nil {
		elog.Error("taker.orderTraded", "err", err)
		return receipt2, kvs2, err
	}

	receipt, kvs, err := s.settlement(maker, &matchDetail)
	if err != nil {
		elog.Error("settlement", "err", err)
		return receipt, kvs, err
	}

	kvs = append(kvs, kvs2...)
	kvs = append(kvs, kvs3...)
	receipt = append(receipt, receipt2...)
	receipt = append(receipt, receipt3...)

	return receipt, kvs, nil
}

func (s *SpotTrader) calcTradeBalance(order *et.SpotOrder) int64 {
	if order.GetBalance() >= s.order.order.GetBalance() {
		return s.order.order.GetBalance()
	}
	return order.GetBalance()
}

func (s *SpotTrader) calcTradeInfo(maker *spotMaker, balance int64) et.MatchInfo {
	var info et.MatchInfo
	info.Matched = balance
	info.LeftBalance = balance
	info.RightBalance = SafeMul(balance, maker.order.order.GetLimitOrder().Price, s.cfg.GetCoinPrecision())
	info.FeeTaker = SafeMul(info.RightBalance, int64(s.order.order.TakerRate), s.cfg.GetCoinPrecision())
	info.FeeMaker = SafeMul(info.RightBalance, int64(maker.order.order.Rate), s.cfg.GetCoinPrecision())
	return info
}

// settlement
// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
func (s *SpotTrader) settlement(maker *spotMaker, tradeBalance *et.MatchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	if s.acc.acc.Id == maker.acc.acc.Id {
		return s.selfSettlement(maker, tradeBalance)
	}

	copyAcc := dupAccount(s.acc.acc)
	copyAccMaker := dupAccount(maker.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := s.order.order.GetLimitOrder().LeftAsset, s.order.order.GetLimitOrder().RightAsset
	var err error
	if s.order.order.GetLimitOrder().Op == et.OpSell {
		err = s.acc.doTranfer(maker.acc, leftToken, uint64(tradeBalance.LeftBalance))
		if err != nil {
			elog.Error("settlement", "sell.doTranfer1", err)
			return nil, nil, err
		}
		err = maker.acc.doFrozenTranfer(s.acc, rightToken, uint64(tradeBalance.RightBalance))
		if err != nil {
			elog.Error("settlement", "sell.doFrozenTranfer2", err)
			return nil, nil, err
		}
		err = s.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeTaker))
		if err != nil {
			elog.Error("settlement", "sell-fee.doTranfer", err)
			return nil, nil, err
		}
		err = maker.acc.doFrozenTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeMaker))
		if err != nil {
			elog.Error("settlement", "sell-fee.doFrozenTranfer3", err)
			return nil, nil, err
		}
	} else {
		err = s.acc.doTranfer(maker.acc, rightToken, uint64(tradeBalance.RightBalance))
		if err != nil {
			elog.Error("settlement", "buy.doTranfer1", err)
			return nil, nil, err
		}
		err = maker.acc.doFrozenTranfer(s.acc, leftToken, uint64(tradeBalance.LeftBalance))
		if err != nil {
			elog.Error("settlement", "buy.doFrozenTranfer2", err)
			return nil, nil, err
		}
		err = s.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeTaker))
		if err != nil {
			elog.Error("settlement", "buy-fee.doTranfer1", err)
			return nil, nil, err
		}
		err = maker.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeMaker))
		if err != nil {
			elog.Error("settlement", "buy-fee.doTranfer2", err)
			return nil, nil, err
		}
	}

	kvs1 := s.acc.GetKVSet()
	kvs2 := maker.acc.GetKVSet()
	kvs3 := s.accFee.GetKVSet()

	kvs1 = append(kvs1, kvs2...)
	kvs1 = append(kvs1, kvs3...)

	re := et.ReceiptSpotTrade{
		Match: tradeBalance,
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
		MakerOrder: maker.order.order.GetLimitOrder().Order,
	}

	log1 := types.ReceiptLog{
		Ty:  et.TySpotTradeLog,
		Log: types.Encode(&re),
	}
	return []*types.ReceiptLog{&log1}, kvs1, nil
}

// taker/maker the same user
func (s *SpotTrader) selfSettlement(maker *spotMaker, tradeBalance *et.MatchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	copyAcc := dupAccount(s.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := s.order.order.GetLimitOrder().LeftAsset, s.order.order.GetLimitOrder().RightAsset

	// taker 是buy,  maker 是sell, Left 是冻结的. takerFee + makerFee 是活动的
	// taker 是sell, maker 是 buy, Right 是冻结的. makerFee 是冻结的. takerFee是活动的
	if s.order.order.GetLimitOrder().Op == et.OpSell {
		rightAmount := tradeBalance.RightBalance
		rightAmount += tradeBalance.FeeMaker
		err := s.acc.doActive(rightToken, uint64(rightAmount))
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := s.acc.doActive(leftToken, uint64(tradeBalance.LeftBalance))
		if err != nil {
			return nil, nil, err
		}
	}

	err := s.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeTaker+tradeBalance.FeeMaker))
	if err != nil {
		return nil, nil, err
	}

	kvs1 := s.acc.GetKVSet()
	kvs3 := s.accFee.GetKVSet()
	kvs1 = append(kvs1, kvs3...)

	re := et.ReceiptSpotTrade{
		Match: tradeBalance,
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
		MakerOrder: maker.order.order.GetLimitOrder().Order,
	}

	log1 := types.ReceiptLog{
		Ty:  et.TySpotTradeLog,
		Log: types.Encode(&re),
	}
	return []*types.ReceiptLog{&log1}, kvs1, nil
}

func (s *SpotTrader) orderTraded(matchDetail *et.MatchInfo, order *et.SpotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.Matched

	// fee and AVGPrice
	s.order.order.DigestedFee += matchDetail.FeeTaker
	s.order.order.AVGPrice = caclAVGPrice(s.order.order, s.order.order.GetLimitOrder().Price, matched)

	// status
	if matched == s.order.order.GetBalance() {
		s.order.order.Status = et.Completed
	} else {
		s.order.order.Status = et.Ordered
	}

	// order matched
	s.order.order.Executed = matched
	s.order.order.Balance -= matched

	s.matches.Order = s.order.order
	s.matches.MatchOrders = append(s.matches.MatchOrders, order)
	// receipt-log, order-kvs 在匹配完成后一次性生成, 不需要生成多次
	// kvs := GetOrderKvSet(s.order)
	// logs += s.matches
	return []*types.ReceiptLog{}, []*types.KeyValue{}, nil
}

// 2 -> 1 update, 2 kv
func (m *spotMaker) orderTraded(matchDetail *et.MatchInfo, takerOrder *et.SpotOrder, orderx *spotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.Matched

	// fee and AVGPrice
	m.order.order.DigestedFee += matchDetail.FeeMaker
	m.order.order.AVGPrice = caclAVGPrice(m.order.order, m.order.order.GetLimitOrder().Price, matched)

	m.order.order.UpdateTime = takerOrder.UpdateTime

	// status
	if matched == m.order.order.GetBalance() {
		m.order.order.Status = et.Completed
	} else {
		m.order.order.Status = et.Ordered
	}

	// order matched
	m.order.order.Executed = matched
	m.order.order.Balance -= matched
	kvs := m.order.repo.GetOrderKvSet(m.order.order)
	return []*types.ReceiptLog{}, kvs, nil
}

func (taker *SpotTrader) matchModel(matchorder *et.SpotOrder, statedb dbm.KV) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	matched := taker.calcTradeBalance(matchorder)
	elog.Info("try match", "activeId", taker.order.order.OrderID, "passiveId", matchorder.OrderID, "activeAddr", taker.order.order.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", taker.order.order.GetLimitOrder().Price)

	accMatch, err := newAccountRepo(spotDexName, statedb, taker.acc.db.dbprefix).LoadSpotAccount(matchorder.Addr, matchorder.GetLimitOrder().Order.AccountID)
	if err != nil {
		return nil, nil, err
	}
	maker := spotMaker{
		SpotTrader: SpotTrader{
			acc:   accMatch,
			order: newSpotOrder(matchorder, taker.order.repo),
			cfg:   taker.cfg,
		},
	}

	logs, kvs, err = taker.Trade(&maker)
	elog.Info("try match2", "activeId", taker.order.order.OrderID, "passiveId", matchorder.OrderID, "activeAddr", taker.order.order.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", taker.order.order.GetLimitOrder().Price)
	return logs, kvs, err
}
