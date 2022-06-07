package spot

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

var (
	// mavl-zkspot-dex-   资金帐号
	// mavl-zkspot-spot-  现货帐号
	// 先都用现货帐号
	spotAccountKey = []byte("mavl-zkspot-spot-")
	spotDexName    = "spot"
	dexAccountKey  = []byte("mavl-zkspot-dex-")
	//spotAccountKey =
	//spot           = []byte("spot")
	spotFeeAccountKey = []byte("zkspot-spotfeeaccount") // mavl-manager-{here}
)

type Dex struct {
	dexName   string
	keyPrefix []byte
}

type SpotDex struct {
	Dex
}

func newSpotDex() *SpotDex {
	return &SpotDex{
		Dex: Dex{
			dexName:   spotDexName,
			keyPrefix: spotAccountKey,
		},
	}
}

func genAccountKey(dexType []byte, addr string, id uint64) []byte {
	return []byte(fmt.Sprintf("%s:%d", dexType, id))
}

func LoadSpotAccount(addr string, id uint64, db dbm.KV) (*dexAccount, error) {
	return newSpotDex().LoadAccount(addr, id, db)
}

func (dex *Dex) LoadAccount(addr string, id uint64, db dbm.KV) (*dexAccount, error) {
	key := genAccountKey(dex.keyPrefix, addr, id)
	v, err := db.Get(key)
	if err == types.ErrNotFound {
		acc := NewDexAccount(dex.dexName, id, addr)
		acc.db = db
		return acc, nil
	}
	var acc et.DexAccount
	err = types.Decode(v, &acc)
	if err != nil {
		return nil, err
	}

	return GetDexAccount(&acc, db), nil
}

type dexAccount struct {
	ty  string // spot, future, asset
	acc *et.DexAccount
	db  dbm.KV
}

func NewDexAccount(ty string, id uint64, addr string) *dexAccount {
	return &dexAccount{
		ty: ty,
		acc: &et.DexAccount{
			Id:      id,
			Addr:    addr,
			DexName: ty,
		},
	}
}

func GetDexAccount(acc *et.DexAccount, db dbm.KV) *dexAccount {
	return &dexAccount{acc: acc, db: db}
}

func (acc *dexAccount) findTokenIndex(tid uint32) int {
	for i, token := range acc.acc.Balance {
		if token.Id == tid {
			return i
		}
	}
	return -1
}

func (acc *dexAccount) newToken(tid uint32, amount uint64) int {
	acc.acc.Balance = append(acc.acc.Balance, &et.DexAccountBalance{
		Id:      tid,
		Balance: amount,
	})
	return len(acc.acc.Balance) - 1
}
func (acc *dexAccount) GetBalance(tid uint32) uint64 {
	return acc.getBalance(tid)
}

func (acc *dexAccount) getBalance(tid uint32) uint64 {
	idx := acc.findTokenIndex(tid)
	if idx == -1 {
		return 0
	}
	return acc.acc.Balance[idx].Balance
}

func (acc *dexAccount) getFrozen(tid uint32) uint64 {
	idx := acc.findTokenIndex(tid)
	if idx == -1 {
		return 0
	}
	return acc.acc.Balance[idx].Frozen
}

func (acc *dexAccount) doMint(tid uint32, amount uint64) error {
	idx := acc.findTokenIndex(tid)
	if idx == -1 {
		acc.acc.Balance = append(acc.acc.Balance, &et.DexAccountBalance{
			Id:      tid,
			Balance: amount,
		})
	} else {
		acc.acc.Balance[idx].Balance += amount
	}
	return nil
}

func (acc *dexAccount) doBurn(tid uint32, amount uint64) error {
	idx := acc.findTokenIndex(tid)
	if idx == -1 {
		return et.ErrDexNotEnough
	}

	if acc.acc.Balance[idx].Balance < amount {
		return et.ErrDexNotEnough
	}

	acc.acc.Balance[idx].Balance -= amount
	return nil
}

func (acc *dexAccount) doFrozen(token uint32, amount uint64) error {
	idx := acc.findTokenIndex(token)
	if idx < 0 {
		return et.ErrDexNotEnough
	}
	if acc.acc.Balance[idx].Balance < amount {
		return et.ErrDexNotEnough
	}
	acc.acc.Balance[idx].Balance -= amount
	acc.acc.Balance[idx].Frozen += amount
	return nil
}

