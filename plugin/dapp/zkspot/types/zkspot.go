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

const (
	Add = int32(0)
	Sub = int32(1)
)

//Zksync 执行器名称定义
const Zksync = "zkspot"
const ZkManagerKey = "manager"
const ZkMimcHashSeed = "seed"
const ZkVerifierKey = "verifier"

//msg宽度
const (
	TxTypeBitWidth      = 8  //1byte
	AccountBitWidth     = 32 //4byte
	TokenBitWidth       = 32 //2byte
	NFTAmountBitWidth   = 16
	AmountBitWidth      = 128 //16byte
	AddrBitWidth        = 160 //20byte
	Chain33AddrBitWidth = 256 //20byte
	PubKeyBitWidth      = 256 //32byte
	FeeAmountBitWidth   = 56  //fee op凑满one chunk=128bit，最大10byte

	PacAmountManBitWidth = 35 //amount mantissa part, 比如12340000,只取1234部分，0000用exponent表示
	PacAmountExpBitWidth = 5  //amount exponent part
	PacFeeManBitWidth    = 11 //fee mantissa part
	PacFeeExpBitWidth    = 5  //fee exponent part
	MaxExponentVal       = 32 // 2**5 by exp bit width

	ChunkBitWidth = 128               //one chunk 16 bytes
	ChunkBytes    = ChunkBitWidth / 8 //16 bytes
)

const (
	//BN254Fp=254bit,254-2 bit
	MsgFirstWidth  = 252
	MsgSecondWidth = 252
	MsgThirdWidth  = 248
	MsgWidth       = 752 //94 byte

)

//不同type chunk数量
const (
	DepositChunks       = 5
	Contract2TreeChunks = 3
	Tree2ContractChunks = 3
	TransferChunks      = 2
	Transfer2NewChunks  = 5
	WithdrawChunks      = 3
	ForceExitChunks     = 3
	FullExitChunks      = 3
	SwapChunks          = 4
	NoopChunks          = 1
	SetPubKeyChunks     = 5
	FeeChunks           = 1
	SetProxyAddrChunks  = 5
	MintNFTChunks       = 5
	WithdrawNFTChunks   = 6
	TransferNFTChunks   = 3
)

const (
	//SystemFeeAccountId 此账户作为缺省收费账户
	SystemFeeAccountId = 1
	//SystemNFTAccountId 此特殊账户没有私钥，只记录并产生NFT token资产，不会有小于NFTTokenId的FT token记录
	SystemNFTAccountId = 2
	//SystemNFTTokenId 作为一个NFT token标记 低于NFTTokenId 为FT token id, 高于NFTTokenId为 NFT token id，即从NFTTokenId+1开始作为NFT资产
	SystemNFTTokenId = 256 //2^8,

)

//ERC protocol
const (
	ZKERC1155 = 1
	ZKERC721  = 2
)

const (
	NormalProxyPubKey = 1
	SystemProxyPubKey = 2
	SuperProxyPubKey  = 3
)

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
	types.AllowUserExec = append(types.AllowUserExec, []byte(Zksync))
	//注册合约启用高度
	types.RegFork(Zksync, InitFork)
	types.RegExec(Zksync, InitExecutor)
}

// InitFork defines register fork
func InitFork(cfg *types.Chain33Config) {
	SpotInitFork(cfg)
	cfg.RegisterDappFork(Zksync, "Enable", 0)
}

// InitExecutor defines register executor
func InitExecutor(cfg *types.Chain33Config) {
	types.RegistorExecutor(Zksync, NewType(cfg))
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
