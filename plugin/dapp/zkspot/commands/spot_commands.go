/*Package commands implement dapp client commands*/
package commands

import (
	"math/big"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	"github.com/spf13/cobra"
)

/*
 * 实现合约对应客户端
 */

// Cmd exchange client command

func nftOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zkNFTOrder",
		Short: "create nft sell order transaction",
		Run:   nftOrder,
	}
	NftOrderFlag(cmd)
	return cmd
}

// ratio: p * 1e8, 1e8
func NftOrderFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("leftTokenId", "l", 0, "(nft)left token id")
	cmd.Flags().Uint64P("rightTokenId", "r", 0, "right token id")
	cmd.Flags().Uint64P("price", "p", 0, "price 1e8 lt = p rt ")
	cmd.Flags().Uint64P("amount", "a", 0, "to buy/sell amount of left token")

	// zkorder part
	cmd.Flags().Uint64P("accountId", "", 0, "accountid of self")
	cmd.Flags().StringP("ethAddress", "", "", "eth address of self")

	markRequired(cmd, "leftTokenId", "rightTokenId", "price", "amount", "accountId", "ethAddress")
}

func nftOrder(cmd *cobra.Command, args []string) {
	lt, _ := cmd.Flags().GetUint64("leftTokenId")
	rt, _ := cmd.Flags().GetUint64("rightTokenId")
	price, _ := cmd.Flags().GetUint64("price")
	amount, _ := cmd.Flags().GetUint64("amount")
	op := "sell"
	opInt := 1
	// 业务 buy = buy-Left, sell-Right
	// ratio参数 要求 sell的比较在前   R1:R2 = R:L = price : 1
	buy := lt
	sell := rt
	// r1:r2 = 1:price*1e10
	if op == "2" || op == "sell" {
		opInt = 2
		buy = rt
		sell = lt
	}
	accountid, _ := cmd.Flags().GetUint64("accountId")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")

	zkorder := et.ZkOrder{
		AccountID:  accountid,
		EthAddress: ethAddress,
		TokenSell:  sell,
		TokenBuy:   buy,
		Amount:     et.AmountToZksync(amount),
		Ratio1:     big.NewInt(1).String(),
		Ratio2:     big.NewInt(0).Mul(big.NewInt(int64(price)), big.NewInt(1e10)).String(),
	}
	// sign

	payload := &et.SpotNftOrder{
		LeftAsset:  lt,
		RightAsset: rt,
		Price:      int64(price),
		Amount:     int64(amount),
		Op:         int32(opInt),
		Order:      &zkorder,
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	params := &rpctypes.CreateTxIn{
		Execer:     getExecname(paraName),
		ActionName: "SpotNTFOrder",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}
