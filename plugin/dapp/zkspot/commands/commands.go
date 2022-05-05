/*Package commands implement dapp client commands*/
package commands

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	pt "github.com/33cn/plugin/plugin/dapp/paracross/types"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	et "github.com/33cn/plugin/plugin/dapp/zkspot/types"
	"github.com/33cn/plugin/plugin/dapp/zkspot/wallet"
	zt "github.com/33cn/plugin/plugin/dapp/zksync/types"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

/*
 * 实现合约对应客户端
 */

const ExecName = et.Zksync

// ZksyncCmd zkspot client command
func ZksyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zkspot",
		Short: "zkspot command",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		depositCmd(),
		withdrawCmd(),
		contractToTreeCmd(),
		treeToContractCmd(),
		transferCmd(),
		transferToNewCmd(),
		forceExitCmd(),
		setPubKeyCmd(),
		fullExitCmd(),
		setVerifyKeyCmd(),
		setOperatorCmd(),
		commitProofCmd(),
		getChain33AddrCmd(),
		getAccountTreeCmd(),
		getTxProofCmd(),
		getTxProofByHeightCmd(),
		getLastCommitProofCmd(),
		getAccountByIdCmd(),
		getAccountByEthCmd(),
		getAccountByChain33Cmd(),
		getContractAccountCmd(),
		getTokenBalanceCmd(),
		getZkCommitProofCmd(),
		limitOrderCmd(),
		revokeOrderCmd(),
	)
	return cmd
}

func depositCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "get deposit tx",
		Run:   deposit,
	}
	depositFlag(cmd)
	return cmd
}

func depositFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "deposit tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "deposit amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("ethAddress", "e", "", "deposit ethaddress")
	cmd.MarkFlagRequired("ethAddress")
	cmd.Flags().StringP("chain33Addr", "c", "", "deposit chain33Addr")
	cmd.MarkFlagRequired("chain33Addr")

}

func deposit(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")
	chain33Addr, _ := cmd.Flags().GetString("chain33Addr")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyDepositAction, tokenId, amount, ethAddress, "", chain33Addr, 0, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "Deposit",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func withdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "get withdraw tx",
		Run:   withdraw,
	}
	withdrawFlag(cmd)
	return cmd
}

func withdrawFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "withdraw tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "withdraw amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Uint64P("accountId", "", 0, "withdraw accountId")
	cmd.MarkFlagRequired("accountId")

}

func withdraw(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyWithdrawAction, tokenId, amount, "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "Withdraw",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func treeToContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "treeToContract",
		Short: "get treeToContract tx",
		Run:   treeToContract,
	}
	treeToContractFlag(cmd)
	return cmd
}

func treeToContractFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "treeToContract tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "treeToContract amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Uint64P("accountId", "", 0, "treeToContract accountId")
	cmd.MarkFlagRequired("accountId")

}

func treeToContract(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyTreeToContractAction, tokenId, amount, "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "TreeToContract",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func contractToTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contractToTree",
		Short: "get contractToTree tx",
		Run:   contractToTree,
	}
	contractToTreeFlag(cmd)
	return cmd
}

func contractToTreeFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "contractToTree tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "contractToTree amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Uint64P("accountId", "", 0, "contractToTree accountId")
	cmd.MarkFlagRequired("accountId")

}

func contractToTree(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyContractToTreeAction, tokenId, amount, "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "ContractToTree",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func transferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "get transfer tx",
		Run:   transfer,
	}
	transferFlag(cmd)
	return cmd
}

func transferFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "transfer tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "transfer amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Uint64P("accountId", "", 0, "transfer fromAccountId")
	cmd.MarkFlagRequired("accountId")
	cmd.Flags().Uint64P("toAccountId", "", 0, "transfer toAccountId")
	cmd.MarkFlagRequired("toAccountId")

}

func transfer(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	accountId, _ := cmd.Flags().GetUint64("accountId")
	toAccountId, _ := cmd.Flags().GetUint64("toAccountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyTransferAction, tokenId, amount, "", "", "", accountId, toAccountId)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "Transfer",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func transferToNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transferToNew",
		Short: "get transferToNew tx",
		Run:   transferToNew,
	}
	transferToNewFlag(cmd)
	return cmd
}

func transferToNewFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "transferToNew tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().StringP("amount", "a", "0", "transferToNew amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Uint64P("accountId", "", 0, "transferToNew fromAccountId")
	cmd.MarkFlagRequired("accountId")
	cmd.Flags().StringP("ethAddress", "e", "", "transferToNew toEthAddress")
	cmd.MarkFlagRequired("ethAddress")
	cmd.Flags().StringP("chain33Addr", "c", "", "transferToNew toChain33Addr")
	cmd.MarkFlagRequired("chain33Addr")
}

func transferToNew(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	amount, _ := cmd.Flags().GetString("amount")
	accountId, _ := cmd.Flags().GetUint64("accountId")
	toEthAddress, _ := cmd.Flags().GetString("ethAddress")
	chain33Addr, _ := cmd.Flags().GetString("chain33Addr")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyTransferToNewAction, tokenId, amount, "", toEthAddress, chain33Addr, accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "TransferToNew",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func forceExitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forceExit",
		Short: "get forceExit tx",
		Run:   forceExit,
	}
	forceExitFlag(cmd)
	return cmd
}

func forceExitFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "forceExit tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().Uint64P("accountId", "a", 0, "forceExit accountId")
	cmd.MarkFlagRequired("accountId")

}

func forceExit(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyForceExitAction, tokenId, "0", "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "ForceExit",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func setPubKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setPubKey",
		Short: "get setPubKey tx",
		Run:   setPubKey,
	}
	setPubKeyFlag(cmd)
	return cmd
}

func setPubKeyFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("accountId", "a", 0, "setPubKeyFlag accountId")
	cmd.MarkFlagRequired("accountId")

}

func setPubKey(cmd *cobra.Command, args []string) {
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TySetPubKeyAction, 0, "0", "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "SetPubKey",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func fullExitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fullExit",
		Short: "get fullExit tx",
		Run:   fullExit,
	}
	fullExitFlag(cmd)
	return cmd
}

func fullExitFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("tokenId", "t", 1, "fullExit tokenId")
	cmd.MarkFlagRequired("tokenId")
	cmd.Flags().Uint64P("accountId", "a", 0, "fullExit accountId")
	cmd.MarkFlagRequired("accountId")

}

func fullExit(cmd *cobra.Command, args []string) {
	tokenId, _ := cmd.Flags().GetUint64("tokenId")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	payload, err := wallet.CreateRawTx(zt.TyFullExitAction, tokenId, "0", "", "", "", accountId, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "createRawTx"))
		return
	}
	params := &rpctypes.CreateTxIn{
		Execer:     ExecName,
		ActionName: "FullExit",
		Payload:    payload,
	}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func setVerifyKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vkey",
		Short: "set verify key for zk-proof",
		Run:   verifyKey,
	}
	addVerifyKeyCmdFlags(cmd)
	return cmd
}

func addVerifyKeyCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("vkey", "v", "", "verify key")
	_ = cmd.MarkFlagRequired("vkey")

}

func verifyKey(cmd *cobra.Command, args []string) {
	paraName, _ := cmd.Flags().GetString("paraName")
	vkey, _ := cmd.Flags().GetString("vkey")

	payload := &zt.ZkVerifyKey{
		Key: vkey,
	}
	exec := ExecName
	if strings.HasPrefix(paraName, pt.ParaPrefix) {
		exec = paraName + ExecName
	}
	params := &rpctypes.CreateTxIn{
		Execer:     exec,
		ActionName: "SetVerifyKey",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func setOperatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "set operators for commit zk-proof",
		Run:   setOperator,
	}
	addOperatorCmdFlags(cmd)
	return cmd
}

func addOperatorCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("operator", "o", "", "operators, separate with '-'")
	_ = cmd.MarkFlagRequired("operator")

}

func setOperator(cmd *cobra.Command, args []string) {
	paraName, _ := cmd.Flags().GetString("paraName")
	operator, _ := cmd.Flags().GetString("operator")

	payload := &zt.ZkVerifier{
		Verifiers: strings.Split(operator, "-"),
	}
	exec := ExecName
	if strings.HasPrefix(paraName, pt.ParaPrefix) {
		exec = paraName + ExecName
	}
	params := &rpctypes.CreateTxIn{
		Execer:     exec,
		ActionName: "SetVerifier",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func commitProofCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "commit proof",
		Run:   commitProof,
	}
	addCommitProofCmdFlags(cmd)
	return cmd
}

