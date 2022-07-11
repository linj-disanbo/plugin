/*Package commands implement dapp client commands*/
package commands

import (
	"fmt"
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
		Use:   "sell_nft",
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

func nftTakerOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_nft",
		Short: "create nft buy order transaction",
		Run:   nftTakerOrder,
	}
	NftTakerOrderFlag(cmd)
	return cmd
}

// ratio: p * 1e8, 1e8
func NftTakerOrderFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("order", "o", 0, "(nft) order id")
	// zkorder part
	cmd.Flags().Uint64P("accountId", "", 0, "accountid of self")
	cmd.Flags().StringP("ethAddress", "", "", "eth address of self")

	markRequired(cmd, "order", "accountId", "ethAddress")
}

func nftTakerOrder(cmd *cobra.Command, args []string) {
	orderId, _ := cmd.Flags().GetInt64("order")
	getNftOrder(cmd, args)
	var order2 et.SpotOrder
	if order2.Ty != et.TyNftOrderAction {
		fmt.Printf("%022d the order is not nft sell order", orderId)
		return
	}
	// 业务 buy = buy-Left, sell-Right
	// ratio参数 要求 sell的比较在前   R1:R2 = R:L = price : 1
	accountid, _ := cmd.Flags().GetUint64("accountId")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")

	zkorder := et.ZkOrder{
		AccountID:  accountid,
		EthAddress: ethAddress,
		TokenSell:  order2.GetNftOrder().Order.TokenBuy,
		TokenBuy:   order2.GetNftOrder().Order.TokenSell,
		Amount:     order2.GetNftOrder().Order.Amount,
		Ratio1:     order2.GetNftOrder().Order.Ratio2,
		Ratio2:     order2.GetNftOrder().Order.Ratio1,
	}
	// sign

	payload := &et.SpotNftTakerOrder{
		OrderID: orderId,
		Order:   &zkorder,
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	params := &rpctypes.CreateTxIn{
		Execer:     getExecname(paraName),
		ActionName: "SpotNTFTakerOrder",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func QueryNftOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nft_order",
		Short: "query nft sell order transaction",
		Run:   queryNftOrder,
	}
	queryNftOrderFlag(cmd)
	return cmd
}

// ratio: p * 1e8, 1e8
func queryNftOrderFlag(cmd *cobra.Command) {
	cmd.Flags().Int64P("order", "o", 0, "(nft) order id")
	markRequired(cmd, "order")
}

func queryNftOrder1(cmd *cobra.Command, args []string) *et.SpotOrder {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderId, _ := cmd.Flags().GetInt64("order")

	var params rpctypes.Query4Jrpc

	paraName, _ := cmd.Flags().GetString("paraName")
	params.Execer = getExecname(paraName)
	req := &et.SpotQueryOrder{
		OrderID: orderId,
	}

	params.FuncName = "QueryNftOrder"
	params.Payload = types.MustPBToJSON(req)

	var resp et.SpotOrder
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
	return &resp
}

func queryNftOrder(cmd *cobra.Command, args []string) {
	queryNftOrder1(cmd, args)
}

func getNftOrder(cmd *cobra.Command, args []string) *et.SpotOrder {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	orderId, _ := cmd.Flags().GetInt64("order")

	var params rpctypes.Query4Jrpc

	paraName, _ := cmd.Flags().GetString("paraName")
	params.Execer = getExecname(paraName)
	req := &et.SpotQueryOrder{
		OrderID: orderId,
	}

	params.FuncName = "QueryNftOrder"
	params.Payload = types.MustPBToJSON(req)

	var resp et.SpotOrder
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.RunResult()
	return &resp
}

func nftOrder2Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell_nft2",
		Short: "create nft sell order transaction",
		Run:   nftOrder2,
	}
	NftOrder2Flag(cmd)
	return cmd
}

// ratio: p * 1e8, 1e8
func NftOrder2Flag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("leftTokenId", "l", 0, "(nft)left token id")
	cmd.Flags().Uint64P("rightTokenId", "r", 0, "right token id")
	cmd.Flags().Uint64P("price", "p", 0, "price 1e8 lt = p rt ")
	cmd.Flags().Uint64P("amount", "a", 0, "to buy/sell amount of left token")

	// zkorder part
	cmd.Flags().Uint64P("accountId", "", 0, "accountid of self")
	cmd.Flags().StringP("ethAddress", "", "", "eth address of self")

	markRequired(cmd, "leftTokenId", "rightTokenId", "price", "amount", "accountId", "ethAddress")
}

func nftOrder2(cmd *cobra.Command, args []string) {
	lt, _ := cmd.Flags().GetUint64("leftTokenId")
	rt, _ := cmd.Flags().GetUint64("rightTokenId")
	price, _ := cmd.Flags().GetUint64("price")
	amount, _ := cmd.Flags().GetUint64("amount")
	op := "sell"
	opInt := 2
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
		Amount:     et.NftAmountToZksync(amount),
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
		ActionName: "NftOrder2",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func nftTakerOrder2Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy_nft2",
		Short: "create nft buy order transaction",
		Run:   nftTakerOrder2,
	}
	NftTakerOrder2Flag(cmd)
	return cmd
}

// ratio: p * 1e8, 1e8
func NftTakerOrder2Flag(cmd *cobra.Command) {
	cmd.Flags().Int64P("order", "o", 0, "(nft) order id")
	// zkorder part
	cmd.Flags().Uint64P("accountId", "", 0, "accountid of self")
	cmd.Flags().StringP("ethAddress", "", "", "eth address of self")

	markRequired(cmd, "order", "accountId", "ethAddress")
}

func nftTakerOrder2(cmd *cobra.Command, args []string) {
	orderId, _ := cmd.Flags().GetInt64("order")
	order2 := getNftOrder(cmd, args)
	if order2 == nil {
		fmt.Println("get nft order failed")
	}
	if order2.Ty != et.TyNftOrder2Action {
		fmt.Printf("%022d the order is not nft sell order", orderId)
		return
	}
	// 业务 buy = buy-Left, sell-Right
	// ratio参数 要求 sell的比较在前   R1:R2 = R:L = price : 1
	accountid, _ := cmd.Flags().GetUint64("accountId")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")

	zkorder := et.ZkOrder{
		AccountID:  accountid,
		EthAddress: ethAddress,
		TokenSell:  order2.GetNftOrder().Order.TokenBuy,
		TokenBuy:   order2.GetNftOrder().Order.TokenSell,
		Amount:     order2.GetNftOrder().Order.Amount,
		Ratio1:     order2.GetNftOrder().Order.Ratio2,
		Ratio2:     order2.GetNftOrder().Order.Ratio1,
	}
	// sign

	payload := &et.SpotNftTakerOrder{
		OrderID: orderId,
		Order:   &zkorder,
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	params := &rpctypes.CreateTxIn{
		Execer:     getExecname(paraName),
		ActionName: "NftTakerOrder2",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}
