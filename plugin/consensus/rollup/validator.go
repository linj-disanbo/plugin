package rollup

import (
	"bytes"
	"encoding/hex"
	"sync"

	"github.com/33cn/chain33/system/crypto/secp256k1"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/types"
	"github.com/33cn/plugin/plugin/crypto/bls"
	rtypes "github.com/33cn/plugin/plugin/dapp/rollup/types"
)

type validator struct {
	lock             sync.RWMutex
	enable           bool
	commitRoundIndex int32
	blsKey           crypto.PrivKey
	signTxKey        crypto.PrivKey
	commitAddr       string
	validators       map[string]int
	valPubHash       []byte
	timeout          bool
	blsDriver        crypto.Crypto
	status           *rtypes.RollupStatus
	exit             chan struct{}
}

func getPrivKey(cryptoName, privKey string) (crypto.Crypto, crypto.PrivKey) {

	if privKey == "" {
		panic("Rollup empty validator privKey")
	}
	driver, err := crypto.Load(cryptoName, -1)
	if err != nil {
		panic("RollUp load crypto driver err:" + err.Error())
	}
	privByte, err := common.FromHex(privKey)
	if err != nil {
		panic("RollUp decode hex key err:" + err.Error())
	}
	key, err := driver.PrivKeyFromBytes(privByte)
	if err != nil {
		panic("RollUp priv key from bytes err:" + err.Error())
	}

	return driver, key
}

func (v *validator) init(cfg Config, valPubs *rtypes.ValidatorPubs, status *rtypes.RollupStatus) {

	if cfg.CommitInterval <= 0 {
		cfg.CommitInterval = 30
	}

	v.exit = make(chan struct{})
	v.blsDriver, v.blsKey = getPrivKey(bls.Name, cfg.ValidatorBlsKey)
	_, v.signTxKey = getPrivKey(secp256k1.Name, cfg.CommitTxKey)
	v.updateValidators(valPubs)
	v.updateRollupStatus(status)

}

func (v *validator) isMyCommitTurn() (int64, bool) {

	v.lock.RLock()
	defer v.lock.RUnlock()

	nextCommitRound := v.status.GetCommitRound() + 1
	roundIdx := int32(nextCommitRound % int64(len(v.validators)))

	if v.commitRoundIndex == roundIdx {
		return nextCommitRound, true
	}

	waitTime := types.Now().Unix() - v.status.Timestamp
	// 达到一半超时, 即触发由上一个提交者代理提交
	if waitTime >= rtypes.RollupCommitTimeout/2 && v.status.CommitAddr == v.commitAddr {
		return nextCommitRound, true
	}

	// 超时情况, 任意其他节点代理提交
	if waitTime >= rtypes.RollupCommitTimeout {
		return nextCommitRound, true
	}

	return -1, false
}

func (v *validator) updateRollupStatus(status *rtypes.RollupStatus) {

	v.lock.Lock()
	defer v.lock.Unlock()

	v.status = status
}

func (v *validator) getValidatorCount() int {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return len(v.validators)
}

func (v *validator) updateValidators(valPubs *rtypes.ValidatorPubs) {
	v.lock.Lock()
	defer v.lock.Unlock()

	hash := common.Sha256(types.Encode(valPubs))
	// 数据没有变更, 直接返回
	if bytes.Equal(v.valPubHash, hash) {
		return
	}
	// 更新验证节点
	v.valPubHash = hash
	v.validators = make(map[string]int, len(valPubs.GetBlsPubs()))

	for i, pub := range valPubs.GetBlsPubs() {
		pub = rtypes.FormatHexPubKey(pub)
		v.validators[pub] = i
	}

	blsPub := hex.EncodeToString(v.blsKey.PubKey().Bytes())
	idx, ok := v.validators[blsPub]

	v.enable = ok
	v.commitRoundIndex = int32(idx)
	if !v.enable {
		close(v.exit)
	}
}

func (v *validator) validateSignMsg(sign *rtypes.ValidatorSignMsg) bool {

	v.lock.RLock()
	defer v.lock.RUnlock()
	pub := hex.EncodeToString(sign.PubKey)

	_, ok := v.validators[pub]
	if !ok {
		rlog.Error("validateSignMsg invalid node", "round", sign.CommitRound, "pub", pub)
		return false
	}

	if err := v.blsDriver.Validate(sign.MsgHash, sign.PubKey, sign.Signature); err != nil {
		rlog.Error("validateSignMsg invalid sign",
			"round", sign.CommitRound, "pub", pub, "err", err)
		return false
	}
	return true
}

func (v *validator) sign(round int64, batch *rtypes.BlockBatch) *rtypes.ValidatorSignMsg {

	msg := common.Sha256(types.Encode(batch))
	sign := &rtypes.ValidatorSignMsg{}
	sign.Signature = v.blsKey.Sign(msg).Bytes()
	sign.PubKey = v.blsKey.PubKey().Bytes()
	sign.CommitRound = round
	sign.MsgHash = msg

	return sign
}

type aggreSignFunc = func(set *validatorSignMsgSet) (pubs [][]byte, aggreSign []byte)

func (v *validator) aggregateSign(set *validatorSignMsgSet) (pubs [][]byte, aggreSign []byte) {

	if set == nil {
		return nil, nil
	}
	valCount := v.getValidatorCount()
	// 2/3 共识, 向上取整
	minSignCount := valCount * 2 / 3
	if valCount%3 != 0 {
		minSignCount++
	}
	if len(set.others)+1 < minSignCount {
		rlog.Debug("aggregateSign", "commitRound", set.self.CommitRound,
			"valCount", valCount, "signCount", len(set.others)+1)
		return nil, nil
	}

	pubs = make([][]byte, 0, len(set.others)+1)
	signs := make([]crypto.Signature, 0, len(set.others)+1)

	s, _ := v.blsDriver.SignatureFromBytes(set.self.Signature)
	signs = append(signs, s)
	pubs = append(pubs, set.self.PubKey)
	for i := 0; i < len(set.others); {
		sign := set.others[i]
		// 数据哈希不一致, 非法签名
		if !bytes.Equal(sign.MsgHash, set.self.MsgHash) {

			set.others = append(set.others[:i], set.others[i+1:]...)
			rlog.Error("aggregateSign msgHash not equal", "commitRound", set.self.CommitRound,
				"selfHash", hex.EncodeToString(set.self.MsgHash),
				"otherHash", hex.EncodeToString(sign.MsgHash))
			continue
		}
		s, _ = v.blsDriver.SignatureFromBytes(sign.GetSignature())
		signs = append(signs, s)
		pubs = append(pubs, sign.PubKey)
		i++
	}

	blsAggre := v.blsDriver.(crypto.AggregateCrypto)
	s, err := blsAggre.Aggregate(signs[:minSignCount])
	if err != nil {
		rlog.Error("aggregateSign", "commitRound", set.self.CommitRound, "aggre err", err)
		return nil, nil
	}

	return pubs[:minSignCount], s.Bytes()
}
