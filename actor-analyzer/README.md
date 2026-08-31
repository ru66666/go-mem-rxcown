# actor-analyzer

解析 protoactor-go 插桩日志，构建 **Actor 父子拓扑图** 与 **Actor 通信热点图**，
辅助分析 Actor 通信模式与性能瓶颈。

插桩与解析完全解耦：本工具只面向"日志文本"这一契约，通过 **Schema** 描述日志格式，
格式变化时改 Schema 即可，无需改代码。

## 快速开始

```bash
# 1) 创建虚拟环境并安装依赖
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt

# 2) 解析 + 生成图与报告（交互式 HTML + CSV + 文本摘要）
.venv/bin/python -m actor_topo analyze \
    --log sample.log --schema schema.yaml --out out/ \
    --by type --metric count --top 20

# 3) 仅解析、导出规范化 JSON（调试 Schema 用）
.venv/bin/python -m actor_topo parse \
    --log sample.log --schema schema.yaml --out records.json
```

生成结果（在 `out/` 下）：

| 文件 | 说明 |
|---|---|
| `topology.html` | 父子拓扑图（交互式，节点大小 ∝ 实例数） |
| `communication.html` | 通信热点图（交互式，边粗细 ∝ 所选度量） |
| `topo_nodes.csv` / `topo_edges.csv` | 拓扑图节点 / 边 |
| `nodes.csv` / `edges.csv` | 通信图节点（出入流量）/ 边（count、bytes、avg） |
| `top_links.csv` | Top-N 热点链路 |
| `actor_stats.csv` | 每个 Actor 的实例数、大小、出入流量 |
| `msg_stats.csv` | 每种消息类型的频次 |

## 命令行参数

```
python -m actor_topo analyze \
    --log 日志文件 --schema schema.yaml --out 输出目录 \
    --by type|pid          # 通信图聚合粒度：type=按 ActorType，pid=按实例
    --metric count|total_bytes|avg_bytes|volume   # 热点度量
    --top 20               # Top-N 数量
    --exclude-nil-sender   # 丢弃 sender 为 nil 的消息（如 *actor.Started）
    --no-html              # 不生成 HTML
    --neo4j-uri / --neo4j-user / --neo4j-password   # 可选：导入 Neo4j
```

## 热点度量（可定制）

"热点"如何衡量是**可配置**的。边在聚合阶段累积 `count`（频次）、`total_bytes`（累计字节）、
`msg_types`（消息类型分布），度量函数从这三者派生标量权重：

| 度量 | 含义 | 计算 |
|---|---|---|
| `count` | 消息频次 | `count` |
| `total_bytes` | 消息大小累计 | `total_bytes` |
| `avg_bytes` | 平均每条消息大小 | `total_bytes / count` |
| `volume` | 频次 × 每条大小 | `count × avg = total_bytes` |

自定义度量：在代码里调用 `actor_topo.metrics.register_metric(name, fn)` 注册一个
`fn(edge_attrs) -> float` 即可，随后用 `--metric name` 选择。

## Schema（解析可配置）

Schema 用 YAML（也支持 JSON）描述日志如何解析。核心是**字段契约**：正则捕获组必须使用
这些规范名，日志格式怎么变、捕获组怎么重排，只改 `pattern` 即可，下游代码不用动：

- Actor 记录：`actor_type`(string) `actor_size`(int) `self`(pid) `parent`(pid)
- Message 记录：`msg_type`(string) `msg_size`(int) `sender`(pid) `self`(pid) `parent`(pid)
- PID 字段：`Address` `Id` `GroupID`(int) `Master`(pid，递归，可为 nil)

```yaml
version: 1
records:
  actor:
    pattern: '^\[ACTOR\]\s+INFO\s+Actor\s+statistics统计\s+ActorType="(?P<actor_type>[^"]*)"\s+...$'
    fields: {actor_type: string, actor_size: int, self: pid, parent: pid}
pid:
  null_literal: "nil"
  fields:
    Address: string
    Id: string
    GroupID: int
    Master: pid
```

要点：

- **pattern 用单引号单行标量书写**，不要用 YAML 折叠标量（`>`/`|`），否则换行会被折成空格破坏正则。
- **类型强转**：`string` 原样；`int` 转整数；`pid` 递归解析。
- **PID 递归解析**：值为 `null_literal`（`"nil"`）→ `None`；否则按顶层逗号切分（跟踪 `{}` 深度）成
  `Key:Value`，依 `pid.fields` 逐字段强转；`Master` 形如 `{...}`/`nil` 时递归。
- **PID 字段名映射**：wire 名（Go `PascalCase`）默认按 `snake_case` 映射到模型字段
  （`Address→address`、`GroupID→group_id` 等）；需要自定义时用 `{type: ..., name: ...}` 显式指定。
- 未知字段会被忽略（前向兼容）；`nil` 的 sender 归入 `(system)`，parent 归入 `(root)`，
  查不到类型的端点归入 `(unknown)`。

## 进阶：Neo4j 图数据库

可选地把拓扑图与通信图导入 Neo4j，用 Cypher 查询、用 Neo4j Browser 可视化：

```bash
.venv/bin/python -m actor_topo analyze --log app.log --schema schema.yaml --out out \
    --neo4j-uri bolt://localhost:7687 --neo4j-user neo4j --neo4j-password secret
```

需先 `pip install neo4j` 并运行 Neo4j 实例。导入后可用 `actor_topo/export/neo4j.py` 里
`QUERIES` 中预置的 Cypher（Top 链路 / 最热 Actor / 最大 Actor / 父链）。

## 说明

- **HTML 依赖 CDN**：pyvis 生成的 HTML 引用 vis-network 的 CDN，浏览需联网；离线可自行内联 vis.js。
- **大日志内存友好**：消息边流式聚合（不保存逐条消息），仅保留 Actor 记录与图结构。
- 拓扑图/通信图均支持 `--by type`（类型级，噪音少）与 `--by pid`（实例级，精确定位）。

## 目录结构

```
actor-analyzer/
├── schema.yaml        # 默认解析 Schema
├── sample.log         # 样例日志
├── actor_topo/        # 包
│   ├── schema.py pid.py parser.py model.py build.py metrics.py report.py cli.py
│   └── export/        # html.py（pyvis）/ neo4j.py（可选）
└── tests/             # pytest 单元测试
```

运行测试：`.venv/bin/python -m pytest -q`
