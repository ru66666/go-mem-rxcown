<!-- 中英对照翻译 · 由 text-raw/2023-Zhao-Let-It-Go.txt 生成 -->

> **阅读说明**
> 本文为**中英对照**译本：每一段**英文原文**以引用块（行首 `>`）给出，紧随其后的**普通段落**为对应的中文译文，两者逐段一一对应。
> 图表题注采用「中文（English）」双语形式；**参考文献列表保留英文原文，未作翻译**。
> 原文由 PDF 抽取，其中丢失的空格与跨行断词已在英文侧还原。

**原文标题**：Let It Go: Relieving Garbage Collection Pain for Latency Critical Applications in Golang
**作者**：Junxian Zhao、Xiaobo Zhou（通讯作者）、Sang-Yoon Chang、Chengzhong Xu
**会议**：HPDC '23 — 第 32 届高性能并行与分布式计算国际研讨会（32nd International Symposium on High-Performance Parallel and Distributed Computing），2023 年 6 月 16–23 日，美国佛罗里达州奥兰多
**DOI**：https://doi.org/10.1145/3588195.3592998

---

# 放手吧：为 Golang 中的延迟关键应用缓解垃圾回收之痛

**Let It Go: Relieving Garbage Collection Pain for Latency Critical Applications in Golang**

**Junxian Zhao**（jzhao@uccs.edu）— 科罗拉多大学科罗拉多斯普林斯分校（University of Colorado, Colorado Springs）

**Xiaobo Zhou∗**（waynexzhou@um.edu.mo）— 澳门大学智慧城市物联网国家重点实验室（IOTSC, University of Macau），澳门特别行政区（Macau S.A.R）

**Sang-Yoon Chang**（schang2@uccs.edu）— 科罗拉多大学科罗拉多斯普林斯分校（University of Colorado, Colorado Springs）

**Chengzhong Xu**（czxu@um.edu.mo）— 澳门大学智慧城市物联网国家重点实验室（IOTSC, University of Macau），澳门特别行政区（Macau S.A.R）

> ∗The corresponding author.

∗通讯作者。

## 摘要（Abstract）

> Garbage Collection (GC) is a representative automatic memory manager widely deployed in
> popular programming languages, such as Java, C#, and Golang (Go). Through GC, these
> languages provide programmers with flexibility and safety. However, GC leads to non-trivial
> overhead in compute and memory resources during application runtime. GC threads compete
> with non-GC threads (mutators) of an application, which particularly impacts latency-critical
> (LC) applications and causes long tail latency. Existing GC approaches do not efficiently
> address the interference, as GC is triggered passively without a global insight of the
> application; or they employ incremental GC to reduce the interference, while the incremental
> progress is not dynamically tailored during GC process according to runtime characteristics,
> which leads to significant performance degradation upon bursty requests.

垃圾回收（Garbage Collection, GC）是一种具有代表性的自动内存管理器，广泛部署于 Java、C# 和 Golang（Go）等主流编程语言中。借助 GC，这些语言为程序员提供了灵活性与安全性。然而，GC 在应用运行期会带来不容忽视的计算与内存资源开销。GC 线程会与应用的非 GC 线程（mutator（应用线程））相互竞争，这对延迟关键（latency critical, LC）应用的影响尤为突出，并导致长尾延迟。现有 GC 方法未能有效解决这种干扰：要么 GC 是被动触发的，缺乏对应用的整体洞察；要么采用增量 GC 来降低干扰，但增量进度在 GC 过程中并未依据运行时特征动态调整，从而在突发请求下导致显著的性能退化。

> We present LEGO, an efficient and non-intrusive GC framework that deploys a novel elastic
> incremental GC mechanism integrated with an adaptive GC scheduler to reduce CPU contention
> between GC and mutators, and improve resource utilization and QoS of LC applications. We
> choose to develop LEGO in Go, as Go is specifically designed for cloud applications and the
> current GC mechanisms for JVM are ineffective for Go due to its unique GPM thread scheduling
> and memory allocation model. LEGO adapts incremental GC into Go to tackle the full GC issue
> and addresses resource contention with a proactive scheduler for adaptive GC triggering.
> Importantly, LEGO leverages the elastic incremental GC mechanism in mitigating the
> interference from unavoidable GC in face of bursts of requests. We implement and evaluate
> LEGO with popular LC applications developed in Go. Results show that compared to the default
> Go GC and a tailored G1 GC, LEGO significantly improves both the tail latency and throughput
> for LC applications.

我们提出 LEGO，一个高效且非侵入式的 GC 框架，它部署了一种新颖的弹性增量 GC 机制，并与自适应 GC 调度器相结合，以降低 GC 与 mutator 之间的 CPU 争用，同时提升 LC 应用的资源利用率与 QoS。我们选择在 Go 中开发 LEGO，因为 Go 是专为云应用设计的，而现有的 JVM GC 机制由于 Go 独特的 GPM 线程调度与内存分配模型，对 Go 而言并不有效。LEGO 将增量 GC 适配到 Go 中，以解决全量 GC 的问题，并通过一个主动式调度器实现自适应 GC 触发，从而缓解资源争用。重要的是，LEGO 利用弹性增量 GC 机制来缓解面对突发请求时不可避免的 GC 所带来的干扰。我们使用流行的 Go 语言 LC 应用实现并评测了 LEGO。结果表明，与默认 Go GC 和经过定制的 G1 GC 相比，LEGO 显著提升了 LC 应用的尾延迟与吞吐量。

---

**版权与出版信息（Copyright and Publication Information）**

本文允许出于个人或课堂教学目的，免费对本文全部或部分内容进行数字或硬拷贝的复制与分发，前提是复制件不得用于营利或商业目的，且首份复制件须保留本声明及完整的引用信息。本文中由他人（而非作者本人）持有的版权必须予以尊重。允许在注明出处的前提下进行摘录。若要以其他方式复制、重新出版、上传至服务器或分发至邮件列表，则须事先获得特别许可和/或支付相应费用。许可申请请联系 permissions@acm.org。

- 会议（Venue）：HPDC '23, June 16–23, 2023, Orlando, FL, USA
- 版权（Copyright）：© 2023 Copyright held by the owner/author(s). Publication rights licensed to ACM.
- ISBN：ACM ISBN 979-8-4007-0155-9/23/06...$15.00
- DOI：https://doi.org/10.1145/3588195.3592998

---

## CCS 概念（CCS CONCEPTS）

> • Software and its engineering → Garbage collection; • Computer systems organization → Cloud computing.

• 软件及其工程 → 垃圾回收；• 计算机系统组织 → 云计算。

## 关键词（KEYWORDS）

> Golang (Go), Garbage Collection (GC), GC Scheduler, Latency Critical (LC), Tail Latency

Golang (Go)，垃圾回收（GC），GC 调度器，延迟关键（LC），尾延迟

## ACM 引用格式（ACM Reference Format）

