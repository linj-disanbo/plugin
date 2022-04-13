package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksopt/types"
)

func (z *zksopt) Exec_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.Deposit(payload)
}

func (z *zksopt) Exec_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.Withdraw(payload)
}

func (z *zksopt) Exec_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.ContractToTree(payload)
}

func (z *zksopt) Exec_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.TreeToContract(payload)
}

func (z *zksopt) Exec_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.Transfer(payload)
}

func (z *zksopt) Exec_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.TransferToNew(payload)
}

func (z *zksopt) Exec_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.ForceExit(payload)
}

func (z *zksopt) Exec_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.SetPubKey(payload)
}

func (z *zksopt) Exec_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.FullExit(payload)
}

func (z *zksopt) Exec_Swap(payload *zt.ZkSwap, tx *types.Transaction, index int) (*types.Receipt, error) {
	//todo swap stub
	return nil, nil
}

func (z *zksopt) Exec_SetVerifyKey(payload *zt.ZkVerifyKey, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.setVerifyKey(payload)
}

func (z *zksopt) Exec_CommitProof(payload *zt.ZkCommitProof, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.commitProof(payload)
}

func (z *zksopt) Exec_SetVerifier(payload *zt.ZkVerifier, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.setVerifier(payload)
}
