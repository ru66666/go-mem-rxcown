> 我需要写一个脚本，该脚本用于构建Actor(基于Go的protoactor-go)之间通信的拓扑图以及热点图。
>
> 我在程序中进行了插桩，插桩会输出两类信息：Actor相关的信息以及Actor之间消息相关的信息。
>
> Actor相关的信息：
> * Actor的类型(ActorType)
> * Actor的大小(ActorSize)
> * Actor对应的PID(Self{Address, Id, GroupID, Master})，可能为nil
> * Actor的父Actor的PID(Parent{Address, Id, GroupID, Master})，可能为nil
> ```
> [ACTOR] INFO Actor statistics统计 ActorType="*" ActorSize=8  Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
> ```
>
> 消息相关的信息：
> * 消息的类型(MsgType)
> * 消息的大小(MsgSize)
> * 消息的发送者(Sender{Address, Id, GroupID, Master})，可能为nil
> * 消息的接受者(Self{Address, Id, GroupID, Master})，可能为nil
> * 当前Actor的Parent PID(Parent{Address, Id, GroupID, Master})，可能为nil
> ```
> [ACTOR] INFO Receive统计 MsgType="*actor.Started" MsgSize=8 Sender="nil" Self="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Parent="nil"
> ```
>
> 我希望收集这些信息来辅助我了解我的程序中Actor之间的通信模式和性能瓶颈。为了实现这一目标，我计划编写一个脚本，该脚本将解析插桩输出的日志文件，并生成以下内容：
> 1. Actor拓扑图：展示不同Actor类别之间的父子关系
> 2. Actor通信热点图：展示Actor之间的通信拓扑，并且展示热点消息或者通信链路（这里“热点”可以用频次来衡量，也可以用消息的大小来衡量，消息发送的频次*每条消息的大小。“热点”的衡量方法最好可以做成可以修改和定制的），以便识别性能瓶颈和高频通信的Actor。
>
> 请你协助我实现上边这些内容，可以用Python实现。
> 进阶的方法可以考虑使用图数据库来反映父子拓扑关系和通信拓扑关系，并通过一些工具来辅助进行可视化查看或查询。
> 另外，每个插桩的输出可能随着我的需求而发生变换，你可以先按照当前的格式来解析，但是要求最终能实现可配置的（可以考虑通过Schema的形式）。


# Actor 拓扑图 & 通信热点图 分析脚本实现计划

## Context（背景）

程序基于 Go 的 protoactor-go，已做插桩，日志中输出两类可解析记录（Actor 统计、消息 Receive 统计）。
目标是解析这些日志，生成：(1) Actor 父子拓扑图；(2) Actor 通信热点图（按可定制度量衡量"热点"），
以辅助分析通信模式与性能瓶颈。

关键约束（来自需求方）：
- **插桩与解析解耦**：本脚本只面向"日志文本"这一契约，不依赖 Go 代码。
- **解析可配置（Schema 驱动）**：插桩格式将来会变，改 Schema 即可适配，尽量不改代码。
- 图数据库是"进阶"能力：做成**可选后端**，有 Neo4j 实例才启用。
- 通信图节点**支持按 ActorType 聚合与按实例 PID 聚合两种粒度**（参数切换）。
- v1 交付：**交互式 HTML（pyvis）** + **文本/CSV 报告**；可选 Neo4j 导入/查询。

## 目录结构（本仓库内 `actor-analyzer/`）

```
actor-analyzer/
├── plan.md                 # 本方案记录
├── README.md               # 使用说明 + Schema 字段契约
├── requirements.txt        # networkx / pyvis / pyyaml；neo4j 标注为可选
├── schema.yaml             # 默认解析 Schema（对应当前日志格式）
├── sample.log              # 合成样例日志（用于冒烟测试 + 用户可替换为真实日志）
├── .venv/                  # 虚拟环境（python -m venv 创建，不入库）
├── actor_topo/
│   ├── __init__.py
│   ├── schema.py           # 加载/校验 Schema（YAML/JSON）
│   ├── pid.py              # PID 递归解析（含 nil / 嵌套 Master）
│   ├── parser.py           # 日志行 → 规范化记录（两遍：先 Actor 后 Message）
│   ├── model.py            # dataclass：PID / ActorRecord / MessageRecord
│   ├── build.py            # 构建 NetworkX 图（拓扑图 + 通信图，type/pid 两粒度）
│   ├── metrics.py          # 热点度量注册表（可扩展）
│   ├── report.py           # CSV + 文本 Top-N 报告
│   ├── export/
│   │   ├── __init__.py
│   │   ├── html.py         # pyvis 交互式 HTML（边粗细=热点权重）
│   │   └── neo4j.py        # 可选：Neo4j 导入 + 若干 Cypher 查询（lazy import）
│   └── cli.py              # argparse 命令行入口
└── tests/
    ├── test_pid.py / test_parser.py / test_build.py / test_metrics.py
```

