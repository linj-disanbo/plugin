package executor

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
func genAccountKey(dexType []byte, id uint64) []byte {
	return []byte(fmt.Sprintf("%s:016%d:%016d", dexType, id))
}

func LoadSpotAccount(addr string, id uint64, db dbm.KV) (*dexAccount, error) {
	return newSpotDex().LoadAccount(addr, id, db)
}

func (dex *Dex) LoadAccount(addr string, id uint64, db dbm.KV) (*dexAccount, error) {
	key := genAccountKey(dex.keyPrefix, id)
	v, err := db.Get(key)
	if err == types.ErrNotFound {
		return NewDexAccount(dex.dexName, id, addr), nil
	}
	var acc et.DexAccount
	err = types.Decode(v, &acc)
	if err != nil {
		return nil, err
	}

	return GetDexAccount(&acc), nil
}

type dexAccount struct {
	ty  string // spot, future, asset
	acc *et.DexAccount
}

// 先写逻辑
// TODO 需要增加 kv , receipt的处理
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

func GetDexAccount(acc *et.DexAccount) *dexAccount {
	return &dexAccount{acc: acc}
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

func (acc *dexAccount) Mint(tid uint32, amount uint64) {
	idx := acc.findTokenIndex(tid)
	if idx == -1 {
		acc.acc.Balance = append(acc.acc.Balance, &et.DexAccountBalance{
			Id:      tid,
			Balance: amount,
		})
	} else {
		acc.acc.Balance[idx].Balance += amount
	}
}

func (acc *dexAccount) Burn(tid uint32, amount uint64) error {
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

// 撮合 包含 1个交换, 和两个手续费
// 币的源头是是从balance/frozen 中转 看balance 的中值是否为frozen
// 币的目的一般到 balance即可, 如果有到frozen的 提供额外的函数或参数

func (acc *dexAccount) Swap(accTo *dexAccount, got, gave *et.DexAccountBalance) error {
	err := acc.Tranfer(accTo, gave)
	if err != nil {
		return err
	}
	return acc.Withdraw(accTo, got)
}

func (acc *dexAccount) Tranfer(accTo *dexAccount, b *et.DexAccountBalance) error {
	idx := acc.findTokenIndex(b.Id)
	if idx < 0 {
		return et.ErrDexNotEnough
	}
	idxTo := accTo.findTokenIndex(b.Id)
	if idxTo < 0 {
		idxTo = acc.newToken(b.Id, 0)
	}
	if b.Balance > 0 {
		if acc.acc.Balance[idx].Balance < b.Balance {
			return et.ErrDexNotEnough
		}
		acc.acc.Balance[idx].Balance -= b.Balance
		accTo.acc.Balance[idxTo].Balance += b.Balance
	}
	if b.Frozen > 0 {
		if acc.acc.Balance[idx].Frozen < b.Frozen {
			return et.ErrDexNotEnough
		}
		acc.acc.Balance[idx].Frozen -= b.Frozen
		accTo.acc.Balance[idxTo].Balance += b.Balance
	}
	return nil
}

func (acc *dexAccount) Withdraw(accTo *dexAccount, b *et.DexAccountBalance) error {
	return accTo.Tranfer(acc, b)
}

func (acc *dexAccount) Frozen(token uint32, amount uint64) error {
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

func (acc *dexAccount) Active(token uint32, amount uint64) error {
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

func (acc *dexAccount) FrozenTranfer(accTo *dexAccount, tid uint32, amount uint64) error {
	b := et.DexAccountBalance{
		Id:     tid,
		Frozen: amount,
	}
	return acc.Tranfer(accTo, &b)
}
