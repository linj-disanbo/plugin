package executor

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/pkg/errors"
)

var (
	zklog = log15.New("module", "exec.zkspot")
)

// Action action struct
type Action struct {
	statedb   dbm.KV
	txhash    []byte
	fromaddr  string
	blocktime int64
	height    int64
	execaddr  string
	localDB   dbm.KVDB
	index     int
	api       client.QueueProtocolAPI
}

//NewAction ...
func NewAction(z *zkspot, tx *types.Transaction, index int) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &Action{
		statedb:   z.GetStateDB(),
		txhash:    hash,
		fromaddr:  fromaddr,
		blocktime: z.GetBlockTime(),
		height:    z.GetHeight(),
		execaddr:  dapp.ExecAddress(string(tx.Execer)),
		localDB:   z.GetLocalDB(),
		index:     index,
		api:       z.GetAPI(),
	}
}

//GetIndex get index
func (a *Action) GetIndex() int64 {
	return a.height*types.MaxTxsPerBlock + int64(a.index)
}

func (a *Action) Deposit(payload *zt.ZkDeposit) (*types.Receipt, error, uint64) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue
	var err error

	err = checkParam(payload.Amount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam"), 0
	}

	zklog.Info("start zkspot deposit", "eth", payload.EthAddress, "chain33", payload.Chain33Addr)
	//只有管理员能操作
	cfg := a.api.GetConfig()
	if !isSuperManager(cfg, a.fromaddr) && !isVerifier(a.statedb, a.fromaddr) {
		return nil, errors.Wrapf(types.ErrNotAllow, "from addr is not manager"), 0
	}

	//TODO set chainID
	lastPriority, err := getLastEthPriorityQueueID(a.statedb, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "get eth last priority queue id"), 0
	}
	lastPriorityId, ok := big.NewInt(0).SetString(lastPriority.GetID(), 10)
	if !ok {
		return nil, errors.Wrapf(types.ErrInvalidParam, fmt.Sprintf("getID =%s", lastPriority.GetID())), 0
	}
	//if lastPriorityId.Int64()+1 != payload.GetEthPriorityQueueId() {
	//	return nil, errors.Wrapf(types.ErrNotAllow, "eth last priority queue id=%d,new=%d", lastPriorityId, payload.GetEthPriorityQueueId())
	//}

	//转换10进制
	payload.Chain33Addr = zt.HexAddr2Decimal(payload.Chain33Addr)
	payload.EthAddress = zt.HexAddr2Decimal(payload.EthAddress)

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo"), 0
	}

	leaf, err := GetLeafByChain33AndEthAddress(a.statedb, payload.GetChain33Addr(), payload.GetEthAddress(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByChain33AndEthAddress"), 0
	}

	tree, err := getAccountTree(a.statedb, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.getAccountTree"), 0
	}
	zklog.Info("zkspot deposit", "tree", tree)

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyDepositAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		SigData:     payload.Signature,
	}

	//leaf不存在就添加
	if leaf == nil {
		zklog.Info("zkspot deposit add leaf")
		operationInfo.AccountID = tree.GetTotalIndex() + 1
		//添加之前先计算证明
		receipt, err := calProof(a.statedb, info, operationInfo.AccountID, payload.TokenId)
		if err != nil {
			return nil, errors.Wrapf(err, "calProof"), 0
		}

		before := getBranchByReceipt(receipt, operationInfo, payload.EthAddress, payload.Chain33Addr, nil, "0", operationInfo.AccountID)

		kvs, localKvs, err = AddNewLeaf(a.statedb, a.localDB, info, payload.GetEthAddress(), payload.GetTokenId(), payload.GetAmount(), payload.GetChain33Addr())
		if err != nil {
			return nil, errors.Wrapf(err, "db.AddNewLeaf"), 0
		}
		receipt, err = calProof(a.statedb, info, operationInfo.AccountID, payload.TokenId)
		if err != nil {
			return nil, errors.Wrapf(err, "calProof"), 0
		}

		after := getBranchByReceipt(receipt, operationInfo, payload.EthAddress, payload.Chain33Addr, nil, receipt.Token.Balance, operationInfo.AccountID)
		rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
		kv := &types.KeyValue{
			Key:   getHeightKey(a.height),
			Value: rootHash,
		}
		kvs = append(kvs, kv)

		branch := &zt.OperationPairBranch{
			Before: before,
			After:  after,
		}
		operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
		zklog := &zt.ZkReceiptLog{
			OperationInfo: operationInfo,
			LocalKvs:      localKvs,
		}
		receiptLog := &types.ReceiptLog{Ty: zt.TyDepositLog, Log: types.Encode(zklog)}
		logs = append(logs, receiptLog)
	} else {
		operationInfo.AccountID = leaf.GetAccountId()

		receipt, err := calProof(a.statedb, info, leaf.AccountId, payload.TokenId)
		if err != nil {
			return nil, errors.Wrapf(err, "calProof"), 0
		}

		var balance string
		if receipt.Token == nil {
			balance = "0"
		} else {
			balance = receipt.Token.Balance
		}
		before := getBranchByReceipt(receipt, operationInfo, payload.EthAddress, payload.Chain33Addr, leaf.PubKey, balance, operationInfo.AccountID)

		kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), payload.GetAmount(), zt.Add)
		if err != nil {
			return nil, errors.Wrapf(err, "db.UpdateLeaf"), 0
		}
		receipt, err = calProof(a.statedb, info, leaf.AccountId, payload.TokenId)
		if err != nil {
			return nil, errors.Wrapf(err, "calProof"), 0
		}
		after := getBranchByReceipt(receipt, operationInfo, payload.EthAddress, payload.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
		rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
		kv := &types.KeyValue{
			Key:   getHeightKey(a.height),
			Value: rootHash,
		}
		kvs = append(kvs, kv)

		branch := &zt.OperationPairBranch{
			Before: before,
			After:  after,
		}
		operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
		zklog := &zt.ZkReceiptLog{
			OperationInfo: operationInfo,
			LocalKvs:      localKvs,
		}
		receiptLog := &types.ReceiptLog{Ty: zt.TyDepositLog, Log: types.Encode(zklog)}
		logs = append(logs, receiptLog)
	}
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	//add priority part
	r := makeSetEthPriorityIdReceipt(0, lastPriorityId.Int64(), payload.EthPriorityQueueId)
	return mergeReceipt(receipts, r), nil, operationInfo.AccountID
}

