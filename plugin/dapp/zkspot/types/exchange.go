package types

import (
	"fmt"

	"github.com/33cn/chain33/types"
)

/*
 * 交易相关类型定义
 * 交易action通常有对应的log结构，用于交易回执日志记录
 * 每一种action和log需要用id数值和name名称加以区分
 */

// action类型id和name，这些常量可以自定义修改
const (
	TyUnknowAction = iota + 200
	TyLimitOrderAction
	TyMarketOrderAction
	TyRevokeOrderAction
	TyExchangeBindAction
	TyEntrustOrderAction
	TyEntrustRevokeOrderAction

	NameLimitOrderAction         = "LimitOrder"
	NameMarketOrderAction        = "MarketOrder"
	NameRevokeOrderAction        = "RevokeOrder"
	NameExchangeBindAction       = "ExchangeBind"
	NameEntrustOrderAction       = "EntrustOrder"
	NameEntrustRevokeOrderAction = "EntrustRevokeOrder"

	FuncNameQueryMarketDepth      = "QueryMarketDepth"
	FuncNameQueryHistoryOrderList = "QueryHistoryOrderList"
	FuncNameQueryOrder            = "QueryOrder"
	FuncNameQueryOrderList        = "QueryOrderList"
)

// log类型id值
const (
	TyUnknowLog = iota + 2000
	TyLimitOrderLog
	TyMarketOrderLog
	TyRevokeOrderLog

	TyExchangeBindLog
	TxSpotTradeLog

	// account logs
	TyDexAccountFrozen
)

// OP
const (
	OpBuy = iota + 1
	OpSell
)

//order status
const (
	Ordered = iota
	Completed
	Revoked
)

//const
const (
	ListDESC = int32(0)
	ListASC  = int32(1)
	ListSeek = int32(2)
)

const (
	//Count 单次list还回条数
	Count = int32(10)
	//MaxMatchCount 系统最大撮合深度
	MaxMatchCount = 100
)

var (
	//定义log的id和具体log类型及名称，填入具体自定义log类型
	//ForkFix Forks
	//ForkFix1 = "ForkFix1"

	ForkParamV1 = "ForkParamV1"
	ForkParamV2 = "ForkParamV2"
	ForkParamV3 = "ForkParamV3"
	ForkParamV4 = "ForkParamV4"
	ForkParamV5 = "ForkParamV5"
	ForkParamV6 = "ForkParamV6"
	ForkParamV7 = "ForkParamV7"
	ForkParamV8 = "ForkParamV8"
	ForkParamV9 = "ForkParamV9"
)

// SpotInitFork defines register fork
func SpotInitFork(cfg *types.Chain33Config) {
	//cfg.RegisterDappFork(ExchangeX, ForkFix1, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV1, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV2, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV3, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV4, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV5, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV6, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV7, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV8, 0)
	cfg.RegisterDappFork(Zksync, ForkParamV9, 0)
}

var MverPrefix = "mver.exec.sub." + Zksync // [mver.exec.sub.exchange]

type Econfig struct {
	Banks     []string
	Coins     []CoinCfg
	Exchanges map[string]*Trade // 现货交易、杠杠交易
}

type CoinCfg struct {
	Coin   string
	Execer string
	Name   string
}

// 交易对配置
type Trade struct {
	Symbol       string
	PriceDigits  int32
	AmountDigits int32
	Taker        int32
	Maker        int32
	MinFee       int64
}

func (f *Econfig) GetFeeAddr() string {
	return f.Banks[0]
}

func (f *Econfig) GetFeeAddrID() uint64 {
	return 1
}

func (f *Econfig) GetCoinName(asset *Asset) string {
	for _, v := range f.Coins {
		if v.Coin == asset.GetSymbol() && v.Execer == asset.GetExecer() {
			return v.Name
		}
	}
	return asset.Symbol
}

// TODO
func (f *Econfig) GetSymbol(left, right uint32) string {
	return fmt.Sprintf("%v_%v", left, right)
}

func (f *Econfig) GetTrade(left, right uint32) *Trade {
	symbol := f.GetSymbol(left, right)
	c, ok := f.Exchanges[symbol]
	if !ok {
		return nil
	}
	return c
}

func (t *Trade) GetPriceDigits() int32 {
	if t == nil {
		return 0
	}
	return t.PriceDigits
}

func (t *Trade) GetAmountDigits() int32 {
	if t == nil {
		return 0
	}
	return t.AmountDigits
}

func (t *Trade) GetTaker() int32 {
	if t == nil {
		return 100000
	}
	return t.Taker
}

func (t *Trade) GetMaker() int32 {
	if t == nil {
		return 100000
	}
	return t.Maker
}

func (t *Trade) GetMinFee() int64 {
	if t == nil {
		return 0
	}
	return t.MinFee
}
