package types

import (
	"reflect"

	"github.com/33cn/chain33/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
)

/*
 * 交易相关类型定义
 * 交易action通常有对应的log结构，用于交易回执日志记录
 * 每一种action和log需要用id数值和name名称加以区分
 */

// action类型id和name，这些常量可以自定义修改
const (
// zk action type  0 - 1000
// zk log id 0 - 1000
// github.com/33cn/plugin/plugin/dapp/zksync/types/zksync.go
)

//ExecName 执行器名称定义
const ExecName = "zkspot"

var (

	//定义actionMap
	actionMap = map[string]int32{
		// zk
		//NameNoopAction:           TyNoopAction,
		zt.NameDepositAction:        zt.TyDepositAction,
		zt.NameWithdrawAction:       zt.TyWithdrawAction,
		zt.NameContractToTreeAction: zt.TyContractToTreeAction,
		zt.NameTreeToContractAction: zt.TyTreeToContractAction,
		zt.NameTransferAction:       zt.TyTransferAction,
		zt.NameTransferToNewAction:  zt.TyTransferToNewAction,
		zt.NameForceExitAction:      zt.TyForceExitAction,
		zt.NameSetPubKeyAction:      zt.TySetPubKeyAction,
		zt.NameFullExitAction:       zt.TyFullExitAction,
		zt.NameSwapAction:           zt.TySwapAction,
		zt.NameSetVerifyKeyAction:   zt.TySetVerifyKeyAction,
		zt.NameCommitProofAction:    zt.TyCommitProofAction,
		zt.NameSetVerifierAction:    zt.TySetVerifierAction,
		zt.NameSetFeeAction:         zt.TySetFeeAction,
		zt.NameMintNFTAction:        zt.TyMintNFTAction,
		zt.NameWithdrawNFTACTION:    zt.TyWithdrawNFTAction,
		zt.NameTransferNFTAction:    zt.TyTransferNFTAction,
		// spot
		NameLimitOrderAction:         TyLimitOrderAction,
		NameMarketOrderAction:        TyMarketOrderAction,
		NameRevokeOrderAction:        TyRevokeOrderAction,
		NameExchangeBindAction:       TyExchangeBindAction,
		NameEntrustOrderAction:       TyEntrustOrderAction,
		NameEntrustRevokeOrderAction: TyEntrustRevokeOrderAction,
		// spot nft
		NameNftOrderAction:      TyNftOrderAction,
		NameNftTakerOrderAction: TyNftTakerOrderAction,
	}
	//定义log的id和具体log类型及名称，填入具体自定义log类型
	logMap = map[int64]*types.LogInfo{
		// zk
		//TyNoopLog:           {Ty: reflect.TypeOf(ZkReceiptLeaf{}), Name: "TyNoopLog"},
		zt.TyDepositLog:        {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyDepositLog"},
		zt.TyWithdrawLog:       {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyWithdrawLog"},
		zt.TyContractToTreeLog: {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyContractToTreeLog"},
		zt.TyTreeToContractLog: {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyTreeToContractLog"},
		zt.TyTransferLog:       {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyTransferLog"},
		zt.TyTransferToNewLog:  {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyTransferToNewLog"},
		zt.TyForceExitLog:      {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyForceExitLog"},
		zt.TySetPubKeyLog:      {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TySetPubKeyLog"},
		zt.TyFullExitLog:       {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyFullExitLog"},
		zt.TySwapLog:           {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TySwapLog"},
		zt.TyFeeLog:            {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyFeeLog"},
		zt.TyMintNFTLog:        {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyMintNFTLog"},
		zt.TyWithdrawNFTLog:    {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyWithdrawNFTLog"},
		zt.TyTransferNFTLog:    {Ty: reflect.TypeOf(zt.ZkReceiptLog{}), Name: "TyTransferNFTLog"},

		zt.TySetVerifyKeyLog:       {Ty: reflect.TypeOf(zt.ReceiptSetVerifyKey{}), Name: "TySetVerifyKey"},
		zt.TyCommitProofLog:        {Ty: reflect.TypeOf(zt.ReceiptCommitProof{}), Name: "TyCommitProof"},
		zt.TySetVerifierLog:        {Ty: reflect.TypeOf(zt.ReceiptSetVerifier{}), Name: "TySetVerifierLog"},
		zt.TySetEthPriorityQueueId: {Ty: reflect.TypeOf(zt.ReceiptEthPriorityQueueID{}), Name: "TySetEthPriorityQueueID"},
		zt.TySetFeeLog:             {Ty: reflect.TypeOf(zt.ReceiptSetFee{}), Name: "TySetFeeLog"},

		// spot
		TyLimitOrderLog:   {Ty: reflect.TypeOf(ReceiptSpotMatch{}), Name: "TyLimitOrderLog"},
		TyMarketOrderLog:  {Ty: reflect.TypeOf(ReceiptSpotMatch{}), Name: "TyMarketOrderLog"},
		TyRevokeOrderLog:  {Ty: reflect.TypeOf(ReceiptSpotMatch{}), Name: "TyRevokeOrderLog"},
		TyExchangeBindLog: {Ty: reflect.TypeOf(ReceiptDexBind{}), Name: "TyExchangeBindLog"},
		TySpotTradeLog:    {Ty: reflect.TypeOf(ReceiptSpotTrade{}), Name: "TySpotTradeLog"},
		// dex account
		TyDexAccountFrozen: {Ty: reflect.TypeOf(ReceiptDexAccount{}), Name: "TyDexAccountFrozen"},
		TyDexAccountActive: {Ty: reflect.TypeOf(ReceiptDexAccount{}), Name: "TyDexAccountActive"},
		TyDexAccountBurn:   {Ty: reflect.TypeOf(ReceiptDexAccount{}), Name: "TyDexAccountBurn"},
		TyDexAccountMint:   {Ty: reflect.TypeOf(ReceiptDexAccount{}), Name: "TyDexAccountMint"},
		// spot nft
		TyNftOrderLog:      {Ty: reflect.TypeOf(ReceiptSpotMatch{}), Name: "TyNftOrderLog"},
		TyNftTakerOrderLog: {Ty: reflect.TypeOf(ReceiptSpotMatch{}), Name: "TyNftTakerOrderLog"},
	}

	FeeMap = map[int64]string{
		zt.TyWithdrawAction:      "1000000",
		zt.TyTransferAction:      "100000",
		zt.TyTransferToNewAction: "100000",
		zt.TyForceExitAction:     "1000000",
		zt.TyFullExitAction:      "1000000",
		zt.TySwapAction:          "100000",
		zt.TyMintNFTAction:       "100",
		zt.TyWithdrawNFTAction:   "100",
		zt.TyTransferNFTAction:   "100",
	}
)

// init defines a register function
func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(ExecName))
	//注册合约启用高度
	types.RegFork(ExecName, InitFork)
	types.RegExec(ExecName, InitExecutor)
}

// InitFork defines register fork
func InitFork(cfg *types.Chain33Config) {
	SpotInitFork(cfg)
	cfg.RegisterDappFork(ExecName, "Enable", 0)
}

// InitExecutor defines register executor
func InitExecutor(cfg *types.Chain33Config) {
	types.RegistorExecutor(ExecName, NewType(cfg))
}

//ZksyncType ...
type ZksyncType struct {
	types.ExecTypeBase
}

//NewType ...
func NewType(cfg *types.Chain33Config) *ZksyncType {
	c := &ZksyncType{}
	c.SetChild(c)
	c.SetConfig(cfg)
	return c
}

// GetPayload 获取合约action结构
func (e *ZksyncType) GetPayload() types.Message {
	return &ZksyncAction1{}
}

// GetTypeMap 获取合约action的id和name信息
func (e *ZksyncType) GetTypeMap() map[string]int32 {
	return actionMap
}

// GetLogMap 获取合约log相关信息
func (e *ZksyncType) GetLogMap() map[int64]*types.LogInfo {
	return logMap
}
