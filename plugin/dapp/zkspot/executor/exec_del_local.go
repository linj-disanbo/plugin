package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
)

func (z *zkspot) ExecDelLocal_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_MintNFT(payload *zt.ZkMintNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_WithdrawNFT(payload *zt.ZkWithdrawNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_TransferNFT(payload *zt.ZkTransferNFT, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zkspot) ExecDelLocal_CommitProof(payload *zt.ZkCommitProof, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}