func getBranchByReceipt(receipt *zt.ZkReceiptLeaf, info *zt.OperationInfo, ethAddr string, chain33Addr string, pubKey *zt.ZkPubKey, balance string, accountId uint64) *zt.OperationMetaBranch {
	return getBranchByReceipt1(receipt, info, ethAddr, chain33Addr, pubKey, balance, accountId, info.TokenID)
}

func getBranchByReceipt1(receipt *zt.ZkReceiptLeaf, info *zt.OperationInfo, ethAddr string, chain33Addr string, pubKey *zt.ZkPubKey, balance string, accountId uint64, tokenId uint64) *zt.OperationMetaBranch {
	info.Roots = append(info.Roots, receipt.TreeProof.RootHash)

	treePath := &zt.SiblingPath{
		Path:   receipt.TreeProof.ProofSet,
		Helper: receipt.TreeProof.GetHelpers(),
	}
	accountW := &zt.AccountWitness{
		ID:          accountId,
		EthAddr:     ethAddr,
		Chain33Addr: chain33Addr,
		PubKey:      pubKey,
		Sibling:     treePath,
	}

	//token不存在生成默认TokenWitness
	if receipt.GetTokenProof() == nil {
		accountW.TokenTreeRoot = "0"
		return &zt.OperationMetaBranch{
			AccountWitness: accountW,
			TokenWitness: &zt.TokenWitness{
				ID:      uint64(tokenId),
				Balance: "0",
			},
		}
	}
	accountW.TokenTreeRoot = receipt.GetTokenProof().RootHash

	tokenPath := &zt.SiblingPath{
		Path:   receipt.TokenProof.ProofSet,
		Helper: receipt.TokenProof.GetHelpers(),
	}
	tokenW := &zt.TokenWitness{
		ID:      uint64(tokenId),
		Balance: balance,
		Sibling: tokenPath,
	}

	branch := &zt.OperationMetaBranch{
		AccountWitness: accountW,
		TokenWitness:   tokenW,
	}
	return branch
}

func generateTreeUpdateInfo(db dbm.KV) (*TreeUpdateInfo, error) {
	updateMap := make(map[string][]byte)
	val, err := db.Get(GetAccountTreeKey())
	if err != nil {
		//没查到就先初始化
		if err == types.ErrNotFound {
			tree := NewAccountTree(db)
			updateMap[string(GetAccountTreeKey())] = types.Encode(tree)
			return &TreeUpdateInfo{updateMap: updateMap}, nil
		} else {
			return nil, err
		}
	}
	var tree zt.AccountTree
	err = types.Decode(val, &tree)
	if err != nil {
		return nil, err
	}
	updateMap[string(GetAccountTreeKey())] = types.Encode(&tree)
	return &TreeUpdateInfo{updateMap: updateMap}, nil
}

func (a *Action) Withdraw(payload *zt.ZkWithdraw) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue
	fee := zt.FeeMap[zt.TyWithdrawAction]
	//加上手续费
	amountInt, _ := new(big.Int).SetString(payload.Amount, 10)
	feeInt, _ := new(big.Int).SetString(fee, 10)
	totalAmount := new(big.Int).Add(amountInt, feeInt).String()
	err := checkParam(totalAmount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}

	leaf, err := GetLeafByAccountId(a.statedb, payload.GetAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if leaf == nil {
		return nil, errors.New("account not exist")
	}
	err = authVerification(payload.GetSignature().PubKey, leaf.GetPubKey())
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	token, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.AccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	err = checkAmount(token, payload.GetAmount())
	if err != nil {
		return nil, errors.Wrapf(err, "db.checkAmount")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyWithdrawAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		FeeAmount:   fee,
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)

	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), totalAmount, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//取款之后计算证明
	receipt, err = calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)

	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyWithdrawLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}

	feeReceipt, err := a.MakeFeeLog(fee, info, payload.TokenId, payload.Signature)
	receipts = mergeReceipt(receipts, feeReceipt)
	return receipts, nil
}

func checkAmount(token *zt.TokenBalance, amount string) error {
	if token != nil {
		balance, _ := new(big.Int).SetString(token.Balance, 10)
		need, _ := new(big.Int).SetString(amount, 10)
		if balance.Cmp(need) >= 0 {
			return nil
		} else {
			return errors.New("balance not enough")
		}
	}
	//token为nil
	return errors.New("balance not enough")
}

