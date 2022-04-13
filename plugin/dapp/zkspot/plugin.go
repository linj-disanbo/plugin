package wasm

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/plugin/plugin/dapp/zksopt/commands"
	"github.com/33cn/plugin/plugin/dapp/zksopt/executor"
	"github.com/33cn/plugin/plugin/dapp/zksopt/rpc"
	"github.com/33cn/plugin/plugin/dapp/zksopt/types"
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
