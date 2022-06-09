package spot

import (
	"encoding/hex"
	"reflect"

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
	order *et.SpotOrder
	fee   *spotFee
	//takerFee int32
	//makerFee int32

	//
	matches *et.ReceiptSpotMatch
	accFee  *DexAccount
}

func (s *SpotTrader) GetOrder() *et.SpotOrder {
	return s.order
}

type spotMaker struct {
	SpotTrader
}

// buy 按最大量判断余额是否够
// 因为在吃单时, 价格是变动的, 所以实际锁定的量是会浮动的
// 实现上, 按最大量判断余额是否够, 在成交时, 按实际需要量扣除. 最后变成挂单时, 进行锁定
func (s *SpotTrader) CheckTokenAmountForLimitOrder(order *et.SpotOrder) error {
	precision := s.cfg.GetCoinPrecision()
	or := order.GetLimitOrder()
	if or.GetOp() == et.OpBuy {
		amount := SafeMul(or.GetAmount(), or.GetPrice(), precision)
		fee := calcMtfFee(amount, int32(order.TakerRate))
		total := SafeAdd(amount, int64(fee))

		if s.acc.getBalance(or.RightAsset) < uint64(total) {
			elog.Error("limit check right balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "b", s.acc.getBalance(or.RightAsset), "need", total)
			return et.ErrAssetBalance
		}
		return nil
	}

	/* if payload.GetOp() == et.OpSell */
	amount := or.GetAmount()
	if s.acc.getBalance(or.LeftAsset) < uint64(amount) {
		elog.Error("limit check left balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "b", s.acc.getBalance(or.LeftAsset), "need", amount)
		return et.ErrAssetBalance
	}

	return nil
}

func (s *SpotTrader) FrozenForLimitOrder() (*types.Receipt, error) {
	or := s.order.GetLimitOrder()
	if or.GetOp() == et.OpSell {
		receipt, err := s.acc.Frozen(or.LeftAsset, uint64(s.order.Balance))
		if err != nil {
			elog.Error("limit frozen left balance", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", s.order.Balance)
			return nil, et.ErrAssetBalance
		}
		return receipt, err
	}

	precision := s.cfg.GetCoinPrecision()
	amount := SafeMul(s.order.Balance, or.GetPrice(), precision)
	fee := calcMtfFee(amount, int32(s.order.Rate))
	total := SafeAdd(amount, fee)

	receipt, err := s.acc.Frozen(or.RightAsset, uint64(total))
	if err != nil {
		elog.Error("FrozenForLimitOrder", "addr", s.acc.acc.Addr, "avail", s.acc.acc.Balance, "need", total)
		return nil, et.ErrAssetBalance
	}
	return receipt, nil
}

func (s *SpotTrader) Trade(maker *spotMaker) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	balance := s.calcTradeBalance(maker.order)
	matchDetail := s.calcTradeInfo(maker, balance)

	receipt3, kvs3, err := maker.orderTraded(&matchDetail, s.order)
	if err != nil {
		elog.Error("maker.orderTraded", "err", err)
		return receipt3, kvs3, err
	}

	receipt2, kvs2, err := s.orderTraded(&matchDetail, maker.order)
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
	if order.GetBalance() >= s.order.GetBalance() {
		return s.order.GetBalance()
	}
	return order.GetBalance()
}

func (s *SpotTrader) calcTradeInfo(maker *spotMaker, balance int64) et.MatchInfo {
	var info et.MatchInfo
	info.Matched = balance
	info.LeftBalance = balance
	info.RightBalance = SafeMul(balance, maker.order.GetLimitOrder().Price, s.cfg.GetCoinPrecision())
	info.FeeTaker = SafeMul(info.RightBalance, int64(s.order.TakerRate), s.cfg.GetCoinPrecision())
	info.FeeMaker = SafeMul(info.RightBalance, int64(maker.order.Rate), s.cfg.GetCoinPrecision())
	return info
}

// settlement
// LeftToken: seller -> buyer
// RightToken: buyer -> seller
// RightToken: buyer, seller -> fee-bank
func (s *SpotTrader) settlement(maker *spotMaker, tradeBalance *et.MatchInfo) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	if s.acc.acc.Addr == maker.acc.acc.Addr {
		return s.selfSettlement(maker, tradeBalance)
	}

	copyAcc := dupAccount(s.acc.acc)
	copyAccMaker := dupAccount(maker.acc.acc)
	copyFeeAcc := dupAccount(s.accFee.acc)

	leftToken, rightToken := s.order.GetLimitOrder().LeftAsset, s.order.GetLimitOrder().RightAsset
	var err error
	if s.order.GetLimitOrder().Op == et.OpSell {
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
		MakerOrder: maker.order.GetLimitOrder().Order,
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

	leftToken, rightToken := s.order.GetLimitOrder().LeftAsset, s.order.GetLimitOrder().RightAsset
	err := s.acc.doActive(leftToken, uint64(tradeBalance.LeftBalance))
	if err != nil {
		return nil, nil, err
	}
	// taker 是buy, takerFee是活动的, makerFee 是活动的
	// taker 是sell, takerFee是活动的, makerFee 是冻结的
	rightAmount := tradeBalance.RightBalance
	if s.order.GetLimitOrder().Op == et.OpSell {
		rightAmount += tradeBalance.FeeMaker
	}
	err = s.acc.doActive(rightToken, uint64(rightAmount))
	if err != nil {
		return nil, nil, err
	}
	err = s.acc.doTranfer(s.accFee, rightToken, uint64(tradeBalance.FeeTaker+tradeBalance.FeeMaker))
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
		MakerOrder: maker.order.GetLimitOrder().Order,
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
	s.order.DigestedFee += matchDetail.FeeTaker
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

	s.matches.Order = s.order
	s.matches.MatchOrders = append(s.matches.MatchOrders, order)
	// receipt-log, order-kvs 在匹配完成后一次性生成, 不需要生成多次
	// kvs := GetOrderKvSet(s.order)
	// logs += s.matches
	return []*types.ReceiptLog{}, []*types.KeyValue{}, nil
}

func (m *spotMaker) orderTraded(matchDetail *et.MatchInfo, takerOrder *et.SpotOrder) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	matched := matchDetail.Matched

	// fee and AVGPrice
	m.order.DigestedFee += matchDetail.FeeMaker
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

func (m *matcher) matchModel(matchorder *et.SpotOrder, taker *SpotTrader) ([]*types.ReceiptLog, []*types.KeyValue, error) {
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
		SpotTrader: SpotTrader{
			acc:   accMatch,
			order: matchorder,
			cfg:   m.api.GetConfig(),
		},
	}

	logs, kvs, err = taker.Trade(&maker)
	elog.Info("try match2", "activeId", taker.order.OrderID, "passiveId", matchorder.OrderID, "activeAddr", taker.order.Addr, "passiveAddr",
		matchorder.Addr, "amount", matched, "price", taker.order.GetLimitOrder().Price)
	return logs, kvs, err
}

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

func GetOrderKvSet(order *et.SpotOrder) (kvset []*types.KeyValue) {
	kvset = append(kvset, &types.KeyValue{Key: calcOrderKey(order.OrderID), Value: types.Encode(order)})
	return kvset
}

func (a *Spot) LoadUser(fromaddr string, accountID uint64) (*SpotTrader, error) {
	acc, err := LoadSpotAccount(fromaddr, accountID, a.env.GetStateDB())
	if err != nil {
		elog.Error("executor/exchangedb LoadSpotAccount load taker account", "err", err)
		return nil, err
	}

	return &SpotTrader{
		acc: acc,
		cfg: a.env.GetAPI().GetConfig(),
	}, nil
}

func (a *Spot) CreateLimitOrder(fromaddr string, acc *SpotTrader, payload *et.SpotLimitOrder, entrustAddr string) (*et.SpotOrder, error) {
	fees, err := a.GetSpotFee(fromaddr, payload.LeftAsset, payload.RightAsset)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	acc.fee = fees

	order := createLimitOrder(payload, entrustAddr,
		[]orderInit{a.initLimitOrder(), fees.initLimitOrder()})

	err = acc.CheckTokenAmountForLimitOrder(order)
	if err != nil {
		return nil, err
	}
	acc.order = order

	return order, nil
}

//GetIndex get index
func (a *Spot) GetIndex() int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (a.env.GetHeight()*types.MaxTxsPerBlock + int64(a.tx.Index)) * 1e4
}

