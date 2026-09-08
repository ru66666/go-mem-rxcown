# Go 类型关系分析器设计

## 目标与边界

工具命名为 `type-analyzer`，实现为独立 Go CLI。它将一个 Go 项目及其可解析的依赖包装载为
`go/types` 语义包图，抽取**具名类型**、声明的方法，以及 `IMPLEMENTS`、`EMBEDS`、`REFERENCES`
三类类型关系，批量写入 Neo4j。

“项目范围”是根项目各 `patterns` 匹配的包和它们的可达依赖；默认也保留标准库，因而接口实现可以跨
项目、第三方库和标准库查询。匿名 struct/interface、内建类型和无法得到 `*types.TypeName` 的类型不建
`Type` 节点。这样每个节点都具有稳定、可定位的声明来源。

## 命令行和产物

```
type-analyzer analyze --config config.yaml
type-analyzer schema  --uri bolt://localhost:7687 --user neo4j \
  --password-env TYPE_ANALYZER_NEO4J_PASSWORD --file schema.cypher
type-analyzer query   --file queries.cypher                 # 仅打印内置查询文件
```

`analyze` 会先在 stdout 输出汇总（包、类型、方法、各边数量和诊断），然后按批次写库。配置格式见
[`config.example.yaml`](config.example.yaml)；图 schema 和可直接粘贴的查询分别在
[`schema.cypher`](schema.cypher)、[`queries.cypher`](queries.cypher)。密码只从环境变量读取。

建议目录结构：

```
type-analyzer/
  cmd/type-analyzer/main.go
  internal/config/      # YAML 解析、配置校验
  internal/load/        # module/vendor 两种 packages.Load 策略
  internal/model/       # 与 Neo4j 解耦的节点、边、诊断 DTO
  internal/extract/     # types + AST 抽取和稳定 ID
  internal/store/neo4j/ # UNWIND/MERGE 批量导入
  config.example.yaml schema.cypher queries.cypher
```

推荐依赖：`golang.org/x/tools/go/packages`、`gopkg.in/yaml.v3` 和 `neo4j-go-driver/v5`。

## 包装载

### 有 `go.mod`

以 `target.root` 为 `packages.Config.Dir` 调用 `packages.Load`，模式至少包含
`NeedName | NeedFiles | NeedCompiledGoFiles | NeedSyntax | NeedTypes | NeedTypesInfo | NeedImports |
NeedDeps | NeedModule`。`BuildFlags` 由 `build.tags` 生成，`Env` 合并 `build.env`。从初始
`patterns` 的包沿 `Imports` 遍历，去重后得到所有已装载包；不能只遍历初始 `./...`，否则漏掉外部
依赖。

若发现 `packages.Package.Errors`：`strict: true` 立即失败；否则将包含 package、position、message 的
诊断写入 stderr/汇总并继续处理可用类型。对带 cgo 或平台条件编译的仓库，用户须在配置中给出与构建一致的
`GOOS`、`GOARCH`、tags 等环境。

### 无 `go.mod`、依赖位于 `vendor/`

现代 module 模式不能可靠地将此目录当作独立 vendor 项目处理。`vendor.legacy_import_path` 因而是
有意要求的显式输入。工具先检查根目录是否精确位于
`$GOPATH/src/<legacy_import_path>`；若是，直接设置该 `GOPATH` 并加载，完全不复制项目。否则创建临时目录
`GOPATH`，将根项目复制为 `$TMP/src/<legacy_import_path>`，设置 `GO111MODULE=off`、`GOPATH=$TMP`，并以
`<legacy_import_path>/...` 加载。Go 工具链此时会遵守传统 vendor 查找顺序（项目/父目录的 `vendor`），
可解析 vendor 内部的嵌套 vendor。

复制而非软链接是必要的：`go list <import-path>/...` 会刻意忽略 GOPATH 中的 symlink 目录。分析结束后
移除**仅由本程序创建**的临时目录。原项目、vendor 和 module cache 都不写入。若项目含
relative import、缺少 vendor 依赖或无法确定旧 import path，报告明确错误而不伪造关系。`mode: auto`
优先检测 `go.mod`，否则仅当 `vendor/` 存在时采用此路径。

## 语义抽取规则

扫描每个已加载包的 `Types.Scope().Names()`，仅处理名称对象为 `*types.TypeName` 且底层为
`*types.Interface` 或 `*types.Struct` 的具名类型。ID 为：

```
<package import path>.<declared type name>
```

包路径而非包名保证重名包不冲突；别名会先经 `types.Unalias` 归一。源位置优先 `TypeName.Pos()`，从
`token.FileSet.PositionFor(pos, true)` 得到真实文件路径与行号。