func (a *Action) ContractToTree(payload *zt.ZkContractToTree) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	//因为合约balance需要/1e10，因此要先去掉精度
	amountInt, _ := new(big.Int).SetString(payload.Amount, 10)
	payload.Amount = new(big.Int).Mul(new(big.Int).Div(amountInt, big.NewInt(1e10)), big.NewInt(1e10)).String()

	err := checkParam(payload.Amount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}

	leaf, err := GetLeafByAccountId(a.statedb, payload.GetAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if leaf == nil {
		return nil, errors.New("account:" + strconv.FormatUint(payload.AccountId, 10) + " not exist")
	}

	err = authVerification(payload.GetSignature().PubKey, leaf.GetPubKey())
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyContractToTreeAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	var balance string
	if receipt.Token == nil {
		balance = "0"
	} else {
		balance = receipt.Token.Balance
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, balance, operationInfo.AccountID)

	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), payload.GetAmount(), zt.Add)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//更新合约账户
	accountKvs, err := a.UpdateContractAccount(a.fromaddr, payload.GetAmount(), payload.GetTokenId(), zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateContractAccount")
	}
	kvs = append(kvs, accountKvs...)
	//存款到叶子之后计算证明
	receipt, err = calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)

	zkspotlog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyContractToTreeLog, Log: types.Encode(zkspotlog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

func (a *Action) TreeToContract(payload *zt.ZkTreeToContract) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue
	//因为合约balance需要/1e10，因此要先去掉精度
	amountInt, _ := new(big.Int).SetString(payload.Amount, 10)
	payload.Amount = new(big.Int).Mul(new(big.Int).Div(amountInt, big.NewInt(1e10)), big.NewInt(1e10)).String()

	err := checkParam(payload.Amount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}
	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}
	leaf, err := GetLeafByAccountId(a.statedb, payload.GetAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if leaf == nil {
		return nil, errors.New("account not exist")
	}
	err = authVerification(payload.Signature.PubKey, leaf.GetPubKey())
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	token, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.AccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	err = checkAmount(token, payload.GetAmount())
	if err != nil {
		return nil, errors.Wrapf(err, "db.checkAmount")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyTreeToContractAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)

	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), payload.GetAmount(), zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//更新合约账户
	accountKvs, err := a.UpdateContractAccount(a.fromaddr, payload.GetAmount(), payload.GetTokenId(), zt.Add)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateContractAccount")
	}
	kvs = append(kvs, accountKvs...)
	//从叶子取款之后计算证明
	receipt, err = calProof(a.statedb, info, payload.AccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyTreeToContractLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

func (a *Action) UpdateContractAccount(addr string, amount string, tokenId uint64, option int32) ([]*types.KeyValue, error) {
	accountdb, _ := account.NewAccountDB(a.api.GetConfig(), zt.Zksync, strconv.Itoa(int(tokenId)), a.statedb)
	contractAccount := accountdb.LoadAccount(addr)
	change, _ := new(big.Int).SetString(amount, 10)
	//accountdb去除末尾10位小数
	shortChange := new(big.Int).Div(change, big.NewInt(1e10)).Int64()
	if option == zt.Sub {
		if contractAccount.Balance < shortChange {
			return nil, errors.New("balance not enough")
		}
		contractAccount.Balance -= shortChange
	} else {
		contractAccount.Balance += shortChange
	}

	kvs := accountdb.GetKVSet(contractAccount)
	zlog.Info("zkspot UpdateContractAccount", "key", string(kvs[0].GetKey()), "account", contractAccount)
	return kvs, nil
}

func (a *Action) Transfer(payload *zt.ZkTransfer) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	fee := zt.FeeMap[zt.TyTransferAction]
	//加上手续费
	amountInt, _ := new(big.Int).SetString(payload.Amount, 10)
	feeInt, _ := new(big.Int).SetString(fee, 10)
	totalAmount := new(big.Int).Add(amountInt, feeInt).String()

	err := checkParam(totalAmount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}
	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}
	fromLeaf, err := GetLeafByAccountId(a.statedb, payload.GetFromAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if fromLeaf == nil {
		return nil, errors.New("account not exist")
	}
	err = authVerification(payload.Signature.PubKey, fromLeaf.PubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	fromToken, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.FromAccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	err = checkAmount(fromToken, payload.GetAmount())
	if err != nil {
		return nil, errors.Wrapf(err, "db.checkAmount")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyTransferAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		FeeAmount:   fee,
		SigData:     payload.Signature,
		AccountID:   payload.FromAccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.FromAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	before := getBranchByReceipt(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId)

	//更新fromLeaf
	fromKvs, fromLocal, err := UpdateLeaf(a.statedb, a.localDB, info, fromLeaf.GetAccountId(), payload.GetTokenId(), totalAmount, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	kvs = append(kvs, fromKvs...)
	localKvs = append(localKvs, fromLocal...)
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.FromAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)

	toLeaf, err := GetLeafByAccountId(a.statedb, payload.ToAccountId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if toLeaf == nil {
		return nil, errors.New("account not exist")
	}

	//更新之前先计算证明
	receipt, err = calProof(a.statedb, info, payload.ToAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	var balance string
	if receipt.Token == nil {
		balance = "0"
	} else {
		balance = receipt.Token.Balance
	}
	before = getBranchByReceipt(receipt, operationInfo, toLeaf.EthAddress, toLeaf.Chain33Addr, toLeaf.PubKey, balance, payload.ToAccountId)

	//更新toLeaf
	tokvs, toLocal, err := UpdateLeaf(a.statedb, a.localDB, info, toLeaf.GetAccountId(), payload.GetTokenId(), payload.GetAmount(), zt.Add)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	kvs = append(kvs, tokvs...)
	localKvs = append(localKvs, toLocal...)
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.GetToAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	after = getBranchByReceipt(receipt, operationInfo, toLeaf.EthAddress, toLeaf.Chain33Addr, toLeaf.PubKey, receipt.Token.Balance, payload.ToAccountId)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch = &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyTransferLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)

	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}

	feeReceipt, err := a.MakeFeeLog(fee, info, payload.TokenId, payload.Signature)
	receipts = mergeReceipt(receipts, feeReceipt)
	return receipts, nil
}

func (a *Action) TransferToNew(payload *zt.ZkTransferToNew) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	fee := zt.FeeMap[zt.TyTransferToNewAction]
	//加上手续费
	amountInt, _ := new(big.Int).SetString(payload.Amount, 10)
	feeInt, _ := new(big.Int).SetString(fee, 10)
	totalAmount := new(big.Int).Add(amountInt, feeInt).String()

	err := checkParam(totalAmount)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}

	//转换10进制
	payload.ToChain33Address = zt.HexAddr2Decimal(payload.ToChain33Address)
	payload.ToEthAddress = zt.HexAddr2Decimal(payload.ToEthAddress)

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}
	fromLeaf, err := GetLeafByAccountId(a.statedb, payload.GetFromAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if fromLeaf == nil {
		return nil, errors.New("account not exist")
	}
	err = authVerification(payload.Signature.PubKey, fromLeaf.PubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	fromToken, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.FromAccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	err = checkAmount(fromToken, payload.GetAmount())
	if err != nil {
		return nil, errors.Wrapf(err, "db.checkAmount")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyTransferToNewAction,
		TokenID:     payload.TokenId,
		Amount:      payload.Amount,
		FeeAmount:   fee,
		SigData:     payload.Signature,
		AccountID:   payload.FromAccountId,
	}

	toLeaf, err := GetLeafByChain33AndEthAddress(a.statedb, payload.GetToChain33Address(), payload.GetToEthAddress(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByChain33AndEthAddress")
	}
	if toLeaf != nil {
		return nil, errors.New("to account already exist")
	}
	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.GetFromAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	before := getBranchByReceipt(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId)

	//更新fromLeaf
	fromkvs, fromLocal, err := UpdateLeaf(a.statedb, a.localDB, info, fromLeaf.GetAccountId(), payload.GetTokenId(), totalAmount, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	kvs = append(kvs, fromkvs...)
	localKvs = append(localKvs, fromLocal...)
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.GetFromAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)

	tree, err := getAccountTree(a.statedb, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.getAccountTree")
	}
	accountId := tree.GetTotalIndex() + 1
	//更新之前先计算证明
	receipt, err = calProof(a.statedb, info, accountId, payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	before = getBranchByReceipt(receipt, operationInfo, payload.ToEthAddress, payload.ToChain33Address, nil, "0", accountId)

	//新增toLeaf
	tokvs, toLocal, err := AddNewLeaf(a.statedb, a.localDB, info, payload.GetToEthAddress(), payload.GetTokenId(), payload.GetAmount(), payload.GetToChain33Address())
	if err != nil {
		return nil, errors.Wrapf(err, "db.AddNewLeaf")
	}
	kvs = append(kvs, tokvs...)
	localKvs = append(localKvs, toLocal...)
	//新增之后计算证明
	receipt, err = calProof(a.statedb, info, accountId, payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after = getBranchByReceipt(receipt, operationInfo, payload.ToEthAddress, payload.ToChain33Address, nil, receipt.Token.Balance, accountId)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch = &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyTransferToNewLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)

	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}

	feeReceipt, err := a.MakeFeeLog(fee, info, payload.TokenId, payload.Signature)
	receipts = mergeReceipt(receipts, feeReceipt)
	return receipts, nil
}

func (a *Action) ForceExit(payload *zt.ZkForceExit) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	fee := zt.FeeMap[zt.TyForceExitAction]

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}
	leaf, err := GetLeafByAccountId(a.statedb, payload.AccountId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	if leaf == nil {
		return nil, errors.New("account not exist")
	}

	token, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.AccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	//token不存在时，不需要取
	if token == nil {
		return nil, errors.New("token not find")
	}

	//加上手续费
	amountInt, _ := new(big.Int).SetString(token.Balance, 10)
	feeInt, _ := new(big.Int).SetString(fee, 10)
	//存量不够手续费时，不能取
	if amountInt.Cmp(feeInt) <= 0 {
		return nil, errors.New("no enough fee")
	}
	exitAmount := new(big.Int).Sub(amountInt, feeInt).String()

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyForceExitAction,
		TokenID:     payload.TokenId,
		Amount:      exitAmount,
		FeeAmount:   fee,
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.GetAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)

	//更新fromLeaf
	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), token.Balance, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.GetAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyForceExitLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}

	feeReceipt, err := a.MakeFeeLog(fee, info, payload.TokenId, payload.Signature)
	receipts = mergeReceipt(receipts, feeReceipt)
	return receipts, nil
}