func (a *Spot) initLimitOrder() func(*et.SpotOrder) *et.SpotOrder {
	return func(order *et.SpotOrder) *et.SpotOrder {
		order.OrderID = a.GetIndex()
		order.Index = a.GetIndex()
		order.CreateTime = a.env.GetBlockTime()
		order.UpdateTime = a.env.GetBlockTime()
		order.Hash = hex.EncodeToString(a.tx.Hash)
		order.Addr = a.tx.From
		return order
	}
}

// config

func ParseConfig(cfg *types.Chain33Config, height int64) (*et.Econfig, error) {
	banks, err := ParseStrings(cfg, "banks", height)
	if err != nil || len(banks) == 0 {
		return nil, err
	}
	coins, err := ParseCoins(cfg, "coins", height)
	if err != nil {
		return nil, err
	}
	exchanges, err := ParseSymbols(cfg, "exchanges", height)
	if err != nil {
		return nil, err
	}
	return &et.Econfig{
		Banks:     banks,
		Coins:     coins,
		Exchanges: exchanges,
	}, nil
}

func ParseStrings(cfg *types.Chain33Config, tradeKey string, height int64) (ret []string, err error) {
	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid val", "val", val, "key", tradeKey)
		return nil, et.ErrCfgFmt
	}

	for _, v := range datas {
		one, ok := v.(string)
		if !ok {
			elog.Error("invalid one", "one", one, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}
		ret = append(ret, one)
	}
	return
}