> Junxian Zhao, Xiaobo Zhou, Sang-Yoon Chang, and Chengzhong Xu. 2023. Let It Go: Relieving
> Garbage Collection Pain for Latency Critical Applications in Golang. In Proceedings of the 32nd
> International Symposium on High-Performance Parallel and Distributed Computing (HPDC '23),
> June 16–23, 2023, Orlando, FL, USA. ACM, New York, NY, USA, 12 pages.
> https://doi.org/10.1145/3588195.3592998

Junxian Zhao, Xiaobo Zhou, Sang-Yoon Chang, and Chengzhong Xu. 2023. Let It Go: Relieving Garbage Collection Pain for Latency Critical Applications in Golang. 载于第 32 届高性能并行与分布式计算国际研讨会论文集（In Proceedings of the 32nd International Symposium on High-Performance Parallel and Distributed Computing, HPDC '23），2023 年 6 月 16–23 日，美国佛罗里达州奥兰多。ACM，美国纽约，12 页。https://doi.org/10.1145/3588195.3592998

## 1 引言（Introduction）

> Cloud computing has achieved remarkable success over the last decade due to features including
> resource sharing, flexibility, and scalability. Cloud systems also require fast and safe concurrency
> provided by programming languages. This is where Golang (a.k.a Go) comes for, which is also
> known as Language of Cloud. Designed for high concurrency, Go emphasizes concurrent
> programming for cloud systems. It is being used to build popular cloud applications [5–7, 41]
> at a massive scale.

过去十年间，云计算凭借资源共享、灵活性与可扩展性等特性取得了显著成功。云系统还要求编程语言提供快速且安全的并发能力。这正是 Golang（又称 Go）的用武之地，它也被称为“云的语言”（Language of Cloud）。Go 为高并发而设计，强调面向云系统的并发编程。它正被用于大规模构建流行的云应用 [5–7, 41]。

> As many modern programming languages, Go is an auto-memory management language. Instead
> of explicitly removing obsolete objects, it deploys GC for memory management. GC is executed
> automatically in background to reclaim memory for the runtime heap. During GC runtime, it
> may pause the entire application, a phenomenon known as Stop The World (STW) [18] in other
> languages such as Java.

与许多现代编程语言一样，Go 是一种自动内存管理语言。它并非显式地移除废弃对象，而是部署 GC 来进行内存管理。GC 在后台自动执行，为运行时堆回收内存。在 GC 运行期间，它可能会暂停整个应用，这一现象在 Java 等其他语言中被称为 Stop-The-World（STW，全停顿）[18]。

> Motivation. Go’s unique GPM model is designed to improve thread-switching performance.
> Different from the state-of-the-art GCs, such as G1 GC [24], ZGC [20] and Shenandoah GC [19]
> for the Java Virtual Machine (JVM), Go employs a concurrent, non-generational, and
> non-incremental GC. This corresponds to less STW overhead, a runtime heap layout without
> distinction between young and old objects, and the ability to run GC on the partial memory heap.
> Go also leverages the TCmalloc memory allocator [26], which allocates objects at the thread
> level, whereas JVM allocates objects from a universal heap when the thread-level stack buffer is
> full. Furthermore, Go GC is non-copy-and-compact. These features make existing GC
> optimizations for JVM not suitable or inefficient for Go.

动机。Go 独特的 GPM 模型旨在提升线程切换性能。与 Java 虚拟机（JVM）上最先进的 GC（如 G1 GC [24]、ZGC [20] 和 Shenandoah GC [19]）不同，Go 采用的是并发、非分代且非增量的 GC。这对应于更低的 STW 开销、不区分年轻对象与老年代对象的运行时堆布局，以及在部分内存堆上运行 GC 的能力。Go 还利用了 TCmalloc 内存分配器 [26]，它在线程粒度上分配对象，而 JVM 在线程级栈缓冲（stack buffer）耗尽时从全局堆中分配对象。此外，Go GC 是非复制-压缩式（non-copy-and-compact）的。这些特性使得针对 JVM 的现有 GC 优化对 Go 并不适用或效率低下。

> We find Go applications often experience significant latency, particularly tail latency in face of
> CPU interference and bursty requests. GC threads interfere with non-GC threads (mutators),
> which results in performance degradation of latency-critical (LC) applications. Our experiments
> show that GC interference on LC applications such as key-value stores developed in Go can
> cause up to 4× increase of the 99th percentile (p99) query latency compared to when the
> key-value stores are running without GC. The tail latency becomes more severe when the input
> requests are bursty. Even worse, our experiments find that GC is more likely (more than 2×) to
> be triggered in face of bursty requests.

我们发现，面对 CPU 干扰与突发请求时，Go 应用常常经历显著的延迟，尤其是尾延迟。GC 线程会干扰非 GC 线程（mutator），从而导致延迟关键（LC）应用的性能退化。我们的实验表明，对于用 Go 开发的键值存储等 LC 应用，GC 干扰可使第 99 百分位（p99）查询延迟相比这些键值存储无 GC 运行时增加最多 4 倍。当输入请求具有突发性时，尾延迟会变得更加严重。更糟糕的是，我们的实验发现，面对突发请求时 GC 更有可能（超过 2 倍）被触发。

> Therefore, Go demands for a new GC mechanism and a scheduler that take into account
> interference of resource usage introduced by mutators of the concurrent GC. Specifically, Go GC
> should satisfy the following requirements for maintaining service level objective (SLO) of LC
> applications.

因此，Go 需要一种新的 GC 机制与调度器，将并发 GC 的 mutator 所引入的资源使用干扰纳入考量。具体而言，为维持 LC 应用的服务等级目标（SLO），Go GC 应满足以下要求。

> • LC applications should maintain low tail latency for meeting its SLO objectives.

• LC 应用应保持较低的尾延迟，以满足其 SLO 目标。

> • The new GC should reduce its interference on mutators of the application, especially upon
> bursty requests.

• 新的 GC 应减少对应用 mutator 的干扰，尤其是在突发请求时。

> • The new GC needs to be transparent and generally applicable to all applications developed in
> Go. Thus, the source code of applications need not be modified.

• 新的 GC 需要是透明的，并且普遍适用于所有用 Go 开发的应用。因此，应用的源代码无需修改。

> Limitation of state-of-the-art approaches. There are work in memory management focusing on
> GC optimization, mostly in JVM. Previous works [32, 37, 38, 51] introduced new algorithms and
> modified heap layout to reduce GC overhead. Recent works [28, 50] propose to schedule with
> fine-grained hardware resources, i.e. CPU and memory, for different threads in GC. There are
> also studies [44, 45] that focus on GC on far-memory and resource disaggregated systems.
> However, due to the semantic gap between JVM and Go designs, most of the existing
> optimizations are not available or directly applicable for Go. Importantly, the existing works
> lack runtime adaptivity for incremental GC design.

现有方法的局限。内存管理领域已有大量聚焦于 GC 优化的工作，且大多针对 JVM。先前的工作 [32, 37, 38, 51] 引入了新算法并修改堆布局，以降低 GC 开销。近期的工作 [28, 50] 提出为 GC 中的不同线程以细粒度硬件资源（即 CPU 和内存）进行调度。也有研究 [44, 45] 关注远端内存（far-memory）与资源解耦系统上的 GC。然而，由于 JVM 与 Go 设计之间存在语义鸿沟，现有优化大多对 Go 不可用或无法直接适用。重要的是，现有工作缺乏针对增量 GC 设计的运行时自适应能力。

> Challenge I - pre-fixed incremental volume. The production G1 GC deploys an incremental GC
> mechanism for JVM, where the GC pause time and reclaimed memory volume are pre-fixed when
> an application is started up or during a non-GC period. However, G1 GC cannot elastically
> decide the volume of the current GC during GC runtime. In face of bursty requests, GC activities
> dramatically increase the latency of an application violating its SLO. This is unavoidable in the
> design of the current incremental GC mechanism.

挑战 I——预设的增量规模。生产环境中的 G1 GC 为 JVM 部署了增量 GC 机制，其 GC 暂停时间与回收内存量在应用启动时或非 GC 期间即被预先固定。然而，G1 GC 无法在 GC 运行期间弹性地决定当次 GC 的规模。面对突发请求时，GC 活动会急剧增加应用延迟，从而违背其 SLO。这在当前增量 GC 机制的设计中是不可避免的。

> Challenge II - passive GC triggering. The state-of-the-art GCs (such as G1 GC, ZGC and
> Shenandoah GC) only trigger GC progress early enough to avoid Out Of Memory (OOM) errors.
> They do not consider the resource usage of GC progress at runtime. The default GC in Go limits
> the total CPU time usage of GC progress to a 30% threshold, which does not delicately match the
> runtime CPU usage. GC progress tends to compete CPU resources with mutators dynamically.
> However, the Go scheduler triggers GC even when the CPU usage is higher than 70%, which
> can lead to severe performance degradation in latency-critical applications.

挑战 II——被动的 GC 触发。最先进的 GC（如 G1 GC、ZGC 和 Shenandoah GC）只会在为避免内存不足（Out Of Memory, OOM）错误所需的时间点之前触发 GC 进展。它们并不考虑 GC 进展在运行时的资源使用情况。Go 中的默认 GC 将 GC 进展的总 CPU 时间占用限制在 30% 的阈值，这与运行时的 CPU 使用情况并不精细匹配。GC 进展倾向于与 mutator 动态争抢 CPU 资源。然而，Go 调度器即便在 CPU 使用率高于 70% 时仍会触发 GC，这可能在延迟关键应用中导致严重的性能退化。

> Challenge III - integration into Go. The existing GC optimizations are difficult to be directly
> transplanted into Go, as Go employs a different memory layout, i.e., GPM model, and it uses
> TCmalloc and non-copy-and-compact GC. Currently, the default Go GC only supports full GC.

挑战 III——集成到 Go 中。现有 GC 优化难以直接移植到 Go，因为 Go 采用了不同的内存布局（即 GPM 模型），并且使用 TCmalloc 与非复制-压缩式 GC。目前，默认的 Go GC 仅支持全量 GC。

> Key insights and contributions. In this paper, we present an efficient, non-intrusive GC
> scheduling framework, LEGO that deploys a novel elastic incremental GC mechanism integrated
> with an adaptive GC scheduler to reduce CPU contention between GC and mutators, and improve
> resource utilization and QoS of LC applications developed in Go.

关键洞见与贡献。本文提出一个高效、非侵入式的 GC 调度框架 LEGO，它部署了一种新颖的弹性增量 GC 机制，并与自适应 GC 调度器相结合，以降低 GC 与 mutator 之间的 CPU 争用，并提升用 Go 开发的 LC 应用的资源利用率与 QoS。

> To address the first two challenges (pre-fixed incremental volume and passive GC triggering),
> LEGO designs a unique adaptive incremental GC mechanism that proactively triggers GC and
> dynamically tailors the incremental progress during GC according to runtime characteristics.
> Specifically, it obtains GC processing volume at runtime in face of detecting a burst of requests,
> and proactively triggers and adjusts the GC processing volume based on the available resources.
> To facilitate its adaptive incremental GC, LEGO designs Quad-Color marking algorithm for
> dynamically GC triggering.

为解决前两个挑战（预设的增量规模与被动的 GC 触发），LEGO 设计了一种独特的自适应增量 GC 机制，它主动触发 GC，并依据运行时特征在 GC 期间动态裁剪增量进度。具体而言，它在检测到请求突发时于运行时获取 GC 处理量，并基于可用资源主动触发与调整 GC 处理量。为支撑其自适应增量 GC，LEGO 设计了四色标记（Quad-Color marking）算法以动态触发 GC。

> To address the third challenge, LEGO adds new data structures (write barrier and P&G-level
> remember set) for recording object reference modifications and Goroutine migration during GC
> runtime. Furthermore, it adds a P-level reference counter to reduce the overhead of Goroutine
> migration and maintain the consistency of object reference during GC.

为解决第三个挑战，LEGO 新增了数据结构（写屏障以及 P 级与 G 级记忆集（P&G-level remember set）），用于记录 GC 运行期间的对象引用修改与 Goroutine 迁移。此外，它增加了 P 级引用计数器，以降低 Goroutine 迁移的开销并维持 GC 期间对象引用的一致性。

> This paper makes the following contributions.

本文做出了以下贡献。

> • We conduct in-depth characterization and analysis of Go GC interference on LC applications,
> especially its performance impact upon bursty requests. We apply a statistic approach that
> leverages on the obtained runtime characteristics for diagnosing GC interference.

• 我们对 Go GC 在 LC 应用上的干扰进行了深入刻画与分析，尤其是其在突发请求下的性能影响。我们采用一种统计方法，利用所获取的运行时特征来诊断 GC 干扰。

> • We design an elastic incremental GC mechanism with a proactive GC scheduler. Furthermore,
> we develop a Quad-Color marking algorithm for dynamically triggering the incremental GC
> mechanism.

• 我们设计了一种带有主动式 GC 调度器的弹性增量 GC 机制。此外，我们开发了四色标记算法，用于动态触发该增量 GC 机制。

> • We implement LEGO in Go runtime. It is transparent to applications and the OS. LEGO
> enables flexible and robust GC, which can be deployed in any mark-and-sweep GC system
> including JVM GC.

• 我们在 Go 运行时中实现了 LEGO。它对应用与操作系统均是透明的。LEGO 实现了灵活且鲁棒的 GC，可部署于任何标记-清扫式 GC 系统，包括 JVM GC。

> We evaluate LEGO with popular Go-developed cloud applications (Badger, Bolt, Rosedb, and
> TiDB) and benchmark workloads. Experiment results show that LEGO can reduce the average
> 99th percentile latency up to 35.6% and 42.7%, and improve the system throughput by 33% and
> 42% compared to the default Go GC and a tailored G1 GC, respectively. LEGO software artifact
> is available at https://github.com/lotusknight/LEGO.

我们使用流行的 Go 云应用（Badger、Bolt、Rosedb 和 TiDB）与基准负载对 LEGO 进行了评测。实验结果表明，与默认 Go GC 和经过定制的 G1 GC 相比，LEGO 可分别将平均第 99 百分位延迟最多降低 35.6% 与 42.7%，并将系统吞吐量分别提升 33% 与 42%。LEGO 的软件产物可在 https://github.com/lotusknight/LEGO 获取。

> Limitations of the proposed approach. LEGO considers LC applications that have bursty
> requests. It reschedules GC with transiently available system resources. It is possible though
> uncommon that an LC application receives consistent high volume of traffic. In this scenario, GC
> may be triggered frequently or even continuously. LEGO would be less or ineffective compared
> with the default Go GC. Note LEGO can be transiently switched to the default Go GC. Currently,
> LEGO supports Go programs. Its principle and design can be applied to any programming
> language runtimes including JVM that employ automatic memory management.

所提方法的局限性。LEGO 面向存在突发请求的 LC 应用。它利用短暂可用的系统资源重新调度 GC。LC 应用也可能（尽管不常见）接收到持续高流量的请求。在这种场景下，GC 可能被频繁甚至持续触发，LEGO 相比默认 Go GC 效果会更差甚至无效。请注意，LEGO 可以临时切换回默认 Go GC。目前，LEGO 支持 Go 程序。其原理与设计可应用于任何采用自动内存管理的编程语言运行时，包括 JVM。

> In the rest of the paper, Section 2 presents background and motivational studies. Sections 3 and
> 4 describe LEGO design and implementation. LEGO is evaluated in Section 5. We discuss related
> work in Section 6 and conclude in Section 7.

本文其余部分的组织如下：第 2 节介绍背景与动机研究。第 3 节和第 4 节描述 LEGO 的设计与实现。第 5 节对 LEGO 进行评测。第 6 节讨论相关工作，第 7 节进行总结。



## 2 背景与动机（Background and Motivation）

### 2.1 Go 线程调度与堆布局（Go Thread Scheduling and Heap Layout）

> Thread scheduling. Go introduces a unique GPM model as abstracts for flexible and low-overhead
> runtime thread management and concurrent programming. Figure 1 shows the abstract model.
> G (Goroutine) is the basic execution unit in Go, which is detached from an OS-level thread to
> reduce thread switching overhead. Different Gs run in parallel during runtime. Similar to a
> process in Linux, each G has execution states such as *runnable* and *running*.
> The solid gray G represents the current running Goroutine, while
> a white blank G is not being executed by any thread. Multiple Ms
> represent OS-level threads. Each P (processor) must be attached
> to a M (machine). Go runtime can reschedule a P to a different M.
> Each P has a local queue for *runnable* Gs. By running on an M,
> P executes Gs from its *runnable* queue. Compared with runtime
> thread switching based on system calls, G switching is more flexible
> as P reuses the same OS thread (M) for running different Gs. With
> the GPM model, Go reduces overhead in creating and scheduling
> threads.

线程调度。Go 引入了独特的 GPM 模型，作为灵活且低开销的运行时线程管理与并发编程的抽象。图 1 给出了该抽象模型。G（Goroutine，Go 协程）是 Go 中的基本执行单元，它与操作系统级线程解绑，以降低线程切换开销。不同的 G 在运行时并行执行。与 Linux 中的进程类似，每个 G 都具有执行状态，例如可运行（*runnable*）和运行中（*running*）。实心灰色的 G 表示当前正在运行的 Goroutine，而空白的 G 则未被任何线程执行。多个 M 表示操作系统级线程。每个 P（processor，处理器）必须绑定到一个 M（machine，机器）之上。Go 运行时可以把 P 重新调度到不同的 M 上。每个 P 都有一个用于存放可运行 G 的本地队列。P 通过运行在某个 M 上，从其可运行队列中取出 G 执行。与基于系统调用的运行时线程切换相比，G 的切换更加灵活，因为 P 复用同一个 OS 线程（M）来运行不同的 G。借助 GPM 模型，Go 降低了创建和调度线程的开销。

**图 1：Go GPM 抽象模型（Fig. 1: Go GPM abstract model）** — 图中元素：M、P、G。

> Heap layout and memory management. Figure 2 shows the
> process of heap layout and memory management. Data allocation
> at runtime happens on the G stack and runtime heap. If data on
> the stack is shared by multiple Gs, it will be placed on the heap.
> Go deploys TCmalloc allocator to pre-allocate memory from heap
> for each P. Each P has its own cache, *mcache*, for different sizes of
> objects. Each allocation size set is called *mspan*. When a G performs
> memory allocation, it actually operates on the corresponding *mspan*
> from the current P. P requests more *mspan* from a set of *mspan* on
> *mcentral*. A *mcentral* is a data structure that holds a set of free
> memory blocks with different sizes. It is used for initializing a set
> of *mspan* with different sizes from the runtime heap.

堆布局与内存管理。图 2 展示了堆布局与内存管理的过程。运行时的数据分配发生在 G 栈和运行时堆上。如果栈上的数据被多个 G 共享，那么它会被放置到堆上。Go 采用 TCmalloc 分配器，为每个 P 从堆中预分配内存。每个 P 都有自己的缓存 mcache，用于存放不同大小的对象。每一种分配大小集合称为 mspan。当 G 执行内存分配时，它实际上是在操作来自当前 P 的相应 mspan。P 会从 mcentral 上的一组 mspan 中申请更多的 mspan。mcentral 是一种数据结构，它持有一组大小不同的空闲内存块，用于从运行时堆中初始化一组不同大小的 mspan。

**图 2：Go 堆布局与内存管理（Fig. 2: Go heap layout and memory management）** — 图中元素：Runtime Heap、Span Sets、Span、Global、P-Level、G-Level、Object。

**表 1：各 GC 阶段的耗时（ms）（Table 1: Time spent on each GC phase (ms).）**

| Sweep Termination 清扫终止 (A) | Sweep Termination 清扫终止 (B) | Concurrent Mark 并发标记 (A) | Concurrent Mark 并发标记 (B) | Mark Termination 标记终止 (A) | Mark Termination 标记终止 (B) |
| --- | --- | --- | --- | --- | --- |
| 0.09 | 0.09 | 28 | 0.58 | 0.005 | 0.005 |
| 0.10 | 0.10 | 27 | 0.58 | 0.008 | 0.008 |
| 0.11 | 0.10 | 32 | 0.57 | 0.017 | 0.017 |
| 0.13 | 0.12 | 23 | 0.60 | 0.011 | 0.01 |

### 2.2 垃圾回收（Garbage Collection, GC）

> Go GC semantics. Fig 3 shows four phases in Go GC, 1) Sweep
> termination (STW), 2) Mark phase (GCmark), 3) Mark termination
> (STW, GCmarktermination), and 4) Sweep phase (GCoff). In each
> GC cycle (phases 1 to 3), Go scans the whole heap and marks
> alive objects. Two phases (STWs and Concurrent Marking) are
> executed during this period. Concurrent Marking time is much
> longer (>100X) than STW time. During the Concurrent Marking
> phase, mutators are running in parallel with GC processes. Each
> allocation by a mutator is followed by a number of scan operations,
> *AllocateAssist*. After a complete GC cycle, in phase 4 Go sweeps and
> reallocates obsolete objects when a new object is allocated, which
> is also concurrent. Note Go does not deploy a generational GC
> mechanism, instead, the objects in runtime heap are non-movable.
> They are allocated in their final positions, which is different from
> the traditional GCs in JVM. Thus, JVM GCs cannot be directly
> transplanted to Go since they have different basic GC logics, i.e.
> JVM adopts copy-and-compact GC.

Go GC 语义。图 3 展示了 Go GC 的四个阶段：1）清扫终止（Sweep termination，STW）；2）标记阶段（Mark phase，GCmark）；3）标记终止（Mark termination，STW，GCmarktermination）；4）清扫阶段（Sweep phase，GCoff）。在每一个 GC 周期（阶段 1 至 3）中，Go 扫描整个堆并标记存活对象。该期间会执行两个阶段（Stop-The-World（STW，全停顿）与并发标记）。并发标记的时间远长于 STW 时间（>100 倍）。在并发标记阶段，mutator（应用线程）与 GC 过程并行运行。mutator 的每一次分配都会伴随若干次扫描操作，即 *AllocateAssist*。在一个完整的 GC 周期之后，Go 在阶段 4 中清扫，并在分配新对象时重新分配已废弃的对象，该过程同样是并发的。需要注意的是，Go 并未采用分代 GC 机制；相反，运行时堆中的对象是不可移动的。它们被分配在最终的位置上，这与 JVM 中的传统 GC 不同。因此，由于基本的 GC 逻辑不同（即 JVM 采用复制-整理（copy-and-compact）式 GC），JVM GC 无法直接移植到 Go 中。

> Tri-color marking. All tracing-based GCs use tri-color marking
> algorithm [10]. It traces reachable objects from RootSets. When the
> marking phase starts, it puts all objects in RootSets to a list which
> grays them. GC worker threads fetch objects from the list and trace
> pointers from the objects. Objects with untraced pointers are grayed
> and put into the list recursively. Other reachable objects are marked
> black. Three colors, white, gray and black, represent different states
> of marking: 1) a white object is not currently reachable, 2), a black
> object is reachable and all its pointers are traced, 3) a gray object is
> marked, while it has pointers not exhausted by scan worker threads.
> Gray objects are put in a work list for further tracing. When the list
> of gray objects is empty, the Concurrent Marking phase is finished.
> The white objects left are unreachable and can be swept in the later
> phase.

