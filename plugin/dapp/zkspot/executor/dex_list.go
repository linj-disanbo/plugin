package executor

import (
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// 清单, 做为零知识证明和各个交易所之间沟通的桥梁
// 如1 存款,
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

type TodoItem struct {
	ty         int
	oneofvalue interface{}
}

type TodoList struct {
	items []TodoItem
}

// A (accountid = 1) 存入 1000 usdt (id = 1)
func SampleDeposit() TodoList {
	b := et.DexAccountBalance{
		Id:      1,    // USDT
		Balance: 1000, // 存入 1000USDT
		Frozen:  0,
	}

	acc := et.DexAccount{
		Id:      666,
		Addr:    "0x12334567",
		Balance: []*et.DexAccountBalance{&b},
	}
	i := TodoItem{ty: ListZkDeposit, oneofvalue: acc}
	ll := TodoList{items: []TodoItem{i}}
	return ll
}

type TMatch struct {
	acc  et.DexAccount
	acc2 et.DexAccount
	got  et.DexAccountBalance
	gave et.DexAccountBalance
}

type TFee struct {
	acc       et.DexAccount
	fee       et.DexAccountBalance
	sysFeeAcc et.DexAccount
}

type TMatchPkg struct {
	tx1, tx2 string // tx hash
	match    TMatch
	f1, f2   TFee
}

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
func SampleSpotMatch() TodoList {
	fee1 := et.DexAccountBalance{
		Id:      1, // USDT
		Balance: 1, // 1USDT
		Frozen:  0,
	}

	fee2 := et.DexAccountBalance{
		Id:      1, // USDT
		Balance: 0,
		Frozen:  1, // 挂单币是锁定的 1USDT, 对零知识证明, 可以直接将值调整到 Balance 中
	}

	got := et.DexAccountBalance{
		Id:      1, // USDT
		Balance: 100,
		Frozen:  0,
	}

	gave := et.DexAccountBalance{
		Id:      2,
		Balance: 1,
		Frozen:  0,
	}

	accA := et.DexAccount{
		Id:   666, //
		Addr: "0x12334567",
	}

	accB := et.DexAccount{
		Id:   777, //
		Addr: "0x777",
	}

	accsys := et.DexAccount{
		Id:   111, //
		Addr: "0xSys",
	}

	f1 := TFee{
		acc:       accA,
		sysFeeAcc: accsys,
		fee:       fee1,
	}
	f2 := TFee{
		acc:       accB,
		sysFeeAcc: accsys,
		fee:       fee2,
	}
	m := TMatch{
		acc:  accA,
		acc2: accB,
		got:  got,
		gave: gave,
	}

	match := TodoItem{ty: ListSpotMatch, oneofvalue: TMatchPkg{
		match: m, f1: f1, f2: f2,
	}}
	ll := TodoList{items: []TodoItem{match}}
	return ll
}

// 撮合 包含 1个交换, 和两个手续费
// 币的源头是是从balance/frozen 中转 看balance 的中值是否为frozen
// 币的目的一般到 balance即可, 如果有到frozen的 提供额外的函数或参数
