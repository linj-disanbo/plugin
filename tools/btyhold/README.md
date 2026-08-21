# btyscan — bty 主币余额 / 合约账户扫描工具

基于 chain33 store 数据库存储, 遍历出:

1. **所有持有 bty 的地址**(主账户 `mavl-coins-bty-<addr>`);
2. **把每个合约(执行器)当成一个账户**, 列出其持有的总余额/冻结/内部账户数
   (`mavl-coins-bty-exec-<合约地址>:<用户地址>`);
3. **查询某个合约内部不同账户的余额**(钻取)。

同时提供**在线模式**(连节点公开 JSON-RPC), 无需停节点即可列出合约账户汇总。

## 背景: chain33 账户在 DB 里的存储

- 余额存在 **stateDB(mavl 状态树)**, 不在 localDB(`LODB-*`)。
- key 布局(币执行器 `coins`, 币符号 `bty`):
  - 主账户: `mavl-coins-bty-<addr>` → proto `types.Account`
  - 合约账户: `mavl-coins-bty-exec-<合约地址>:<用户地址>` → proto `types.Account`
- 零余额账户不会删除, 因此全量遍历得到的是"历史上出现过的所有地址",
  **当前持有者**需过滤 `Balance+Frozen > 0`(工具默认已过滤)。
- 一个地址的"真实持有量" = 主账户余额 + 他在所有合约里的余额/冻结之和。

## 存储布局支持(自动识别, 也可 `--layout` 强制)

| 布局 | 判断方式 | 扫描方式 |
|---|---|---|
| kvmvccmavl + MVCC 全量数据 | `.-mvcc-.d.mavl-coins-bty-` 前缀存在 | 遍历所有历史版本, 按 key 取最大版本(完整、可靠, **默认优先**) |
| kvmvccmavl + mvccIter last-key | `.-mvcc-.l.` 前缀存在 | 直接读最新值(快, 但**只覆盖 enableMVCCIter 启用后写过的 key**, 中途开启配置的节点不完整; 仅 `--layout mvcciter` 显式使用) |
| 纯 mavl(带 prune) | `_mrhp_` 前缀存在 | 取最新根哈希, 迭代状态树叶子区间 |
| 纯 mavl(未开 prune) | 裸 key 是节点哈希, 无法自动识别 | 需 `--layout mavl --statehash 0x...`(最新 stateHash 可用 `chain33.GetLastHeader` RPC 获取) |

