# Go Green Tea GC 调研

> 调研日期：2026-09-18  
> 主题：Green Tea GC 的动机、算法、实现、评估、适用边界，以及对本项目“Actor-like 业务生命周期感知内存优化”的启示。

## 1. 结论摘要

Green Tea GC 不是一套新的完整回收器，而是 Go 并发三色标记—清扫 GC 的**新标记/扫描算法**。它不改变对象是否存活的判定，也不把 Go 改造成分代、移动或压缩式 GC；它主要改变的是：发现一个待扫描对象以后，GC 如何组织后续扫描工作。

传统 Go GC 以对象为工作项，沿指针图立即扩散，访问地址容易在整个堆中跳跃。Green Tea 改为以内存页/小对象 span 为主要工作项，在每个 span 内分别记录“已发现”和“已扫描”位，暂缓扫描以积累同一 span 上的多个对象，再按内存顺序批量处理。其直接收益来自三点：

1. 提高空间和时间局部性，减少随机内存访问与 cache miss；
2. 工作队列从大量对象指针变为更少的 span，降低多核并行时的队列竞争；
3. 规则的位图与连续内存扫描允许在新型 x86 上使用 AVX-512 向量化。

官方报告中，不带向量化时，GC-heavy 工作负载的 GC CPU 通常下降 **10%–40%**，部分微基准达到 **10%–50%**；Go 1.26 发布说明给出的现实预期也是 GC-heavy 程序 GC overhead 下降 **10%–40%**。但这不是端到端性能承诺：若程序原本只有 10% CPU 花在 GC，GC 本身下降 10%–40%，折算到总 CPU 约为 1%–4%。

版本状态如下：

| Go 版本 | Green Tea 状态 | 使用方式 |
|---|---|---|
| Go 1.25 | 实验特性 | 构建时设置 `GOEXPERIMENT=greenteagc` |
| Go 1.26 | 默认启用 | 可在构建时用 `GOEXPERIMENT=nogreenteagc` 回退；新版 amd64 可使用向量化扫描 |
| Go 1.27.1（调研日当前稳定版） | Green Tea 已是常规实现的一部分 | 当前源码仍默认开启，并仍保留 `GOEXPERIMENT=nogreenteagc` 关闭路径；这与 Go 1.26 发布说明中“预计 1.27 删除开关”的计划不一致，应以所用工具链实测为准 |

对本项目最重要的判断是：

- Green Tea 能降低“每轮全堆标记”的单位扫描成本，尤其值得在多核、指针密集、同尺寸小对象多的 Actor/消息/状态对象场景中验证；
- Green Tea **不能**让仍可达的事务、DO、StableData 或 mailbox 对象提前死亡，也不感知 `releaseTrans`、DO 去活或业务空闲窗口；
- 因此它应当作为项目的**新基线与底层加速器**，不能替代生命周期域、及时断引用、业务提示、GC pacing 或调度协同；
- 本项目的真正创新空间仍在 Green Tea 上方：把“某批内存何时失去业务价值、何时集中变为垃圾、何时有 CPU 窗口”转化为通用运行时信号，再与默认的内存安全触发机制组合。

## 2. 背景：传统 Go GC 的成本在哪里

Go GC 是并发、非移动的 tracing mark-sweep collector。一次回收从 roots 出发遍历可达对象，标记未结束前，业务 goroutine（mutator）仍可并发运行；标记结束后，未被访问的对象由 sweep 阶段回收。Go 官方成本模型将每轮 GC CPU 近似写成：

```text
单轮 GC CPU ≈ 固定成本 + 活跃堆/roots × 每字节标记扫描成本
总 GC CPU ≈ GC 次数 × 单轮成本
```