## Schema 设计（可配置解析的核心）

Schema 用 YAML 描述"日志行如何解析"。每种记录一个正则（命名捕获组）+ 字段类型映射；PID 用通用的
"结构体字面量"递归解析器（比单一正则更稳，天然支持 `Master:{nil}` 嵌套）。

```yaml
version: 1
records:
  actor:
    pattern: >-
      ^\[ACTOR\]\s+INFO\s+Actor\s+statistics统计\s+
      ActorType="(?P<actor_type>[^"]*)"\s+ActorSize=(?P<actor_size>\d+)\s+
      Self="(?P<self>[^"]*)"\s+Parent="(?P<parent>[^"]*)"\s*$
    fields: {actor_type: string, actor_size: int, self: pid, parent: pid}
  message:
    pattern: >-
      ^\[ACTOR\]\s+INFO\s+Receive统计\s+
      MsgType="(?P<msg_type>[^"]*)"\s+MsgSize=(?P<msg_size>\d+)\s+
      Sender="(?P<sender>[^"]*)"\s+Self="(?P<self>[^"]*)"\s+Parent="(?P<parent>[^"]*)"\s*$
    fields: {msg_type: string, msg_size: int, sender: pid, self: pid, parent: pid}
pid:
  null_literal: "nil"
  fields: {Address: string, Id: string, GroupID: int, Master: pid}   # Master 递归
```

**字段契约（稳定部分）**：正则捕获组必须使用这些**规范名**，即使日志格式变化、捕获组写法重排，
下游代码只认这些名字——因此改格式只改 `pattern`，不改代码：
- Actor 记录：`actor_type`(string) `actor_size`(int) `self`(pid) `parent`(pid)
- Message 记录：`msg_type`(string) `msg_size`(int) `sender`(pid) `self`(pid) `parent`(pid)
- PID 字段：`Address` `Id` `GroupID`(int) `Master`(pid，递归，可为 nil)

类型强转：`string`→原样；`int`→`int()`；`pid`→走 `pid.py` 递归解析。
**PID 解析规则**：值 == `null_literal`（`"nil"`）→ `None`；否则按顶层逗号切分（跟踪 `{}` 深度）成
`Key:Value`，依 `pid.fields` 逐字段强转；`Value` 形如 `{...}` 或 `nil` 且类型为 `pid` → 递归。

## 数据模型（model.py，与 Schema 解耦）

```python
@dataclass
class PID:
    address: str; id: str; group_id: int; master: "PID | None"
    def identity(self) -> str: return f"{self.address}/{self.id}"   # 实例唯一键

@dataclass
class ActorRecord:
    actor_type: str; actor_size: int; self: PID | None; parent: PID | None

@dataclass
class MessageRecord:
    msg_type: str; msg_size: int; sender: PID | None; self: PID | None; parent: PID | None
```

## 解析流程（parser.py，两遍、流式、内存友好）

1. 逐行读日志；跳过不匹配任意 `pattern` 的行（统计 skipped 行数，报告里体现）。
2. **第一遍**：收集所有 ActorRecord，建 `pid_identity -> ActorType/Size` 映射（同 id 重复时 last-wins）。
3. **第二遍**：处理 MessageRecord，用该映射把 `self`/`sender` PID 解析为类型（用于 type 粒度聚合）。
   消息的发送/接收者若为 nil 或查不到类型 → 归入合成节点（见下）。

## 图构建（build.py）

用 `networkx.DiGraph`，两图 × 两粒度（`--by type|pid`）：

1. **拓扑图**（父子关系）：
   - 节点 = ActorType（type 粒度）或 PID 实例（pid 粒度）。
   - 每条 ActorRecord：`parent` 非 nil 时 `parent_type → actor_type` 建边；查不到父类型 → `(unknown)`
     合成节点；`parent` 为 nil → `(root)` 合成节点。
   - 节点属性：实例数、`actor_size` 的 sum/avg/max；边属性：父子对计数。

