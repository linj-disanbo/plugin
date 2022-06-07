package executor

import (
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	exchangetypes "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

/*
 * 执行器相关定义
 * 重载基类相关接口
 */

var (
	//日志
	elog = log.New("module", et.ExecName+".executor")
)

// CheckTx 实现自定义检验交易接口，供框架调用
func SpotCheckTx(cfg *types.Chain33Config, tx *types.Transaction, index int) error {
	//发送交易的时候就检查payload,做严格的参数检查
	var exchange exchangetypes.SpotAction
	types.Decode(tx.GetPayload(), &exchange)
	if exchange.Ty == exchangetypes.TyLimitOrderAction {
		limitOrder := exchange.GetLimitOrder()
		return et.CheckLimitOrder(cfg, limitOrder)
	}
	if exchange.Ty == exchangetypes.TyMarketOrderAction {
		return types.ErrActionNotSupport
	}
	return nil
}