func calProof(statedb dbm.KV, info *TreeUpdateInfo, accountId uint64, tokenId uint64) (*zt.ZkReceiptLeaf, error) {
	receipt := &zt.ZkReceiptLeaf{}

	leaf, err := GetLeafByAccountId(statedb, accountId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	receipt.Leaf = leaf

	token, err := GetTokenByAccountIdAndTokenId(statedb, accountId, tokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	receipt.Token = token

	leafProof, err := CalLeafProof(statedb, leaf, info)
	if err != nil {
		return nil, errors.Wrapf(err, "CalLeafProof")
	}
	receipt.TreeProof = leafProof

	tokenProof, err := CalTokenProof(statedb, leaf, token, info)
	if err != nil {
		return nil, errors.Wrapf(err, "CalTokenProof")
	}
	receipt.TokenProof = tokenProof

	return receipt, nil
}

func (a *Action) SetPubKey(payload *zt.ZkSetPubKey) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue
	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}

	leaf, err := GetLeafByAccountId(a.statedb, payload.GetAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByEthAddress")
	}
	if leaf == nil {
		return nil, errors.New("account not exist")
	}

	//校验预存的地址是否和公钥匹配
	hash := mimc.NewMiMC(zt.ZkMimcHashSeed)
	hash.Write(zt.Str2Byte(payload.PubKey.X))
	hash.Write(zt.Str2Byte(payload.PubKey.Y))
	if zt.Byte2Str(hash.Sum(nil)) != leaf.Chain33Addr {
		return nil, errors.New("not your account")
	}

	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TySetPubKeyAction,
		TokenID:     leaf.TokenIds[0],
		Amount:      "0",
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.AccountId, leaf.TokenIds[0])
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, nil, receipt.Token.Balance, operationInfo.AccountID)

	kvs, localKvs, err = UpdatePubKey(a.statedb, a.localDB, info, payload.GetPubKey(), payload.AccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.AccountId, leaf.TokenIds[0])
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, payload.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TySetPubKeyLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