三色标记。所有基于追踪（tracing）的 GC 都使用三色标记算法 [10]。它从 RootSets 出发追踪可达对象。当标记阶段开始时，它把 RootSets 中的所有对象放入一个列表，从而将它们置为灰色。GC 工作线程从该列表中取出对象，并追踪对象中的指针。带有未被追踪指针的对象会被置为灰色，并递归地放入该列表。其他可达对象被标记为黑色。白、灰、黑三种颜色代表不同的标记状态：1）白色对象当前不可达；2）黑色对象可达，且其所有指针都已被追踪；3）灰色对象已被标记，但仍存在未被扫描工作线程处理完的指针。灰色对象被放入一个工作列表以便进一步追踪。当灰色对象列表为空时，并发标记阶段就结束了。剩下的白色对象不可达，可以在后续阶段被清扫。

### 2.3 GC 特征刻画与分析（GC Characterization and Analysis）

> CPU interference. Discord developer logs give the CPU usage
> and response latency from a Discord chatting server written in
> Go [1]. There are peaks of both CPU usage and response time
> every 2 minutes when Go GC is triggered. By testing with different
> configurations, it turns out that concurrent Marking of Go GC
> costs significant hardware resources and leads to interference on
> mutators.

CPU 干扰。Discord 开发者日志给出了一个用 Go 编写的 Discord 聊天服务器的 CPU 使用率与响应延迟 [1]。每当 Go GC 被触发时，CPU 使用率与响应时间每 2 分钟就会出现一次峰值。通过在不同配置下进行测试，结果表明 Go GC 的并发标记会消耗大量硬件资源，并对 mutator 造成干扰。

**图 3：Go GC 在一次执行时间内的四个阶段周期（Fig. 3: Go GC cycles with four phases during execution time.）**

图中标注：GC cycle（GC 周期）、non-GC period（非 GC 阶段）、GC period（GC 阶段）、execution time（执行时间）；图例：1 Sweep termination（清扫终止）、2 Mark phase (GC period)（标记阶段（GC 阶段））、3 Mark termination（标记终止）、4 Sweep phase (non-GC period)（清扫阶段（非 GC 阶段））。

> Reference impact. We build two microbenchmarks (A and B)
> using Go. Microbenchmark A stores a map with key (integer) and
> value (a reference to structure). The time spent on four GC cycles are
> 28.095ms, 27.108ms, 32.127ms, and 23.141 ms, respectively. For each
> cycle, we show the time elapse of each phase in Table 1. Obviously,
> the concurrent marking phase dominates the overall GC overhead.
> Microbenchmark B changes the value to integer. Go GC ignores
> scalar (int, float) when tracing references during marking. The time
> spent on 4 GC cycles are 0.675ms, 0.688ms, 0.687ms, and 0.73ms,
> respectively, much smaller than those in A. We can observe that the
> overhead of concurrent marking is severely impacted by the number
> of references. To mitigate the overhead in Goroutine immigration,
> we reduce the amount of metadata for recording references between
> different Goroutines by introducing a P-level bit mask technique.

引用的影响。我们用 Go 构建了两个微基准测试（A 与 B）。微基准测试 A 存储一个 map，其键为整数、值为（指向结构体的引用）。四个 GC 周期所耗的时间分别为 28.095ms、27.108ms、32.127ms 和 23.141 ms。对于每个周期，我们在表 1 中给出了各阶段的耗时。显然，并发标记阶段在整体 GC 开销中占主导地位。微基准测试 B 将值改为整数。Go GC 在标记阶段追踪引用时会忽略标量（int、float）。其 4 个 GC 周期的耗时分别为 0.675ms、0.688ms、0.687ms 和 0.73ms，远小于 A 中的耗时。我们可以观察到，并发标记的开销严重受到引用数量的影响。为降低 Goroutine 迁移（Goroutine immigration）带来的开销，我们通过引入 P 级位掩码（bit mask）技术，减少了用于记录不同 Goroutine 之间引用的元数据量。

> Memory demand and usage. Both CPU usage and memory de-
> mand of an application fluctuate at runtime. Here, we use Poisson
> distribution to generate random input volume. Case studies are
> executed on Badger [4], a key-value (KV) store written in Go. Fig-
> ure 4 shows the input volume and total memory usage per second.
> The vertical green lines show that the memory usage drops sharply
> each time GC is triggered, because only little data are reused for
> the KV store. The memory demand in each second consists of the
> basic memory usage of the KV store (about 200MB) and the input
> volume, which is pretty low compared to the total memory usage.
> Though the default Go GC triggering avoids OOM errors, it does
> not consider CPU interference when GC is triggered, which often
> causes high tail latency for LC applications. As the memory de-
> mand fluctuates from time to time, it is unnecessary to trigger GC
> as late as possible. The memory demand is not always beyond a
> high threshold, which allows us to trigger GC dynamically as the
> full GC is not required. We leverage the opportunities to proactively
> trigger GC and thus avoid CPU interference in face of a burst of
> requests.

内存需求与使用。应用程序的 CPU 使用率和内存需求在运行时都会波动。这里，我们使用泊松分布来生成随机的输入量。案例研究在 Badger [4] 上执行，Badger 是一个用 Go 编写的键值（KV）存储。图 4 展示了每秒的输入量与总内存使用量。竖直的绿线表明，每次 GC 被触发时内存使用量都会急剧下降，因为该 KV 存储复用的数据很少。每一秒的内存需求由 KV 存储的基础内存使用量（约 200MB）和输入量构成，而后者与总内存使用量相比相当低。尽管 Go 默认的 GC 触发机制能够避免 OOM 错误，但它在触发 GC 时并未考虑 CPU 干扰，这往往会给延迟关键（LC）应用带来高尾延迟。由于内存需求时刻波动，因此并无必要尽可能晚地触发 GC。内存需求并非始终高于某个高阈值，这让我们可以动态地触发 GC，因为并不需要完整 GC。我们利用这些机会主动触发 GC，从而在面对突发请求时避免 CPU 干扰。

### 2.4 案例研究（Case Studies）

> We consider two different settings of LC application Badger, 1) input
> stable requests at the server maximum capacity 5 * 10^4 request per
> second (RPS), and 2) input bursty requests following a Poisson
> distribution. In each interval (one second), Badger sends *n* requests
> where *n* obeys a Poisson distribution (λ = 5 * 10^4). We use the
> second setup to emulate a real-world scenario. The server setting
> is identical to that in Section 5.

我们考虑 LC 应用 Badger 的两种不同设置：1）输入为稳定请求，速率为服务器最大容量每秒 5 * 10^4 个请求（RPS）；2）输入为遵循泊松分布的突发请求。在每个时间区间（1 秒）内，Badger 发送 *n* 个请求，其中 *n* 服从泊松分布（λ = 5 * 10^4）。我们使用第二种设置来模拟真实场景。服务器设置与第 5 节相同。

**图 4：Badger 在随机输入量下的内存需求与实际使用量；三处急剧下降由 GC 触发导致。堆大小设为 2,400 MB（Fig. 4: Memory demand and actual usage for Badger with Random input volume. Three sharp declines are due to GC Triggering. Heap size is set to 2,400 MB.）**

坐标轴与图例：X 轴 Time (sec)（时间，秒），刻度 0–90；左 Y 轴 Input Volume (MB) / Input Volume (MB/s)（输入量，MB / MB/s），刻度 0–350；右 Y 轴 Memory Usage (MB)（内存使用量，MB），刻度 500–2000；图例：GC（GC 触发点）、Input Volume (MB/s)（输入量）、Memory Usage (MB)（内存使用量）。

**图 5：静态输入量下的 P99 尾延迟（Fig. 5: P99 tail latency under static input volume.）**

坐标轴与图例：X 轴 Execution Time (sec)（执行时间，秒），刻度 0–60；Y 轴 Latency (milisec)（延迟，毫秒；原文拼写为 milisec），刻度 0–200；图例：GC（GC 触发点）、p99 tail latency（p99 尾延迟）。

**图 6：GC 触发会影响输入量（Fig. 6: GC triggering affects the input volume.）**

坐标轴与图例：X 轴 Time (sec)（时间，秒），刻度 0–600；Y 轴 Input Volume (K)（输入量，K），刻度 0–300；图例：GC Trigger w/ Input Vol.（随输入量变化的 GC 触发）、Input Vol.（输入量）。

**表 2：GC 触发时机与输入量的统计（Table 2: Statistics of GC trigger timing with input volume.）**

| | Input Volume 输入量 <50,000 | Input Volume 输入量 >=50,000 |
| --- | --- | --- |
| GC # GC 触发次数 | 67 (29.9%) | 157 (70.1%) |
| Input # 输入次数 | 3893 (64.2%) | 2167 (35.8%) |

> Figure 5 depicts the 99th tail latency due to setting 1. Each time
> Go GC is triggered, the tail latency increases to more than 4X of the
> average p99 tail latency. As Go GC runs concurrently with mutators,
> it competes CPU resources and prolongs the application response.
> While the average CPU usage is only around 40% without GC, the
> CPU usage is more than 90% upon GC triggering. Though the GC
> pause and concurrent marking time are pretty small, the overall
> overhead of GC in the tail latency is nontrivial.

图 5 描绘了设置 1 下的第 99 百分位尾延迟。每当 Go GC 被触发时，尾延迟会升高到平均 p99 尾延迟的 4 倍以上。由于 Go GC 与 mutator 并发运行，它会争抢 CPU 资源并延长应用的响应时间。在不发生 GC 时平均 CPU 使用率仅为约 40%，而在 GC 触发时 CPU 使用率超过 90%。尽管 GC 暂停时间和并发标记时间都相当小，但 GC 对尾延迟的整体开销却不容忽视。

> After learning the impact of CPU interference due to GC, we
> examine how the input volume affects GC trigger timing in setting 2.
> We run the case study for 100 minutes and record the real-time RPS
> and timing of GC triggering. Figure 6 shows the results in random
> 10 minutes. The y-axis of blue lines shows the RPS value when GCs
> are triggered (red darts). The orange dash line (y=50K) represents
> the server maximum capacity. If the RPS is higher than 50K, the LC
> application cannot reach its SLO due to the bursty traffic condition.
> We can clearly see that most of red darts are scattered above the
> orange dashed line. We collect the data of the whole 100 minutes
> experiment and show the statistics in Table 2.

在了解了 GC 导致 CPU 干扰的影响之后，我们在设置 2 中考察输入量如何影响 GC 的触发时机。我们运行该案例研究 100 分钟，并记录实时的 RPS 与 GC 触发时机。图 6 展示了随机选取的 10 分钟内的结果。蓝色折线的纵轴显示 GC 被触发（红色箭头）时的 RPS 值。橙色虚线（y=50K）代表服务器的最大容量。如果 RPS 高于 50K，则该 LC 应用因突发流量而无法达到其 SLO。我们可以清楚地看到，大多数红色箭头散布在橙色虚线之上。我们收集了整个 100 分钟实验的数据，并在表 2 中给出统计结果。

> Table 2 shows that more than 70% of GCs are triggered upon
> large input volumes, while large input volumes only occupy 36% of
> all input volumes. The results show that GC tends to be triggered
> when a burst of requests arrives.

表 2 表明，超过 70% 的 GC 是在大输入量下触发的，而大输入量仅占全部输入量的 36%。结果表明，GC 倾向于在一批请求突发到达时被触发。

> We examine the interesting phenomenon and reveal that the
> current Go GC use a heap usage increment ratio mechanism to
> predict how early enough does it trigger GC to avoid OOM errors.
> Thus, a burst of requests represents a higher rate of heap usage
> increment ratio, which in return results in a higher probability
> of reaching the GC threshold since a large number of objects are
> allocated in the heap. As a result, more requests are involved when
> GC is running during a burst of requests, which leads to more
> significant tail latency.

我们考察了这一有趣的现象，并揭示出当前的 Go GC 使用堆使用量增长比例机制来预测它应当多早触发 GC 以避免 OOM 错误。因此，一批突发请求意味着更高的堆使用量增长比例，这反过来又使得达到 GC 阈值的概率更高，因为堆中分配了大量对象。结果是，当 GC 在请求突发期间运行时涉及更多请求，从而导致更显著的尾延迟。

> The state-of-the-art GCs, including both Go GC and JVM GCs,
> are triggered passively, whether in a fixed time interval or in a pre-
> dictive way to avoid OOM errors. Such a GC triggering mechanism
> is unable to address the high tail latency when a burst of requests
> arrives. To tackle the identified issues, we propose and develop
> LEGO, a proactive GC, currently implemented for Go GC, with
> different GC triggering policies. To mitigate the interference from
> GC, we design and develop an elastic incremental GC mechanism
> that minimizes the GC overhead on mutators.

当前最先进的各类 GC（包括 Go GC 与 JVM GC）都是被动触发的，无论是以固定时间间隔触发，还是以预测的方式触发以避免 OOM 错误。当一批请求突发到达时，这样的 GC 触发机制无法解决高尾延迟问题。为解决上述已识别的问题，我们提出并开发了 LEGO——一种主动式 GC，目前针对 Go GC 实现，具有不同的 GC 触发策略。为缓解来自 GC 的干扰，我们设计并实现了一种弹性增量 GC 机制，以将对 mutator 的 GC 开销降至最低。


## 3 LEGO 设计（LEGO Design）

> LEGO consists of two important components, an elastic incremental GC mechanism and an
> augmented Go GC scheduler. Figure 7 shows its design structure. The elastic incremental GC
> mechanism consists of an incremental GC mechanism and an elastic GC controller. The GC
> scheduler determines the timing and volume of GC, both before and during a GC period. It
> also decides which policy is used for GC triggering. The elastic incremental GC mechanism
> receives signals from the GC scheduler and executes the adaptive GC process.

LEGO 由两个重要组件构成：弹性增量 GC 机制（elastic incremental GC mechanism）与增强的 Go GC 调度器（augmented Go GC scheduler）。图 7 展示了其设计结构。弹性增量 GC 机制由增量 GC 机制与弹性 GC 控制器（elastic GC controller）组成。GC 调度器决定 GC 的时机与执行量，无论是在 GC 周期之前还是 GC 周期期间。它还决定采用哪种策略来触发 GC。弹性增量 GC 机制接收来自 GC 调度器的信号，并执行自适应的 GC 过程。

### 3.1 Go 增量 GC（Incremental Go GC）

> The current Go GC supports full GC only, which is triggered on the whole runtime heap and is
> unable to be paused or stopped during a GC period. LEGO features incremental Go GC,
> tailoring for its unique GPM model and memory allocator. To adapt incremental GC into Go,
> LEGO splits GC and non-GC objects. Go uses TCmalloc for memory allocation and allocates
> objects at the thread level. LEGO makes use of it to depart GC into the P level to minimize
> the modification in Go runtime. Without rescheduling of Goroutines, all Go routines in one P
> share the same P-level cache, mspan. LEGO triggers GC on P-level to minimize the number of
> cross-region references. However, Go scheduler reschedules Gs into idle Ps for load balancing,
> which asks for a specific solution to maintaining the validity of reference tracing during GC.
> Section 4 further illustrates how LEGO tackles the Goroutine immigration issue.

