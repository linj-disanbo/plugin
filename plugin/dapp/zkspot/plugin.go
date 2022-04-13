package wasm

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/plugin/plugin/dapp/zkspot/commands"
	"github.com/33cn/plugin/plugin/dapp/zkspot/executor"
	"github.com/33cn/plugin/plugin/dapp/zkspot/rpc"
	"github.com/33cn/plugin/plugin/dapp/zkspot/types"
)

// 本来想直接把代码分开, 模块化的, 要改动的太多了
// 先换个分支实现和验证功能
func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.Zksync,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.ZksyncCmd,
		RPC:      rpc.Init,
	})
}
