package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
)

//ExecLocal_Deposit asset withdraw local db process
func (z *zkspot) ExecLocal_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

//ExecLocal_Withdraw asset withdraw local db process
func (z *zkspot) ExecLocal_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

// ExecLocal_Transfer asset transfer local db process
func (z *zkspot) ExecLocal_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

//ExecLocal_Authorize asset withdraw local db process
func (z *zkspot) ExecLocal_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_MintNFT(payload *zt.ZkMintNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_WithdrawNFT(payload *zt.ZkWithdrawNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zkspot) ExecLocal_TransferNFT(payload *zt.ZkTransferNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}


func (z *zkspot) ExecLocal_CommitProof(payload *zt.ZkCommitProof, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execCommitProofLocal(payload, tx, receiptData, index)
}