func addCommitProofCmdFlags(cmd *cobra.Command) {
	cmd.Flags().Uint64P("start", "s", 0, "block start ")
	_ = cmd.MarkFlagRequired("start")
	cmd.Flags().Uint64P("end", "e", 0, "block end ")
	_ = cmd.MarkFlagRequired("end")
	cmd.Flags().StringP("old", "o", "0", "old tree hash")
	_ = cmd.MarkFlagRequired("old")
	cmd.Flags().StringP("new", "n", "0", "new tree hash")
	_ = cmd.MarkFlagRequired("new")
	cmd.Flags().StringP("pubdata", "d", "0", "pub datas, separate with '-'")
	_ = cmd.MarkFlagRequired("pubdata")
	cmd.Flags().StringP("public", "i", "0", "public input")
	_ = cmd.MarkFlagRequired("public")
	cmd.Flags().StringP("proof", "p", "0", "proof")
	_ = cmd.MarkFlagRequired("proof")

}

func commitProof(cmd *cobra.Command, args []string) {
	paraName, _ := cmd.Flags().GetString("paraName")
	start, _ := cmd.Flags().GetUint64("start")
	end, _ := cmd.Flags().GetUint64("end")
	old, _ := cmd.Flags().GetString("old")
	new, _ := cmd.Flags().GetString("new")
	pubdata, _ := cmd.Flags().GetString("pubdata")
	public, _ := cmd.Flags().GetString("public")
	proof, _ := cmd.Flags().GetString("proof")

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	payload := &zt.ZkCommitProof{
		BlockStart:  start,
		BlockEnd:    end,
		OldTreeRoot: old,
		NewTreeRoot: new,
		PublicInput: public,
		Proof:       proof,
		PubDatas:    strings.Split(pubdata, "-"),
	}
	exec := ExecName
	if strings.HasPrefix(paraName, pt.ParaPrefix) {
		exec = paraName + ExecName
	}
	params := &rpctypes.CreateTxIn{
		Execer:     exec,
		ActionName: "CommitProof",
		Payload:    types.MustPBToJSON(payload),
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func getChain33AddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getChain33Addr",
		Short: "get chain33 address by privateKey",
		Run:   getChain33Addr,
	}
	getChain33AddrFlag(cmd)
	return cmd
}

func getChain33AddrFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("private", "k", "", "private key")
}

func getChain33Addr(cmd *cobra.Command, args []string) {
	privateKeyString, _ := cmd.Flags().GetString("private")
	privateKeyBytes, err := common.FromHex(privateKeyString)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "hex.DecodeString"))
		return
	}
	privateKey, err := eddsa.GenerateKey(bytes.NewReader(privateKeyBytes))

	hash := mimc.NewMiMC(zt.ZkMimcHashSeed)
	hash.Write(zt.Str2Byte(privateKey.PublicKey.A.X.String()))
	hash.Write(zt.Str2Byte(privateKey.PublicKey.A.Y.String()))
	fmt.Println(hex.EncodeToString(hash.Sum(nil)))
}

func getAccountTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tree",
		Short: "get current accountTree",
		Run:   getAccountTree,
	}
	getAccountTreeFlag(cmd)
	return cmd
}

func getAccountTreeFlag(cmd *cobra.Command) {
}

func getAccountTree(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := new(types.ReqNil)

	params.FuncName = "GetAccountTree"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.AccountTree
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getTxProofCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proof",
		Short: "get tx proof",
		Run:   getTxProof,
	}
	getTxProofFlag(cmd)
	return cmd
}

func getTxProofFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("height", "", 0, "zkspot proof height")
	cmd.MarkFlagRequired("height")
	cmd.Flags().Uint32P("index", "i", 0, "tx index")
	cmd.MarkFlagRequired("index")
}

func getTxProof(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	height, _ := cmd.Flags().GetUint64("height")
	index, _ := cmd.Flags().GetUint32("index")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		BlockHeight: height,
		TxIndex:     index,
	}

	params.FuncName = "GetTxProof"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.OperationInfo
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getTxProofByHeightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proofs",
		Short: "get block proofs",
		Run:   getTxProofByHeight,
	}
	getTxProofByHeightFlag(cmd)
	return cmd
}

func getTxProofByHeightFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("height", "", 0, "zkspot proof height")
	cmd.MarkFlagRequired("height")
}

func getTxProofByHeight(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	height, _ := cmd.Flags().GetUint64("height")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		BlockHeight: height,
	}

	params.FuncName = "GetTxProofByHeight"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.ZkQueryResp
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getLastCommitProofCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proofstate",
		Short: "get last committed proof",
		Run:   getLastCommitProof,
	}

	return cmd
}

func getLastCommitProof(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName

	params.FuncName = "GetLastCommitProof"
	params.Payload = types.MustPBToJSON(&types.ReqNil{})

	var resp zt.CommitProofState
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getAccountByIdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "get zkspot account by id",
		Run:   getAccountById,
	}
	getAccountByIdFlag(cmd)
	return cmd
}

func getAccountByIdFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("accountId", "a", 0, "zkspot accountId")
	cmd.MarkFlagRequired("accountId")
}

func getAccountById(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	accountId, _ := cmd.Flags().GetUint64("accountId")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		AccountId: accountId,
	}

	params.FuncName = "GetAccountById"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.Leaf
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getAccountByEthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accountE",
		Short: "get zkspot account by ethAddress",
		Run:   getAccountByEth,
	}
	getAccountByEthFlag(cmd)
	return cmd
}

func getAccountByEthFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("ethAddress", "e", " ", "zkspot account ethAddress")
	cmd.MarkFlagRequired("ethAddress")
}

func getAccountByEth(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		EthAddress: ethAddress,
	}

	params.FuncName = "GetAccountByEth"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.ZkQueryResp
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getAccountByChain33Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accountC",
		Short: "get zkspot account by chain33Addr",
		Run:   getAccountByChain33,
	}
	getAccountByChain33Flag(cmd)
	return cmd
}

func getAccountByChain33Flag(cmd *cobra.Command) {
	cmd.Flags().StringP("chain33Addr", "c", "", "zkspot account chain33Addr")
	cmd.MarkFlagRequired("chain33Addr")
}

func getAccountByChain33(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	chain33Addr, _ := cmd.Flags().GetString("chain33Addr")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		Chain33Addr: chain33Addr,
	}

	params.FuncName = "GetAccountByChain33"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.ZkQueryResp
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getContractAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contractAccount",
		Short: "get zkspot contractAccount by chain33WalletAddr and token symbol",
		Run:   getContractAccount,
	}
	getContractAccountFlag(cmd)
	return cmd
}

func getContractAccountFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("chain33Addr", "c", "", "chain33 wallet address")
	cmd.MarkFlagRequired("chain33Addr")
	cmd.Flags().StringP("token", "t", "", "token symbol")
	cmd.MarkFlagRequired("token")
}

func getContractAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	chain33Addr, _ := cmd.Flags().GetString("chain33Addr")
	token, _ := cmd.Flags().GetString("token")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		TokenSymbol:       token,
		Chain33WalletAddr: chain33Addr,
	}

	params.FuncName = "GetZkContractAccount"
	params.Payload = types.MustPBToJSON(req)

	var resp types.Account
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getTokenBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "get zkspot tokenBalance by accountId and tokenId",
		Run:   getTokenBalance,
	}
	getTokenBalanceFlag(cmd)
	return cmd
}

func getTokenBalanceFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("accountId", "a", 0, "zkspot account id")
	cmd.MarkFlagRequired("accountId")
	cmd.Flags().Uint64P("token", "t", 1, "zkspot token id")
	cmd.MarkFlagRequired("token")
}

func getTokenBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	accountId, _ := cmd.Flags().GetUint64("accountId")
	token, _ := cmd.Flags().GetUint64("token")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		TokenId:   token,
		AccountId: accountId,
	}

	params.FuncName = "GetTokenBalance"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.ZkQueryResp
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func getZkCommitProofCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zkCommit",
		Short: "get zkcommit proof by proofId",
		Run:   getZkCommitProof,
	}
	getZkCommitProofFlag(cmd)
	return cmd
}

func getZkCommitProofFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("proofId", "p", 0, "commit proof id")
	cmd.MarkFlagRequired("proofId")
}