当前的 Go GC 仅支持 full GC，它作用于整个运行时堆，并且在 GC 周期内无法暂停或停止。LEGO 针对 Go 特有的 GPM 模型与内存分配器，实现了增量式 Go GC。为将增量 GC 适配到 Go 中，LEGO 将 GC 对象与非 GC 对象分离。Go 使用 TCmalloc 进行内存分配，并在线程级别分配对象。LEGO 利用这一点将 GC 下沉到 P 级别，以尽可能减少对 Go 运行时（runtime）的修改。在不发生 Goroutine 重新调度的情况下，同一个 P 中的所有 Go routine 共享同一份 P 级缓存 `mspan`。LEGO 在 P 级别触发 GC，以尽量减少跨区域引用的数量。然而，Go 调度器会为负载均衡把 G 重新调度到空闲的 P 上，这就需要一种专门的解决方案来在 GC 期间维持引用追踪的有效性。第 4 节将进一步说明 LEGO 如何处理 Goroutine 迁移问题。

### 3.2 弹性 GC 控制器（Elastic GC Controller）

> LEGO introduces the elastic GC controller for two goals, 1) using the incremental GC
> mechanism that executes controllable GCs within a shorter time period and involves less
> resource interference, 2) dynamically determining the volume of GC to be triggered in each
> GC cycle.

LEGO 引入弹性 GC 控制器是为了两个目标：1）利用增量 GC 机制，在更短的时间段内执行可控的 GC，并带来更少的资源干扰；2）动态确定每个 GC 周期中需要触发的 GC 量。

> The second goal is crucial in tackling bursty requests. The elastic incremental GC grants
> LEGO the capability to adaptively execute GC based on the system runtime characteristics. It
> can shorten and stop an ongoing GC when a burst of requests arrives, and execute a full GC
> when CPU resource usage is not intense. However, the state-of-the-art GC marking algorithm,
> tri-color, is unable to dynamically determine the volume of incremental GC during a GC
> period. The tri-color marking algorithm can only distinguish marked, unmarked, and ongoing
> objects during a GC period. The two states, scannable (all objects involved in the chosen
> heap area during a GC period) and unscannable (vice versa), are pre-fixed in all
> state-of-the-art GCs. It is impossible to switch the state of unscannable objects to the
> state of scannable objects. To adaptively decide the volume of GC during a GC period, LEGO
> extends the tri-color marking algorithm with the fourth color.

第二个目标对于应对突发请求至关重要。弹性增量 GC 赋予 LEGO 依据系统运行时特征自适应执行 GC 的能力。当一批突发请求到来时，它可以缩短并停止正在进行的 GC；而当 CPU 资源使用不紧张时，它可以执行一次完整的 full GC。然而，当前最先进的 GC 标记算法——三色（tri-color）标记算法——无法在 GC 周期内动态确定增量 GC 的量。三色标记算法在 GC 周期内只能区分 marked、unmarked 与 ongoing 三类对象。而 scannable（在 GC 周期内被选中的堆区域所涉及的所有对象）与 unscannable（反之）这两种状态，在所有最先进的 GC 中都是预先固定的。无法把 unscannable 对象的状态切换为 scannable 对象的状态。为了在 GC 周期内自适应地决定 GC 的量，LEGO 在三色标记算法的基础上扩展出第四种颜色。

### 3.3 四色算法（Quad-Color Algorithm）

> To support the elastic incremental GC mechanism, we introduce a new Quad-Color algorithm
> during GC mark phase. It is used for the elastic GC controller. Figure 8 shows the Quad-Color
> state transform workflow.

为支持弹性增量 GC 机制，我们在 GC 标记（mark）阶段引入了一种新的四色（Quad-Color）算法。该算法服务于弹性 GC 控制器。图 8 展示了四色状态转换流程。

> We introduce several terminologies. ∼T is the collection of Ps not included in the current
> scanned or scanning Ps, while T is the collection of Ps included in the current scanned or
> scanning Ps. Set {∼T + T} contains all Ps of the application. When GC starts, GC controller
> picks Ps from the P Queue, ∼T. All Ps in the queue are ranked by the GC scheduler, the
> earlier with the higher priority.

我们引入若干术语。∼T 是当前不属于 scanned 或 scanning 的 P 所构成的集合，而 T 是当前属于 scanned 或 scanning 的 P 所构成的集合。集合 {∼T + T} 包含应用的全部 P。当 GC 启动时，GC 控制器从 P 队列 ∼T 中选取 P。队列中的所有 P 都由 GC 调度器排序，越靠前的优先级越高。

> LEGO introduces the fourth color, light gray, extending on the original tri-color algorithm
> (white, gray, and black). The light gray color is used for objects which should not be deleted
> when a GC period is fully finished or stopped ahead of time.

LEGO 在原有三色算法（white、gray 与 black）的基础上扩展出第四种颜色：light gray（浅灰）。light gray 用于标记那些在 GC 周期完整结束或被提前停止时不应被删除的对象。

**图 7：LEGO 的设计结构（灰色：模块；白色：数据结构）（Fig. 7: The LEGO design structure. (Gray: Modules; White: Data Structures)）**

> GC Scheduler / Trigger GC / Ps Queue / Dequeue Ps
> Choose & Rank Ps (Before GC), Add Ps (During GC) / Stop GC
> Monitor / Run-time Characteristics / CPU / Memory
> P-level Span List / P-Bit Mask compare / G-level Rem Set / Heap Increment / Request
> Data Flow / Control Flow
> Elastic GC Controller / Incremental GC Mechanism

（图内标注：GC 调度器 / 触发 GC / P 队列 / P 出队；选择并排序 P（GC 前）、添加 P（GC 期间）/ 停止 GC；监控器 / 运行时特征 / CPU / 内存；P 级 Span 列表 / P 位掩码比较 / G 级 Rem Set / 堆增量 / 请求；数据流 / 控制流；弹性 GC 控制器 / 增量 GC 机制。）

**图 8：四色状态转换流程（Fig. 8: Quad-Color state transform workflow）**

> White / Gray / Gray / White
> Gray / Black / White / Light Gray
> Add new P, Object Referenced By it
> As normal / As normal / As normal
> Partially Mark / Totally Mark
> When finish, move all
> Start & Add P, Object Referenced by ∼T ?  No / Yes

（图内标注：白 / 灰 / 灰 / 白；灰 / 黑 / 白 / 浅灰；新增 P 及其所引用的对象；按常规处理 / 按常规处理 / 按常规处理；部分标记（Partially Mark）/ 完全标记（Totally Mark）；完成时全部移动；开始并添加 P：对象是否被 ∼T 引用？ 否 / 是。注：PDF 提取时图中标注顺序存在错乱，此处按原样保留并给出中文对照。）

> When new Ps are added to T each time, LEGO does a comparison with the previous T for objects
> in the Ps. To do the comparison, remember sets are used for recording cross-P references. The
> process is described in Section 4. Objects are colored light gray only if they are referenced
> by Ps in ∼T. Following the policies below, light-grayed objects may be transformed into
> different colors.

每当有新的 P 加入 T 时，LEGO 都会将这些 P 中的对象与先前的 T 进行比较。为完成该比较，使用 remember set 记录跨 P 引用。该过程在第 4 节中描述。仅当对象被 ∼T 中的 P 引用时，才会被着色为 light gray。按照以下策略，被着为 light gray 的对象可能转换为不同的颜色。

> - white: an object is only referenced by new T after new Ps are added.

- white：对象在新 P 加入之后仅被新的 T 引用。

> - black: an object is marked and all its references are marked.

- black：对象已被标记，且其所有引用也已被标记。

> - gray: an object is marked but its references are not marked, or the current mark phase is
>   finished.

- gray：对象已被标记但其引用尚未标记，或者当前标记阶段已经结束。

> If an object is not referenced by the current T, it is still uncertain whether it might be
> used by ∼T. The elastic GC controller fades a light-grayed object to a white one only when
> the object is not referenced by ∼T. The fourth color empowers LEGO with the capability to
> dynamically terminate a GC period for the LC application.

如果一个对象未被当前的 T 引用，仍不能确定它是否可能被 ∼T 使用。只有当对象不被 ∼T 引用时，弹性 GC 控制器才会把一个 light gray 对象褪为 white。第四种颜色赋予 LEGO 为 LC 应用动态终止 GC 周期的能力。

### 3.4 GC 调度器（GC Scheduler）

> The GC scheduler fetches runtime characteristics from the monitor, including the CPU usage,
> heap usage, and heap increment ratio of the application. Based on the runtime
> characteristics, the scheduler determines the timing and the volume of GC triggering. We
> consider three scheduling policies.

GC 调度器从监控器获取运行时特征，包括应用的 CPU 使用率、堆使用量与堆增量比。基于运行时特征，调度器确定 GC 触发的时机与触发量。我们考虑了三种调度策略。

> - **A - Earliest First.** The GC scheduler triggers GC as early as possible when CPU usage is
>   below a threshold. Policy A increases the probability of triggering GC at low CPU usages,
>   which can be favorable for a LC application with bursty requests. However, the policy may
>   also lead to a high GC frequency when the application is under low workload. To reduce the
>   frequency of unnecessary GCs, a minimum heap size threshold is introduced for GC
>   triggering.

- **A - 最早优先（Earliest First）。** 当 CPU 使用率低于某个阈值时，GC 调度器会尽可能早地触发 GC。策略 A 提高了在低 CPU 使用率时触发 GC 的概率，这对具有突发请求的 LC 应用是有利的。然而，当应用处于低负载时，该策略也可能导致较高的 GC 频率。为降低不必要 GC 的频率，引入了最小堆大小阈值作为 GC 触发的条件。

> - **B - Default Augmented.** The GC scheduler triggers GC when the heap size reaches a
>   similar threshold as in the default Go GC. It makes sure that the threshold can avoid OOM
>   errors. This policy compromises the CPU usage on GC and the tail latency of the
>   application.

- **B - 默认增强（Default Augmented）。** 当堆大小达到与默认 Go GC 相近的阈值时，GC 调度器触发 GC。它确保该阈值能够避免 OOM 错误。该策略在 GC 的 CPU 使用开销与应用的尾延迟之间做出折中。

> - **C - Predictive.** GC scheduler uses ARIMA [34] model to predict the input pattern and
>   trigger GC under low request pressure. With correct predictions, this policy can offer the
>   best performance among all three policies. However, the policy also requires training
>   datasets in prior for each running application.

- **C - 预测式（Predictive）。** GC 调度器使用 ARIMA [34] 模型预测输入模式，并在请求压力较低时触发 GC。在预测正确的情况下，该策略在三种策略中能提供最佳性能。然而，该策略还需要为每个正在运行的应用预先准备训练数据集。

> **GC volume determination.** The coordination of GC volume with the quad-color algorithm is
> achieved by monitoring runtime CPU usage during GC and limiting the number of Ps
> accordingly. This ensures that the CPU is not fully occupied by GC threads with the assigned
> Ps. Furthermore, the GC scheduler monitors runtime CPU usage to determine whether to add more
> Ps into T during the current GC period. In the first two policies, the scheduler naively
> references runtime CPU usage to make decisions. The threshold varies depending on different
> applications and system setups, so we add a parameter for user customization. In the
> experimental setup, we empirically set the CPU usage threshold at 40%, as this is the average
> CPU usage when the testbed server reaches its maximum throughput. A higher threshold may
> result in more CPU interference on mutators.

**GC 量的确定（GC volume determination）。** GC 量与四色算法的协同是通过在 GC 期间监控运行时 CPU 使用率并据此限制 P 的数量来实现的。这确保了 CPU 不会被分配到 P 的 GC 线程完全占满。此外，GC 调度器监控运行时 CPU 使用率，以决定在当前 GC 周期内是否将更多 P 加入 T。在前两种策略中，调度器直接参考运行时 CPU 使用率来做决策。该阈值随应用与系统配置的不同而变化，因此我们增加了一个参数供用户自定义。在实验环境中，我们依据经验把 CPU 使用率阈值设置为 40%，因为这是测试服务器达到最大吞吐量时的平均 CPU 使用率。更高的阈值可能导致对 mutator（应用线程）更严重的 CPU 干扰。

> In Predictive policy, LEGO leverages prediction to select the most suitable time for GC
> triggering. The GC scheduler directs LEGO to determine how many Ps to execute while
> maintaining low CPU interference. With the reliable workload prediction of ARIMA, LEGO is
> more likely to trigger full GC instead of incremental GC. Using full GC for memory reclaim is
> more effective when the CPU resource is abundant, while using incremental GC is complementary
> to the full GC under high CPU utilizations.

在预测式策略中，LEGO 借助预测来选择最合适的 GC 触发时机。GC 调度器指导 LEGO 在保持低 CPU 干扰的同时确定要执行多少个 P。借助 ARIMA 可靠的负载预测，LEGO 更倾向于触发 full GC 而非增量 GC。在 CPU 资源充裕时，使用 full GC 进行内存回收更为有效；而在高 CPU 利用率下，使用增量 GC 是对 full GC 的补充。

> **GC triggering timing.** The default full GC executes faster than the incremental GC, as it
> needs less metadata and scans all Ps at the same time. Thus, the threshold of GC triggering
> in the default Go GC is the baseline, i.e., latest GC triggering. Under this condition, LEGO
> switches back to full GC to avoid OOM errors. In LEGO, the concurrent marking time is
> prolonged while the CPU usage is decreased. When LEGO faces bursty requests and GC is
> unavoidable, it triggers GC at the lowest level, i.e., one P at a time. Under this condition,
> we need to ensure the memory is not over the limit. The time to the full GC threshold depends
> on the number of Ps in the application. LEGO calculates the average time consumption on each
> P’s marking, and multiplies it with the number of Ps to get the latest time interval ahead of
> the default full GC for triggering the elastic incremental GC mechanism.

**GC 触发时机（GC triggering timing）。** 默认的 full GC 比增量 GC 执行得更快，因为它需要的元数据更少，并且会同时扫描所有 P。因此，默认 Go GC 中的 GC 触发阈值就是基线，即最迟的 GC 触发时机。在此条件下，LEGO 会切换回 full GC 以避免 OOM 错误。在 LEGO 中，并发标记时间被拉长，同时 CPU 使用率降低。当 LEGO 面临突发请求且 GC 不可避免时，它会在最低层级触发 GC，即一次只处理一个 P。在这种情况下，我们需要确保内存不超出限制。到达 full GC 阈值的时间取决于应用中 P 的数量。LEGO 计算每个 P 标记的平均耗时，并将其与 P 的数量相乘，从而得到在默认 full GC 之前触发弹性增量 GC 机制的最迟时间间隔。

> **Ps enqueue rule.** The GC scheduler is also responsible for the ranking of Ps. Here, LEGO
> uses the least recently scanned list to store all Ps. With the list, LEGO ensures that all Ps
> have equal opportunity to be chosen during the execution.

**P 入队规则（Ps enqueue rule）。** GC 调度器还负责 P 的排序。在此，LEGO 使用最近最少扫描（least recently scanned）列表来存储所有 P。借助该列表，LEGO 确保所有 P 在执行过程中都有平等的机会被选中。

## 4 实现（Implementation）

> We implement LEGO upon Golang 1.16.6 on Ubuntu 21.04. Here, we discuss the important data
> structures and illustrate how to address the Goroutine immigration issue.