func (a *Action) FullExit(payload *zt.ZkFullExit) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	fee := zt.FeeMap[zt.TyFullExitAction]

	//只有管理员能操作
	cfg := a.api.GetConfig()
	if !isSuperManager(cfg, a.fromaddr) && !isVerifier(a.statedb, a.fromaddr) {
		return nil, errors.Wrapf(types.ErrNotAllow, "from addr is not manager")
	}

	//fullexit last priority id 不能为空
	lastPriority, err := getLastEthPriorityQueueID(a.statedb, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "get eth last priority queue id")
	}
	lastId, ok := big.NewInt(0).SetString(lastPriority.GetID(), 10)
	if !ok {
		return nil, errors.Wrapf(types.ErrInvalidParam, fmt.Sprintf("getID =%s", lastPriority.GetID()))
	}

	//if lastId.Int64()+1 != payload.GetEthPriorityQueueId() {
	//	return nil, errors.Wrapf(types.ErrNotAllow, "eth last priority queue id=%s,new=%d", lastPriority.ID, payload.GetEthPriorityQueueId())
	//}

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}
	leaf, err := GetLeafByAccountId(a.statedb, payload.AccountId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	if leaf == nil {
		return nil, errors.New("account not exist")
	}

	token, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.AccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}

	//token不存在时，不需要取
	if token == nil {
		return nil, errors.New("token not find")
	}

	//加上手续费
	amountInt, _ := new(big.Int).SetString(token.Balance, 10)
	feeInt, _ := new(big.Int).SetString(fee, 10)
	//存量不够手续费时，不能取
	if amountInt.Cmp(feeInt) <= 0 {
		return nil, errors.New("no enough fee")
	}
	exitAmount := new(big.Int).Sub(amountInt, feeInt).String()

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TyFullExitAction,
		TokenID:     payload.TokenId,
		Amount:      exitAmount,
		FeeAmount:   fee,
		SigData:     payload.Signature,
		AccountID:   payload.AccountId,
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.GetAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)

	//更新fromLeaf
	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), payload.GetTokenId(), token.Balance, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.GetAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	zklog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyFullExitLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)
	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	//add priority part
	r := makeSetEthPriorityIdReceipt(0, lastId.Int64(), payload.EthPriorityQueueId)

	feeReceipt, err := a.MakeFeeLog(fee, info, payload.TokenId, payload.Signature)
	receipts = mergeReceipt(receipts, feeReceipt)
	return mergeReceipt(receipts, r), nil
}

//验证身份
func authVerification(signPubKey *zt.ZkPubKey, leafPubKey *zt.ZkPubKey) error {
	if signPubKey == nil || leafPubKey == nil {
		return errors.New("set your pubKey")
	}
	if signPubKey.GetX() != leafPubKey.GetX() || signPubKey.GetY() != leafPubKey.GetY() {
		return errors.New("not your account")
	}
	return nil
}

//检查参数
func checkParam(amount string) error {
	if amount == "" || amount == "0" || strings.HasPrefix(amount, "-") {
		return types.ErrAmount
	}
	return nil
}

