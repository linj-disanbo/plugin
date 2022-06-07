package executor

import (
	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/33cn/plugin/plugin/dapp/zkspot/executor/spot"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
	"github.com/pkg/errors"
)

// Action action struct
type SpotAction struct {
	statedb   dbm.KV
	blocktime int64
	height    int64
	localDB   dbm.KVDB
	api       client.QueueProtocolAPI
	txinfo    *et.TxInfo
}

//NewTxInfo ...
func NewTxInfo(tx *types.Transaction, index int) *et.TxInfo {
	return &et.TxInfo{
		Hash:     tx.Hash(),
		From:     tx.From(),
		To:       tx.GetTo(),
		ExecAddr: dapp.ExecAddress(string(tx.Execer)),
		Index:    index,
		Tx:       tx,
	}
}

//NewAction ...
func NewSpotDex(e *zkspot, tx *types.Transaction, index int) *SpotAction {
	return &SpotAction{
		txinfo:    NewTxInfo(tx, index),
		statedb:   e.GetStateDB(),
		blocktime: e.GetBlockTime(),
		height:    e.GetHeight(),
		localDB:   e.GetLocalDB(),
		api:       e.GetAPI(),
	}
}

//GetIndex get index
func (a *SpotAction) GetIndex() int64 {
	// Add four zeros to match multiple MatchOrder indexes
	return (a.height*types.MaxTxsPerBlock + int64(a.txinfo.Index)) * 1e4
}

type zktree struct {
}

func (z *zktree) getAccount(statedb dbm.KV, acccountID uint64) (*zt.Leaf, error) {
	info, err := getTreeUpdateInfo(statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.getTreeUpdateInfo")
	}
	leaf, err := GetLeafByAccountId(statedb, acccountID, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}

	return leaf, nil
}

func (z *zktree) checkAuth(acc *zt.Leaf, pub *zt.ZkPubKey) error {
	err := authVerification(pub, acc.GetPubKey())
	if err != nil {
		return errors.Wrapf(err, "authVerification")
	}
	return nil
}

//LimitOrder ...
func (a *SpotAction) LimitOrder(base *dapp.DriverBase, payload *et.SpotLimitOrder, entrustAddr string) (*types.Receipt, error) {
	cfg := a.api.GetConfig()
	err := et.CheckLimitOrder(cfg, payload)
	if err != nil {
		return nil, err
	}

	var zktree1 zktree
	zkAcc, err := zktree1.getAccount(a.statedb, payload.Order.AccountID)
	if err != nil {
		return nil, err
	}
	err = zktree1.checkAuth(zkAcc, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	spot1 := spot.NewSpot(base, &et.TxInfo{})
	acc, err := spot1.LoadUser(a.txinfo.From, payload.Order.AccountID)
	if err != nil {
		return nil, err
	}

	order, err := spot1.CreateLimitOrder(a.txinfo.From, acc, payload, entrustAddr)
	if err != nil {
		return nil, err
	}
	_ = order // set to order trader

	fees, err := spot1.GetFees(a.txinfo.From, payload.LeftAsset, payload.RightAsset)
	if err != nil {
		elog.Error("executor/exchangedb getFees", "err", err)
		return nil, err
	}
	_ = fees

	accFee, err := spot.LoadSpotAccount("fees.addr", 1 /*fees.id,*/, a.statedb)
	if err != nil {
		elog.Error("executor/exchangedb LoadSpotAccount load fee account", "err", err)
		return nil, err
	}
	_ = accFee
	taker := acc

	receipt1, err := spot1.MatchLimitOrder(payload, entrustAddr, taker)
	if err != nil {
		return nil, err
	}

	if taker.GetOrder().Status != et.Completed {
		receipt3, err := taker.FrozenForLimitOrder()
		if err != nil {
			return nil, err
		}
		receipt1 = mergeReceipt(receipt1, receipt3)
	}
	return receipt1, nil
}

//QueryMarketDepth 这里primaryKey当作主键索引来用，
//The first query does not need to fill in the value, pay according to the price from high to low, selling orders according to the price from low to high query

// 使用 chain33 地址为key
// 同样提供: account 基本和 token 级别的信息

// 现在为了实现简单: 只有一个交易所,
// 所以 资金帐号和现货交易所帐号是同一个

// 存款交易是系统代为存入的, 存到指定帐号上, 不是签名帐号中

// 用户帐号定义
// dex1 -> accountid -> tokenids 是一个对象
//  理论上, 对象越小越快, 但交易涉及两个资产. 如果一个资产是一个对象的. 要处理两个对象.
//  先实现再说
func (a *SpotAction) Deposit(payload *zt.ZkDeposit, accountID uint64) (*types.Receipt, error) {

	chain33Addr := payload.GetChain33Addr()
	amount, err := et.AmountFromZksync(payload.GetAmount())
	if err != nil {
		return nil, err
	}

	// TODO tid 哪里定义, 里面不需要知道tid 是什么, 在合约里 id1 换 id2

	acc, err := spot.LoadSpotAccount(chain33Addr, accountID, a.statedb)
	if err != nil {
		return nil, err
	}

	return acc.Mint(uint32(payload.TokenId), amount)
}

func (a *SpotAction) CalcMaxActive(accountID uint64, token uint32, amount string) (uint64, error) {
	acc, err := spot.LoadSpotAccount(a.txinfo.From, accountID, a.statedb)
	if err != nil {
		return 0, err
	}
	return acc.GetBalance(token), nil
}

func (a *SpotAction) Withdraw(payload *zt.ZkWithdraw, amountWithFee uint64) (*types.Receipt, error) {
	// TODO amountWithFee to chain33amount
	chain33Addr := a.txinfo.From
	/*
		amount := payload.GetAmount()
		amount2, ok := big.NewInt(0).SetString(amount, 10)
		if !ok {
			return nil, et.ErrAssetBalance
		}
		_ = amount2
	*/
	// TODO tid 哪里定义, 里面不需要知道tid 是什么, 在合约里 id1 换 id2

	acc, err := spot.LoadSpotAccount(chain33Addr, payload.AccountId, a.statedb)
	if err != nil {
		return nil, err
	}

	return acc.Burn(uint32(payload.TokenId), amountWithFee)
}

//

func (a SpotAction) newEntrust() *spot.Entrust {
	e := spot.NewEntrust(a.txinfo.From, a.height, a.statedb)
	e.SetDB(a.statedb, &dbprefix{})
	return e
}

func (a *SpotAction) ExchangeBind(payload *et.SpotExchangeBind) (*types.Receipt, error) {
	e := a.newEntrust()
	return e.Bind(payload)
}

func (a *SpotAction) EntrustOrder(d *dapp.DriverBase, payload *et.SpotEntrustOrder) (*types.Receipt, error) {
	e := a.newEntrust()
	err := e.CheckBind(payload.Addr)
	if err != nil {
		return nil, err
	}
	limitOrder := &et.SpotLimitOrder{
		LeftAsset:  payload.LeftAsset,
		RightAsset: payload.RightAsset,
		Price:      payload.Price,
		Amount:     payload.Amount,
		Op:         payload.Op,
		Order:      payload.Order,
	}

	return a.LimitOrder(d, limitOrder, payload.Addr)
}
