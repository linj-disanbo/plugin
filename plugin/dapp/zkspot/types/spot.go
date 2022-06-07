package types

import (
	"github.com/33cn/chain33/types"
)

/*
 * 交易相关类型定义
 * 交易action通常有对应的log结构，用于交易回执日志记录
 * 每一种action和log需要用id数值和name名称加以区分
 */

// action类型id和name，这些常量可以自定义修改
const (
	TySpotNilAction = iota + 1000
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
	TySpotUnknowLog = iota + 1000
	TyLimitOrderLog
	TyMarketOrderLog
	TyRevokeOrderLog

	TyExchangeBindLog
	TySpotTradeLog

	// account logs
	TyDexAccountFrozen
	TyDexAccountActive
	TyDexAccountBurn
	TyDexAccountMint
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

// config part
var MverPrefix = "mver.exec.sub." + ExecName // [mver.exec.sub.zkspot]