我们在 Ubuntu 21.04 上基于 Golang 1.16.6 实现了 LEGO。在此，我们讨论重要的数据结构，并说明如何解决 Goroutine 迁移（Goroutine immigration）问题。

> The elastic incremental GC mechanism asks for delicate information of cross references during
> runtime. Cross references record the references between different Gs or even between
> different Ps. We change write barriers in Go to keep the records of these updates. Three
> levels of metadata are introduced, G, P, and heap levels. G-level remember sets record all
> references from the source G out of the current G. P-level masks record how many Ps are
> referenced by the current P. There are also objects that are moved from the G stack and
> private cache to the universal heap. We add a function to record the relationship between G
> and P.

弹性增量 GC 机制需要运行时精细的交叉引用（cross reference）信息。交叉引用记录了不同 G 之间乃至不同 P 之间的引用关系。我们修改了 Go 中的写屏障（write barrier），以保留这些更新的记录。我们引入了三个层级的元数据：G 级、P 级与堆级。G 级 remember set 记录从源 G 出发、指向当前 G 之外的所有引用。P 级掩码记录当前 P 引用了多少个 P。此外，还有一些对象会从 G 栈与私有缓存移动到全局堆中。我们增加了一个函数来记录 G 与 P 之间的关系。

> **Remember set.** Each cross reference is at G level or the heap. A remember set records the
> reference from its destination to the source, which is necessary for incremental GC execution
> and Goroutine immigration. LEGO records the references by changing the default write barrier
> and adding entries in the remember set. To distinguish different types of objects in Figure 8,
> LEGO introduces remember sets at both G level and the universal heap. For G level, it records
> the cross references from objects in other Gs to the objects in the current G. The heap level
> remember set records all the objects that escaped from Gs. With the remember set, LEGO can
> categorize objects into different states during GC. Meanwhile, as Go scheduler reschedules Gs
> for load balancing, the remember set is also necessary to be used for maintaining the cross
> reference tracing logics consistently.

**Remember set（记忆集）。** 每个交叉引用要么位于 G 级，要么位于堆级。remember set 记录从其目标到源的引用，这对增量 GC 的执行与 Goroutine 迁移都是必需的。LEGO 通过修改默认写屏障并在 remember set 中添加条目来记录这些引用。为区分图 8 中不同类型的对象，LEGO 在 G 级与全局堆两个层级都引入了 remember set。对于 G 级，它记录从其他 G 中的对象到当前 G 中对象的交叉引用。堆级 remember set 则记录所有从 G 中逃逸（escape）出去的对象。借助 remember set，LEGO 可以在 GC 期间将对象归类为不同的状态。同时，由于 Go 调度器会为负载均衡重新调度 G，remember set 对于一致地维护交叉引用追踪逻辑也是必需的。

> **Bit mask.** Since LEGO triggers GC in the P-level manner, we deploy a P-level mask to reduce
> the overhead of comparison when new Ps are added during GC runtime. The mask is made of two
> parts, a reference mask and a reference counting table. P-level bit mask records the
> cross-reference from the other Ps. Each bit in the mask represents the corresponding P, which
> owns at least one Goroutine where its objects have references pointing to the current P. The
> table records the number of Gs referenced by those Ps, which are used to address Goroutine
> immigration issue. The Reference mask is a bit mask that only paints the bit of reference
> source granularity.

**Bit mask（位掩码）。** 由于 LEGO 以 P 级方式触发 GC，我们部署了 P 级掩码，以降低 GC 运行期间新增 P 时的比较开销。该掩码由两部分组成：引用掩码（reference mask）与引用计数表（reference counting table）。P 级位掩码记录来自其他 P 的交叉引用。掩码中的每一位代表相应的 P，该 P 至少拥有一个 Goroutine，且其对象存在指向当前 P 的引用。计数表记录这些 P 所引用的 G 的数量，用于解决 Goroutine 迁移问题。引用掩码是一种位掩码，仅标记引用来源粒度的位。

> **Goroutine immigration.** For load balancing, the Goroutine scheduler schedules ready state
> Goroutines from busy Ps to idle Ps. For example, a Goroutine G1 immigrates from P1 to P2. All
> existing objects in the immigrated Goroutine are allocated in P1’s mspan MS1, and new objects
> will still be allocated on MS1 if there are empty slots. As a result, a mspan from P1 is
> shared by G1, while G1 belongs to P2. If P2 is included in T while P1 is not, we cannot
> recycle the currently unreferenced objects on MS1, as they might be referenced by objects in
> P1. Without an effective way, Goroutine immigration would corrupt the incremental GC logic
> and lead to memory-leak issues.

**Goroutine 迁移（Goroutine immigration）。** 为进行负载均衡，Goroutine 调度器会把处于 ready 状态的 Goroutine 从繁忙的 P 调度到空闲的 P。例如，一个 Goroutine G1 从 P1 迁移到 P2。被迁移的 Goroutine 中所有已经存在的对象都分配在 P1 的 `mspan` MS1 上，如果 MS1 上还有空槽位，新对象仍将分配在 MS1 上。结果是，来自 P1 的一个 `mspan` 被 G1 共享，而 G1 却属于 P2。如果 P2 被包含在 T 中而 P1 没有，我们就无法回收 MS1 上当前未被引用的对象，因为它们可能被 P1 中的对象引用。若没有有效的解决方法，Goroutine 迁移会破坏增量 GC 的逻辑并导致内存泄漏问题。

> A direct record of all cross-references will be a catastrophe when new Ps are added into T.
> Thus, LEGO employs P-level bit masks with the reference counting table. By comparing the bit
> masks with the current T, LEGO categorizes the objects into four states based on the
> quad-color state transform workflow, as illustrated in Fig 8. The reference counting table is
> used when a Goroutine is removed from its current P. Then, all related reference counting
> numbers are reduced by 1. If some value reaches zero, the corresponding mask bit is cleaned.
> This is processed during Goroutine immigration, which mitigates its impact on GC performance.

当新的 P 被加入 T 时，直接记录所有交叉引用将是一场灾难。因此，LEGO 采用带有引用计数表的 P 级位掩码。通过将位掩码与当前的 T 进行比较，LEGO 依据四色状态转换流程把对象归类为四种状态，如图 8 所示。引用计数表在 Goroutine 从其当前 P 中被移除时使用。此时所有相关的引用计数值减 1。如果某个值降为 0，则清除相应的掩码位。该处理在 Goroutine 迁移期间完成，从而减轻其对 GC 性能的影响。


## 5 评测（Evaluation）

### 5.1 实验设置（Setup）

> **Testbed.** We use two servers in the evaluation. Each server is equipped with two
> Intel(R) Xeon(R) Gold 6143 CPUs @ 2.80GHz, 64 GB DRAM, and 1TB HDD.

**测试平台（Testbed）。** 评测中使用两台服务器。每台服务器配备两颗 Intel(R) Xeon(R) Gold 6143 CPU @ 2.80GHz、64 GB DRAM 以及 1TB HDD。

> **LC applications.** We use four databases developed in Go as LC applications, Badger [4],
> Bolt [22], Rosedb [8], and TiDB [9]. Badger, an embedded and fast key-value store, is
> designed for high performance in both reads and writes. Badger is implemented by
> log-structured merge-tree (LSM Tree). Bolt provides a simple, fast, and reliable database
> for projects that do not require a full database server. Bolt is implemented by B+ Tree.
> Rosedb is a fast, reliable, and embedded NoSQL database similar to Rocksdb. Note that
> Badger, Bolt, and RoseDB are key-value stores. We choose TiDB for SQL evaluation. TiDB is
> an open-source distributed SQL database that supports Hybrid transactional and analytical
> processing workloads. It is MySQL compatible and provides strong consistency and high
> availability. Four databases are running on Ubuntu 21.04. The heap size of each database is
> set to default 100% unless defined differently.

**LC 应用。** 我们使用四个用 Go 开发的数据库作为 LC 应用：Badger [4]、Bolt [22]、Rosedb [8] 和 TiDB [9]。Badger 是一个嵌入式的高速键值存储，其设计目标是在读和写两方面都具备高性能。Badger 采用日志结构合并树（log-structured merge-tree, LSM Tree）实现。Bolt 为那些不需要完整数据库服务器的项目提供简单、快速且可靠的数据库。Bolt 采用 B+ Tree 实现。Rosedb 是一个快速、可靠、嵌入式的 NoSQL 数据库，与 Rocksdb 类似。注意 Badger、Bolt 和 RoseDB 都是键值存储。我们选择 TiDB 用于 SQL 评测。TiDB 是一个开源的分布式 SQL 数据库，支持混合事务与分析处理（Hybrid transactional and analytical processing）负载。它与 MySQL 兼容，并提供强一致性与高可用性。四个数据库都运行在 Ubuntu 21.04 上。除非另有说明，每个数据库的堆大小设置为默认的 100%。

> **Microbenchmark.** We first evaluate LEGO on LC applications with a microbenchmark. We
> use Poisson distribution tool FIO [2] to randomly generate input volumes per second with
> the mean value equal to the maximum server capacity. We use two CPUs for evaluation. The
> startup heap size is set to 2 GB. Unless mentioned, the Poisson request pattern is used
> throughout the whole evaluation.

**微基准测试。** 我们首先使用微基准测试在 LC 应用上评测 LEGO。我们使用泊松分布工具 FIO [2] 随机生成每秒的输入量，其均值等于服务器的最大容量。评测使用两个 CPU。启动堆大小设置为 2 GB。除非特别说明，整个评测过程均使用泊松请求模式。

> **Benchmarks.** We evaluate LEGO using Badger, Bolt and TiDB on benchmark Go-YCSB [3].
> Note Rosedb is not supported by Go-YCSB, which is a Go port of YCSB [17]. YCSB is an
> open-source specification and program suite, which is specifically used to compare the
> relative performance of NoSQL database systems. Go-YCSB fully supports all YCSB generators
> and the core workload so we can do the basic benchmarks with Go. We choose 50% READ and 50%
> UPDATE queries in the evaluation, which is a typical setup used in [30]. To evaluate the
> bursty request issue, we control the *target* value of requests per second. Dynamic
> *target* is not supported in Go-YCSB, so we add it in Go-YCSB. We use 30 CPUs for
> evaluation, and the startup heap size is set to 30GB.

**基准测试。** 我们使用 Badger、Bolt 和 TiDB 在基准测试工具 Go-YCSB [3] 上评测 LEGO。注意 Go-YCSB 不支持 Rosedb，Go-YCSB 是 YCSB [17] 的 Go 版本。YCSB 是一套开源的规范与程序套件，专门用于比较 NoSQL 数据库系统的相对性能。Go-YCSB 完整支持所有 YCSB 生成器和核心负载，因此我们可以用 Go 完成基本基准测试。在评测中我们选择 50% READ 和 50% UPDATE 查询，这是 [30] 中采用的典型配置。为了评测突发请求问题，我们控制每秒请求数的 *target* 值。Go-YCSB 不支持动态 *target*，因此我们在 Go-YCSB 中加入了该功能。评测使用 30 个 CPU，启动堆大小设置为 30GB。

> **Approaches.** We use three approaches in performance evaluation, 1) the default Go GC
> (noted as Go GC), 2) a passive incremental version of LEGO (noted as passive LEGO), which
> emulates the production G1 GC used in JVM, 3) and LEGO. Unless mentioned, LEGO uses policy
> B, Default Augmented, in the GC Scheduler.

**对比方法。** 我们在性能评测中使用三种方法：1）默认的 Go GC（记为 Go GC），2）LEGO 的被动式增量版本（记为 passive LEGO），它模拟 JVM 中生产环境所使用的 G1 GC，3）LEGO。除非特别说明，LEGO 在 GC 调度器中使用策略 B，即 Default Augmented。

> **Metrics.** In primary, we measure the tail latency and throughput of the LC
> applications. We also form a metric to study the relationship between GC triggering and
> bursty conditions. We record the sum of the number of GCs triggered under high CPU usage
> scenarios (higher than the threshold) and under low CPU usage scenarios (lower than the
> threshold). The default maximum RPS of the server is set to be the threshold that can
> maintain the SLO of LC applications. The metric, called *RatioBurst* (*RB*), is the ratio
> of GC triggered under high CPU usage scenarios and under low CPU usage scenarios, which is
> multiplied by the proportion of bursty requests in the total. A high *RB* means more
> proportions of GCs are triggered under high CPU usages. We use a metric of GC burstiness to
> study the benefit of GC triggering under low CPU usages.

**度量指标。** 我们主要测量 LC 应用的尾延迟和吞吐量。我们还构造了一个指标来研究 GC 触发与突发状况之间的关系。我们记录在高 CPU 利用率场景（高于阈值）和低 CPU 利用率场景（低于阈值）下触发 GC 的数量之和。服务器的默认最大 RPS 被设定为能够维持 LC 应用 SLO 的阈值。该指标称为 *RatioBurst*（*RB*），它是在高 CPU 利用率场景下触发的 GC 与在低 CPU 利用率场景下触发的 GC 之比，再乘以突发请求在总请求中所占的比例。*RB* 较高意味着有更大比例的 GC 在高 CPU 利用率下被触发。我们使用 GC 突发性（GC burstiness）这一指标来研究在低 CPU 利用率下触发 GC 的收益。

> In the following, we first evaluate the tail latency and the throughput of the three LC
> applications. We also compare the passive GC mechanism with LEGO, and discuss different
> scheduling policies used in LEGO GC scheduler. We then conduct sensitivity analysis by
> examining the performance of LEGO under different memory pressure levels and heap sizes.
> Finally, we discuss the overhead of LEGO.

在下文中，我们首先评测三个 LC 应用的尾延迟和吞吐量。我们还将被动式 GC 机制与 LEGO 进行比较，并讨论 LEGO GC 调度器中使用的不同调度策略。随后，我们通过考察 LEGO 在不同内存压力水平和堆大小下的表现进行敏感性分析。最后，我们讨论 LEGO 的开销。

### 5.2 尾延迟（The Tail Latency）

> We evaluate the tail latency of the LC applications with both the microbenchmark and
> Go-YCSB benchmark. Experiments are conducted in two configurations, 1) LC applications
> using LEGO, and 2) LC applications using the default Go GC. Under each configuration, we
> run the experiment for two hours and measure the average values as well as the real-time
> values during GC runtime. Due to the different implementations and CPU usages of the three
> LC applications, their RPS are different in meeting the maximum server capacity.

我们使用微基准测试和 Go-YCSB 基准测试来评测 LC 应用的尾延迟。实验在两种配置下进行：1）LC 应用使用 LEGO，2）LC 应用使用默认的 Go GC。在每种配置下，我们运行实验两个小时，并测量平均值以及 GC 运行期间的实时值。由于三个 LC 应用的实现方式和 CPU 利用率不同，它们在满足服务器最大容量时的 RPS 也不同。

