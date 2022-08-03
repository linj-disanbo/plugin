package executor

import (
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// 清单, 做为零知识证明和各个交易所之间沟通的桥梁
// 如1 存款/取款/手续费,
//   1. 在零知识证明存款合法后, 在树上对帐号进行存款
//   2. 零知识证明存款生成存款的清单
//   3. 根据清单, 在交易所中进行等额的存款
// 如2 现货撮合
//   1. 在现货合约中, 进行撮合, 如果进行了2次撮合即 A 用户的Tx0, 和B 用户的Tx1 , C用户的Tx2 进行了撮合 (B, C 提交交易时没有撮合, 变成的市场挂单)
//   2. 在现货合约中, 根据撮合结果, 对用户在现货合约中的帐号进行资产的交换和收取手续费
//   3. 在现货合约中, 生成资产调整的清单
//   4. 根据清单, 对零知识证明上对帐号进行调整

const (
	// 存款
	ListZkDeposit = 1
	// 提款
	ListZkWithdraw = 2
	// 现货撮合
	ListSpotMatch = 1001
)

// 相关的交易为 A签名发送的 tx1(卖bty), B签名发送的tx2(买bty)
// 撮合 100usdt 交易 1bty, 并且收取 A B 各 1usdt的手续费

// 撮合
//          A    B  feesysacc
// BTY     -1    +1
// USDT    +100   -100

// 手续费1
//          A    B  feesysacc
// BTY     0    0     0
// USDT    -1   0     +1

// 手续费1
//          A    B  feesysacc
// BTY     0     0        0
// USDT    0     -1       +1

// 结算后状态
//          A    B  feesysacc
// BTY     -1    +1
// USDT    +99   -101  +2

// 是否需要将清单合并成帐号变化
// BTY-id = 2, USDT-id = 1

// 撮合 包含 1个交换, 和两个手续费
// 币的源头是是从balance/frozen 中转 看balance 的中值是否为frozen
// 币的目的一般到 balance即可, 如果有到frozen的 提供额外的函数或参数

func GetSpotMatch(receipt *types.Receipt) *types.Receipt {
	receipt2 := &types.Receipt{Logs: []*types.ReceiptLog{}}
	for _, l := range receipt.Logs {
		if l.Ty != et.TySpotTradeLog {
			continue
		}
		receipt2.Logs = append(receipt2.Logs, l)
	}
	return receipt2
}