`GOGC`、heap goal 和 `GOMEMLIMIT` 主要影响“多久运行一轮”；Green Tea 主要降低式子中的“每字节标记扫描成本”。官方 GC 指南同时强调，live heap 是程序行为的属性，GC 不能直接消除仍然可达的 live objects。[Go GC Guide](https://go.dev/doc/gc-guide)

### 2.1 对象级图洪泛的局部性问题

传统并行标记把灰色对象指针放入 work buffer。worker 取出一个对象，扫描其中的指针，再把新发现对象放回队列。这在逻辑上类似 DFS/BFS 图洪泛，但“图上相邻”并不意味着“内存地址相邻”。因此扫描可能呈现：

```text
对象 A（span 1）→ 对象 B（span 93）→ 对象 C（span 7）→ 对象 D（span 301）
```

每个对象的工作很小，下一次读取又依赖刚刚读到的指针，CPU 很难提前并行发出足够的内存请求。官方设计跟踪中给出的剖析结果是：平均约 **85% 的 GC 时间位于核心 scan loop**，而 scan loop 中 **超过 35% 的 CPU 周期仅用于等待内存访问**。[Green Tea 设计与实现跟踪 #73581](https://github.com/golang/go/issues/73581)

问题随硬件演进加剧：

- CPU 核数增加，但单核可获得的内存带宽没有同比增加；
- NUMA 使远端内存访问更昂贵；
- 多个 GC worker 频繁访问共享 work list，会形成争用；
- 不规则、短小的对象扫描难以利用 SIMD。

### 2.2 Green Tea 要优化的边界

Green Tea 优化“已经决定要进行一次 mark cycle 后，怎样完成标记”。它不直接解决：

- 何时触发下一轮 GC；
- 当前是否处于业务高峰、事务等待期或 CPU 空闲期；
- 对象虽然业务上无用、但仍被 map、timer、mailbox 或 callback 引用的问题；
- 堆碎片与对象搬迁；
- 按请求、事务或 Actor 隔离与批量释放内存。

因此，Green Tea 属于**内存拓扑感知/微体系结构感知**，而不是本文献调研主题中的**业务语义感知**。

## 3. 核心设计

### 3.1 从“对象队列”改为“span 队列”

Green Tea 的口号是 **work with pages, not objects**。原型把工作块定义为 small-object span：8 KiB 对齐、大小为 8 KiB、内部对象属于同一 size class；大对象继续沿用对象级路径。当前 64 位实现的 span 批扫描路径覆盖 16～512 B 对象：判定条件是 `heapBitsInSpan(size) && size >= 16`，而 64 位平台的小对象 heap bitmap 上限为 512 B。这里的范围属于内部实现细节，不是稳定 API，后续版本仍可能调整。[原型说明](https://github.com/golang/go/issues/73581)；[当前 Green Tea 实现](https://go.dev/src/runtime/mgcmark_greenteagc.go)；[小对象元数据实现](https://go.dev/src/runtime/mbitmap.go)

其变化可以概括为：

| 维度 | 传统对象图洪泛 | Green Tea |
|---|---|---|
| 主要工作项 | 待扫描对象指针 | 待扫描 span/page 的代表项 |
| 局部元数据 | mark bit | mark/seen bit + scan bit + span ownership 状态 |
| 队列策略 | 对象 work buffer，偏 LIFO | span 队列偏 FIFO，并可跨 P work stealing |
| 扫描粒度 | 一个对象 | 一个 span 中本轮累计的多个对象 |
| 访问顺序 | 跟随对象图跳转 | 在 span 内尽量按地址顺序连续扫描 |
| SIMD 机会 | 很弱 | 位图及连续页扫描可向量化 |

### 3.2 两组位图：发现与扫描解耦

每个适用的 span 为对象槽维护两种状态：

- `marks`（官方博客称 seen）：已经从 root 或另一个对象发现该对象；
- `scans`：该对象的字段已经被扫描。

三色语义仍然成立：

```text
white：mark=0, scan=0   尚未发现
grey ：mark=1, scan=0   已发现，等待扫描
black：mark=1, scan=1   已扫描
```

发现对象时，GC 原子地设置其 mark bit。如果该 span 尚未入队，则获取 span 的扫描 ownership，并把 span 加入本地队列；后续发现同一 span 上的其他对象只需设置相应 mark bit，无需反复添加对象级工作项。

span 出队时，GC 计算：

```text
本轮待扫描对象 = marks &^ scans
新的 scans      = scans | marks
```

随后把待扫描对象映射为需要检查的机器字，与 allocator 已维护的 pointer/scalar bitmap 求交，只访问活对象中真正可能包含指针的字段。当前 Go 源码中的 `spanInlineMarkBits`、`tryDeferToSpanScan`、`scanSpan` 和 `spanSetScans` 分别对应局部位图、延迟到 span 扫描、span 扫描和位图差集处理。[Go 1.26 Green Tea 源码](https://github.com/golang/go/blob/go1.26.0/src/runtime/mgcmark_greenteagc.go)

### 3.3 为什么使用 FIFO

对象 work buffer 的 LIFO 有利于快速沿一条边继续图遍历，但对 Green Tea 而言，过早处理刚入队的 span 会使其只积累一个对象。原型比较过 FIFO、LIFO、稀疏优先、稠密优先、随机及地址顺序，FIFO 获得了最高的平均扫描密度。

其直觉是：span 在队尾等待期间，其他对象的扫描可能继续发现该 span 上更多对象；等它出队时，可以一次处理多个相邻对象。一个 span 可在一轮 mark phase 中多次入队，因为第一次扫描后又可能发现新的、尚未扫描的对象。

### 3.4 P-local span queue 与 work stealing

最终实现为每个 P 的 `gcWork` 增加 `spanq`：先消费 P-local 工作以减少全局争用，缺乏本地工作时再从其他 P 偷取 span。源码记录的优先顺序是：

1. P-local 对象 work buffer；
2. P-local span queue；
3. 非本地对象 work buffer；
4. 非本地 span queue；
5. 从其他 P steal span。

先处理对象级 work buffer 还有一个作用：让 span 多等待一会儿，增加同 span 对象的积累概率。span 队列本身采用 local ring 加 single-producer/multi-consumer chain，定期 spill 一部分工作以产生并行度。[Go 1.26 `mgcwork.go`](https://github.com/golang/go/blob/go1.26.0/src/runtime/mgcwork.go)

### 3.5 稀疏扫描与单对象快速路径

批量机制有固定成本：读取/合并两组位图、生成 active-object bitmap、展开 active-word bitmap。若一个 span 每次只有一个对象可扫描，Green Tea 可能比旧算法更慢。

实现为此增加两级回退：

- span 入队期间若确认只有一个新 mark，直接扫描代表对象，跳过完整 bitmap merge/diff；
- 已经形成多个 mark、但密度不足以使用向量化 packed scan 时，使用局限在该 span 内的稀疏对象扫描。

Go 1.26 源码还以大约 `objsMarked < nelems/8` 作为 packed scan 的密度判断之一；这是具体版本的实现阈值，不应视为长期 API。[`scanSpan` 实现](https://github.com/golang/go/blob/go1.26.0/src/runtime/mgcmark_greenteagc.go#L3643)

官方博客称，即使一次仅覆盖页中约 **2%** 的对象，整体也可能已经胜过传统图洪泛，但收益取决于对象拓扑和硬件。[Green Tea 官方博客](https://go.dev/blog/greenteagc)

### 3.6 AVX-512 向量扫描

Go 1.26 在较新的 amd64 平台上为小对象扫描加入向量化。高层流程为：

1. 装入 seen/mark bitmap 与 scanned bitmap；
2. 计算本轮 active objects；
3. 按对象大小把“一对象一位”展开为“一机器字一位”的 active words；
4. 与页的 pointer/scalar bitmap 求交；
5. 连续读取 active pointer words，并把发现的指针放入缓冲区。

AVX-512 的宽寄存器可以容纳一页的关键元数据；GFNI 等位操作用于高效完成按 size class 的 bit expansion。官方发布说明把该路径限定为 Intel Ice Lake、AMD Zen 4 及更新平台，并预计额外带来约 **10% 的 GC overhead 降低**，不是所有 amd64 或 ARM 平台都能获得这部分收益。[Go 1.26 Release Notes](https://go.dev/doc/go1.26)

## 4. Green Tea 没有改变什么

这部分对正确评估项目价值很重要。

| 机制 | 是否由 Green Tea 改变 | 说明 |
|---|---:|---|
| 并发 mark-sweep 总体框架 | 否 | 仍是 Go 的并发三色标记与清扫 |
| 存活判定 | 否 | 仍以从 GC roots 的可达性为准 |
| 写屏障与并发正确性 | 原则不变 | 新算法仍要在 mutator 并发修改对象图时维持三色不变式 |
| GC 触发/pacer | 否 | 仍由 heap goal、分配速率、GOGC、GOMEMLIMIT 等控制 |
| 分代回收 | 否 | Green Tea 不是 young/old generation collector |
| 对象移动/压缩 | 否 | 不通过搬迁对象改善碎片或长期物理聚集度 |
| 业务生命周期 | 否 | 不知道事务、请求、Actor、DO 或去活事件 |
| live heap 大小 | 通常不直接改变 | 仍可达对象必须被保留；更快完成 mark 可能间接改变 floating garbage 与峰值 |

特别是，“以 page/span 为扫描单位”不等于“整页对象一起判死”或“region 一次释放”。对象仍逐个保持精确标记语义，span 只是组织扫描工作的批处理单位。

## 5. 官方评估结果

### 5.1 原型测试环境与数据集

官方设计跟踪在低核和高核、amd64 和 arm64 Linux VM 上测试，主要包括：

- GC-heavy 微基准：`golang.org/x/benchmarks/garbage`、Computer Language Benchmarks Game 的 `binary-trees`；
- Go `bent` 与 `sweet` 套件；
- Intel Cascade Lake 16-core VM；
- Intel Sapphire Rapids 88-core VM；
- Google Axion arm64 16-core 与 72-core VM。

GC-heavy 微基准中，Green Tea 相对旧 GC 的 GC CPU 降低 **10%–50%**，且核数越多一般收益越大；L1/L2 cache miss 大约减半。这支持了“局部性 + 多核队列扩展性”两个核心假设。[官方设计评估](https://github.com/golang/go/issues/73581)

### 5.2 代表性应用

| 工作负载 | 堆拓扑特征 | 结果与解释 |
|---|---|---|
| Tile38 | 高 fan-out、堆中大量树对象，可快速在同一 span 积累工作 | 吞吐、延迟和内存均改善；GC overhead 降低约 **35%** |
| Bleve index | 低 fan-out 二叉树且频繁旋转，100+ MiB 堆上的布局被打乱 | 总体接近持平；约一半 span scan 只有一个对象；16-core amd64 的 GC overhead 约回退 **2%**，但 72/88-core 因扩展性更好而明显改善；16-core arm64 有小幅回退 |
| Go compiler | 对该变化不敏感 | 观测到不稳定的约 **0.5%** 小幅回退，原因当时尚未完全确定 |

这组对比给出一个重要结论：Green Tea 能利用应用已经具备的分配局部性，但不能凭空创造局部性。对象在逻辑图上相邻且在相近时间、相同 size class 中分配时，效果更好；长期原地变异、随机链接和频繁树旋转会削弱收益。

### 5.3 Go 1.26 的正式预期

Go 团队在内部生产环境部署后，于 Go 1.26 将 Green Tea 默认启用。官方给出的现实预期是：对 GC-heavy 实际程序，GC overhead 下降 **10%–40%**；在支持向量扫描的新 amd64 上，可能再降低约 **10% 的 GC overhead**。[Go 1.26 发布公告](https://go.dev/blog/go1.26)；[Go 1.26 Release Notes](https://go.dev/doc/go1.26)

需要正确解读三个“百分比”：

```text
若 baseline 总 CPU = 100
其中 GC CPU = 10
Green Tea 让 GC CPU 降 20%
则节约约 2 个总 CPU 单位，而不是端到端快 20%
```

此外，更短的 mark phase 会减少 floating garbage，也会更早暴露其他瓶颈，因此 GC CPU、RSS、吞吐和尾延迟不一定按同一比例变化。

## 6. 局限与风险

### 6.1 算法性局限

- **扫描密度依赖**：同一 span 不能积累多个待扫描对象时，位图与排队成本难以摊销。
- **应用局部性上限**：跨堆随机链接、低 fan-out、频繁拓扑变换会使地址局部性变差。
- **主要优化小对象**：大对象本来就更容易摊销扫描开销，Green Tea 的主要优化目标是数量多、单次工作小的小对象。
- **只降低扫描成本**：若瓶颈来自长期 live set、保留根、分配速率、mark assist、内存上限或业务积压，Green Tea 只能覆盖其中一部分。
- **不保证降低 RSS**：更快回收有时会降低峰值，但没有对象搬迁/压缩，RSS 仍受 span 利用率、scavenger 和可达对象影响。

### 6.2 已报告的回退案例

Go issue 中存在真实工作负载回退。例如一个 darwin/arm64 Chromium 分布式构建场景报告 Go 1.26 默认 Green Tea 为 7m31s，关闭后为 6m54s；该 issue 已关闭，但报告说明“默认更快”不能替代应用级 A/B。[Issue #77824](https://github.com/golang/go/issues/77824)

另一个 Go 1.25 期间的 Hugo 低内存限制案例也曾报告明显 slowdown，表明在强内存压力和特定对象图下，需要同时观察 GC CPU、assist、cycle 频率和 wall time，而不是只看单一指标。[Issue #74923](https://github.com/golang/go/issues/74923)

向量化路径还扩大了硬件特性检测的重要性。Go 1.26.x 曾跟踪 Windows/amd64 上 AVX-512 子特性检查相关的非法指令问题，并安排补丁版本处理。[Issue #79871](https://github.com/golang/go/issues/79871)；[Go 1.26 backport #79920](https://github.com/golang/go/issues/79920)。这类实现 bug 与算法收益是两个层面，但意味着生产升级应使用最新 patch release，而非停留在早期 1.26.0。

## 7. 如何在本项目中验证

### 7.1 核心实验问题

本项目不应只问“Green Tea 开启后快不快”，而应回答：

1. SMC/UPC 等 Actor-like 场景中，小对象和指针对象占比多大？
2. 相同 size class 的事务、消息、等待事件、Actor 状态是否在相近时间分配？
3. Green Tea 的 span scan 是否能形成足够密度，还是经常退化为单对象扫描？
4. GC CPU 降低后，业务 goroutine 是否更快推进事务并释放 DO/StableData，从而打断“GC 抢 CPU—事务变慢—live set 增大”的反馈链？
5. 峰值内存主要来自 GC 来不及收，还是来自对象仍然可达？若是后者，Green Tea 不会直接解决。

### 7.2 建议的 A/B 矩阵

建议先使用仍受支持的最新 patch release。调研日对应 Go 1.27.1 与 Go 1.26.8；当前 1.27 源码仍保留 `nogreenteagc`，但 Go 1.26 对该 A/B 开关有更明确的发布说明承诺。[Go Release History](https://go.dev/doc/devel/release)

| 组别 | 工具链与配置 | 目的 |
|---|---|---|
| A | Go 1.26 默认 Green Tea | 新基线 |
| B | 同一 Go 1.26，`GOEXPERIMENT=nogreenteagc` | 隔离 Green Tea 算法影响 |
| C | Green Tea + 及时断引用/去活清理 | 验证减少 live set 与更快扫描的叠加收益 |
| D | Green Tea + 生命周期提示/pacing | 验证业务时机感知收益 |
| E | Green Tea + 生命周期域/批量回收原型 | 验证减少进入全堆 GC 的对象量 |

各组必须固定：源码、编译参数、`GOMAXPROCS`、CPU 亲和性、NUMA 策略、`GOGC`、`GOMEMLIMIT`、输入负载、预热期与统计窗口。分别覆盖：

- 稳态负载；
- 突发去活/批量会话删除；
- 高并发积压；
- 低内存上限；
- 目标鲲鹏/ARM 平台与可比较的 amd64 平台。

### 7.3 指标

至少同时收集以下指标：

| 类别 | 指标 |
|---|---|
| 业务 | 吞吐、事务完成时间、消息 p50/p95/p99/p99.9、超时率 |
| 内存 | RSS、heap live、heap goal、heap objects、峰值、回收后平台值 |
| GC | cycle 次数、GC CPU、mark/assist CPU、pause、scan bytes、每轮回收字节 |
| 调度 | runnable goroutines、Actor mailbox 长度/字节、GC worker 与 mutator CPU 竞争 |
| 微体系结构 | cycles、instructions、IPC、L1/L2/LLC miss、内存带宽、NUMA remote access |
| 业务生命周期 | active/retired transaction domains、active DO、StableData 字节、事务结束到引用释放的延迟 |

Go 官方建议先用 CPU profile、execution trace、`GODEBUG=gctrace=1` 和 `runtime/metrics` 定位 GC 成本；硬件计数器再用于验证 Green Tea 的 cache/locality 假设。[Go GC Guide](https://go.dev/doc/gc-guide)

一个实用的判定链是：

```text
Green Tea 是否降低 cache miss / GC scan CPU？
  ├─ 否：检查 span 扫描密度、对象拓扑、平台与版本
  └─ 是：业务 p99 / 吞吐是否改善？
       ├─ 否：瓶颈已转移到业务锁、调度、IO 或其他 runtime 路径
       └─ 是：峰值内存是否改善？
            ├─ 否：峰值主要由 live/retained objects 构成
            └─ 是：量化更短 mark phase 与更快事务推进各自贡献
```

## 8. 对本项目方案的具体启示

### 8.1 Green Tea 应成为基线，不应成为项目终点

本项目若基于 Go 1.26/1.27 开展，Green Tea 已经存在。任何 GC 改造的收益都必须与 Green Tea 基线比较，否则可能把上游已经提供的局部性收益误记为项目贡献。

同时，项目不宜复制 Green Tea 的底层 span 扫描机制。更有价值的工作是利用本项目独有的 Actor/事务语义，减少需要由全堆 GC 发现和扫描的对象，或更合理地安排 GC 时机。

### 8.2 生命周期优化与 Green Tea 是乘法关系

可以把一次 GC 成本粗略拆成：

```text
总 GC 成本
≈ GC cycle 次数
× 每轮需扫描的活跃对象/指针量
× 每单位扫描成本
```

- Green Tea 主要降低“每单位扫描成本”；
- 及时清理 `TransSet`、WaitingEvents、timer、callback、mailbox 和 Actor registry 引用，降低“每轮需扫描量”；
- `DomainRetire`、retired bytes、CPU idle window 等信号，可影响“何时运行/运行几次”；
- transaction/DO domain 的批量回收有机会让部分对象根本不进入全堆图遍历。

三类方法不是替代关系，组合后才有机会同时降低 GC CPU、峰值内存与尾延迟。

### 8.3 分配布局本身值得成为优化变量

Green Tea 的效果取决于同一 span 中能否积累多个同时待扫描的对象。这提示项目除了研究“何时回收”，还应研究“相关对象如何布局”：

- 同一事务或同一 DO 的同尺寸小对象是否在时间上集中分配；
- 对象池是否把逻辑相关对象打散到长期存活的 span；
- mailbox payload、事务状态和 StableData 的 size class 是否导致大量稀疏 span；
- 可否在不破坏安全和 API 的前提下，使生命周期相近的对象具有更好的 allocation locality。

但应注意：Green Tea 只按 size class 的物理 span 聚合，不认识“事务 A”。生命周期域设计若要与它协同，需要显式测量，而不能假定同事务对象天然共页。

### 8.4 span scan density 可作为新的诊断量

Go 的 `gctrace` 在高 debug level 和源码内部已有 dense/sparse span scan 统计。项目若维护定制 runtime，可以把以下观测以低开销指标暴露出来：

- 每次 span scan 的 active object 数；
- 单对象快速路径比例；
- dense/sparse scan 数量与各 size class 分布；
- 每个业务阶段的 scan bytes / reclaimed bytes；
- 生命周期优化前后，span density 和 cache miss 是否变化。

这能回答“业务对象布局是否让 Green Tea 真正工作”，比只看 GC pause 更有解释力。

### 8.5 业务信号必须是 soft hint

Green Tea 没有改变 Go GC 的内存安全职责。本项目即使引入：

- `TransactionRetired(bytes)`；
- `DoBecameIdle`；
- `ExpectedIdleWindow(deadline)`；
- `ReclaimableBytesEstimate`；

也应让它们只影响 pacing、worker 配额或机会式提前回收。接近 heap goal/GOMEMLIMIT、分配无法前进或 GC 饥饿时，运行时必须忽略业务提示并强制保证回收进度。业务层也不能通过“事务已结束”的声明绕过 Go 可达性规则；在引用仍存在时直接回收会破坏内存安全。

### 8.6 对 ARM/鲲鹏平台要单独评估

Green Tea 的通用 locality 和 span queue 设计可在 ARM 上受益，但 Go 1.26 官方额外的 SIMD 收益面向新型 amd64 AVX-512，不能外推到鲲鹏。官方原型在高核 arm64 上受益明显，但也在 16-core arm64 的困难工作负载上观察到小幅回退。因此本项目的目标硬件测试至少要拆分：

- 普通标记扫描收益；
- 多核队列扩展性；
- cache/NUMA 行为；
- 是否存在与 amd64 不同的稀疏扫描成本。

### 8.7 推荐研究路线

按风险从低到高，建议：

1. **建立 Green Tea 基线**：在目标负载、目标 ARM 服务器上完成 Go 1.26 default/nogreenteagc A/B；
2. **定位 live 与 garbage**：把 heap peak 分解为真正 live、逻辑已死但仍被引用、已不可达但尚未回收三类；
3. **先做引用卫生**：缩短事务结束、DO Free/Delete、StableData 置空、timer/callback/mailbox 清理之间的延迟；
4. **暴露通用生命周期信号**：只传达 active/retired domain、预计可回收字节和空闲窗口，不把 SMC/UPC 业务名写入 runtime；
5. **接入 pacing/调度**：在不突破 GOMEMLIMIT 和安全线的条件下择机提前 GC，或降低高峰期 GC worker 干扰；
6. **再探索域式分配/批量回收**：无法证明局部所有权或发生跨域逃逸时，回退到标准 Go heap；
7. **最后联合优化布局**：验证生命周期相近对象共 span 是否能同时提高 Green Tea density 和整域释放效率。

## 9. 与已有业务感知 GC 工作的关系

| 工作 | 主要优化维度 | 与 Green Tea 的关系 |
|---|---|---|
| Green Tea | 一轮 GC 内如何更高效地标记/扫描 | 底层扫描引擎 |
| LEGO / Let It Go | 将 Go GC 工作拆细，并在低 CPU 区间执行 | 可在 Green Tea 之上改变工作粒度与时机，但两者源码改造可能相互交叉 |
| iGC / OppZGC | 根据 CPU、heap pressure、预期收益提前触发 | 可借鉴其控制器，Green Tea 降低控制器估计的单轮成本 |
| Monk | 降低并发 GC 对延迟关键 mutator 的 CPU 干扰 | 可借鉴 worker 优先级/饥饿回退，不改变 span 扫描算法 |
| Iso / 生命周期域 | 让请求私有对象局部回收或批量退休 | 减少进入 Green Tea 全堆扫描的对象量，互补性最强 |
| 本项目拟议的事务/DO 信号 | 报告对象何时集中失去业务价值 | 可用于触发与调度，Green Tea 负责随后更便宜地完成标准全堆标记 |

Green Tea 提供了一个很有价值的设计范式：**不要改变正确性语义，先改变工作组织方式，使元数据局部化、工作可批处理、并行队列可扩展，并为稀疏最坏情况保留快速回退。** 本项目若引入生命周期域或业务提示，也应遵循同样原则：通用抽象、精确语义、局部快速路径、安全回退和可观测性缺一不可。

## 10. 资料与可信度说明

本调研优先使用 Go 官方发布说明、官方博客、Go 源码和 Go issue tracker。Green Tea 尚没有一篇正式同行评审论文；最权威的算法说明是 Michael Knyszek 与 Austin Clements 维护的实现 issue、官方博客及对应源码。原型 issue 的数据适合说明机制和早期评估，Go 1.26 发布说明适合说明正式版本状态；个别用户 issue 只作为风险案例，不能代表总体性能。

主要来源：

1. Michael Knyszek, Austin Clements. [The Green Tea Garbage Collector](https://go.dev/blog/greenteagc), Go Blog, 2025-10-29.
2. Michael Knyszek, Austin Clements. [runtime: green tea garbage collector, issue #73581](https://github.com/golang/go/issues/73581), 2025.
3. Go Team. [Go 1.26 Release Notes](https://go.dev/doc/go1.26), 2026.
4. Go Team. [Go 1.26 is released](https://go.dev/blog/go1.26), 2026-02-10.
5. Go Team. [A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide).
6. Go source. [`src/runtime/mgcmark_greenteagc.go` at go1.26.0](https://github.com/golang/go/blob/go1.26.0/src/runtime/mgcmark_greenteagc.go).
7. Go source. [`src/runtime/mgcwork.go` at go1.26.0](https://github.com/golang/go/blob/go1.26.0/src/runtime/mgcwork.go).
8. Go Team. [Go 1.27 Release Notes](https://go.dev/doc/go1.27), 2026.
9. Go Team. [Go Release History](https://go.dev/doc/devel/release), accessed 2026-09-18.
10. Go source. [Current experiment defaults](https://go.dev/src/internal/buildcfg/exp.go) and [`internal/goexperiment.Flags`](https://go.dev/src/internal/goexperiment/flags.go).
