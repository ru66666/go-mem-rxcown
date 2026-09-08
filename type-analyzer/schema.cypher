// Neo4j 5+ schema；先执行本文件，再执行 type-analyzer analyze。
//
// graph_id 是一次“项目快照”的命名空间。Type/Method 的 id 在 graph_id 内唯一，
// 所以同一 Neo4j 库可保存多个项目或同一项目的多次版本快照。
// 所有 Type/Method 节点及所有业务关系都必须带 graph_id。

// -----------------------------------------------------------------------------
// 节点标签、身份与属性契约
// -----------------------------------------------------------------------------
// (:AnalysisRun {
//   id,                         // graph_id，全库唯一
//   project_root,               // 扫描时的绝对根路径
//   project_import_path,        // module path 或 legacy_import_path
//   load_mode,                  // module | vendor
//   go_version, started_at, completed_at, status,
//   package_count, type_count, method_count, diagnostic_count, config_hash
// })
//
// (:Type {
//   graph_id, id,               // id = <package import path>.<declared name>
//   name,                       // 声明名称，不带包前缀
//   qualified_name,             // 与 id 相同，便于人读查询
//   kind,                       // interface | non_interface
//   package_name, package_path,
//   file_path, line, column,    // 声明的真实源位置
//   is_project_package,         // 包是否由初始 pattern 命中
//   is_stdlib,                  // package_path 是否属于标准库
//   type_params                 // []string，例如 ["T comparable"]
// })
//
// (:Method {
//   graph_id, id,               // <type id>#<method name>
//   name, signature,            // signature 使用 import path 作限定符
//   receiver_type_id,           // 所属 Type 的 id；接口方法也填写接口 id
//   receiver_pointer,           // concrete method 的 receiver 是否为 *T；接口为 false
//   file_path, line, column
// })
//
// 位置未知时 file_path=""、line=0、column=0，属性仍必须写入，不能用 null。

CREATE CONSTRAINT analysis_run_identity IF NOT EXISTS
FOR (r:AnalysisRun) REQUIRE r.id IS UNIQUE;

CREATE CONSTRAINT type_identity IF NOT EXISTS
FOR (t:Type) REQUIRE (t.graph_id, t.id) IS UNIQUE;

CREATE CONSTRAINT method_identity IF NOT EXISTS
FOR (m:Method) REQUIRE (m.graph_id, m.id) IS UNIQUE;

// -----------------------------------------------------------------------------
// 索引：覆盖内置查询及常见的包/位置过滤。
// -----------------------------------------------------------------------------
CREATE INDEX type_name IF NOT EXISTS
FOR (t:Type) ON (t.graph_id, t.name);

CREATE INDEX type_package IF NOT EXISTS
FOR (t:Type) ON (t.graph_id, t.package_path);

CREATE INDEX type_file IF NOT EXISTS
FOR (t:Type) ON (t.graph_id, t.file_path);

CREATE INDEX method_name IF NOT EXISTS
FOR (m:Method) ON (m.graph_id, m.name);

CREATE INDEX method_receiver IF NOT EXISTS
FOR (m:Method) ON (m.graph_id, m.receiver_type_id);

// -----------------------------------------------------------------------------
// 关系契约。关系属性 id 是同一对 Type 之间多个字段关系的稳定去重键。
// 导入使用 MERGE 时须按该 id 合并 REFERENCES/EMBEDS；不要仅按两端点合并。
// -----------------------------------------------------------------------------
// (:Type)-[:IN_ANALYSIS {graph_id}]->(:AnalysisRun)
// (:Method)-[:IN_ANALYSIS {graph_id}]->(:AnalysisRun)
//
// (:Type)-[:DECLARES_METHOD {graph_id}]->(:Method)
//
// (:Type {kind:'non_interface'})-[:IMPLEMENTS {
//   graph_id, value_impl, pointer_impl
// }]->(:Type {kind:'interface'})
// 其中 value_impl/pointer_impl 均为 boolean；至少一个为 true。
//
// (:Type)-[:EMBEDS {
//   graph_id, id,              // <source type id>#embed#<field index>#<target type id>
//   field_name, field_index, pointer_depth,
//   file_path, line, column
// }]->(:Type)
// field_name 是 Go 声明中的嵌入字段名；pointer_depth 只会是 0 或 1。
//
// (:Type)-[:REFERENCES {
//   graph_id, id,              // <source type id>#field#<field index>#<reference path>#<target type id>
//   field_name, field_index, pointer_depth, reference_path,
//   file_path, line, column
// }]->(:Type)
// pointer_depth 是到达目标类型前经过的 * 数；reference_path 记录容器路径，例如
// "direct"、"slice.element"、"map.value"、"map.value.slice"、"type_arg[0]"。
// 普通字段绝不创建 EMBEDS；嵌入字段绝不创建 REFERENCES。

// Neo4j Community 不支持关系唯一约束。以下三个关系索引不是正确性前提，
// 但可加快按 graph_id 清理和按字段查询；Neo4j 5+ 可执行。
CREATE INDEX embeds_graph IF NOT EXISTS
FOR ()-[r:EMBEDS]-() ON (r.graph_id);

CREATE INDEX references_graph IF NOT EXISTS
FOR ()-[r:REFERENCES]-() ON (r.graph_id);

CREATE INDEX implements_graph IF NOT EXISTS
FOR ()-[r:IMPLEMENTS]-() ON (r.graph_id);