> Table 3 shows all results under the five configurations. We can observe that LEGO renders
> the lowest tail latency in average as well as during GC runtime for all five
> configurations. Compared with the default Go GC, for the microbenchmark LEGO can reduce the
> p99 (p99.9) latency by 13.3% (33.3%), 18.1% (40.3%), 12.7% (25.6%), and 21.6% (33.8%) for
> Badger, Bolt, TiDB, and Rosedb, respectively. For Go-YCSB, LEGO can reduce the p99 (p99.9)
> latency by 35.6% (36.7%), 19.7% (23.4%), and 9.5% (17.3%) for Badger, Bolt, and TiDB
> respectively. Badger uses LSM tree, which speeds up write operations while leading to
> multiple read operations in reading a modified entry. Bolt employs B+ Tree to strengthen its
> capability in concurrent readings. The trade-off between write and read operations makes
> Badger more memory consuming. Thus, compared with Bolt, Badger uses more memory and
> triggers more GCs in face of the same number of incoming requests. Due to more rescheduled
> GCs, LEGO obtains more significant improvement for Badger than for Bolt.

表 3 给出了五种配置下的全部结果。我们可以观察到，在全部五种配置下，无论平均值还是 GC 运行期间，LEGO 都呈现出最低的尾延迟。与默认的 Go GC 相比，在微基准测试中，LEGO 对 Badger、Bolt、TiDB 和 Rosedb 分别将 p99（p99.9）延迟降低了 13.3%（33.3%）、18.1%（40.3%）、12.7%（25.6%）和 21.6%（33.8%）。对于 Go-YCSB，LEGO 对 Badger、Bolt 和 TiDB 分别将 p99（p99.9）延迟降低了 35.6%（36.7%）、19.7%（23.4%）和 9.5%（17.3%）。Badger 使用 LSM 树，这加快了写操作，但在读取一条被修改的记录时会引发多次读操作。Bolt 采用 B+ Tree 来增强其并发读取能力。写操作与读操作之间的权衡使 Badger 更加消耗内存。因此，与 Bolt 相比，Badger 在面对相同数量的到达请求时会使用更多内存并触发更多 GC。由于有更多 GC 被重新调度，LEGO 对 Badger 的改进比对 Bolt 更为显著。

**图 9：三种方法的 GC 触发情况（Fig. 9: GC triggering of the three approaches）** 坐标轴与图例：x 轴 `Time (sec)`（0–300），y 轴 `Input Volume (K)`（0–3000）；图例 `Go GC`、`LEGO`、`Passive LEGO`、`Input Volume`。

> Among the LC applications, LEGO achieves relatively less improvement for TiDB due to its
> storage engine optimization. TiKV serves as the underlying storage engine for TiDB,
> utilizing RocksDB for data storage and management. TiKV supports multi-version concurrency
> control, which allows multiple versions of the same row to be stored in RocksDB. To reduce
> resource usage, TiKV records the concurrency control statistics. The optimization allows
> TiDB to check the table properties before examining each region row by row, resulting in
> fewer memory objects generated in the runtime heap and reducing the GC overhead for memory
> reclaim.

在这些 LC 应用中，由于其存储引擎的优化，LEGO 对 TiDB 取得的改进相对较小。TiKV 作为 TiDB 的底层存储引擎，利用 RocksDB 进行数据存储和管理。TiKV 支持多版本并发控制（multi-version concurrency control），允许同一行的多个版本存储在 RocksDB 中。为减少资源使用，TiKV 记录并发控制的统计信息。该优化使 TiDB 能够在逐行检查每个 region 之前先检查表属性，从而在运行时堆中生成更少的内存对象，并降低内存回收的 GC 开销。

> We also evaluate the tail latency during GC runtime. Under the seven configurations,
> compared with Go GC, LEGO reduces the p95 (p99) latency during GC runtime by 54.8%
> (62.3%), 65.7% (70.1%), 36.4% (56.8%), 61.5% (70.3%), 71.9% (81.8%), 75.8% (81.2%), 70.7%
> (82.9%), respectively. Note even with LEGO, the tail latency can still be non-trivial
> during GC runtime, as the GC process owns higher priority compared to the mutators.

我们还评测了 GC 运行期间的尾延迟。在七种配置下，与 Go GC 相比，LEGO 将 GC 运行期间的 p95（p99）延迟分别降低了 54.8%（62.3%）、65.7%（70.1%）、36.4%（56.8%）、61.5%（70.3%）、71.9%（81.8%）、75.8%（81.2%）和 70.7%（82.9%）。注意，即使使用 LEGO，由于 GC 进程相比 mutator 拥有更高的优先级，GC 运行期间的尾延迟仍可能相当可观。

### 5.3 被动式 GC 与主动式 GC（Passive GC v.s. Proactive GC）

> We now investigate the performance of using LEGO without its GC scheduler (noted as
> passive LEGO) to emulate G1 GC, a production concurrent GC that is the default GC on JDK 9.
> We choose G1 GC instead of more recently proposed JVM GCs for performance comparison
> because a recent study finds those new GC mechanisms indeed incur higher tail latency [51]
> than G1 GC does. Passive LEGO is set to trigger GC with 50% volume of the runtime heap. We
> use Badger with Go-YCSB benchmark in the performance evaluation.

现在我们研究在不使用 GC 调度器的情况下使用 LEGO（记为 passive LEGO）来模拟 G1 GC 的性能，G1 GC 是 JDK 9 上默认的生产级并发 GC。我们选择 G1 GC 而不是更新近提出的 JVM GC 进行性能比较，因为近期的一项研究发现，那些新的 GC 机制确实比 G1 GC 产生更高的尾延迟 [51]。Passive LEGO 被设置为在运行时堆达到 50% 容量时触发 GC。我们在性能评测中使用 Badger 与 Go-YCSB 基准测试。

**表 3：采用微基准测试与 GO-YCSB 时 LC 应用的尾延迟（ms）与吞吐量（Table 3: The tail latency (ms) and throughput of LC applications with the microbenchmark and GO-YCSB）**

| Benchmarks 基准测试 | LC Applications LC 应用 | GC types GC 类型 | p99（Average Application Latency and Throughput 平均应用延迟与吞吐量） | p99.9 | Average 平均 | Throughput (RPS) 吞吐量（RPS） | p95（Latency during GC Runtime GC 运行期间延迟） | p99 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Microbench | Badger | Go GC | 12.1 | 19.8 | 6.7 | 50K/s | 42 | 114 |
| Microbench | Badger | LEGO | 10.5 | 13.2 | 6.8 | 50K/s | 18 | 43 |
| Microbench | Bolt | Go GC | 18.4 | 31.5 | 9.3 | 60K/s | 67 | 216 |
| Microbench | Bolt | LEGO | 14.9 | 18.8 | 9.5 | 60K/s | 23 | 65 |
| Microbench | TiDB | Go GC | 8.7 | 17.2 | 5.3 | 50K/s | 33 | 95 |
| Microbench | TiDB | LEGO | 7.6 | 12.8 | 5.4 | 50K/s | 21 | 41 |
| Microbench | Rosedb | Go GC | 22.7 | 37.0 | 11.5 | 50K/s | 83 | 263 |
| Microbench | Rosedb | LEGO | 17.8 | 24.5 | 11.7 | 50K/s | 32 | 78 |
| Go-YCSB | Badger | Go GC | 28.4 | 38.7 | 12.7 | 750K/s | 124 | 453 |
| Go-YCSB | Badger | LEGO | 18.3 | 24.5 | 12.9 | 750K/s | 35 | 87 |
| Go-YCSB | Bolt | Go GC | 21.9 | 27.4 | 10.9 | 900K/s | 128 | 350 |
| Go-YCSB | Bolt | LEGO | 17.6 | 21.0 | 11.0 | 900K/s | 31 | 69 |
| Go-YCSB | TiDB | Go GC | 10.6 | 23.1 | 7.3 | 750K/s | 92 | 188 |
| Go-YCSB | TiDB | LEGO | 9.0 | 19.1 | 7.5 | 750K/s | 27 | 51 |

> Figure 9 shows the input volume of Badger in RPS in five minutes of experimentation. Three
> types of points show the timing and the input volume when GC is triggered due to the three
> approaches. Go GC and LEGO incur similar numbers of GC triggering, while the passive LEGO
> incurs about 2x of GC triggering because it cannot dynamically adjust the GC volume at
> runtime. When the volume is set at one half of the heap, the times of GC triggering
> increase dramatically. Without an adaptive GC scheduler, the passive GC tends to trigger GC
> under high input volumes, similarly to Go GC. On the other hand, LEGO triggers the majority
> of GCs under low input volumes (i.e. under low CPU usages). There are two exceptions
> occurring at 171s and 248s, when LEGO is triggered under bursty conditions. Two following
> GCs are triggered at 173s and 252s. During the first 2 GCs, LEGO limits the GC volume for
> low latency, and later triggers the two extra GCs when the CPU usage is low.

图 9 显示了 Badger 在五分钟实验中的输入量（以 RPS 计）。三类点显示了三种方法触发 GC 的时机和输入量。Go GC 和 LEGO 触发 GC 的次数相近，而 passive LEGO 触发的 GC 次数约为其 2 倍，因为它无法在运行时动态调整 GC 容量。当容量被设置为堆的一半时，GC 触发次数急剧增加。在没有自适应 GC 调度器的情况下，被动式 GC 倾向于在高输入量下触发 GC，与 Go GC 类似。另一方面，LEGO 的大部分 GC 都在低输入量（即在低 CPU 利用率）下触发。有两个例外发生在 171s 和 248s，此时 LEGO 在突发状况下被触发。随后两次 GC 分别在 173s 和 252s 被触发。在前两次 GC 期间，LEGO 为保持低延迟而限制了 GC 容量，之后在 CPU 利用率较低时触发了这两次额外的 GC。

> Table 4 gives the experimental statistics in the number of GC triggering, the average input
> volume at GC triggering, *RB*, and p99 tail latency. The p99 tail latency due to LEGO,
> passive LEGO and Go GC are 17.9ms, 31.2ms, and 27.3ms, respectively. Passive LEGO yields
> the worst outcome. Due to the extra data recording, a naive incremental GC (passive LEGO)
> increases the resource usage and reduces the memory reclaim volume in each GC cycle.
> Without a proactive GC scheduler, passive LEGO incurs 14% longer p99 tail latency. LEGO
> triggers slightly more GCs than Go GC does, while its *RB* and the average input volume at
> GC triggering are dramatically smaller than those of the other two approaches. A smaller
> *RB* means fewer GC triggered under high CPU usages. Compared with Go GC and the tailored
> G1 GC, LEGO improves the p99 tail latency by 34.5% and 42.7% respectively.

表 4 给出了 GC 触发次数、GC 触发时的平均输入量、*RB* 以及 p99 尾延迟的实验统计数据。LEGO、passive LEGO 和 Go GC 的 p99 尾延迟分别为 17.9ms、31.2ms 和 27.3ms。Passive LEGO 的结果最差。由于额外的数据记录，朴素的增量式 GC（passive LEGO）增加了资源使用，并减少了每个 GC 周期中的内存回收量。在没有主动式 GC 调度器的情况下，passive LEGO 的 p99 尾延迟增加了 14%。LEGO 触发的 GC 比 Go GC 略多，而它的 *RB* 和 GC 触发时的平均输入量都远小于另外两种方法。较小的 *RB* 意味着在高 CPU 利用率下触发的 GC 更少。与 Go GC 和定制的 G1 GC 相比，LEGO 分别将 p99 尾延迟改进了 34.5% 和 42.7%。

> Overall, LEGO demonstrates significant improvement over both Go GC and G1 GC (emulated by
> passive LEGO).

总体而言，LEGO 相对 Go GC 和 G1 GC（由 passive LEGO 模拟）都展现出显著的改进。


### 5.4 应用吞吐量（Application Throughput）

> We run LC applications Badger (Bolt) with varying RPS, ranging from 500K to 1M (1.3M) with
> 50K (100K) as a step. We study the application throughput due to the three approaches. We
> collect the p99 tail latency at each RPS and study how LEGO improves the application
> throughput, compared with the default Go GC and the passive LEGO.

我们以变化的 RPS 运行 LC 应用 Badger（Bolt），RPS 从 500K 到 1M（1.3M），步长为 50K（100K）。我们研究这三种方法带来的应用吞吐量。我们在每个 RPS 上采集 p99 尾延迟，并研究 LEGO 相比默认的 Go GC 和 passive LEGO 如何改进应用吞吐量。

**表 4：三种方法 GC 触发的统计（Table 4: Statistics of GC triggering of three approaches）**

| Statistics 统计项 | LEGO | Passive LEGO 被动式 LEGO | Go GC |
| --- | --- | --- | --- |
| Number of GC triggering GC 触发次数 | 14 | 26 | 11 |
| Average Input at GC (K) GC 触发时的平均输入量（K） | 355 | 1104 | 1334 |
| *RB* | 0.331 | 0.955 | 1.031 |
| p99 (ms) | 17.9 | 31.2 | 27.3 |

**表 5：两种 RPS 值下三种策略的统计（Table 5: Statistics of three policies under two RPS values）**

| Input Volume 输入量 | Statistics 统计项 | Policy A 策略 A | Policy B 策略 B | Policy C 策略 C |
| --- | --- | --- | --- | --- |
| 750K RPS/sec | GC triggering times GC 触发次数 | 360 | 156 | 146 |
| 750K RPS/sec | Avg Input at GC (K) GC 触发时的平均输入量（K） | 193 | 341 | 289 |
| 750K RPS/sec | *RB* | 0.072 | 0.310 | 0.181 |
| 750K RPS/sec | p99 (ms) | 18.1 | 17.7 | 17.5 |
| 1500 RPS/sec | GC triggering times GC 触发次数 | 835 | 340 | 311 |
| 1500 RPS/sec | Avg Input at GC (K) GC 触发时的平均输入量（K） | 470 | 683 | 552 |
| 1500 RPS/sec | *RB* | 0.372 | 1.396 | 1.023 |
| 1500 RPS/sec | p99 (ms) | 47.8 | 54.2 | 43.3 |

> Figure 10 shows the relationship between the p99 tail latency and the application
> throughput (RPS) of the three approaches. The red line shows the p99 tail latency of
> 28.4ms (21.9ms), when the default Go GC runs at the maximum server capacity at 750K (900K)
> RPS. Due to the overhead of the incremental GC, the passive LEGO reaches a similar p99
> tail latency when RPS is decreased to 700K (800K). All three approaches lead to increasing
> application throughput as more requests are processed per second. Without rescheduling the
> GC process, the p99 tail latency of Go GC increases much faster than LEGO does. LEGO
> reaches a similar p99 tail latency of 29.1ms (21.6ms), when the RPS is increased to
> 1,000K (1,100K) RPS for Badger (Bolt). Thus, compared with the default Go GC and passive
> LEGO, LEGO improves the throughput of Badger (Bolt) by 33% (22%) and 42% (38%),
> respectively, under similar tail latency.