func ParseCoins(cfg *types.Chain33Config, tradeKey string, height int64) (coins []et.CoinCfg, err error) {
	coins = make([]et.CoinCfg, 0)

	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid coins", "val", val, "type", reflect.TypeOf(val))
		return nil, et.ErrCfgFmt
	}

	for _, e := range datas {
		v, ok := e.(map[string]interface{})
		if !ok {
			elog.Error("invalid coins one", "one", v, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}

		coin := et.CoinCfg{
			Coin:   v["coin"].(string),
			Execer: v["execer"].(string),
			Name:   v["name"].(string),
		}
		coins = append(coins, coin)
	}
	return
}

func ParseSymbols(cfg *types.Chain33Config, tradeKey string, height int64) (symbols map[string]*et.Trade, err error) {
	symbols = make(map[string]*et.Trade)

	val, err := cfg.MG(et.MverPrefix+"."+tradeKey, height)
	if err != nil {
		return nil, err
	}

	datas, ok := val.([]interface{})
	if !ok {
		elog.Error("invalid Symbols", "val", val, "type", reflect.TypeOf(val))
		return nil, et.ErrCfgFmt
	}

	for _, e := range datas {
		v, ok := e.(map[string]interface{})
		if !ok {
			elog.Error("invalid Symbols one", "one", v, "key", tradeKey)
			return nil, et.ErrCfgFmt
		}

		symbol := v["symbol"].(string)
		symbols[symbol] = &et.Trade{
			Symbol:       symbol,
			PriceDigits:  int32(formatInterface(v["priceDigits"])),
			AmountDigits: int32(formatInterface(v["amountDigits"])),
			Taker:        int32(formatInterface(v["taker"])),
			Maker:        int32(formatInterface(v["maker"])),
			MinFee:       formatInterface(v["minFee"]),
		}
	}
	return
}

func formatInterface(data interface{}) int64 {
	switch data.(type) {
	case int64:
		return data.(int64)
	case int32:
		return int64(data.(int32))
	case int:
		return int64(data.(int))
	default:
		return 0
	}
}
