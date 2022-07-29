package executor

import (
	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/33cn/plugin/plugin/dapp/zkspot/executor/spot"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// zkNftDex   swap evmxgo-nft  zkspot-usdt
type zkNftDex struct {
	statedb   dbm.KV
	blocktime int64
	height    int64
	localDB   dbm.KVDB
	api       client.QueueProtocolAPI
	txinfo    *et.TxInfo
	env       *dapp.DriverBase
}

//NewZkNftDex  ...
func NewZkNftDex(e *zkspot, tx *types.Transaction, index int) *zkNftDex {
	return &zkNftDex{
		txinfo:    NewTxInfo(tx, index),
		env:       &e.DriverBase,
		statedb:   e.GetStateDB(),
		blocktime: e.GetBlockTime(),
		height:    e.GetHeight(),
		localDB:   e.GetLocalDB(),
		api:       e.GetAPI(),
	}
}

func (a *zkNftDex) getFeeAcc() (*spot.DexAccount, error) {
	accountID := uint64(et.SystemFeeAccountId)
	z1 := &zktree{}
	leaf, err := z1.getAccount(a.statedb, accountID)
	if err != nil {
		return nil, err
	}
	acc, err := spot.LoadSpotAccount(leaf.ChainAddr, accountID, a.statedb, &dbprefix{})
	if err != nil {
		elog.Error("LoadSpotAccount load taker account", "err", err)
		return nil, err
	}
	return acc, nil
}

// 现在一个交易所, 资金帐号和现货交易所帐号是同一个
// 在多个交易所的情况下, 会有一个资金帐号和多个交易所帐号
// zkNftDex 的 ustd 管理 同 zkSpotDex
// TODO 重提部分， 从交易所中提取出来
// 相关功能 Deposit CalcMaxActive Withdraw

// 委托功能 也同 zkSpotDex,  现在已经有两种不同的方式委托
// zk L1 的委托， dex 中的委托

// exec local 同
// func (a *zkNftDex) execLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {

//NftOrder ...
func (a *zkNftDex) NftOrder(base *dapp.DriverBase, payload *et.SpotNftOrder, entrustAddr string) (*types.Receipt, error) {
	cfg := a.api.GetConfig()
	err := et.CheckNftOrder2(cfg, payload)
	if err != nil {
		return nil, err
	}

	// 这个是为了 卖出的一方有zk L1的账号
	err = checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, err
	}

	spot1, err := spot.NewNftSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	//err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	return spot1.CreateNftOrder(a.txinfo.From, payload, entrustAddr)
}

//NftTakerOrder ...
func (a *zkNftDex) NftTakerOrder(base *dapp.DriverBase, payload *et.SpotNftTakerOrder, entrustAddr string) (*types.Receipt, error) {
	//cfg := a.api.GetConfig()
	err := checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, err
	}

	spot1, err := spot.NewNftSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	//err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	// 下面流程是否要放到 spot1中
	taker, err := spot1.LoadNftTrader(a.txinfo.From, payload.Order.AccountID)
	if err != nil {
		return nil, err
	}

	return spot1.TradeNft(a.txinfo.From, taker, payload, entrustAddr)
}
