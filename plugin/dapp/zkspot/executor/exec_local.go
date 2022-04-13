package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksopt/types"
)

//ExecLocal_Deposit asset withdraw local db process
func (z *zksopt) ExecLocal_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

//ExecLocal_Withdraw asset withdraw local db process
func (z *zksopt) ExecLocal_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

// ExecLocal_Transfer asset transfer local db process
func (z *zksopt) ExecLocal_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

//ExecLocal_Authorize asset withdraw local db process
func (z *zksopt) ExecLocal_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoLocalZksync(tx, receiptData, index)
}

func (z *zksopt) ExecLocal_CommitProof(payload *zt.ZkCommitProof, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execCommitProofLocal(payload, tx, receiptData, index)
}
