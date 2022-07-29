package spot

import (
	"math/big"

	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

type NftTrader struct {
	cfg   *types.Chain33Config
	acc   *DexAccount
	order *spotOrder
	fee   *SpotFee
	//takerFee int32
	//makerFee int32

	//
	matches *et.ReceiptSpotMatch
	accFee  *DexAccount
}

func (s *NftTrader) GetOrder() *spotOrder {
	return s.order
}

func (s *NftTrader) GetAccout() *DexAccount {
	return s.acc
}

func (s *NftTrader) CheckTokenAmountForLimitOrder(tid uint64, total int64) error {
	if s.acc.getBalance(tid) < uint64(total) {
		elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "b", s.acc.getBalance(tid), "need", total)
		return et.ErrAssetBalance
	}
	return nil
}

// precision = 1
func (s *NftTrader) FrozenForNftOrder() (*types.Receipt, error) {
	asset, amount := s.order.order.GetNftOrder().LeftAsset, s.order.order.GetBalance()

	receipt, err := s.acc.Frozen(asset, uint64(amount))
	if err != nil {
		elog.Error("FrozenForNftOrder", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", amount)
		return nil, et.ErrAssetBalance
	}
	return receipt, nil
}

func (s *NftTrader) Trade(maker *NftTrader) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(maker.order.order)
	matchDetail := s.calcTradeInfo(maker, balance)

	receipt3, kvs3, err := maker.makerOrderTraded(matchDetail, s.order.order.UpdateTime)
	if err != nil {
		elog.Error("maker.orderTraded", "err", err)
		return receipt3, kvs3, err
	}

	receipt2, kvs2, err := s.takerOrderTraded(matchDetail, maker.order.order)
	if err != nil {
		elog.Error("taker.orderTraded", "err", err)
		return receipt2, kvs2, err
	}

	receipt, kvs, err := s.settlement(maker, matchDetail)
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

//
func (s *NftTrader) calcTradeBalance(order *et.SpotOrder) int64 {
	return order.GetBalance()
}

func nftSafeMul(x *big.Int, y, coinPrecision int64) *big.Int {
	res := big.NewInt(0).Mul(x, big.NewInt(y))
	return big.NewInt(0).Div(res, big.NewInt(coinPrecision))
}

func (s *NftTrader) calcTradeInfo(maker *NftTrader, balance int64) *et.MatchInfo {
	price := maker.order.order.GetNftOrder().Price
	var info et.MatchInfo
	info.Matched = balance
	info.LeftBalance = balance
	info.Price = price
	info.RightBalance = SafeMul(info.Price, balance, 1)
	info.FeeTaker = SafeMul(info.RightBalance, int64(s.order.order.TakerRate), s.cfg.GetCoinPrecision())
	info.FeeMaker = SafeMul(info.RightBalance, int64(maker.order.order.Rate), s.cfg.GetCoinPrecision())
	return &info
}

// settlement
// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
func (s *NftTrader) settlement(maker *NftTrader, tradeBalance *et.MatchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	if s.acc.acc.Addr == maker.acc.acc.Addr {
		return s.selfSettlement(maker, tradeBalance)
	}

	copyAcc := dupAccount(s.acc.acc)
	copyAccMaker := dupAccount(maker.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := maker.order.order.GetNftOrder().LeftAsset, maker.order.order.GetNftOrder().RightAsset
	var err error
	// op always := et.OpBuy
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
func (s *NftTrader) selfSettlement(maker *NftTrader, tradeBalance *et.MatchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
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

func (s *NftTrader) orderUpdate(matchDetail *et.MatchInfo) {
	s.order.orderUpdate(matchDetail)
}

func (s *NftTrader) takerOrderTraded(matchDetail *et.MatchInfo, order *et.SpotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	s.orderUpdate(matchDetail)
	s.matches.Order = s.order.order
	s.matches.MatchOrders = append(s.matches.MatchOrders, order)
	// receipt-log, order-kvs 在匹配完成后一次性生成, 不需要生成多次
	// kvs := GetOrderKvSet(s.order)
	// logs += s.matches
	return []*types.ReceiptLog{}, []*types.KeyValue{}, nil
}

// 2 -> 1 update, 2 kv
func (m *NftTrader) makerOrderTraded(matchDetail *et.MatchInfo, blocktime int64) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	m.orderUpdate(matchDetail)
	m.order.order.UpdateTime = blocktime
	kvs := m.order.repo.GetOrderKvSet(m.order.order)
	return []*types.ReceiptLog{}, kvs, nil
}
