package types

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/33cn/chain33/types"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/pkg/errors"
)

func Str2Byte(v string) []byte {
	var f fr.Element
	f.SetString(v)
	b := f.Bytes()
	return b[:]
}

func Byte2Str(v []byte) string {
	var f fr.Element
	f.SetBytes(v)
	return f.String()
}

func Byte2Uint64(v []byte) uint64 {
	return new(big.Int).SetBytes(v).Uint64()
}

// HexAddr2Decimal 16进制地址转10进制
func HexAddr2Decimal(addr string) string {
	addrInt, _ := new(big.Int).SetString(strings.ToLower(addr), 16)
	return addrInt.String()
}

// DecimalAddr2Hex 10进制地址转16进制
func DecimalAddr2Hex(addr string) string {
	addrInt, _ := new(big.Int).SetString(strings.ToLower(addr), 10)
	return hex.EncodeToString(addrInt.Bytes())
}

func SplitNFTContent(contentHash string) (*big.Int, *big.Int, string, error) {
	hexContent := strings.ToLower(contentHash)
	if hexContent[0:2] == "0x" || hexContent[0:2] == "0X" {
		hexContent = hexContent[2:]
	}

	if len(hexContent) != 64 {
		return nil, nil, "", errors.Wrapf(types.ErrInvalidParam, "contentHash not 64 len, %s", hexContent)
	}
	part1, ok := big.NewInt(0).SetString(hexContent[:32], 16)
	if !ok {
		return nil, nil, "", errors.Wrapf(types.ErrInvalidParam, "contentHash.preHalf hex err, %s", hexContent[:32])
	}
	part2, ok := big.NewInt(0).SetString(hexContent[32:], 16)
	if !ok {
		return nil, nil, "", errors.Wrapf(types.ErrInvalidParam, "contentHash.postHalf hex err, %s", hexContent[32:])
	}
	return part1, part2, hexContent, nil
}

// eth precision : 1e18, chain33 precision : 1e8
const (
	precisionDiff = 1e10
)

func AmountFromZksync(s string) (uint64, error) {
	zkAmount, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return 0, ErrAssetAmount
	}
	chain33Amount := new(big.Int).Div(zkAmount, big.NewInt(precisionDiff))
	if !chain33Amount.IsUint64() {
		return 0, ErrAssetAmount
	}
	return chain33Amount.Uint64(), nil
}

func AmountToZksync(a uint64) string {
	amount := new(big.Int).Mul(new(big.Int).SetUint64(a), big.NewInt(precisionDiff))
	return amount.String()
}
