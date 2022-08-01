package spot

import (
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// 同步处理 order account 变化， 并收取fee
type NftSpotTraderHelper struct {
	cfg   *types.Chain33Config
	acc   *DexAccount
	order *spotOrder
	fee   *SpotFee

	matches  *et.ReceiptSpotMatch
	accFee   *DexAccount
	execAddr string
}

func (s *NftSpotTraderHelper) Trade(spot *NftSpot, makerOrder *spotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(makerOrder.order)
	matchDetail := s.calcTradeInfo(makerOrder, balance)

	receipt3, kvs3, err := makerOrder.Traded(matchDetail, s.order.order.UpdateTime)
	if err != nil {
		elog.Error("maker.orderTraded", "err", err)
		return receipt3, kvs3, err
	}

	receipt2, kvs2, err := s.takerOrderTraded(matchDetail, makerOrder.order)
	if err != nil {
		elog.Error("taker.orderTraded", "err", err)
		return receipt2, kvs2, err
	}

	makerNftAcc, err := spot.leftAccDb.NewAccount(makerOrder.order.Addr, uint64(makerOrder.order.GetNftOrder().Order.AccountID), nil) // TODO makerOrder.order.GetNftOrder().LeftAsset)
	if err != nil {
		elog.Error("load maker nft account", "err", err)
		return receipt2, kvs2, err
	}
	takerNftAcc, err := spot.leftAccDb.NewAccount(s.order.order.Addr, uint64(s.order.order.GetNftTakerOrder().Order.AccountID), nil) // TODO makerOrder.order.GetNftOrder().LeftAsset)
	if err != nil {
		elog.Error("load taker nft account", "err", err)
		return receipt2, kvs2, err
	}

	makerAcc, err := spot.accountdb.LoadAccount(s.order.order.Addr, uint64(s.order.order.GetNftTakerOrder().Order.AccountID))
	if err != nil {
		elog.Error("load makerAcc  account", "err", err)
		return receipt2, kvs2, err
	}

	receipt, kvs, err := s.settlement(takerNftAcc, makerNftAcc, makerAcc, matchDetail, makerOrder)
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

func (s *NftSpotTraderHelper) calcTradeBalance(order *et.SpotOrder) int64 {
	return order.GetBalance()
}

func (s *NftSpotTraderHelper) calcTradeInfo(makerOrder *spotOrder, balance int64) *et.MatchInfo {
	price := makerOrder.order.GetNftOrder().Price
	var info et.MatchInfo
	info.Matched = balance
	info.LeftBalance = balance
	info.Price = price
	info.RightBalance = SafeMul(info.Price, balance, 1)
	info.FeeTaker = SafeMul(info.RightBalance, int64(s.order.order.TakerRate), s.cfg.GetCoinPrecision())
	info.FeeMaker = SafeMul(info.RightBalance, int64(makerOrder.order.Rate), s.cfg.GetCoinPrecision())
	return &info
}

// settlement
// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
func (s *NftSpotTraderHelper) settlement(takerNft, makerNft *NftAccount, makerAcc *DexAccount, tradeBalance *et.MatchInfo, makerOrder *spotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	// not support taker/maker the same user
	if s.acc.acc.Id == makerNft.accid {
		return nil, nil, types.ErrNotSupport
	}

	copyAcc := dupAccount(s.acc.acc)
	copyAccMaker := dupAccount(makerAcc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	_, rightToken := makerOrder.order.GetNftOrder().LeftAsset, makerOrder.order.GetNftOrder().RightAsset
	var err error
	// op always := et.OpBuy
	err = s.acc.doTranfer(makerAcc, rightToken, uint64(tradeBalance.RightBalance))
	if err != nil {
		elog.Error("settlement", "buy.doTranfer1", err)
		return nil, nil, err
	}
	receiptN, err := makerNft.accdb.accdb.ExecTransferFrozen(makerNft.address, takerNft.address, s.execAddr, tradeBalance.LeftBalance)
	if err != nil {
		elog.Error("settlement", "buy.doFrozenTranfer2", err)
		return nil, nil, err
	}
	err = s.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeTaker))
	if err != nil {
		elog.Error("settlement", "buy-fee.doTranfer1", err)
		return nil, nil, err
	}
	err = makerAcc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeMaker))
	if err != nil {
		elog.Error("settlement", "buy-fee.doTranfer2", err)
		return nil, nil, err
	}

	kvs1 := s.acc.GetKVSet()
	kvs2 := makerAcc.GetKVSet()
	kvs3 := s.accFee.GetKVSet()

	kvs1 = append(kvs1, kvs2...)
	kvs1 = append(kvs1, kvs3...)
	kvs1 = append(kvs1, receiptN.KV...)

	re := et.ReceiptSpotTrade{
		Match: tradeBalance,
		Prev: &et.TradeAccounts{
			Taker: copyAcc,
			Maker: copyAccMaker,
			Fee:   copyFeeAcc,
		},
		Current: &et.TradeAccounts{
			Taker: s.acc.acc,
			Maker: makerAcc.acc,
			Fee:   s.accFee.acc,
		},
		MakerOrder: makerOrder.order.GetNftOrder().Order,
	}

	log1 := types.ReceiptLog{
		Ty:  et.TySpotTradeLog,
		Log: types.Encode(&re),
	}
	var logs []*types.ReceiptLog
	logs = append(logs, &log1)
	logs = append(logs, receiptN.Logs...)

	return []*types.ReceiptLog{&log1}, kvs1, nil
}

func (s *NftSpotTraderHelper) takerOrderTraded(matchDetail *et.MatchInfo, order *et.SpotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	s.order.orderUpdate(matchDetail)
	s.matches.Order = s.order.order
	s.matches.MatchOrders = append(s.matches.MatchOrders, order)
	// receipt-log, order-kvs 在匹配完成后一次性生成, 不需要生成多次
	// kvs := GetOrderKvSet(s.order)
	// logs += s.matches
	return []*types.ReceiptLog{}, []*types.KeyValue{}, nil
}

func (s *NftSpotTraderHelper) CheckTokenAmountForLimitOrder(tid uint64, total int64) error {
	if s.acc.getBalance(tid) < uint64(total) {
		elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "b", s.acc.getBalance(tid), "need", total)
		return et.ErrAssetBalance
	}
	return nil
}
