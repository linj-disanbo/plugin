package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

func (z *zkspot) Exec_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	r, err := action.Deposit(payload)
	if err != nil {
		return r, err
	}
	action2 := NewSpotAction2(z, tx, index)
	r2, err := action2.Deposit(payload)
	if err != nil {
		return r, err
	}
	return mergeReceipt(r, r2), nil
}

func (z *zkspot) Exec_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.Withdraw(payload)
}

func (z *zkspot) Exec_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.ContractToTree(payload)
}

func (z *zkspot) Exec_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.TreeToContract(payload)
}

func (z *zkspot) Exec_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.Transfer(payload)
}

func (z *zkspot) Exec_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.TransferToNew(payload)
}

func (z *zkspot) Exec_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.ForceExit(payload)
}

func (z *zkspot) Exec_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.SetPubKey(payload)
}

func (z *zkspot) Exec_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.FullExit(payload)
}

func (z *zkspot) Exec_Swap(payload *zt.ZkSwap, tx *types.Transaction, index int) (*types.Receipt, error) {
	//todo swap stub
	return nil, nil
}

func (z *zkspot) Exec_SetVerifyKey(payload *zt.ZkVerifyKey, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.setVerifyKey(payload)
}

func (z *zkspot) Exec_CommitProof(payload *zt.ZkCommitProof, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.commitProof(payload)
}

func (z *zkspot) Exec_SetVerifier(payload *zt.ZkVerifier, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := NewAction(z, tx, index)
	return action.setVerifier(payload)
}