图 10 显示了三种方法的 p99 尾延迟与应用吞吐量（RPS）之间的关系。红线表示默认 Go GC 在服务器最大容量 750K（900K）RPS 下运行时的 p99 尾延迟为 28.4ms（21.9ms）。由于增量式 GC 的开销，passive LEGO 在 RPS 降至 700K（800K）时才达到相近的 p99 尾延迟。随着每秒处理的请求增多，三种方法都会带来应用吞吐量的提升。在不重新调度 GC 进程的情况下，Go GC 的 p99 尾延迟增长得比 LEGO 快得多。对于 Badger（Bolt），当 RPS 增加到 1,000K（1,100K）时，LEGO 才达到相近的 p99 尾延迟 29.1ms（21.6ms）。因此，在相近的尾延迟下，与默认的 Go GC 和 passive LEGO 相比，LEGO 分别将 Badger（Bolt）的吞吐量提升了 33%（22%）和 42%（38%）。

**图 10：三种方法的应用吞吐量（Fig. 10: Application throughput of three approaches）** 坐标轴与图例：左图 Badger，x 轴 `Throughput in RPS (K)`（600–1000），y 轴 `P99 Latency (ms)`（20–60）；右图 Bolt，x 轴 `Throughput in RPS (K)`（600–1200），y 轴（10–60）；图例 `LEGO`、`GoGC`、`passive LEGO`。

### 5.5 调度策略（Scheduling Policies）

> We examine the three policies of LEGO GC scheduler, Earliest First (A), Default Augmented
> (B), and Predictive (C) under two input conditions, 750K PRS as the normal traffic
> condition and 1,500K RPS as the heavy condition. The heap size thresholds of A are set at
> 50% and 30% under the normal and heavy conditions, respectively. C is trained by ARIMA in
> a 10-hour input RPS record and predicts the next 1-hour RPS.

我们在两种输入条件下考察 LEGO GC 调度器的三种策略：Earliest First（A）、Default Augmented（B）和 Predictive（C），其中 750K PRS 作为正常流量条件，1,500K RPS 作为重载条件。A 的堆大小阈值在正常和重载条件下分别设为 50% 和 30%。C 使用 ARIMA 在 10 小时的输入 RPS 记录上进行训练，并预测未来 1 小时的 RPS。

> Table 5 shows policy C outperforms the two other policies, with 17.5ms and 43.3ms in the
> p99 tail latency under normal and heavy-duty conditions, respectively. Compared with A and
> B in normal (heavy) conditions, C reduces the tail latency by 3.4% (9.5%) and 1.2% (20.2%),
> respectively. Accurate prediction of incoming traffic volume enables C to trigger the least
> times of GC. It avoids the two extra GC triggering due to Policy B (at 171s and 248s in
> Figure 9). It also increases the opportunity in triggering GC under low CPU usages. The
> performance difference due to policies A and B depends on the incoming traffic conditions.
> Under normal conditions, B obtains shorter p99 tail latency (17.7ms) than A does (18.1ms).
> Though A has the lowest *RB* among all three approaches, the frequency of GC triggering is
> so high that incurs significant CPU interference. Under heavy traffic conditions, A obtains
> smaller p99 tail latency (47.8ms) than B does (54.2ms). A triggers GC as early as possible
> when both thresholds of heap size and CPU usage are met. It yields the higher possibility
> in triggering GC under low CPU usages.

表 5 显示策略 C 优于另外两种策略，在正常和重载条件下 p99 尾延迟分别为 17.5ms 和 43.3ms。在正常（重载）条件下，与 A 和 B 相比，C 分别将尾延迟降低了 3.4%（9.5%）和 1.2%（20.2%）。对到达流量体积的准确预测使 C 的 GC 触发次数最少。它避免了策略 B 带来的两次额外 GC 触发（图 9 中 171s 和 248s 处）。它还提高了在低 CPU 利用率下触发 GC 的机会。策略 A 与 B 的性能差异取决于到达流量条件。在正常条件下，B 的 p99 尾延迟（17.7ms）比 A（18.1ms）更短。尽管 A 在三种方法中 *RB* 最低，但其 GC 触发频率过高，造成了显著的 CPU 干扰。在重载流量条件下，A 的 p99 尾延迟（47.8ms）比 B（54.2ms）更小。A 在堆大小和 CPU 利用率两个阈值都满足时尽可能早地触发 GC。这带来了在低 CPU 利用率下触发 GC 的更高可能性。

> Overall, Policy C yields the best performance with correct predictions. Policy B fits under
> a normal traffic condition, while Policy A works better under heavy traffic conditions.

总体而言，在预测正确的情况下策略 C 表现最佳。策略 B 适用于正常流量条件，而策略 A 在重载流量条件下表现更好。

### 5.6 内存压力敏感性（Sensitivity of Memory Pressure）

> We now examine LEGO under different memory pressures. We run Badger on Go-YCSB with
> different heap sizes to evaluate the latency reduction by LEGO compared to the default Go
> GC. Figure 11 shows the percentile of the tail latency reduction when the size of the
> runtime heap is set to 25%, 50%, 100%, 150%, and 200%, respectively. We can see that LEGO
> obtains just marginal improvement for the p99 and p99.9 tail latency under 150% and 200%
> heap sizes. However, as the heap size decreases, the latency reduction due to LEGO improves
> significantly. The rationale is that there are more frequent GC triggering with small heap
> sizes, which in return increases the CPU interference of GC and mutators. LEGO yields
> higher possibility in triggering GC under low CPU usages, which mitigates the CPU
> contention with mutators. As a result, LEGO significantly reduces the tail latency. For
> example, when the heap size is set at 50%, the latency reductions for p99 and p99.9 latency
> due to LEGO are 46.2% and 53.4%, respectively.

现在我们考察 LEGO 在不同内存压力下的表现。我们在 Go-YCSB 上以不同的堆大小运行 Badger，以评测 LEGO 相比默认 Go GC 的延迟降低幅度。图 11 显示了运行时堆大小分别设置为 25%、50%、100%、150% 和 200% 时尾延迟降低的百分比。我们可以看到，在 150% 和 200% 堆大小下，LEGO 对 p99 和 p99.9 尾延迟仅有边际改进。然而，随着堆大小减小，LEGO 带来的延迟降低显著改善。其原因在于，堆较小时 GC 触发更频繁，进而增加了 GC 与 mutator 的 CPU 干扰。LEGO 在低 CPU 利用率下触发 GC 的可能性更高，从而缓解了与 mutator 的 CPU 争用。因此，LEGO 显著降低了尾延迟。例如，当堆大小设置为 50% 时，LEGO 对 p99 和 p99.9 延迟的降低分别为 46.2% 和 53.4%。

**图 11：不同堆大小下的延迟降低（Fig. 11: Latency reduction with different heap sizes）** 坐标轴与图例：x 轴为堆大小 `25%`、`50%`、`100%`、`150%`、`200%`，y 轴 `Reduction (%)`（0–60）；图例 `p99 Tail Latency`、`p99.9 Tail Latency`。

**表 6：多租户环境中的统计（Table 6: Statistics in multi-tenant environments）**

