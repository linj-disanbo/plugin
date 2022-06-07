package executor

import (
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	exchangetypes "github.com/33cn/plugin/plugin/dapp/zkspot/types"
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