func getLastEthPriorityQueueID(db dbm.KV, chainID uint32) (*zt.EthPriorityQueueID, error) {
	key := getEthPriorityQueueKey(chainID)
	v, err := db.Get(key)
	//未找到返回-1
	if isNotFound(err) {
		return &zt.EthPriorityQueueID{ID: "-1"}, nil
	}
	if err != nil {
		return nil, err
	}
	var id zt.EthPriorityQueueID
	err = types.Decode(v, &id)
	if err != nil {
		zklog.Error("getLastEthPriorityQueueID.decode", "err", err)
		return nil, err
	}

	return &id, nil
}

func makeSetEthPriorityIdReceipt(chainId uint32, prev, current int64) *types.Receipt {
	key := getEthPriorityQueueKey(chainId)
	log := &zt.ReceiptEthPriorityQueueID{
		Prev:    prev,
		Current: current,
	}
	return &types.Receipt{
		Ty: types.ExecOk,
		KV: []*types.KeyValue{
			{Key: key, Value: types.Encode(&zt.EthPriorityQueueID{ID: big.NewInt(current).String()})},
		},
		Logs: []*types.ReceiptLog{
			{
				Ty:  zt.TySetEthPriorityQueueId,
				Log: types.Encode(log),
			},
		},
	}
}

func mergeReceipt(receipt1, receipt2 *types.Receipt) *types.Receipt {
	if receipt2 != nil {
		receipt1.KV = append(receipt1.KV, receipt2.KV...)
		receipt1.Logs = append(receipt1.Logs, receipt2.Logs...)
	}

	return receipt1
}

func (a *Action) MakeFeeLog(amount string, info *TreeUpdateInfo, tokenId uint64, sign *zt.ZkSignature) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue
	var err error

	//todo 手续费收款方accountId可配置
	leaf, err := GetLeafByAccountId(a.statedb, 1, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}

	if leaf == nil {
		return nil, errors.New("account not exist")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		OpIndex:     1,
		TxType:      zt.TyFeeAction,
		TokenID:     tokenId,
		Amount:      amount,
		SigData:     sign,
		AccountID:   leaf.GetAccountId(),
	}

	//leaf不存在就添加

	receipt, err := calProof(a.statedb, info, leaf.AccountId, tokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	var balance string
	if receipt.Token == nil {
		balance = "0"
	} else {
		balance = receipt.Token.Balance
	}
	before := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, balance, leaf.GetAccountId())

	kvs, localKvs, err = UpdateLeaf(a.statedb, a.localDB, info, leaf.GetAccountId(), tokenId, amount, zt.Add)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	receipt, err = calProof(a.statedb, info, leaf.AccountId, tokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	after := getBranchByReceipt(receipt, operationInfo, leaf.EthAddress, leaf.Chain33Addr, leaf.PubKey, receipt.Token.Balance, operationInfo.AccountID)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	feelog := &zt.ZkReceiptLog{
		OperationInfo: operationInfo,
		LocalKvs:      localKvs,
	}
	receiptLog := &types.ReceiptLog{Ty: zt.TyFeeLog, Log: types.Encode(feelog)}
	logs = append(logs, receiptLog)

	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

// 处理撮合结果
//  如果是 按不同的交易类型来处理的话, 零知识证明部分的代码, 会随交易的多样化, 需要也写很多函数来支持.
//  所以结果最好以 结算的形式作为参数.
//  不同交易的结果, 转化为有限的几种结算
//    主动结算: (用户地址发起的交易)    如: 撮合
//    被动结算: (系统特定帐号发起的交易) 如: 永续中暴仓, 和资金费
//  结算的列表以结果的形式体现帐号的变化, 和具体的业务无关
func (a *Action) SpotMatch(payload *et.SpotLimitOrder, list *types.Receipt) (*types.Receipt, error) {
	receipt := &types.Receipt{}
	for _, tradeRaw := range list.Logs {
		switch tradeRaw.Ty {
		case et.TySpotTradeLog:
			var trade et.ReceiptSpotTrade
			err := types.Decode(tradeRaw.Log, &trade)
			if err != nil {
				return nil, err
			}
			receipt2, err := a.Swap(payload, &trade)
			if err != nil {
				return nil, err
			}
			receipt = mergeReceipt(receipt, receipt2)
		default:
			//
		}
	}
	return receipt, nil
}

// A 和 B 交换 = transfer(A,B) + transfer(B,A) + transfer(A,fee) + transfer(B,fee)
// A 和 A 交换 = transfer(A,A) 0 + transfer(A,A) 0 + transfer(A,fee) + transfer(B,fee)
// fee 先不处理, 因为交易本身就收了手续费
func (a *Action) Swap(payload1 *et.SpotLimitOrder, trade *et.ReceiptSpotTrade) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue

	info, err := generateTreeUpdateInfo(a.statedb)
	if err != nil {
		return nil, errors.Wrapf(err, "db.generateTreeUpdateInfo")
	}

	operationInfo := &zt.OperationInfo{
		BlockHeight: uint64(a.height),
		TxIndex:     uint32(a.index),
		TxType:      zt.TySwapAction,
		TokenID:     uint64(payload1.LeftAsset),
		Amount:      new(big.Int).SetInt64(payload1.Amount).String(),
		FeeAmount:   "0",
		SigData:     payload1.GetOrder().Signature,
		AccountID:   payload1.Order.AccountID,
	}

	// A 和 B 交易, 构造4个transfer, 使用transfer 实现
	// A 和 A 交易, 构造4个transfer, 0 swap *2  收取手续费*2
	zklog.Debug("swapGenTransfer", "trade-buy", trade.MakerOrder.TokenBuy, "trade-sell", trade.MakerOrder.TokenSell)
	transfers := a.swapGenTransfer(payload1, trade)
	zklog.Debug("swapGenTransfer", "tokenid0", transfers[0].TokenId, "tokenid1", transfers[1].TokenId)
	// operationInfo, localKvs 通过 zklog 获得
	zklog := &zt.ZkReceiptLog{OperationInfo: operationInfo}
	//for _, transfer1 := range transfers {
	receipt1, err := a.swapByTransfer(transfers[0], payload1, trade, info, zklog)
	if err != nil {
		return nil, err
	}
	logs = append(logs, receipt1.Logs...)
	kvs = append(kvs, receipt1.KV...)
	receipt2, err := a.swapByTransfer(transfers[1], payload1, trade, info, zklog)
	if err != nil {
		return nil, err
	}
	logs = append(logs, receipt2.Logs...)
	kvs = append(kvs, receipt2.KV...)

	receiptLog := &types.ReceiptLog{Ty: zt.TySwapLog, Log: types.Encode(zklog)}
	logs = append(logs, receiptLog)

	feelog1, err := a.MakeFeeLog(transfers[2].AmountIn, info, transfers[2].TokenId, transfers[2].Signature)
	if err != nil {
		return nil, err
	}

	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	receipts = mergeReceipt(receipts, feelog1)
	return receipts, nil
}

