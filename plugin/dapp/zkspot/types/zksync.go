package types

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