func (acc *dexAccount) doActive(token uint32, amount uint64) error {
	idx := acc.findTokenIndex(token)
	if idx < 0 {
		return et.ErrDexNotEnough
	}
	if acc.acc.Balance[idx].Frozen < amount {
		return et.ErrDexNotEnough
	}
	acc.acc.Balance[idx].Balance += amount
	acc.acc.Balance[idx].Frozen -= amount
	return nil
}

func dupAccount(acc *et.DexAccount) *et.DexAccount {
	copyAcc := *acc
	var bs []*et.DexAccountBalance
	for _, b := range acc.Balance {
		copyB := *b
		bs = append(bs, &copyB)
	}
	copyAcc.Balance = bs
	return &copyAcc
}

// GetKVSet account to statdb kv
func (acc *dexAccount) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(acc.acc)
	key := genAccountKey(spotAccountKey, acc.acc.Addr, acc.acc.Id)
	acc.db.Set(key, value)

	kvset = make([]*types.KeyValue, 1)
	kvset[0] = &types.KeyValue{
		Key:   key,
		Value: value,
	}
	return kvset
}

func (acc *dexAccount) Frozen(token uint32, amount uint64) (*types.Receipt, error) {
	return acc.updateWithFunc(token, amount, acc.doFrozen, et.TyDexAccountFrozen)
}

func (acc *dexAccount) Active(token uint32, amount uint64) (*types.Receipt, error) {
	return acc.updateWithFunc(token, amount, acc.doActive, et.TyDexAccountActive)
}

func (acc *dexAccount) Mint(token uint32, amount uint64) (*types.Receipt, error) {
	return acc.updateWithFunc(token, amount, acc.doMint, et.TyDexAccountMint)
}

func (acc *dexAccount) Burn(token uint32, amount uint64) (*types.Receipt, error) {
	return acc.updateWithFunc(token, amount, acc.doBurn, et.TyDexAccountBurn)
}

type updateDexAccount func(token uint32, amount uint64) error

func (acc *dexAccount) updateWithFunc(token uint32, amount uint64, f updateDexAccount, logType int32) (*types.Receipt, error) {
	copyAcc := dupAccount(acc.acc)
	err := f(token, amount)
	if err != nil {
		return nil, err
	}
	receiptlog := et.ReceiptDexAccount{
		Prev:    copyAcc,
		Current: acc.acc,
	}

	return acc.genReceipt(logType, acc, &receiptlog), nil
}

func (acc *dexAccount) genReceipt(ty int32, acc1 *dexAccount, r *et.ReceiptDexAccount) *types.Receipt {
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r),
	}
	kv := acc.GetKVSet()
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}
}

// two account operator

// 撮合 包含 1个交换, 和两个手续费
// 币的源头是是从balance/frozen 中转 看balance 的中值是否为frozen
// 币的目的一般到 balance即可, 如果有到frozen的 提供额外的函数或参数
/*
func (acc *dexAccount) Swap(accTo *dexAccount, got, gave *et.DexAccountBalance) error {
	err := acc.Tranfer(accTo, gave)
	if err != nil {
		return err
	}
	return acc.Withdraw(accTo, got)
}
*/

func (acc *dexAccount) doTranfer(accTo *dexAccount, token uint32, balance uint64) error {
	idx := acc.findTokenIndex(token)
	if idx < 0 {
		return et.ErrDexNotEnough
	}
	idxTo := accTo.findTokenIndex(token)
	if idxTo < 0 {
		idxTo = accTo.newToken(token, 0)
	}

	if acc.acc.Balance[idx].Balance < balance {
		return et.ErrDexNotEnough
	}

	acc.acc.Balance[idx].Balance -= balance
	accTo.acc.Balance[idxTo].Balance += balance

	return nil
}

func (acc *dexAccount) doFrozenTranfer(accTo *dexAccount, token uint32, amount uint64) error {
	idx := acc.findTokenIndex(token)
	if idx < 0 {
		return et.ErrDexNotEnough
	}
	idxTo := accTo.findTokenIndex(token)
	if idxTo < 0 {
		idxTo = accTo.newToken(token, 0)
	}

	if acc.acc.Balance[idx].Frozen < amount {
		return et.ErrDexNotEnough
	}

	acc.acc.Balance[idx].Frozen -= amount
	accTo.acc.Balance[idxTo].Balance += amount
	return nil
}