// 将参加放到 ZkTransfer, 可以方便的修改 Transfer的实现
func (a *Action) swapByTransfer(payload *et.ZkTransferWithFee, payload1 *et.SpotLimitOrder, trade *et.ReceiptSpotTrade, info *TreeUpdateInfo, zklog *zt.ZkReceiptLog) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	var localKvs []*types.KeyValue

	operationInfo := zklog.OperationInfo
	//加上手续费
	amountOutTmp, _ := new(big.Int).SetString(payload.AmountOut, 10)
	amountOut := amountOutTmp.String()

	amountInTmp, _ := new(big.Int).SetString(payload.AmountIn, 10)
	amountIn := amountInTmp.String()

	err := checkParam(amountOut)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}
	err = checkParam(amountIn)
	if err != nil {
		return nil, errors.Wrapf(err, "checkParam")
	}

	fromLeaf, err := GetLeafByAccountId(a.statedb, payload.GetFromAccountId(), info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if fromLeaf == nil {
		return nil, errors.New("account not exist")
	}

	err = authVerification(payload.Signature.PubKey, fromLeaf.PubKey)
	if err != nil {
		return nil, errors.Wrapf(err, "authVerification")
	}

	fromToken, err := GetTokenByAccountIdAndTokenId(a.statedb, payload.FromAccountId, payload.TokenId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetTokenByAccountIdAndTokenId")
	}
	err = checkAmount(fromToken, amountOut)
	if err != nil {
		return nil, errors.Wrapf(err, "db.checkAmount")
	}

	//更新之前先计算证明
	receipt, err := calProof(a.statedb, info, payload.FromAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	before := getBranchByReceipt1(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId, payload.TokenId)
	// after
	//更新fromLeaf
	fromKvs, fromLocal, err := UpdateLeaf(a.statedb, a.localDB, info, fromLeaf.GetAccountId(), payload.TokenId, amountOut, zt.Sub)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	kvs = append(kvs, fromKvs...)
	localKvs = append(localKvs, fromLocal...)
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.FromAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}

	after := getBranchByReceipt1(receipt, operationInfo, fromLeaf.EthAddress, fromLeaf.Chain33Addr, fromLeaf.PubKey, receipt.Token.Balance, payload.FromAccountId, payload.TokenId)

	branch := &zt.OperationPairBranch{
		Before: before,
		After:  after,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch)
	// 2before
	toLeaf, err := GetLeafByAccountId(a.statedb, payload.ToAccountId, info)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GetLeafByAccountId")
	}
	if toLeaf == nil {
		return nil, errors.New("account not exist")
	}

	//更新之前先计算证明
	receipt, err = calProof(a.statedb, info, payload.ToAccountId, payload.TokenId)
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	var balance string
	if receipt.Token == nil {
		balance = "0"
	} else {
		balance = receipt.Token.Balance
	}
	before2 := getBranchByReceipt1(receipt, operationInfo, toLeaf.EthAddress, toLeaf.Chain33Addr, toLeaf.PubKey, balance, payload.ToAccountId, payload.TokenId)
	// 2after
	//更新toLeaf
	tokvs, toLocal, err := UpdateLeaf(a.statedb, a.localDB, info, toLeaf.GetAccountId(), payload.GetTokenId(), amountIn, zt.Add)
	if err != nil {
		return nil, errors.Wrapf(err, "db.UpdateLeaf")
	}
	kvs = append(kvs, tokvs...)
	localKvs = append(localKvs, toLocal...)
	//更新之后计算证明
	receipt, err = calProof(a.statedb, info, payload.GetToAccountId(), payload.GetTokenId())
	if err != nil {
		return nil, errors.Wrapf(err, "calProof")
	}
	after2 := getBranchByReceipt1(receipt, operationInfo, toLeaf.EthAddress, toLeaf.Chain33Addr, toLeaf.PubKey, receipt.Token.Balance, payload.ToAccountId, payload.TokenId)
	rootHash := zt.Str2Byte(receipt.TreeProof.RootHash)
	kv := &types.KeyValue{
		Key:   getHeightKey(a.height),
		Value: rootHash,
	}
	kvs = append(kvs, kv)

	branch2 := &zt.OperationPairBranch{
		Before: before2,
		After:  after2,
	}
	operationInfo.OperationBranches = append(operationInfo.GetOperationBranches(), branch2)

	// 返回 operationInfo (本来就是引用zklog) localKvs
	zklog.LocalKvs = append(zklog.LocalKvs, localKvs...)

	receipts := &types.Receipt{Ty: types.ExecOk, KV: kvs, Logs: logs}
	return receipts, nil
}

