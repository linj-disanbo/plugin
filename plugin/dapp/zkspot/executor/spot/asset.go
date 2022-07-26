package spot

import (
	"fmt"

	"github.com/33cn/chain33/account"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

//  support kinds of asset
type AssetAccount interface {
	GetCoinPrecision() int64
	TransferFrozen(to string, amountt int64) (*types.Receipt, error)
	Frozen(amount int64) (*types.Receipt, error)
	Transfer(to string, amountt int64) (*types.Receipt, error)
	UnFrozen(amount int64) (*types.Receipt, error)

	CheckBalance(amount int64) error
}

// support nft asset from evm contract
type NftAccount struct {
	accdb   *EvmxgoNftAccountRepo
	address string
	accid   uint64
	nftid   uint64
	symbol  string
}

type EvmxgoNftAccountRepo struct {
	cfg     *types.Chain33Config
	statedb dbm.KV
	//symbol  string
	accdb *account.DB
}

func newNftAccountRepo(db dbm.KV, cfg *types.Chain33Config) (*EvmxgoNftAccountRepo, error) {
	return &EvmxgoNftAccountRepo{
		statedb: db,
		cfg:     cfg}, nil
}

func (accdb *EvmxgoNftAccountRepo) NewAccount(addr string, accid uint64, nftid uint64) (*NftAccount, error) {
	var err error
	symbol := fmt.Sprintf("%d", nftid)
	if accdb.accdb == nil {
		accdb.accdb, err = account.NewAccountDB(accdb.cfg, "evmxgo", symbol, accdb.statedb)
		if err != nil {
			return nil, err
		}
	}
	return &NftAccount{accdb: accdb, address: addr, accid: accid, nftid: nftid, symbol: symbol}, nil
}

// support go token from go contract
type TokenAccount struct {
	accdb   *TokenAccountRepo
	address string
	accid   uint64
	execer  string
	symbol  string

	acc *account.DB
}

func (acc *TokenAccount) GetCoinPrecision() int64 {
	return 1e8
}

func (acc *TokenAccount) TransferFrozen(to string, amountt int64) (*types.Receipt, error) {
	return acc.acc.ExecTransferFrozen(acc.address, to, acc.accdb.execAddr, amountt)
}
func (acc *TokenAccount) Frozen(amount int64) (*types.Receipt, error) {
	return acc.acc.ExecFrozen(acc.address, acc.accdb.execAddr, amount)
}
func (acc *TokenAccount) Transfer(to string, amount int64) (*types.Receipt, error) {
	return acc.acc.ExecTransfer(acc.address, to, acc.accdb.execAddr, amount)
}
func (acc *TokenAccount) UnFrozen(amount int64) (*types.Receipt, error) {
	return acc.acc.ExecActive(acc.address, acc.accdb.execAddr, amount)
}

func (acc *TokenAccount) CheckBalance(amount int64) error {
	balance := acc.acc.LoadExecAccount(acc.address, acc.accdb.execAddr)
	if balance.Balance < amount {
		elog.Error("TokenAccount balance", "balance", balance.Balance, "need", amount)
		return et.ErrAssetBalance
	}
	return nil
}

type TokenAccountRepo struct {
	cfg      *types.Chain33Config
	statedb  dbm.KV
	execAddr string
}

func newTokenAccountRepo(db dbm.KV, cfg *types.Chain33Config, execAddr string) (*TokenAccountRepo, error) {
	return &TokenAccountRepo{
		statedb:  db,
		cfg:      cfg,
		execAddr: execAddr}, nil
}

func (accdb *TokenAccountRepo) NewAccount(addr string, accid uint64, asset *et.Asset) (*TokenAccount, error) {
	acc := &TokenAccount{accdb: accdb, address: addr, accid: accid, execer: asset.GetTokenAsset().Execer, symbol: asset.GetTokenAsset().Symbol}
	var err error
	acc.acc, err = account.NewAccountDB(accdb.cfg, asset.GetTokenAsset().Execer, asset.GetTokenAsset().Symbol, accdb.statedb)
	if err != nil {
		return nil, err
	}

	return acc, nil
}
