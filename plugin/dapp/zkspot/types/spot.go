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
	TyNftOrderAction
	TyNftTakerOrderAction
	TyNftOrder2Action
	TyNftTakerOrder2Action

	NameLimitOrderAction         = "LimitOrder"
	NameMarketOrderAction        = "MarketOrder"
	NameRevokeOrderAction        = "RevokeOrder"
	NameExchangeBindAction       = "ExchangeBind"
	NameEntrustOrderAction       = "EntrustOrder"
	NameEntrustRevokeOrderAction = "EntrustRevokeOrder"
	NameNftOrderAction           = "NftOrder"
	NameNftTakerOrderAction      = "NftTakerOrder"
	NameNftOrder2Action          = "NftOrder2"
	NameNftTakerOrder2Action     = "NftTakerOrder2"

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

	TyNftOrderLog = iota + 1100
	TyNftTakerOrderLog
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

	QeuryCountLmit = 20
	PriceLimit     = 1e16
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
	cfg.RegisterDappFork(ExecName, ForkParamV1, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV2, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV3, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV4, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV5, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV6, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV7, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV8, 0)
	cfg.RegisterDappFork(ExecName, ForkParamV9, 0)
}

// config part
var MverPrefix = "mver.exec.sub." + ExecName // [mver.exec.sub.zkspot]

//CheckPrice price  1<=price<=1e16
func CheckPrice(price int64) bool {
	if price > int64(PriceLimit) || price < 1 {
		return false
	}
	return true
}

//CheckOp ...
func CheckOp(op int32) bool {
	if op == OpBuy || op == OpSell {
		return true
	}
	return false
}

//CheckCount ...
func CheckCount(count int32) bool {
	return count <= QeuryCountLmit && count >= 0
}

//CheckAmount 最小交易 1coin
func CheckAmount(amount, coinPrecision int64) bool {
	if amount < 1 || amount >= types.MaxCoin*coinPrecision {
		return false
	}
	return true
}

//CheckDirection ...
func CheckDirection(direction int32) bool {
	if direction == ListASC || direction == ListDESC {
		return true
	}
	return false
}

//CheckStatus ...
func CheckStatus(status int32) bool {
	if status == Ordered || status == Completed || status == Revoked {
		return true
	}
	return false
}

//CheckExchangeAsset
func CheckExchangeAsset(coinExec string, left, right uint64) bool {
	if left == right {
		return false
	}
	return true
}

func CheckLimitOrder(cfg *types.Chain33Config, limitOrder *SpotLimitOrder) error {
	left := limitOrder.GetLeftAsset()
	right := limitOrder.GetRightAsset()
	price := limitOrder.GetPrice()
	amount := limitOrder.GetAmount()
	op := limitOrder.GetOp()
	if !CheckExchangeAsset(cfg.GetCoinExec(), left, right) {
		return ErrAsset
	}
	if !CheckPrice(price) {
		return ErrAssetPrice
	}
	if !CheckAmount(amount, cfg.GetCoinPrecision()) {
		return ErrAssetAmount
	}
	if !CheckOp(op) {
		return ErrAssetOp
	}
	return nil
}

func CheckNftOrder(cfg *types.Chain33Config, limitOrder *SpotNftOrder) error {
	left := limitOrder.GetLeftAsset()
	right := limitOrder.GetRightAsset()
	price := limitOrder.GetPrice()
	amount := limitOrder.GetAmount()
	if !CheckExchangeAsset(cfg.GetCoinExec(), left, right) {
		return ErrAsset
	}
	if !CheckPrice(price) {
		return ErrAssetPrice
	}
	if !CheckAmount(amount, cfg.GetCoinPrecision()) {
		return ErrAssetAmount
	}
	if !(CheckIsNFTToken(left) && CheckIsNFTToken(right)) {
		return ErrAsset
	}
	return nil
}

func CheckNftOrder2(cfg *types.Chain33Config, limitOrder *SpotNftOrder) error {
	//left := limitOrder.GetLeftAsset()
	right := limitOrder.GetRightAsset()
	price := limitOrder.GetPrice()
	amount := limitOrder.GetAmount()
	//if !CheckExchangeAsset(cfg.GetCoinExec(), left, right) {
	//	return ErrAsset
	//}
	if !CheckPrice(price) {
		return ErrAssetPrice
	}
	if !CheckAmount(amount, cfg.GetCoinPrecision()) {
		return ErrAssetAmount
	}
	if CheckIsNFTToken(right) {
		return ErrAsset
	}
	return nil
}