func (a *Action) swapGenTransfer(payload1 *et.SpotLimitOrder, trade *et.ReceiptSpotTrade) []*et.ZkTransferWithFee {
	// A 和 A 交易
	var transfers []*et.ZkTransferWithFee
	if trade.Current.Maker.Addr == trade.Current.Taker.Addr {
		return a.selfSwapGenTransfer(payload1, trade)
	}

	// A 和 B 交易
	// Sell
	takerTokenID, makerTokenID := payload1.LeftAsset, payload1.RightAsset
	takerPay, makerRcv := trade.Match.LeftBalance, trade.Match.LeftBalance

	rightBalance := trade.Match.RightBalance
	takerRcv := rightBalance - trade.Match.FeeTaker
	makerPay := rightBalance + trade.Match.FeeMaker
	fee := trade.Match.FeeMaker + trade.Match.FeeTaker

	if payload1.Op == et.OpBuy {
		takerTokenID, makerTokenID = makerTokenID, takerTokenID
		takerRcv, makerPay = trade.Match.LeftBalance, trade.Match.LeftBalance

		takerPay, makerRcv = rightBalance+trade.Match.FeeTaker, rightBalance-trade.Match.FeeMaker
	}

	taker1 := &et.ZkTransferWithFee{
		TokenId:       uint64(takerTokenID),
		AmountOut:     new(big.Int).SetInt64(takerPay).String(),
		FromAccountId: payload1.Order.AccountID,
		ToAccountId:   trade.Current.Maker.Id,
		Signature:     payload1.Order.Signature,
		AmountIn:      new(big.Int).SetInt64(makerRcv).String(),
	}
	maker1 := &et.ZkTransferWithFee{
		TokenId:       uint64(makerTokenID),
		AmountOut:     new(big.Int).SetInt64(makerPay).String(),
		FromAccountId: trade.Current.Maker.Id,
		ToAccountId:   payload1.Order.AccountID,
		Signature:     trade.MakerOrder.Signature,
		AmountIn:      new(big.Int).SetInt64(takerRcv).String(),
	}
	fee1 := &et.ZkTransferWithFee{
		TokenId:       uint64(payload1.RightAsset),
		AmountOut:     new(big.Int).SetInt64(0).String(),
		FromAccountId: payload1.Order.AccountID,
		ToAccountId:   trade.Current.Fee.Id,
		Signature:     payload1.Order.Signature,
		AmountIn:      new(big.Int).SetInt64(fee).String(),
	}

	transfers = append(transfers, taker1)
	transfers = append(transfers, maker1)
	transfers = append(transfers, fee1)
	elog.Error("swapGenTransfer", "takerPay", takerPay, "takerRcv", takerRcv,
		"makerPay", makerPay, "makerRcv", makerRcv)
	return transfers
}

// A 和 A 交易时, 也需要构造4个transfer
// maker/taker 由于是同一个帐号, 所以takerPay makerPay 为0
func (a *Action) selfSwapGenTransfer(payload1 *et.SpotLimitOrder, trade *et.ReceiptSpotTrade) []*et.ZkTransferWithFee {
	// A 和 A 交易, 构造3个transfer, 使用transfer 实现
	var transfers []*et.ZkTransferWithFee
	leftPay, rightPay := int64(0), trade.Match.FeeTaker+trade.Match.FeeMaker

	left := &et.ZkTransferWithFee{
		TokenId:       uint64(payload1.LeftAsset),
		AmountOut:     new(big.Int).SetInt64(leftPay).String(),
		FromAccountId: payload1.Order.AccountID,
		ToAccountId:   trade.Current.Maker.Id,
		Signature:     payload1.Order.Signature,
		AmountIn:      new(big.Int).SetInt64(leftPay).String(),
	}
	right := &et.ZkTransferWithFee{
		TokenId:       uint64(payload1.RightAsset),
		AmountOut:     new(big.Int).SetInt64(rightPay).String(),
		FromAccountId: trade.Current.Maker.Id,
		ToAccountId:   payload1.Order.AccountID,
		Signature:     trade.MakerOrder.Signature,
		AmountIn:      new(big.Int).SetInt64(0).String(),
	}
	fee := &et.ZkTransferWithFee{
		TokenId:       uint64(payload1.RightAsset),
		AmountOut:     new(big.Int).SetInt64(0).String(),
		FromAccountId: payload1.Order.AccountID,
		ToAccountId:   trade.Current.Fee.Id,
		Signature:     payload1.Order.Signature,
		AmountIn:      new(big.Int).SetInt64(rightPay).String(),
	}

	transfers = append(transfers, left)
	transfers = append(transfers, right)
	transfers = append(transfers, fee)
	return transfers
}
