# Go 类型关系分析器

`type-analyzer` 是一个 Go CLI。它使用 `go/packages` 装载目标项目及其依赖，并以 `go/types` 的
类型信息构建类型关系图。结果可写入 Neo4j，或导出为 CSV；两种输出共享同一套语义模型。

## 能力

- 识别包级具名类型，标注 `interface` 或 `non_interface`，记录包、文件和源码位置。
- 识别接口实现：分别检查 `T` 与 `*T` 是否实现接口 `I`。
- 识别结构体嵌入：例如 `type A struct { B }` 产生 `A -[:EMBEDS]-> B`。
- 识别普通字段引用：保留字段名、容器路径和 `pointer_depth`，可区分 `B`、`*B`、`[]B`、
  `map[K]B` 与泛型实参中的类型。
- 记录具名类型显式声明的方法；不会将 promoted/inherited method 重复写入。
- 支持有 `go.mod` 的 module 项目和无 `go.mod`、仅有 `vendor/` 的传统 GOPATH 项目。

关系的精确定义、稳定 ID 和幂等导入细节见 [plan.md](plan.md)；Neo4j 约束与属性契约见
[schema.cypher](schema.cypher)。

## 安装与配置

要求 Go 1.23+。在本目录执行：

```bash
go build ./cmd/type-analyzer
cp config.example.yaml config.yaml
```

编辑 `config.yaml` 中的 `target.root`、`target.mode` 和数据库/输出配置。密码不写入 YAML，
由 `neo4j.password_env` 指定的环境变量提供。

最小 module 配置示例：

```yaml
target:
  root: /absolute/path/to/go-project
  mode: module
  patterns: ["./..."]
analysis:
  include_dependencies: true
  include_stdlib: false
  strict: false
output:
  csv_dir: ""
neo4j:
  uri: bolt://localhost:7687
  database: neo4j
  user: neo4j
  password_env: TYPE_ANALYZER_NEO4J_PASSWORD
  graph_id: my-project
```

完整字段及默认值见 [config.example.yaml](config.example.yaml)。`patterns` 遵循 `go list` 包模式；
`build.tags` 与 `build.env` 可传入条件编译标签和 `GOOS`、`GOARCH` 等环境。

## 分析项目

只分析并打印汇总/诊断，不输出数据：

```bash
go run ./cmd/type-analyzer analyze --config config.yaml --no-import
```

默认分析完成后会导入 Neo4j。输出的 `graph_id` 是一次项目快照命名空间；同一数据库可保留多个
`graph_id`，重复导入同一个 `graph_id` 是幂等的。

### Module 项目

`mode: module`（或 `auto` 且项目根目录有 `go.mod`）以项目根目录为工作目录调用 `go/packages`。
初始 `patterns` 的导入依赖会被递归装载。若 `analysis.strict: false`，无法完整 type-check 的包会记录为
诊断，但其余可用包仍会分析。

### 纯 vendor/GOPATH 项目

无 `go.mod` 时使用：

```yaml
target:
  root: /work/gopath/src/example.com/acme/service
  mode: vendor
  patterns: ["./..."]
  vendor:
    legacy_import_path: example.com/acme/service
```

工具以 `GO111MODULE=off` 装载。若 `root` **精确位于**
`$GOPATH/src/<legacy_import_path>`，会直接使用该 GOPATH，不复制任何文件。否则，为避免 Go 工具链
忽略 GOPATH 内 symlink 目录的 `...` 展开，工具会把项目复制到临时 GOPATH 的相应 `src/` 路径，
分析后自动删除临时目录。该逻辑跨平台有效。

## CSV 导出

无需 Neo4j 时，指定目录即可导出 CSV；此模式默认不会连接数据库：

```bash
go run ./cmd/type-analyzer analyze --config config.yaml --csv-dir out/type-graph
```

也可在配置文件设置 `output.csv_dir`。如需同时导出 CSV 和导入 Neo4j，增加 `--also-import`：

```bash
go run ./cmd/type-analyzer analyze --config config.yaml \
  --csv-dir out/type-graph --also-import
```

