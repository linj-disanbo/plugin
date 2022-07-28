package spot

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// account repos -> asset_ty -> account repo
// load user
type accountRepos struct {
	zkRepo    *accountRepo
	tokenRepo *TokenAccountRepo
}

func newAccountRepo11(dexName string, statedb dbm.KV, p et.DBprefix, cfg *types.Chain33Config, execAddr string) (*accountRepos, error) {
	//zkAccID uint64, from string, asset *et.Asset) {

	var repos accountRepos
	var err error
	repos.zkRepo = newAccountRepo(dexName, statedb, p)
	repos.tokenRepo, err = newTokenAccountRepo(statedb, cfg, execAddr)
	if err != nil {
		return nil, err
	}

	return &repos, nil
}

func (repos *accountRepos) LoadAccount(addr string, zkAccID uint64, asset *et.Asset) (AssetAccount, error) {
	switch asset.Ty {
	case et.AssetType_L1Erc20:
		acc1, err := repos.zkRepo.LoadAccount(addr, zkAccID)
		if err != nil {
			return nil, err
		}
		return &ZkAccount{acc: acc1, asset: asset}, nil
	case et.AssetType_Token:
		acc, err := repos.tokenRepo.NewAccount(addr, 1, asset)
		if err != nil {
			return nil, err
		}
		return acc, nil
	}
	panic("not support")

}