|  | LC p99 Tail Latency (ms) LC 应用的 p99 尾延迟（ms） | Batch Jobs (# per hour) 批处理作业（个/小时） | Ave. CPU usage at GC triggering GC 触发时的平均 CPU 利用率 |
| --- | --- | --- | --- |
| LEGO | 25.9 | 175 | 26% |
| Go GC | 32.0 | 172 | 47% |

### 5.7 多租户（Multi-tenancy）

> We conduct performance evaluation of LEGO under multi-tenancy. We use Badger as the LC
> application and Kmeans from Hibench running on Spark as the batch jobs. We use 20 logical
> CPUs (10 physical cores with Hyperthreading) for the experiment, in which 10 logical CPUs
> are assigned to Badger and the other 10 paired logical CPUs are assigned to the batch job
> pool. Thus, there are CPU interference between Badger and batch jobs. Table 6 presents the
> p99 tail latency and the average CPU usage when GC is triggered. The results show that LEGO
> reduces the p99 tail latency by 19.1% compared to the default Go GC. Furthermore, the
> average CPU usage when GC is triggered with LEGO is 44% lower, reducing CPU interference
> from GC threads. LEGO also marginally increases the throughput of batch jobs, as less CPU
> preemption occurs during GC. Note that resource scheduling and performance isolation
> between different applications is out of scope of this study.

我们在多租户下对 LEGO 进行性能评测。我们使用 Badger 作为 LC 应用，并使用运行在 Spark 上的 Hibench 中的 Kmeans 作为批处理作业。实验使用 20 个逻辑 CPU（10 个物理核心，开启超线程），其中 10 个逻辑 CPU 分配给 Badger，另外 10 个配对的逻辑 CPU 分配给批处理作业池。因此，Badger 与批处理作业之间存在 CPU 干扰。表 6 给出了 p99 尾延迟以及 GC 触发时的平均 CPU 利用率。结果表明，与默认的 Go GC 相比，LEGO 将 p99 尾延迟降低了 19.1%。此外，使用 LEGO 时 GC 触发时的平均 CPU 利用率低 44%，减少了来自 GC 线程的 CPU 干扰。LEGO 还略微提高了批处理作业的吞吐量，因为 GC 期间的 CPU 抢占更少。注意，不同应用之间的资源调度和性能隔离不在本研究范围内。

### 5.8 开销（Overhead）

> LEGO introduces the elastic incremental GC mechanism that dynamically determines the GC
> volume at runtime. Compared with the default Go GC, LEGO employs an always-on write barrier
> for object recording, which increases the average request latency by up to 2%. LEGO also
> increases the basic heap usage due to the extra data structure for cross-references, which
> implicitly increases the frequency of GC. However, the benefits of the elastic incremental
> GC mechanism and adaptive GC scheduler significantly outperform the overhead. Note that the
> number of Ps is counted by the default Go runtime. LEGO does not introduce overhead in this
> regard.

LEGO 引入了弹性增量 GC 机制，在运行时动态确定 GC 容量。与默认的 Go GC 相比，LEGO 采用常开的写屏障进行对象记录，这使平均请求延迟最多增加 2%。由于为交叉引用增加了额外的数据结构，LEGO 还增大了基础堆使用量，这隐式地提高了 GC 的频率。然而，弹性增量 GC 机制和自适应 GC 调度器带来的收益显著超过其开销。注意，P 的数量由默认的 Go 运行时统计。LEGO 在这方面不引入开销。


## 6 相关工作（Related Work）

> **LC Application QoS.** Many studies focus on meeting the SLO of LC applications from the perspective of reducing the tail latency [11, 23, 25, 39, 52]. For example, PerfIso [28] is on CPU isolation for CPU sharing between LC applications and batch jobs. FlatStore [16] is a KV store that uses logs in persistent memory for serving data requests. Caladan [25] proposes a new CPU scheduler in the Linux kernel that improves QoS of LC applications, especially the tail latency. Perséphone [23] is a novel kernel-bypass OS scheduler designed to minimize the tail latency for applications executing at the microsecond scale. LEGO tackles the GC interference on LC applications, which is orthogonal to those studies.

**LC 应用的 QoS。** 许多研究从降低尾延迟的角度出发，致力于满足 LC 应用的 SLO [11, 23, 25, 39, 52]。例如，PerfIso [28] 关注的是 LC 应用与批处理作业共享 CPU 时的 CPU 隔离问题。FlatStore [16] 是一种 KV 存储，它利用持久内存中的日志来服务数据请求。Caladan [25] 在 Linux 内核中提出了一种新的 CPU 调度器，用于改善 LC 应用的 QoS，尤其是尾延迟。Perséphone [23] 是一种新颖的绕过内核（kernel-bypass）操作系统调度器，旨在为微秒级执行的应用最小化尾延迟。LEGO 解决的是 GC 对 LC 应用的干扰问题，与上述研究是正交的。

> **GC Optimization.** Most studies on GC are focused on JVM [12, 24, 27, 27, 33, 36–38, 46, 49–51]. LXR GC [51] demonstrates the production G1 GC yields smaller tail latency for LC applications, compared with the concurrent GC mechanisms such as ZGC GC [20] and Shenandoah GC [19]. Though G1 GC incurs more STW pauses, the overall tail latency is indeed reduced. Yak [37] develops a GC algorithm for distributed data processing systems. ROLP [12] profiles object lifetime and uses the profiled information for efficient GC. iGC [50] introduces a proactive GC to reduce the tail latency in cloud environments. Different from the most tracing GCs, LRX GC [51] evolves the original reference counter based GC and achieves significant improvement on LC application QoS. However, the mechanism does not apply for Go, because Go GC is neither generational nor copy-and-compact. Platinum [47] leverages idle cores during STW pauses and grants them to mutators for utilizing idle computing resources. LEGO focuses on Go GC, while its adaptive incremental GC mechanism can also be extended and deployed in other automatic memory management programming languages.

**GC 优化。** 大多数 GC 研究都聚焦于 JVM [12, 24, 27, 27, 33, 36–38, 46, 49–51]。LXR GC [51] 表明，与 ZGC GC [20] 和 Shenandoah GC [19] 等并发 GC 机制相比，生产环境中的 G1 GC 能为 LC 应用带来更小的尾延迟。尽管 G1 GC 会引入更多的 STW 停顿，但整体尾延迟确实降低了。Yak [37] 为分布式数据处理系统开发了一种 GC 算法。ROLP [12] 对对象生命周期进行剖析，并利用剖析得到的信息实现高效 GC。iGC [50] 引入了主动式 GC 以降低云环境中的尾延迟。与大多数追踪式 GC 不同，LRX GC [51] 对原本基于引用计数器的 GC 进行了演进，并在 LC 应用的 QoS 上取得了显著改进。然而，该机制并不适用于 Go，因为 Go GC 既不是分代的，也不是复制-压缩（copy-and-compact）式的。Platinum [47] 利用 STW 停顿期间空闲的核心，将其授予 mutator，以利用空闲的计算资源。LEGO 聚焦于 Go GC，其自适应增量 GC 机制也可以扩展并部署到其他采用自动内存管理的编程语言中。

> **GC in New Systems.** There are GC studies focusing on new systems, such as non-volatile memory (NVM) and far memory systems. For example, Semeru [44] provides a distributed GC that offloads object tracing to memory servers so that tracing is performed close to data in memory-disaggregated systems. Work [48] optimizes GC atop NVM systems by introducing two NVM-aware optimizations, write cache and header map, to tackle the bandwidth limitation of NVM. Memliner [45] improves the performance of far-memory systems by lining up memory accesses, which alleviates the gap between the application QoS and the GC.

**新型系统中的 GC。** 也有一些 GC 研究聚焦于新型系统，例如非易失性内存（NVM）和远端内存（far memory）系统。例如，Semeru [44] 提供了一种分布式 GC，将对象追踪卸载到内存服务器上，从而在内存解耦（memory-disaggregated）系统中实现贴近数据的追踪。文献 [48] 通过在 NVM 系统之上引入两项 NVM 感知的优化——写缓存与头部映射（header map）——来应对 NVM 的带宽限制，从而优化 GC。Memliner [45] 通过对内存访问进行排队来提升远端内存系统的性能，从而缓解应用 QoS 与 GC 之间的鸿沟。

> **Performance Interference and Isolation.** Prior works have intensively tackled performance interference in several ways. Early approaches disallow LC applications from sharing resources with other applications to avoid performance interference [13–15, 31, 39, 40, 42, 43]. Other early approaches avoid co-scheduling applications that are likely to interfere with each other [21, 35]. The third category of research focuses on eliminating interference altogether, by partitioning and isolating resources among co-scheduled services with feedback control based techniques [22, 29, 31, 39]. LEGO tackles the interference between GC threads and mutators of LC applications, which is complementary to the existing studies.

**性能干扰与隔离。** 已有工作从多个方面对性能干扰进行了深入研究。早期方法不允许 LC 应用与其他应用共享资源，以避免性能干扰 [13–15, 31, 39, 40, 42, 43]。另一些早期方法则避免将可能相互干扰的应用协同调度在一起 [21, 35]。第三类研究着眼于彻底消除干扰，借助基于反馈控制的技术在协同调度的服务之间划分并隔离资源 [22, 29, 31, 39]。LEGO 解决的是 LC 应用的 GC 线程与 mutator 之间的干扰，这与现有研究是互补的。

## 7 结论（Conclusion）

> LEGO is motivated by severe performance degradation due to the CPU interference between concurrent GC threads and mutators of LC applications during GC runtime in face of bursty requests. We find that proactively scheduling GC threads under low CPU usages opens up a significant opportunity for application performance improvement. LEGO designs the adaptive incremental GC mechanism for the first time. We develop LEGO in Go, as the current GC mechanisms for JVM are ineffective for Go due to its unique GPM thread scheduling and memory allocation model. Experimental results show that compared to the default Go GC and a tailored G1 GC, LEGO significantly improves both the tail latency and the throughput for LC applications developed in Go.

LEGO 的动机在于：面对突发请求时，LC 应用在 GC 运行期间，并发 GC 线程与 mutator 之间的 CPU 争用会导致严重的性能下降。我们发现，在低 CPU 利用率下主动调度 GC 线程，为提升应用性能带来了重要的机会。LEGO 首次设计了自适应增量 GC 机制。我们用 Go 实现了 LEGO，因为 JVM 现有的 GC 机制对 Go 并不奏效，这是由 Go 独特的 GPM 线程调度与内存分配模型所决定的。实验结果表明，与默认 Go GC 和经过定制的 G1 GC 相比，LEGO 显著改善了用 Go 开发的 LC 应用的尾延迟与吞吐量。

> In future, we plan to extend LEGO to other programming languages that deploy automatic memory management.

未来，我们计划将 LEGO 扩展到其他采用自动内存管理的编程语言。

## 参考文献（References）

> 说明：以下参考文献列表保留英文原文，未作翻译。

[1] 2020. GO GC problem in Discord. https://discord.com/blog/why-discord-is-switching-from-go-to-rust.

[2] 2022. Flexible I/O tester. https://github.com/axboe/fio.

[3] 2022. Go-YCSB. https://github.com/pingcap/go-ycsb.

[4] 2023. Badger. https://pkg.go.dev/github.com/dgraph-io/badger.

[5] 2023. Beego. https://github.com/beego/beego.

[6] 2023. Gin Web Framework. https://gin-gonic.com/.

[7] 2023. Go Case Studies. https://go.dev/solutions/#case-studies.

[8] 2023. Rosedb. https://github.com/roseduan/rosedb.

[9] 2023. TiDB: Open, Unified, Distributed SQL. https://www.pingcap.com/.

[10] 2023. Tri Color Marking. https://en.wikipedia.org/wiki/Tracing_garbage_collection#Tri-color_marking.

[11] Daniel S Berger, Benjamin Berg, Timothy Zhu, Siddhartha Sen, and Mor Harchol-Balter. 2018. RobinHood: Tail Latency Aware Caching–Dynamic Reallocation from Cache-Rich to Cache-Poor. In 13th USENIX Symposium on Operating Systems Design and Implementation (OSDI 18). Oct. 8–10, 2018, CARLSBAD, CA, USA.

[12] Rodrigo Bruno, Duarte Patricio, José Simão, Luis Veiga, and Paulo Ferreira. 2019. Runtime object lifetime profiler for latency sensitive big data applications. In Proceedings of the Fourteenth EuroSys Conference 2019 (EuroSys). March 25-28, 2019, Dresden Germany.

[13] Wei Chen, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2019. Os-augmented oversubscription of opportunistic memory with a user-assisted oom killer. In Proceedings of the 20th International Middleware Conference (Middleware). Dec. 9-13, 2019, Davis CA USA.

[14] Wei Chen, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2019. Pufferfish: Container-driven elastic memory management for data-intensive applications. In Proceedings of the ACM Symposium on Cloud Computing (SoCC). Nov. 20-23, 2019, Santa Cruz CA USA.

[15] Wei Chen, Jia Rao, and Xiaobo Zhou. 2017. Preemptive, low latency datacenter scheduling via lightweight virtualization. In 2017 USENIX Annual Technical Conference (USENIX ATC 17). July 12–14, 2017, Santa Clara, CA.

[16] Youmin Chen, Youyou Lu, Fan Yang, Qing Wang, Yang Wang, and Jiwu Shu. 2020. Flatstore: An efficient log-structured key-value storage engine for persistent memory. In Proceedings of the Twenty-Fifth International Conference on Architectural Support for Programming Languages and Operating Systems (ASPLOS). March 16-20, 2020, Lausanne, Switzerland.

[17] Brian F Cooper, Adam Silberstein, Erwin Tam, Raghu Ramakrishnan, and Russell Sears. 2010. Benchmarking cloud serving systems with YCSB. In Proceedings of the 1st ACM symposium on Cloud computing (SoCC). June 10-11, 2010, Indianapolis Indiana USA.

[18] Oracle Corperation. 2016. Java Garbage Collection Stop The World Event. https://www.oracle.com/technetwork/tutorials/tutorials-1873457.html

[19] Oracle Corporation. 2022. The Shenandoah Garbage Collector. https://wiki.openjdk.org/display/shenandoah/Main

[20] Oracle Corporation. 2022. The Z Garbage Collector. https://openjdk.java.net/projects/zgc/

[21] Christina Delimitrou and Christos Kozyrakis. 2013. Paragon: QoS-aware scheduling for heterogeneous datacenters. ACM SIGPLAN Notices (2013).

[22] Christina Delimitrou and Christos Kozyrakis. 2017. Bolt: I know what you did last summer... in the cloud. ACM SIGARCH Computer Architecture News (2017).

[23] Henri Maxime Demoulin, Joshua Fried, Isaac Pedisich, Marios Kogias, Boon Thau Loo, Linh Thi Xuan Phan, and Irene Zhang. 2021. When Idling is Ideal: Optimizing Tail-Latency for Heavy-Tailed Datacenter Workloads with Perséphone. In Proceedings of the ACM SIGOPS 28th Symposium on Operating Systems Principles (SOSP). Oct. 26-29, 2021, Virtual Event Germany.

[24] David Detlefs, Christine Flood, Steve Heller, and Tony Printezis. 2004. Garbage-first garbage collection. In Proceedings of the 4th international symposium on Memory management (ISMM). Oct. 24-25, 2004, Vancouver BC Canada.

[25] Joshua Fried, Zhenyuan Ruan, Amy Ousterhout, and Adam Belay. 2020. Caladan: Mitigating interference at microsecond timescales. In 14th USENIX Symposium on Operating Systems Design and Implementation (OSDI 20). Nov. 4-6, 2020, virtual.

[26] Sanjay Ghemawat and Paul Menage. 2005. TCMalloc: Thread-caching malloc. http://goog-perftools.sourceforge.net/doc/tcmalloc.html.

[27] Ionel Gog, Jana Giceva, Malte Schwarzkopf, Kapil Vaswani, Dimitrios Vytiniotis, Ganesan Ramalingam, Manuel Costa, Derek G Murray, Steven Hand, and Michael Isard. 2015. Broom: Sweeping out garbage collection from big data systems. In 15th Workshop on Hot Topics in Operating Systems (HotOS XV). May 18-20, 2015, Kartause Ittingen, Switzerland.

[28] Călin Iorgulescu, Reza Azimi, Youngjin Kwon, Sameh Elnikety, Manoj Syamala, Vivek Narasayya, Herodotos Herodotou, Paulo Tomita, Alex Chen, Jack Zhang, et al. 2018. PerfIso: Performance Isolation for Commercial Latency-Sensitive Services. In 2018 USENIX Annual Technical Conference (USENIX ATC 18). July 11–13, 2018, Boston, MA, USA.

[29] Harshad Kasture, Davide B Bartolini, Nathan Beckmann, and Daniel Sanchez. 2015. Rubik: Fast analytical power management for latency-critical systems. In 2015 48th Annual IEEE/ACM International Symposium on Microarchitecture (MICRO). Dec. 5-12, 2015, Waikiki, HI, USA. IEEE.

[30] Harshad Kasture and Daniel Sanchez. 2016. Tailbench: a benchmark suite and evaluation methodology for latency-critical applications. In 2016 IEEE International Symposium on Workload Characterization (IISWC). IEEE, 1–10.

[31] David Lo, Liqun Cheng, Rama Govindaraju, Parthasarathy Ranganathan, and Christos Kozyrakis. 2015. Heracles: Improving resource efficiency at scale. In Proceedings of the 42nd Annual International Symposium on Computer Architecture (ISCA). June 13-17, 2015, Portland Oregon.

[32] Lu Lu, Xuanhua Shi, Yongluan Zhou, Xiong Zhang, Hai Jin, Cheng Pei, Ligang He, and Yuanzhen Geng. 2016. Lifetime-Based Memory Management for Distributed Data Processing Systems. Proc. VLDB Endow. (2016).

[33] Martin Maas, Tim Harris, Krste Asanović, and John Kubiatowicz. 2015. Trash day: Coordinating garbage collection in distributed systems. In 15th Workshop on Hot Topics in Operating Systems (HotOS XV). May 18-20, 2015, Kartause Ittingen, Switzerland.

[34] Spyros Makridakis and Michele Hibon. 1997. ARMA models and the Box–Jenkins methodology. Journal of forecasting 16, 3 (1997), 147–163.

[35] Jason Mars and Lingjia Tang. [n.d.]. Whare-map: Heterogeneity in "homogeneous" warehouse-scale computers. In Proceedings of the 40th Annual International Symposium on Computer Architecture (ISCA). June 23-27, 2013, Tel-Aviv Israel.

[36] Kris Mok. 2011. JVM@Taobao. https://www.slideshare.net/RednaxelaFX/jvm-taobao

[37] Khanh Nguyen, Lu Fang, Guoqing Xu, Brian Demsky, Shan Lu, Sanazsadat Alamian, and Onur Mutlu. 2016. Yak: A High-Performance Big-Data-Friendly Garbage Collector. In 12th USENIX Symposium on Operating Systems Design and Implementation (OSDI 16). Nov. 2–4, 2016, SAVANNAH, GA, USA.

[38] Khanh Nguyen, Kai Wang, Yingyi Bu, Lu Fang, Jianfei Hu, and Guoqing Xu. 2015. Facade: A compiler and runtime for (almost) object-bounded big data applications. ACM SIGARCH Computer Architecture News (2015).

[39] Aidi Pi, Junxian Zhao, Shaoqi Wang, and Xiaobo Zhou. 2021. Memory at your service: fast memory allocation for latency-critical services. In Proceedings of the 22nd International Middleware Conference (Middleware). Dec. 6-10, 2021, Québec city Canada.

[40] Aidi Pi, Xiaobo Zhou, and Chengzhong Xu. 2022. Holmes: SMT Interference Diagnosis and CPU Scheduling for Job Co-Location. In Proceedings of the 31st International Symposium on High-Performance Parallel and Distributed Computing.

[41] David K. Rensin. 2015. Kubernetes - Scheduling the Future at Cloud Scale. http://www.oreilly.com/webops-perf/free/kubernetes.csp

[42] Malte Schwarzkopf, Andy Konwinski, Michael Abd-El-Malek, and John Wilkes. 2013. Omega: Flexible, Scalable Schedulers for Large Compute Clusters. In Proceedings of the 8th ACM European Conference on Computer Systems (EuroSys). April 15-17, 2013, Prague Czech Republic.

[43] Abhishek Verma, Luis Pedrosa, Madhukar Korupolu, David Oppenheimer, Eric Tune, and John Wilkes. 2015. Large-Scale Cluster Management at Google with Borg. In Proceedings of the Tenth European Conference on Computer Systems (EuroSys). April 21-24, 2015, Bordeaux France.

[44] Chenxi Wang, Haoran Ma, Shi Liu, Yuanqi Li, Zhenyuan Ruan, Khanh Nguyen, Michael D Bond, Ravi Netravali, Miryung Kim, and Guoqing Harry Xu. 2020. Semeru: A Memory-Disaggregated Managed Runtime. In 14th USENIX Symposium on Operating Systems Design and Implementation (OSDI 20). Nov. 4-6, 2020, virtual.

[45] Chenxi Wang, Haoran Ma, Shi Liu, Yifan Qiao, Jonathan Eyolfson, Christian Navasca, Shan Lu, and Guoqing Harry Xu. 2022. {MemLiner}: Lining up Tracing and Application for a {Far-Memory-Friendly} Runtime. In 16th USENIX Symposium on Operating Systems Design and Implementation (OSDI 22).

[46] Jingjing Wang and Magdalena Balazinska. 2017. Elastic memory management for cloud data analytics. In 2017 USENIX Annual Technical Conference (USENIX ATC 17). July 12–14, 2017, Santa Clara, CA.

[47] Mingyu Wu, Ziming Zhao, Yanfei Yang, Haoyu Li, Haibo Chen, Binyu Zang, Haibing Guan, Sanhong Li, Chuansheng Lu, and Tongbao Zhang. 2020. Platinum: A {CPU-Efficient} Concurrent Garbage Collector for {Tail-Reduction} of Interactive Services. In 2020 USENIX Annual Technical Conference (USENIX ATC 20).

[48] Yanfei Yang, Mingyu Wu, Haibo Chen, and Binyu Zang. 2021. Bridging the performance gap for copy-based garbage collectors atop non-volatile memory. In Proceedings of the Sixteenth European Conference on Computer Systems (EuroSys).

[49] Junxian Zhao, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2021. Flashbyte: Improving memory efficiency with lightweight native storage. In 2021 IEEE/ACM 21st International Symposium on Cluster, Cloud and Internet Computing (CCGrid). May 10-13, 2021, Melbourne, Australia. IEEE.

[50] Junxian Zhao, Aidi Pi, Xiaobo Zhou, Sang-Yoon Chang, and Chengzhong Xu. 2022. Improving Concurrent GC for Latency Critical Services in Multi-tenant Systems. In Proceedings of the 23rd ACM/IFIP International Middleware Conference (Middleware).

[51] Wenyu Zhao, Stephen M. Blackburn, and Kathryn S. McKinley. 2022. Low-Latency, High-Throughput Garbage Collection. In Proceedings of the 43rd ACM SIGPLAN International Conference on Programming Language Design and Implementation (PLDI) (San Diego, CA, USA).

[52] Timothy Zhu, Michael A Kozuch, and Mor Harchol-Balter. 2017. Workload compactor: Reducing datacenter cost while providing tail latency slo guarantees. In Proceedings of the 2017 Symposium on Cloud Computing (SoCC). Sept. 24-27, 2017, Santa Clara California.


