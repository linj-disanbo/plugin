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

// zkSpotDex struct
type zkSpotDex struct {
	statedb   dbm.KV
	blocktime int64
	height    int64
	localDB   dbm.KVDB
	api       client.QueueProtocolAPI
	txinfo    *et.TxInfo
	env       *dapp.DriverBase
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

//NewZkSpotDex ...
func NewZkSpotDex(e *zkspot, tx *types.Transaction, index int) *zkSpotDex {
	return &zkSpotDex{
		txinfo:    NewTxInfo(tx, index),
		env:       &e.DriverBase,
		statedb:   e.GetStateDB(),
		blocktime: e.GetBlockTime(),
		height:    e.GetHeight(),
		localDB:   e.GetLocalDB(),
		api:       e.GetAPI(),
	}
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

func (z *zktree) checkL2Auth(acc *zt.Leaf, pub *zt.ZkPubKey) error {
	err := authVerification(pub, acc.GetPubKey())
	if err != nil {
		return errors.Wrapf(err, "authVerification")
	}
	return nil
}

func checkL2Auth(statedb dbm.KV, accountID uint64, pub *zt.ZkPubKey) error {
	var zktree1 zktree
	zkAcc, err := zktree1.getAccount(statedb, accountID)
	if err != nil {
		return err
	}
	err = zktree1.checkL2Auth(zkAcc, pub)
	if err != nil {
		return errors.Wrapf(err, "authVerification")
	}
	return nil
}

func (a *zkSpotDex) getFeeAcc() (*spot.SpotFee, error) {
	accountID := uint64(et.SystemFeeAccountId)
	z1 := &zktree{}
	leaf, err := z1.getAccount(a.statedb, accountID)
	if err != nil {
		return nil, err
	}
	return &spot.SpotFee{
		Address: leaf.ChainAddr,
		AccID:   accountID,
	}, nil
}

//LimitOrder ...
func (a *zkSpotDex) LimitOrder(base *dapp.DriverBase, payload *et.SpotLimitOrder, entrustAddr string) (*types.Receipt, error) {
	cfg := a.api.GetConfig()
	err := et.CheckLimitOrder(cfg, payload)
	if err != nil {
		return nil, err
	}

	err = checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, err
	}

	spot1, err := spot.NewSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	// 下面流程是否要放到 spot1中

	taker, err := spot1.LoadUser(a.txinfo.From, payload.Order.AccountID)
	if err != nil {
		return nil, err
	}

	order, err := spot1.CreateLimitOrder(a.txinfo.From, taker, payload, entrustAddr)
	if err != nil {
		return nil, err
	}
	_ = order // set to order trader

	receipt1, err := spot1.MatchLimitOrder(payload, taker)
	if err != nil {
		return nil, err
	}
	return receipt1, nil
}

func (a *zkSpotDex) RevokeOrder(payload *et.SpotRevokeOrder, entrustAddr string) (*types.Receipt, error) {
	spot, err := spot.NewSpot(a.env, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	return spot.RevokeOrder(a.txinfo.From, payload)
}

// 现在一个交易所, 资金帐号和现货交易所帐号是同一个
// 在多个交易所的情况下, 会有一个资金帐号和多个交易所帐号
func (a *zkSpotDex) Deposit(payload *zt.ZkDeposit, accountID uint64, info *zkHandler) (*types.Receipt, error) {
	amount, err := et.AmountFromZksync(payload.GetAmount())
	if err != nil {
		return nil, err
	}
	// 在第一次存钱时, 是不知道用户chainAddr
	//var zktree1 zktree
	//leaf, err := zktree1.getAccount(a.statedb, accountID)
	//if err != nil {
	//	return nil, err
	//}
	acc, err := spot.LoadSpotAccount("leaf.ChainAddr", accountID, a.statedb, &dbprefix{})
	if err != nil {
		return nil, err
	}

	return acc.Mint(payload.TokenId, amount)
}

func (a *zkSpotDex) CalcMaxActive(accountID uint64, token uint64, amount string) (uint64, error) {
	acc, err := spot.LoadSpotAccount(a.txinfo.From, accountID, a.statedb, &dbprefix{})
	if err != nil {
		return 0, err
	}
	return acc.GetBalance(token), nil
}

func (a *zkSpotDex) Withdraw(payload *zt.ZkWithdraw, amountWithFee uint64) (*types.Receipt, error) {
	chain33Addr := a.txinfo.From
	acc, err := spot.LoadSpotAccount(chain33Addr, payload.AccountId, a.statedb, &dbprefix{})
	if err != nil {
		return nil, err
	}

	return acc.Burn(payload.TokenId, amountWithFee)
}

func (a *zkSpotDex) newEntrust() *spot.Entrust {
	e := spot.NewEntrust(a.txinfo.From, a.height, a.statedb)
	e.SetDB(a.statedb, &dbprefix{})
	return e
}

func (a *zkSpotDex) ExchangeBind(payload *et.SpotExchangeBind) (*types.Receipt, error) {
	e := a.newEntrust()
	return e.Bind(payload)
}

func (a *zkSpotDex) EntrustOrder(d *dapp.DriverBase, payload *et.SpotEntrustOrder) (*types.Receipt, error) {
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

func (a *zkSpotDex) execLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	spot, err := spot.NewSpot(a.env, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	return spot.ExecLocal(tx, receiptData, index)
}

//NftOrder ...
func (a *zkSpotDex) NftOrder(base *dapp.DriverBase, payload *et.SpotNftOrder, entrustAddr string) (*types.Receipt, error) {
	cfg := a.api.GetConfig()
	err := et.CheckNftOrder(cfg, payload)
	if err != nil {
		return nil, err
	}

	err = checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, err
	}

	spot1, err := spot.NewSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	// 下面流程是否要放到 spot1中
	taker, err := spot1.LoadTrader(a.txinfo.From, payload.Order.AccountID, nil, nil) // TODO
	if err != nil {
		return nil, err
	}

	return spot1.CreateNftOrder(a.txinfo.From, taker, payload, entrustAddr)
}

//NftOrder ...
func (a *zkSpotDex) NftTakerOrder(base *dapp.DriverBase, payload *et.SpotNftTakerOrder, entrustAddr string) (*types.Receipt, error) {
	//cfg := a.api.GetConfig()
	err := checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
	if err != nil {
		return nil, err
	}

	spot1, err := spot.NewSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	// 下面流程是否要放到 spot1中

	return spot1.TradeNft(a.txinfo.From, payload, entrustAddr)
}

func isSellZkAsset(op int32, left, right *et.Asset) bool {
	asset := left
	if op == et.OpBuy {
		asset = right
	}
	return asset.Ty == et.AssetType_L1Erc20 || asset.Ty == et.AssetType_ZkNft
}

//AssetLimitOrder ...
// TODO create new account for L1
func (a *zkSpotDex) AssetLimitOrder(base *dapp.DriverBase, payload *et.AssetLimitOrder, entrustAddr string) (*types.Receipt, error) {
	cfg := a.api.GetConfig()
	err := et.CheckAssetLimitOrder(cfg, payload)
	if err != nil {
		return nil, err
	}

	if isSellZkAsset(payload.Op, payload.LeftAsset, payload.RightAsset) {
		err = checkL2Auth(a.statedb, payload.Order.AccountID, payload.Order.Signature.PubKey)
		if err != nil {
			return nil, err
		}
	}

	spot1, err := spot.NewSpot(base, a.txinfo, &dbprefix{})
	if err != nil {
		return nil, err
	}
	err = spot1.SetFeeAcc(a.getFeeAcc)
	if err != nil {
		return nil, err
	}

	// 下面流程是否要放到 spot1中
	taker, err := spot1.LoadNewUser(a.txinfo.From, payload.Order.AccountID, payload)
	if err != nil {
		return nil, err
	}

	order, err := spot1.CreateAssetLimitOrder(a.txinfo.From, taker, payload, entrustAddr)
	if err != nil {
		return nil, err
	}
	_ = order // set to order trader

	receipt1, err := spot1.MatchAssetLimitOrder(payload, taker)
	if err != nil {
		return nil, err
	}
	return receipt1, nil
}