| 关系 | 精确定义 | 边属性 |
|---|---|---|
| `IMPLEMENTS` | 对每个非接口具名 `T` 和接口 `I`，检查 `types.Implements(T, I)` 与 `types.Implements(types.NewPointer(T), I)` | `value_impl`, `pointer_impl` |
| `EMBEDS` | struct 的 `Field(i).Embedded()` 为真；终点为字段类型剥去 pointer、alias、实例化后的具名类型 | `field_name`, `pointer_depth`, `file_path`, `line` |
| `REFERENCES` | 非嵌入 struct 字段的类型可归约到具名 struct/interface（递归走 pointer、array、slice、map、chan、签名的 type arguments） | `field_name`, `pointer_depth`, `reference_path`, `file_path`, `line` |
| `DECLARES_METHOD` | 具名类型的显式方法，或 interface 的显式方法；不写入 promoted/inherited method | 无 |

`IMPLEMENTS` 的目标必须是接口节点；边以 `(source,type target)` 合并，所以同时满足值和指针时只留一条
边并置两个布尔值。接口自身不作为 `IMPLEMENTS` 的 source；接口嵌入通过 `EMBEDS` 表达。接口必须先调用
`Complete()`，否则嵌入接口的方法集未展开，`types.Implements` 会不正确。

对于字段，`pointer_depth=0` 表示值字段，`>0` 表示经多少层指针；`reference_path` 记录从字段根到命名
类型的容器路径，例如 `map.value.slice`，避免把复杂容器误报为直接值嵌套。普通字段只会产生
`REFERENCES`，永不产生 `EMBEDS`。一个字段可能含多个具名类型（如泛型实参），每个终点各有一条边。

方法 ID 为 `<receiver type ID>#<method name>`；`signature` 使用 `types.TypeString(sig, qualifier)`，其中
qualifier 返回 import path（而非短包名），使签名可比较且不会歧义。方法的文件和行取 `*types.Func.Pos()`。

## Neo4j 数据模型与幂等导入

所有节点和关系都带 `graph_id`，避免多次扫描互相污染。每个命名空间另有一个 `AnalysisRun` 节点，
记录装载模式、项目 import path、Go 版本、配置摘要和计数，便于审计某次分析。完整的属性契约、约束与
索引见 [`schema.cypher`](schema.cypher)。核心节点属性如下，满足需求文档规定字段：

```
(:Type {
  graph_id, id, name, kind: 'interface'|'non_interface',
  package_name, package_path, file_path, line
})
(:Method { graph_id, id, name, signature, file_path, line })
```

关系为：

```
(:Type)-[:IMPLEMENTS {value_impl, pointer_impl}]->(:Type)
(:Type)-[:DECLARES_METHOD]->(:Method)
(:Type)-[:REFERENCES {field_name, pointer_depth, reference_path, file_path, line}]->(:Type)
(:Type)-[:EMBEDS {field_name, pointer_depth, file_path, line}]->(:Type)
```

导入分为 AnalysisRun、Type、Method、归属/方法边、类型边等 `UNWIND $rows` 批次；每个批次都 `MERGE`
按稳定 ID 对齐，再 `SET` 当前扫描属性。`EMBEDS`/`REFERENCES` 以关系的稳定 `id` 合并，而不是只按端点，
从而不会丢失“同一类型的两个字段引用同一目标”的信息。因此同一 `graph_id` 的重复执行是幂等的。关系每次
扫描前先按 `graph_id` 删除，随后重建；节点采用本次扫描 id 集合进行清理，避免源代码删类型后残留孤儿数据。
清理只限给定的 `graph_id`，绝不能清空整个数据库。

## 性能和正确性

建节点、字段/方法边是线性的。朴素接口实现判定是 `O(ConcreteTypes × Interfaces)`，在大型依赖图上昂贵；
先用接口的显式方法数、`types.AssignableTo` 的前置条件缩小候选，仍以 `types.Implements` 作唯一真值；
并按包/分块并发计算，最终按 ID 排序以得到可复现输出。Neo4j 写入使用配置中的 batch size。

测试不依赖网络：fixture 包覆盖值接收者/指针接收者、嵌入 pointer、普通值/pointer 字段、泛型、别名、
接口嵌入和跨包实现；分别以临时 module 和临时 GOPATH+vendor fixture 运行。集成测试可选：克隆 rclone，
运行 module 场景；`go mod vendor && rm go.mod go.sum` 后，以其 module path 配置 vendor 场景，并对两次
的类型/关系计数和代表性查询结果做对比。
