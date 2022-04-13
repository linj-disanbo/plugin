package wasm

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/plugin/plugin/dapp/zkspot/commands"
	"github.com/33cn/plugin/plugin/dapp/zkspot/executor"
	"github.com/33cn/plugin/plugin/dapp/zkspot/rpc"
	"github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.Zksync,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.ZksyncCmd,
		RPC:      rpc.Init,
	})
}
