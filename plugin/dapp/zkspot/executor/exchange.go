package executor

import (
	log "github.com/33cn/chain33/common/log/log15"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	exchangetypes "github.com/33cn/plugin/plugin/dapp/exchange/types"
)

/*
 * 执行器相关定义
 * 重载基类相关接口
 */

var (
	//日志
	elog = log.New("module", "exchange.executor")
)

type exchange struct {
	drivers.DriverBase
}

// CheckTx 实现自定义检验交易接口，供框架调用
func (e *exchange) CheckTx(tx *types.Transaction, index int) error {
	//发送交易的时候就检查payload,做严格的参数检查
	var exchange exchangetypes.ExchangeAction
	types.Decode(tx.GetPayload(), &exchange)
	cfg := e.GetAPI().GetConfig()
	if exchange.Ty == exchangetypes.TyLimitOrderAction {
		limitOrder := exchange.GetLimitOrder()
		left := limitOrder.GetLeftAsset()
		right := limitOrder.GetRightAsset()
		price := limitOrder.GetPrice()
		amount := limitOrder.GetAmount()
		op := limitOrder.GetOp()
		if !CheckExchangeAsset(cfg.GetCoinExec(), left, right) {
			return exchangetypes.ErrAsset
		}
		if !CheckPrice(price) {
			return exchangetypes.ErrAssetPrice
		}
		if !CheckAmount(amount, cfg.GetCoinPrecision()) {
			return exchangetypes.ErrAssetAmount
		}
		if !CheckOp(op) {
			return exchangetypes.ErrAssetOp
		}
	}
	if exchange.Ty == exchangetypes.TyMarketOrderAction {
		return types.ErrActionNotSupport
	}
	return nil
}

//ExecutorOrder Exec 的时候 同时执行 ExecLocal
func (e *exchange) ExecutorOrder() int64 {
	return drivers.ExecLocalSameTime
}
