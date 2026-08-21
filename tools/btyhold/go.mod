module github.com/33cn/plugin/tools/btyhold

go 1.22

require (
	github.com/33cn/chain33 v1.69.1-0.20260508025622-0fa35083839d
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/syndtr/goleveldb v1.0.1-0.20220614013038-64ee5596c38a
)

// 说明:
//   本工具依赖 chain33 核心库(与 plugin 仓库 go.mod 锁定的版本一致)。
//   离线验证/构建时使用 make build-local / make test(自动切换到 go.mod.local,
//   其中包含指向本地 chain33 源码目录的 replace 指令), 完成后自动恢复本文件。
