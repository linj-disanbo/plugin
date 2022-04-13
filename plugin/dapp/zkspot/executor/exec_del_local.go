package executor

import (
	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksopt/types"
)

func (z *zksopt) ExecDelLocal_Deposit(payload *zt.ZkDeposit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_Withdraw(payload *zt.ZkWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_ContractToTree(payload *zt.ZkContractToTree, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_TreeToContract(payload *zt.ZkTreeToContract, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_Transfer(payload *zt.ZkTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_TransferToNew(payload *zt.ZkTransferToNew, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_ForceExit(payload *zt.ZkForceExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_SetPubKey(payload *zt.ZkSetPubKey, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}

func (z *zksopt) ExecDelLocal_FullExit(payload *zt.ZkFullExit, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return z.execAutoDelLocal(tx, receiptData)
}