2. **通信图**（sender→receiver，有向）：
   - 节点 = ActorType 或 PID 实例。
   - 每条 MessageRecord 聚合到 `sender → self` 边（**流式聚合，不保存逐条消息**）：
     边属性累计 `count`、`total_bytes`、`msg_types`(Counter: msg_type→count)。
     avg_bytes 惰性求 `total_bytes/count`。
   - `sender` 为 nil（如 `*actor.Started` 系统消息）→ 默认归入 `(system)` 合成节点，
     提供 `--exclude-nil-sender` 可丢弃。
   - 节点属性：ActorSize 统计、出/入度、累计出/入字节。

## 热点度量（metrics.py，可扩展）

边聚合出 `(count, total_bytes, msg_types)` 后，权重函数从此派生。注册表 + `--metric` 选择，
用户可自行加函数并注册：

| metric | 含义 | 计算 |
|---|---|---|
| `count` | 频次 | `count` |
| `total_bytes` | 消息大小累计 | `total_bytes` |
| `avg_bytes` | 平均每条消息大小 | `total_bytes/count` |
| `volume` | 频次 × 每条大小 | 即 `count × avg = total_bytes`（显式命名，等价 total_bytes） |

```python
METRICS = {"count": ..., "total_bytes": ..., "avg_bytes": ..., "volume": ...}
def edge_weight(edge_attrs, metric: str) -> float: ...
```

## 输出后端

- **HTML（export/html.py，pyvis）**：拓扑图与通信图各出一份 `*.html`；节点大小 ∝ 节点总出入权重，
  边粗细 ∝ 所选度量，节点按类型着色，hover 显示 count/bytes/top msg_type。pyvis 默认引用 CDN 的
  vis.js（README 说明可本地化）。
- **报告（report.py）**：CSV（`nodes.csv` / `edges.csv` / `top_links.csv` / `actor_stats.csv` /
  `msg_stats.csv`）+ 文本摘要（stdout）：Top-N 热点链路、Top-N 热点消息类型、Top-N 最热 Actor、
  最大 Actor 等。
- **Neo4j（export/neo4j.py，可选）**：仅在提供 `--neo4j-uri/user/password`（或配置）时启用；
  创建 `(:Actor {type,size_avg,...})` 节点与 `-[:PARENT_OF]->`、`-[:SENDS_TO {count,total_bytes,
  avg_bytes,msg_types}]->` 关系；内置若干 Cypher 查询（Top 链路/最热 Actor/最大 Actor）。
  `neo4j` 驱动 lazy import，未装时给提示不报错。

## CLI（cli.py）

```
python -m actor_topo parse   --log logs/app.log --schema schema.yaml --out out/records.json  # 只解析，导规范 JSON（调试 Schema 用）
python -m actor_topo analyze --log logs/app.log --schema schema.yaml --out out/ \
        --by type|pid --metric count|total_bytes|avg_bytes|volume --top 20 \
        [--exclude-nil-sender] [--neo4j-uri ... --neo4j-user ... --neo4j-password ...]
```

## 依赖（requirements.txt）

`networkx>=3.0`、`pyvis>=0.3`、`pyyaml>=6.0`；可选 `neo4j>=5.0`；测试 `pytest`。
Schema 同时支持 YAML/JSON（按扩展名嗅探，JSON 走 stdlib）。

## 验证

1. **单元测试（pytest）**：PID 解析（nil / 完整 / 嵌套 Master）、两种日志行解析、Schema 加载、
   各 metric 计算、type/pid 两粒度图构建、nil-sender/unknown 合成节点、报告生成。
2. **端到端冒烟**：用 `sample.log`（由当前示例行合成，含父子 + 多条消息）跑
   `python -m actor_topo analyze ...`，产出 HTML + CSV，人工核对 Top-N 排序与手算计数一致。
3. **Schema 可配置性验证**：改动 `schema.yaml` 中 pattern 的字段顺序/别名，解析结果不变。
4. **Neo4j（可选）**：有实例时导入后跑一条 Cypher 验证节点/关系数正确。

## 假设与边界

- 当前按给定格式实现默认 `schema.yaml`；格式变化靠改 Schema，不改代码（遵守字段契约即可）。
- `ActorType="*"` 按字面量当作普通类型串处理（如需归一化可后续加 `type_aliases` 映射）。
- 大日志采用两遍 + 流式聚合，不在内存保留逐条消息，控制内存占用。
- 如需静态热力图 PNG / GraphML(Gephi) 导出，后续可加 `export/matrix.py`、`export/graphml.py`，
  本次按已确认范围（HTML + CSV/报告 + 可选 Neo4j）交付，架构已预留后端扩展点。