> **线上实测结论(bityuan 主网 https://mainnet.bityuan.com/api, chain33 v1.68.2, 高度约 4690 万):**
> 节点公开的 `GetTotalCoins` 只返回 1 个主账户、`GetExecBalance` 只能看到部分近期写过的
> 合约账户(last-key 只积累 enableMVCCIter 启用之后的写入)。**权威完整数据必须用离线模式
> 直接扫数据库**(`.-mvcc-.d.` 全量版本去重路径); 在线 RPC 模式仅作参考。

## 构建

依赖 `github.com/33cn/chain33`(版本与 plugin 仓库 go.mod 一致)。

```bash
make build          # 用 go.mod(需环境能解析 chain33 模块)
make build-local    # 用 go.mod.local(replace 到本地 chain33 源码, 离线可用)
```

产物: `btyscan`。

## 用法

```
btyscan <command> [flags]

离线命令(直接读 store 数据库, 需要先停止节点, 或拷贝一份 datadir):
  scan        扫描并输出: 汇总 + bty 持有地址列表 + 合约账户列表
  holders     只输出 bty 持有地址(主账户 balance+frozen>0)
  contracts   只输出合约账户(把每个合约当成一个账户)
  contract    查询某个合约内部不同账户的余额(钻取), 用 --exec 指定合约

在线命令(连节点 JSON-RPC, 不需要停节点):
  rpc-contracts  通过公开 RPC 列出所有合约账户的总余额(无内部地址明细)
  rpc-totals     通过公开 RPC 输出主链 bty 账户总数与总量
```

### 示例(主网, coinSymbol=bty)

```bash
# 1) 停节点(或拷贝 datadir), 全量扫描
btyscan scan --db datadir/mavltree --symbol bty

# 2) 导出 CSV
#    scan --csv 会生成两个文件: holders.csv(持有地址) + holders-contracts.csv(合约账户)
btyscan scan --db datadir/mavltree --symbol bty --csv holders.csv
#    只导出持有地址 / 只导出合约账户
btyscan holders   --db datadir/mavltree --symbol bty --csv holders.csv
btyscan contracts --db datadir/mavltree --symbol bty --csv contracts.csv

# 3) 把每个合约当成一个账户
btyscan contracts --db datadir/mavltree --symbol bty --min 100000000

# 4) 查 ticket 合约内部各账户余额(传执行器名或合约地址都行)
btyscan contract --db datadir/mavltree --exec ticket
btyscan contract --db datadir/mavltree --exec 16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp

# 5) 节点在线时, 用公开 RPC 看合约总余额(仅供参考)
btyscan rpc-contracts --url https://mainnet.bityuan.com/api
btyscan rpc-totals    --url https://mainnet.bityuan.com/api

# 6) 纯 mavl 未开 prune 的库, 需显式指定布局和最新 stateHash
btyscan scan --db datadir/mavltree --layout mavl --statehash 0x<最新stateHash>
```

### 主要参数

| 参数 | 说明 | 默认 |
|---|---|---|
| `--db <dir>` | store 数据库目录(含 store.db) | `datadir/mavltree` |
| `--execer <name>` | 币所在执行器 | `coins` |
| `--symbol <symbol>` | 币符号 | `bty` |
| `--exec <addr\|name>` | 合约地址或执行器名(钻取) | - |
| `--layout <auto\|mvcciter\|mvccdata\|mavl>` | 存储布局 | `auto` |
| `--statehash <hash>` | 最新状态哈希(纯 mavl 用) | - |
| `--min <n>` | 只显示余额>= n satoshi | 0 |
| `--all` | 连零余额主账户一起列出 | false |
| `--csv <file>` | 明细写入 CSV | - |
| `--execers-file <file>` | 每行一个执行器名, 扩展合约名映射 | - |
| `--url <url>` | JSON-RPC 地址 | `http://127.0.0.1:9671` |
| `--count <n>` | RPC 分页大小 | 10000 |

## 已知限制与注意点

1. **离线模式需要停节点**: goleveldb 对数据库持有排它 flock, 运行中的节点会拒绝只读打开。
   可以拷贝一份 datadir(注意拷贝时节点已停止, 否则数据可能不一致)。
2. **MVCC last-key 路径不完整**: 节点若中途才开启 `enableMVCCIter`, `.-mvcc-.l.` 里只有
   近期写过的 key。自动识别**优先走 `.-mvcc-.d.` 全量版本去重**(任何 kvmvccmavl 节点都完整),
   只是大链上扫描较慢(等价于顺序读一遍所有历史版本)。
3. **`enableMVCCPrune`**: 若节点开启了 MVCC 精简, 长期未动的账户其最新版本也可能被裁掉,
   该 key 会缺失; 本工具无解, 属于节点数据裁剪的固有限制。
4. **地址集合 vs 持有集合**: 遍历得到的是历史上出现过的所有地址; `holders` 默认只列
   `balance+frozen>0` 的。真实持有量应把合约内余额也加上(工具分别展示, 不合并)。
5. **合约名映射**: 内置了常见执行器名(ticket/token/trade/paracross/evm 等), 未知合约显示地址;
   自定义执行器用 `--execers-file` 追加。

## 附带脚本: stat_holders.py(统计 holders.csv 持有量)

对 `btyscan ... --csv` 导出的 CSV 做持有量统计(优先用 satoshi 整数列 total, 精确):

```bash
python3 stat_holders.py holders.csv              # 汇总 + 分桶分布 + Top10
python3 stat_holders.py holders.csv --top 20     # Top 20
python3 stat_holders.py holders.csv --no-top     # 不显示 Top
python3 stat_holders.py holders.csv --summary buckets.csv   # 分桶汇总写 CSV
```

输出示例: 地址总数 / 持有者数 / 总持有量 / 最大最小平均中位数 / Top1-Top100 集中度 /
按持有量分桶分布(>=100万, 10万~100万, ..., 0~1, 0) / Top N 名单。
兼容 `holders`/`scan`(addr 列) 与 `contracts`(execAddr 列) 生成的 CSV。

## 代码结构

```
main.go         CLI 入口
store.go        只读打开 goleveldb + dbm.DB 适配器
scan.go         布局识别 + 三种扫描(mvcc-last / mvcc全量 / mavl树)
report.go       汇总、排序、输出(表格 + CSV)
execers.go      执行器名 <-> 合约地址映射
rpc.go          在线 JRPC 模式(GetTotalCoins / GetExecBalance / GetLastHeader)
scan_test.go    三种布局的合成库单测
```
