# 中英对照翻译规范（所有分片翻译者必须遵守）

## 一、总体目标

把论文的英文原文逐段翻译成中文，输出为 **中英对照的 Markdown**：**每一段英文原文，紧跟一段对应的中文译文**。

## 二、排版格式（严格照做）

### 1. 段落

英文原文段落用 **引用块**（行首 `> `）承载；中文译文用 **普通段落**，紧随其后，两段之间空一行。

```markdown
> For resource utilization efficiency, latency critical (LC) services are commonly
> co-located with best-effort batch jobs in datacenter servers.

为了提升资源利用效率，延迟关键（LC）服务通常与尽力而为（best-effort）型批处理作业
混部在同一台数据中心服务器中。
```

- 英文原文的长度以 **原文的自然段落** 为准（一个自然段 = 一个引用块）。不要把一段拆成多段，也不要把多段并成一段。
- 中文译文与英文原文 **一一对应**，不加不减。
- 引用块内换行时，续行同样以 `> ` 开头。

### 2. 标题

章节标题用中英双语形式，层级与原文一致：

```markdown
## 1 引言（Introduction）

### 3.1 iGC 概览（iGC Overview）
```

### 3. 图表

- 图注 / 表注合并为一行，中英对照，格式：
  `**图 3：采用 iGC 后的延迟分布（Fig. 3: Latency distribution with iGC）**`
- 论文中的表格用 Markdown 表格重建；表头保留英文并在括号内给出中文：
  `| Operation 操作 | READ 读 | UPDATE 更新 | ... |`
  表格数据行按原文照录。

### 4. 参考文献

`REFERENCES` 一节：**保留英文原样，不必翻译**（人名、标题、URL 保持原样）。仅在该节开头加一句中文说明：
`> 说明：以下参考文献列表保留英文原文，未作翻译。`

### 5. 需要丢弃的内容（PDF 抽取噪声）

- 页码（如单独成行的 `43`、`169`、`53`）
- 页眉 / 页脚（如 `Middleware '22, November 7–11, 2022, Quebec City, Quebec, Canada Junxian and Aidi, et al.`、`HPDC '23, June 16–23, 2023, Orlando, FL, USA Junxian Zhao, ...`）
- 版面残留的空格与分栏错位
- 版权声明 / ACM 出版信息块可以保留一小段，译为中文即可；`CCS CONCEPTS`、`KEYWORDS` 保留原文并给出中文译文。

### 6. 修复英文原文的抽取缺陷

PDF 抽取在部分区域 **丢失了单词之间的空格**，例如：

```
LCservices.WeusethreeNoSQLdatabasesasLCservices,Cassandra-
HBase-2.2.7, and Solr-8.8.2.
```

翻译时请在引用块中**把英文还原成带空格的正常英文**（如上例应还原为 `LC services. We use three NoSQL databases as LC services, Cassandra-4.0-beta2, HBase-2.2.7, and Solr-8.8.2.`）。
另外要**合并跨行断词**：`interfer-\nence` → `interference`。

### 7. 保留不译的内容

- 数字、百分比、单位、实验参数（如 `p95`、`2.56X`、`8GB`）
- 代码、命令行、配置项、函数名、变量名（如 `GOGC`、`GOMAXPROCS`、`nice()`、`taskset`、`ZGC`、`G1 GC`、`TCmalloc`）
- 缩写首现时给出全称与中文，如 `延迟关键（latency critical, LC）`，其后沿用 `LC`。

## 三、术语对照表（务必统一使用）

| 英文 | 中文 |
| --- | --- |
| garbage collection (GC) | 垃圾回收（GC） |
| garbage collector | 垃圾回收器 |
| mutator | mutator（应用线程）※首现如此，其后用 `mutator` |
| concurrent GC | 并发 GC |
| incremental GC | 增量 GC |
| generational GC | 分代 GC |
| Stop The World (STW) | Stop-The-World（STW，全停顿） |
| tail latency / heavy-tailed latency | 尾延迟 / 重尾延迟 |
| p95 / p99 percentile latency | p95 / p99 百分位延迟（或 p95 / p99 尾延迟） |
| latency critical (LC) | 延迟关键（LC） |
| throughput | 吞吐量 |
| service level objective (SLO) | 服务等级目标（SLO） |
| QoS | 服务质量（QoS） |
| multi-tenant / multi-tenancy | 多租户 |
| co-location / co-located | 混部 / 混合部署 |
| interference | 干扰 |
| resource contention | 资源争用 |
| CPU contention | CPU 争用 |
| semantic gap | 语义鸿沟 |
| Simultaneous Multi-Threading (SMT) | 同时多线程（SMT） |
| hyper-threading (HT) | 超线程（HT） |
| heap | 堆；runtime heap 运行时堆 |
| allocation | 分配 |
| reclaim / reclaim memory | 回收 / 回收内存 |
| scheduler | 调度器 |
| thread placement / CPU affinity | 线程放置 / CPU 亲和性 |
| worker threads | 工作线程 |
| batch jobs | 批处理作业 |
| best-effort | 尽力而为（best-effort） |
| Java Virtual Machine (JVM) | Java 虚拟机（JVM） |
| GPM model | GPM 模型 |
| monitor / assist / mark | （Go GC 三色标记中的）监控/辅助/标记 —— 按上下文译，保留英文原词 |
| CPU utilization | CPU 利用率 |
| cache miss | 缓存缺失 |
| retired instructions | 退休指令数（指令完成数） |
| latency distribution / CDF | 延迟分布 / 累积分布函数（CDF） |
| evaluation | 评测 |
| overhead | 开销 |
| design | 设计 |
| implementation | 实现 |
| workload | 负载 |
| bursty requests | 突发请求 |
| rate limiter | 限流器 |
| backpressure | 背压 |
| steal work / work stealing | 窃取任务 / 工作窃取 |
| memory hierarchy | 存储层次 |
| NUMA | 非一致性内存访问（NUMA） |

## 四、译文质量要求

1. **学术书面语**，不要口语化；不要加入原文没有的解释、评价或补充。
2. 长句按中文习惯拆分重组，但**信息不得增删**。
3. 术语首次出现时中英并列，之后统一用中文或缩写。
4. 数字、实验结果必须与原文完全一致。
5. 不要输出 "以下是翻译" 之类的元话语，直接给正文。