func getZkCommitProof(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	proofId, _ := cmd.Flags().GetUint64("proofId")

	var params rpctypes.Query4Jrpc

	params.Execer = ExecName
	req := &zt.ZkQueryReq{
		ProofId: proofId,
	}

	params.FuncName = "GetCommitProodByProofId"
	params.Payload = types.MustPBToJSON(req)

	var resp zt.ZkCommitProof
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.Query", params, &resp)
	ctx.Run()
}

func limitOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zkLimitOrder",
		Short: "create limit order transaction",
		Run:   limitOrder,
	}
	limitOrderFlag(cmd)
	return cmd
}

func markRequired(cmd *cobra.Command, params ...string) {
	for _, param := range params {
		_ = cmd.MarkFlagRequired(param)
	}
}

// ratio: p * 1e8, 1e8
func limitOrderFlag(cmd *cobra.Command) {
	cmd.Flags().Uint32P("leftTokenId", "l", 0, "left token id")
	cmd.Flags().Uint32P("rightTokenId", "r", 0, "right token id")
	cmd.Flags().Uint64P("price", "p", 0, "price 1e8 lt = p rt ")
	cmd.Flags().Uint64P("amount", "a", 0, "to buy/sell amount of left token")
	cmd.Flags().StringP("op", "o", "1", "1/buy, 2/sell")

	// zkorder part
	cmd.Flags().Uint64P("accountId", "", 0, "accountid of self")
	cmd.Flags().StringP("ethAddress", "", "", "eth address of self")

	markRequired(cmd, "leftTokenId", "rightTokenId", "price", "amount", "op", "accountId", "ethAddress")
}

func limitOrder(cmd *cobra.Command, args []string) {

	lt, _ := cmd.Flags().GetUint32("leftTokenId")
	rt, _ := cmd.Flags().GetUint32("rightTokenId")
	price, _ := cmd.Flags().GetUint64("price")
	amount, _ := cmd.Flags().GetUint64("amount")
	op, _ := cmd.Flags().GetString("op")
	opInt := 1
	// buy: L:R = price/1 = R1:R2
	buy := lt
	sell := rt
	ratio1 := int64(price)
	ratio2 := int64(1e8)
	if op == "2" || op == "sell" {
		opInt = 2
		buy = rt
		sell = lt
		ratio1, ratio2 = ratio2, ratio1
	}
	accountid, _ := cmd.Flags().GetUint64("accountId")
	ethAddress, _ := cmd.Flags().GetString("ethAddress")

	zkorder := et.ZkOrder{
		AccountID:  accountid,
		EthAddress: ethAddress,
		TokenSell:  sell,
		TokenBuy:   buy,
		Amount:     amount,
		Ratio1:     uint32(ratio1),
		Ratio2:     uint32(ratio2),
	}
	// sign

	payload := &et.SpotLimitOrder{
		LeftAsset:  lt,
		RightAsset: rt,
		Price:      int64(price),
		Amount:     int64(amount),
		Op:         int32(opInt),
		Order:      &zkorder,
	}
	exec := ExecName
	paraName, _ := cmd.Flags().GetString("paraName")
	if strings.HasPrefix(paraName, pt.ParaPrefix) {
		exec = paraName + ExecName
	}
	params := &rpctypes.CreateTxIn{
		Execer:     exec,
		ActionName: "LimitOrder",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func revokeOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zkRevokeOrder",
		Short: "create revoke limit order transaction",
		Run:   revokeOrder,
	}
	revokeOrderFlag(cmd)
	return cmd
}

func revokeOrderFlag(cmd *cobra.Command) {
	cmd.Flags().Uint64P("orderid", "o", 0, "order id")
	cmd.MarkFlagRequired("orderid")
}

func revokeOrder(cmd *cobra.Command, args []string) {
	orderid, _ := cmd.Flags().GetUint64("orderid")

	payload := &et.SpotRevokeOrder{
		OrderID: int64(orderid),
	}
	exec := ExecName
	paraName, _ := cmd.Flags().GetString("paraName")
	if strings.HasPrefix(paraName, pt.ParaPrefix) {
		exec = paraName + ExecName
	}
	params := &rpctypes.CreateTxIn{
		Execer:     exec,
		ActionName: "RevokeOrder",
		Payload:    types.MustPBToJSON(payload),
	}
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CreateTransaction", params, nil)
	ctx.RunWithoutMarshal()
}