| 文件 | 内容 |
|---|---|
| `analysis_runs.csv` | `graph_id`、项目根、装载模式、Go 版本 |
| `types.csv` | Type 节点及源码位置、包、类别、泛型参数 |
| `methods.csv` | Method 节点、签名、接收者类型及指针接收者标记 |
| `implements.csv` | `IMPLEMENTS` 边及 value/pointer 实现标记 |
| `embeds.csv` | `EMBEDS` 边、嵌入字段及指针层数 |
| `references.csv` | `REFERENCES` 边、字段、容器路径及指针层数 |
| `diagnostics.csv` | 包装载/type-check 诊断 |

## 用 Docker 启动 Neo4j

以下命令启动 Neo4j 5，保留命名卷中的数据，并开放 Browser（7474）与 Bolt（7687）：

```bash
docker run --detach --name type-analyzer-neo4j \
  --publish 7474:7474 --publish 7687:7687 \
  --env NEO4J_AUTH=neo4j/change-this-password \
  --volume type-analyzer-neo4j-data:/data \
  neo4j:5
```

等待服务启动后，在 <http://localhost:7474> 登录（用户名 `neo4j`，密码为上面设置的值），并在本目录执行：

```bash
export TYPE_ANALYZER_NEO4J_PASSWORD='change-this-password'
go run ./cmd/type-analyzer schema --config config.yaml --file schema.cypher
go run ./cmd/type-analyzer analyze --config config.yaml
```

停止/再次启动容器：

```bash
docker stop type-analyzer-neo4j
docker start type-analyzer-neo4j
```

如需彻底删除容器及其分析数据：

```bash
docker rm -f type-analyzer-neo4j
docker volume rm type-analyzer-neo4j-data
```

`schema` 子命令会应用 [schema.cypher](schema.cypher) 的唯一约束和索引。基础查询位于
[queries.cypher](queries.cypher)，例如将其中的 `$graph_id`、`$type_id` 或 `$name_regex` 作为参数
传给 Neo4j Browser，即可查询接口的实现类型、类型的嵌入/嵌套关系和声明方法。

## 图模型

```text
(:AnalysisRun)<-[:IN_ANALYSIS]-(:Type)-[:DECLARES_METHOD]->(:Method)
                                  |\
                                  | \-[:IMPLEMENTS]->(:Type {kind: 'interface'})
                                  |-[:EMBEDS]->(:Type)
                                  \-[:REFERENCES]->(:Type)
```

`Type.id` 是 `<package import path>.<declared name>`，在 `graph_id` 内唯一。`Method.id` 是
`<type id>#<method name>`。`IMPLEMENTS` 具有 `value_impl`、`pointer_impl` 两个布尔属性；
`EMBEDS`、`REFERENCES` 具有字段名、字段序号、`pointer_depth`、文件和行号，后者另有
`reference_path`。

## CLI 参考

```text
type-analyzer analyze --config config.yaml [--no-import]
                      [--csv-dir DIR] [--also-import]
type-analyzer schema  --config config.yaml [--file schema.cypher]
type-analyzer query   [--file queries.cypher]
```

- `--no-import` 与 `--also-import` 不能同时使用。
- 指定 CSV 目录时不加 `--also-import`，不会访问 Neo4j。
- `schema` 和 `analyze` 的 Neo4j 导入都需要配置 `uri`、`user`、`password_env`。

## 测试

普通单元测试不依赖网络或 Neo4j：

```bash
go test ./...
```

真实项目集成测试使用 `testdata/` 中的 rclone `v1.68.2`（commit
`f2d16ab4c5856098abe9e194c5f4dde6bfc56b66`）。这些大型测试资产不会提交；首次使用前生成：

```bash
./scripts/prepare-rclone-testdata.sh
```

该脚本从 `https://github.com/rclone/rclone.git` 浅克隆 module 副本，复制一份并执行 `go mod vendor`，
再删除 vendor 副本根目录的模块文件。生成结果为：

- `rclone-module` 保留 `go.mod`，验证 module 装载。
- `rclone-vendor` 由 `go mod vendor` 后删除根目录 `go.mod`、`go.sum`，验证纯 vendor 装载。

测试默认跳过大型项目扫描；主动执行：

```bash
TYPE_ANALYZER_RCLONE=1 go test ./internal/integration \
  -run TestRcloneModuleAndVendor -v
```

两个场景均验证 `*backend/local.Fs` 对 `fs.Fs` 的指针实现、`fs.OverrideRemote` 对
`fs.ObjectInfo` 的嵌入，以及 `backend/local.Fs` 对 `Options`（值）和 `*fs.Features`（指针）的字段引用。
