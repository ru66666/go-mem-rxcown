// 参数：$graph_id, $type_id（例如 github.com/rclone/rclone/fs.Fs）
// Q1：一个接口的全部实现；value_impl / pointer_impl 说明何种形式满足接口。
MATCH (c:Type {graph_id: $graph_id})-[r:IMPLEMENTS]->(i:Type {graph_id: $graph_id, id: $type_id})
RETURN c.id, c.name, c.package_path, r.value_impl, r.pointer_impl
ORDER BY c.package_path, c.name;

// Q2：名字符合正则的类型及其一跳嵌入、字段引用关系。
// 参数：$graph_id, $name_regex（例如 '.*Context.*'）
MATCH (a:Type {graph_id: $graph_id})-[r:EMBEDS|REFERENCES]->(b:Type {graph_id: $graph_id})
WHERE a.name =~ $name_regex OR b.name =~ $name_regex
RETURN a, r, b;

// Q3：指定类型的传递嵌入链（避免在有环图上无限扩张，深度上限 12）。
MATCH p=(a:Type {graph_id: $graph_id, id: $type_id})-[:EMBEDS*1..12]->(b:Type)
WHERE b.graph_id = $graph_id
RETURN p;

// Q4：一个类型声明的方法（接口方法也用 DECLARES_METHOD 表示）。
MATCH (t:Type {graph_id: $graph_id, id: $type_id})-[:DECLARES_METHOD]->(m:Method {graph_id: $graph_id})
RETURN m.name, m.signature, m.file_path, m.line
ORDER BY m.name;
