<!-- 中英对照翻译 · 由 text-raw/2022-Zhao-Improving-Concurrent-GC.txt 生成 -->

> **阅读说明**
> 本文为**中英对照**译本：每一段**英文原文**以引用块（行首 `>`）给出，紧随其后的**普通段落**为对应的中文译文，两者逐段一一对应。
> 图表题注采用「中文（English）」双语形式；**参考文献列表保留英文原文，未作翻译**。
> 原文由 PDF 抽取，其中丢失的空格与跨行断词已在英文侧还原。

**原文标题**：Improving Concurrent GC for Latency Critical Services in Multi-tenant Systems
**作者**：Junxian Zhao、Aidi Pi、Xiaobo Zhou（通讯作者）、Sang-Yoon Chang、Chengzhong Xu
**会议**：Middleware '22 — 第 23 届国际中间件会议（23nd International Middleware Conference），2022 年 11 月 7–11 日，加拿大魁北克城
**DOI**：https://doi.org/10.1145/3464298.3493394

---

# 面向多租户系统中延迟关键服务的并发 GC 改进

**Improving Concurrent GC for Latency Critical Services in Multi-tenant Systems**

**Junxian Zhao** — University of Colorado, Colorado Springs, CO, USA
**Aidi Pi** — University of Colorado, Colorado Springs, CO, USA
**Xiaobo Zhou**（通讯作者） — University of Colorado, Colorado Springs, CO, USA
**Sang-Yoon Chang** — University of Colorado, Colorado Springs, CO, USA
**Chengzhong Xu** — University of Macau, Macau, China

## 摘要（ABSTRACT）

> For resource utilization efficiency, latency critical (LC) services are commonly co-located with
> best-effort batch jobs in datacenter servers. Many LC services, such as Cassandra and HBase, run in
> Java Virtual Machine (JVM). We find that LC services often experience heavy-tailed latency due to
> performance interference of the concurrent garbage collection (GC) as well as multi-tenancy.
> The root cause is a semantic gap of resource allocation between JVM and the underlying Linux OS in
> multi-tenant systems. That is, the OS is unaware of the characteristics of different kinds of
> threads in JVM (i.e., GC threads and LC worker threads), which may lead to GC threads competing for
> CPUs; JVM is unaware of the resource utilization in the OS, which may trigger CPU-intensive
> GC operations when CPUs are busy. Furthermore, we find that co-located batch jobs can interfere
> with LC services due to Simultaneous Multi-Threading (SMT).

为了提升资源利用效率，延迟关键（latency critical, LC）服务通常与尽力而为（best-effort）型批处理作业混部在同一台数据中心服务器中。许多 LC 服务（如 Cassandra 和 HBase）运行在 Java 虚拟机（JVM）中。我们发现，由于并发垃圾回收（GC）以及多租户带来的性能干扰，LC 服务经常出现重尾延迟。其根本原因在于多租户系统中 JVM 与底层 Linux 操作系统之间在资源分配上存在语义鸿沟。也就是说，操作系统并不了解 JVM 中不同类型线程（即 GC 线程与 LC 工作线程）的特征，这可能导致 GC 线程争抢 CPU；而 JVM 也不了解操作系统中的资源利用情况，这可能在 CPU 繁忙时触发 CPU 密集型的 GC 操作。此外，我们还发现，混部的批处理作业会通过同时多线程（Simultaneous Multi-Threading, SMT）对 LC 服务造成干扰。

> We propose iGC, a middleware that bridges the semantic gap between JVM and Linux OS and improves
> concurrent GC performance in multi-tenant systems. iGC adaptively triggers GC based on the
> CPU utilization at runtime, which speeds up the GC process and reduces its CPU contention.
> Furthermore, iGC deploys a dynamic CPU scheduling and thread placement strategy to avoid or
> mitigate the interference due to concurrent GC and multi-tenancy, but also improve the cache
> performance. We implement iGC upon two state-of-the-art concurrent GC mechanisms ZGC and G1 GC.
> We conduct its evaluation using three NoSQL databases as LC services. Experimental results show
> that iGC significantly improves the performance of concurrent GC for LC services and the
> throughput in multi-tenant systems. iGC reduces the p95 tail latency by 83%, 37% and 22% for the
> three LC services Cassandra, HBase and Solr, respectively. It also increases the throughput of
> LC services up to 2.56X.

我们提出 iGC，一种弥合 JVM 与 Linux 操作系统之间语义鸿沟、并提升多租户系统中并发 GC 性能的中间件。iGC 依据运行时的 CPU 利用率自适应地触发 GC，从而加快 GC 过程并减少其 CPU 争用。此外，iGC 部署了一种动态 CPU 调度与线程放置策略，以规避或缓解由并发 GC 和多租户引起的干扰，同时改善缓存性能。我们在两种最先进的并发 GC 机制 ZGC 和 G1 GC 之上实现了 iGC。我们使用三种 NoSQL 数据库作为 LC 服务对其进行了评测。实验结果表明，iGC 显著提升了 LC 服务并发 GC 的性能以及多租户系统中的吞吐量。对于 Cassandra、HBase 和 Solr 这三种 LC 服务，iGC 分别将 p95 尾延迟降低了 83%、37% 和 22%。它还将 LC 服务的吞吐量提升至最高 2.56 倍。

> ∗Corresponding author. Email: xzhou@uccs.edu.

∗通讯作者。邮箱：xzhou@uccs.edu。

> Middleware '22, November 7–11, 2022, Quebec City, Quebec, Canada
> © 2022 Association for Computing Machinery.
> ACM ISBN 978-1-4503-8534-3/21/12...$15.00
> https://doi.org/10.1145/3464298.3493394
> This work is licensed under a Creative Commons Attribution International 4.0 License.

Middleware '22，2022 年 11 月 7–11 日，加拿大魁北克市。
© 2022 美国计算机学会（ACM）。
ACM ISBN 978-1-4503-8534-3/21/12...$15.00
https://doi.org/10.1145/3464298.3493394
本作品采用知识共享署名 4.0 国际许可协议（Creative Commons Attribution International 4.0 License）授权。

## CCS 概念（CCS CONCEPTS）

> • Computer systems organization → Cloud computing; • Software and its engineering → Garbage collection.

• 计算机系统组织（Computer systems organization）→ 云计算；• 软件及其工程（Software and its engineering）→ 垃圾回收。

## 关键词（KEYWORDS）

> Garbage collection, Job co-location, Tail latency, Interference

垃圾回收（Garbage collection）、作业混部（Job co-location）、尾延迟（Tail latency）、干扰（Interference）

## ACM 引用格式（ACM Reference Format）

> Junxian Zhao, Aidi Pi, Xiaobo Zhou, Sang-Yoon Chang, and Chengzhong Xu. 2022. Improving
> Concurrent GC for Latency Critical Services in Multi-tenant Systems. In 23nd International
> Middleware Conference (Middleware '22), November 7–11, 2022, Quebec City, Quebec, Canada. ACM,
> New York, NY, USA, 13 pages. https://doi.org/10.1145/3464298.3493394

Junxian Zhao, Aidi Pi, Xiaobo Zhou, Sang-Yoon Chang, and Chengzhong Xu. 2022. Improving Concurrent GC for Latency Critical Services in Multi-tenant Systems. In 23nd International Middleware Conference (Middleware '22), November 7–11, 2022, Quebec City, Quebec, Canada. ACM, New York, NY, USA, 13 pages. https://doi.org/10.1145/3464298.3493394
（中文说明：以上为本文的 ACM 标准引用格式，作者、会议名、出版地与 DOI 等均保留英文原文。）

## 1 引言（Introduction）

> For resource efficiency in cloud computing, latency critical (LC) services such as key-value
> stores and web search are commonly co-located with best-effort batch jobs in datacenter servers.
> However, as applications are co-located and contend for shared resources, multi-tenancy often
> leads to severe interference and unpredictable performance, especially the heavy-tailed latency
> for LC services.

出于云计算中的资源效率考虑，诸如键值存储与网页搜索这类延迟关键（LC）服务，通常与尽力而为型批处理作业混部在数据中心服务器中。然而，随着应用被混部并争抢共享资源，多租户往往会带来严重的干扰与不可预测的性能，尤其是 LC 服务的重尾延迟。

> The heavy-tailed latency problem is significant because cloud applications are undergoing a major
> redesign from large, monolithic services to hundreds or thousands of loosely-coupled
> microservices [6, 25]. A user request of LC services often triggers sub-requests to multiple
> components across many servers and the end-to-end response time is largely determined by the
> slowest individual latency of the sub-requests [17]. As LC services need to maintain stable and
> low latency for user requests, the problem significantly impacts the QoS of cloud computing.

重尾延迟问题之所以重要，是因为云应用正经历一场重大重构：从大型单体服务转向成百上千个松耦合的微服务 [6, 25]。LC 服务的一次用户请求往往会触发跨多台服务器、面向多个组件的子请求，而端到端响应时间在很大程度上取决于这些子请求中最慢的那个单独延迟 [17]。由于 LC 服务需要为用户请求维持稳定且低延迟的表现，该问题显著影响着云计算的 QoS（服务质量）。

> There are growing interests that aim to improve the tail latency of LC services from the
> perspectives of workload placement [55], CPU isolation between LC services and batch jobs [31],
> tail-aware caching [4], and resource partitioning [6, 36]. Recently, Caladan [24] proposes a new
> Linux OS scheduler that reduces the tail latency in multi-tenant systems via fast core allocation
> instead of resource partitioning. Hermes [46] and Perséphone [20] design reservation-based memory
> allocation and kernel-bypass OS scheduling approaches to minimize the tail latency for LC services
> executing at the microsecond scale, respectively.

从负载放置 [55]、LC 服务与批处理作业之间的 CPU 隔离 [31]、尾延迟感知缓存（tail-aware caching）[4] 以及资源划分 [6, 36] 等角度出发，改善 LC 服务尾延迟的工作日益增多。最近，Caladan [24] 提出了一种新的 Linux 操作系统调度器，它通过快速核心分配而非资源划分来降低多租户系统中的尾延迟。Hermes [46] 与 Perséphone [20] 分别设计了基于预留的内存分配方法与内核旁路（kernel-bypass）的操作系统调度方法，以最小化微秒级执行时延的 LC 服务的尾延迟。

> In this paper, we tackle the problem of heavy-tailed latency of LC services from two new aspects.
> First, we identify a new interference causing the heavy-tailed latency in JVM-based LC services,
> that is, the interference between the concurrent GC threads and the worker threads of a LC service.
> Many LC services, such as Cassandra [1], HBase [2], and Solr [3], run in Java Virtual Machine
> (JVM). JVM provides an easy-scalable, generalized platform for different workloads, which fits
> well in cloud environments. While Java provides programmers with convenience such as memory
> management, scalability, and compatibility, it also incurs nontrivial overhead of GC. Traditional
> GC in JVM leads to the Stop The World (STW) problem, which pauses all worker threads [13].
> Concurrent GC, represented by ZGC [16] (since JDK 11) and G1 GC [22] (since JDK 7u4, the default
> in JDK 9), runs most phases of GC concurrently with worker threads of Java applications. It reduces
> the long pause associated with STW and the full GC. However, it creates severe interfere with the
> worker threads of LC services.

在本文中，我们从两个新的方面来应对 LC 服务的重尾延迟问题。第一，我们识别出一种导致基于 JVM 的 LC 服务出现重尾延迟的新干扰，即并发 GC 线程与 LC 服务工作线程之间的干扰。许多 LC 服务，例如 Cassandra [1]、HBase [2] 和 Solr [3]，都运行在 Java 虚拟机（JVM）中。JVM 为不同负载提供了一个易于扩展的通用平台，非常适合云环境。虽然 Java 为程序员带来了内存管理、可扩展性与兼容性等便利，但也不可避免地引入了不可忽视的 GC 开销。JVM 中的传统 GC 会导致 Stop-The-World（STW，全停顿）问题，即暂停所有工作线程 [13]。以 ZGC [16]（自 JDK 11 起）和 G1 GC [22]（自 JDK 7u4 起，JDK 9 中的默认 GC）为代表的并发 GC，使 GC 的大部分阶段与 Java 应用的工作线程并发执行。它减少了与 STW 和 full GC 相关的长时间停顿。然而，它也对 LC 服务的工作线程造成了严重的干扰。

> The root cause is a semantic gap of resource allocation between JVM and the Linux OS in
> multi-tenant systems. 1) JVM is unaware of the available resources in the OS. It triggers GC only
> based on the status of JVM, which often prolongs the execution time of both GC threads and threads
> of batch jobs that share CPUs with the GC threads. 2) The OS is unaware of the characteristics of
> different threads in JVM, which may lead to threads competing for the same set of logical CPUs.
> In such a case, GC threads contend for CPU resource with worker threads of LC services in JVM,
> which prolongs the latency of requests of LC services.

其根本原因在于多租户系统中 JVM 与 Linux 操作系统之间在资源分配上的语义鸿沟。1）JVM 不了解操作系统中可用的资源。它仅依据 JVM 自身的状态触发 GC，这往往会延长 GC 线程以及与 GC 线程共享 CPU 的批处理作业线程的执行时间。2）操作系统不了解 JVM 中不同线程的特征，这可能导致线程争抢同一组逻辑 CPU。在这种情况下，GC 线程会与 JVM 中 LC 服务的工作线程争用 CPU 资源，从而延长 LC 服务请求的延迟。

> Second, servers enable Simultaneous Multi-Threading (SMT) technology to improve system throughput.
> Linux OS may schedule different workloads on the logical CPUs running on one physical core by
> SMT technology. As a result, co-located batch jobs can incur non-trivial SMT interference with
> LC services, leading to cache contention and longer request latency of LC services.

第二，服务器启用同时多线程（SMT）技术以提升系统吞吐量。Linux 操作系统可能借助 SMT 技术，将不同的负载调度到同一个物理核上的多个逻辑 CPU 上运行。其结果是，混部的批处理作业会对 LC 服务造成不可忽视的 SMT 干扰，导致缓存争用以及 LC 服务请求延迟的延长。

> We propose iGC, a middleware that bridges the semantic gap between JVM and the OS and improves
> concurrent GC performance in multi-tenant systems. 1) iGC adaptively triggers GC based on the
> available CPU resource at runtime, which speeds up GC process and reduces its CPU contention.
> 2) iGC deploys a dynamic CPU scheduling and thread placement strategy to avoid or mitigate the
> interference due to concurrent GC and multi-tenancy, but also improve the cache performance at the
> same time.

我们提出 iGC，一种弥合 JVM 与操作系统之间语义鸿沟、并提升多租户系统中并发 GC 性能的中间件。1）iGC 依据运行时可用的 CPU 资源自适应地触发 GC，从而加快 GC 过程并减少其 CPU 争用。2）iGC 部署了一种动态 CPU 调度与线程放置策略，以规避或缓解由并发 GC 和多租户引起的干扰，同时又改善缓存性能。

> Specifically, iGC separates GC threads from worker threads of LC services by assigning different
> CPUs for the GC threads and worker threads, avoiding the interference. To mitigate SMT
> interference, iGC tries to assign threads of batch jobs and worker threads of LC services to
> different cores. Further, iGC limits the resource usage of batch threads when GC threads are
> co-located with batch threads and GC is triggered. As a result, it effectively mitigates the
> contention of memory hierarchy in a multi-tenant system.

具体而言，iGC 通过为 GC 线程与工作线程分配不同的 CPU，将 GC 线程与 LC 服务的工作线程隔离开来，从而避免干扰。为缓解 SMT 干扰，iGC 尝试将批处理作业的线程与 LC 服务的工作线程分配到不同的核上。此外，当 GC 线程与批处理线程混部且 GC 被触发时，iGC 会限制批处理线程的资源使用量。因此，它有效缓解了多租户系统中存储层次的争用。

> We implement iGC as a middleware upon two state-of-the-art concurrent GC mechanisms, ZGC and
> G1 GC. We conduct performance evaluation using three representative NoSQL databases Cassandra,
> HBase, and Solr as the LC services and Spark KMeans as the co-located batch jobs. Experimental
> results show that compared to ZGC, iGC reduces the p95 latency by 83%, 37% and 22% for Cassandra,
> HBase, and Solr, respectively. It reduces the L1 and L2 cache miss rate by 55.7% and 54.4%,
> respectively. It also increases the instruction throughput of the LC services up to 2.56X.
> Compared to G1 GC, iGC reduces the p95 latency by 78% and improves the instruction throughput
> up to 1.45X for Cassandra.

我们将 iGC 作为中间件实现于两种最先进的并发 GC 机制之上，即 ZGC 与 G1 GC。我们使用三种有代表性的 NoSQL 数据库 Cassandra、HBase 和 Solr 作为 LC 服务，并使用 Spark KMeans 作为混部的批处理作业，开展了性能评测。实验结果表明，与 ZGC 相比，iGC 对 Cassandra、HBase 和 Solr 分别将 p95 延迟降低了 83%、37% 和 22%。它将 L1 和 L2 缓存缺失率分别降低了 55.7% 和 54.4%。它还将 LC 服务的指令吞吐量提升至最高 2.56 倍。与 G1 GC 相比，iGC 对 Cassandra 将 p95 延迟降低了 78%，并将指令吞吐量提升至最高 1.45 倍。

**图 1：Cassandra 操作在运行时的延迟（Fig. 1: Latency of Cassandra operations at runtime）**

> The rest of this paper is organized as follows. Section 2 introduces the background and
> motivational studies. Sections 3 and 4 describe iGC design and implementation, respectively.
> Section 5 presents the testbed setup, experimental results and analysis. Related work is presented
> in Section 6. We conclude the work in Section 7.

本文余下部分的组织如下。第 2 节介绍背景与动机研究。第 3 节和第 4 节分别描述 iGC 的设计与实现。第 5 节给出测试平台搭建、实验结果与分析。相关工作在第 6 节介绍。我们在第 7 节总结全文。

## 2 背景与动机（Background and Motivations）

### 2.1 并发 GC 干扰（Concurrent GC Interference）

> Though concurrent GC is effective in mitigating the STW overhead, GC threads compete CPU
> resources with other threads which can severely increase the latency of LC services and reduce the
> system throughput. In a multi-tenant system, GC threads compete with threads of multiple types of
> workloads. 1) Different threads within one JVM compete for CPUs, that is, concurrent GC threads
> compete with worker threads, which prolongs the request latency of the LC service. 2) Threads of
> batch jobs (batch threads) in one JVM share and compete for CPUs with GC threads of a LC service
> in another JVM, which prolongs the GC process.

尽管并发 GC 在缓解 STW 开销方面是有效的，但 GC 线程会与其他线程争用 CPU 资源，这可能严重增加 LC 服务的延迟并降低系统吞吐量。在多租户系统中，GC 线程会与多种类型负载的线程发生争用。1）同一个 JVM 内的不同线程争抢 CPU，即并发 GC 线程与工作线程争用，从而延长了 LC 服务请求的延迟。2）一个 JVM 中的批处理作业线程（batch threads）与另一个 JVM 中 LC 服务的 GC 线程共享并争抢 CPU，从而延长了 GC 过程。

> To study the impact of concurrent GC on latency of LC services, we run Cassandra, a NoSQL
> database, as the LC service on Java 11 with ZGC. We use YCSB [12] as the benchmark to send
> requests to the LC service. The experimental setup is same as that in Section 5.1. We compare the
> default ZGC and ZGC with adaptive thread placement. For the default ZGC, GC threads may be
> scheduled on the same set of CPUs that host worker threads of Cassandra. For the ZGC with adaptive
> thread placement, GC threads are assigned to CPUs that do not host worker threads of Cassandra.

为了研究并发 GC 对 LC 服务延迟的影响，我们在 Java 11 上以 ZGC 运行 NoSQL 数据库 Cassandra 作为 LC 服务。我们使用 YCSB [12] 作为基准测试工具，向该 LC 服务发送请求。实验环境与 5.1 节相同。我们对比了默认的 ZGC 与采用自适应线程放置的 ZGC。对于默认的 ZGC，GC 线程可能被调度到承载 Cassandra 工作线程的同一组 CPU 上。对于采用自适应线程放置的 ZGC，GC 线程被分配到不承载 Cassandra 工作线程的 CPU 上。

> Figure 1(a) shows the latency of the Read and Update operations of Cassandra at runtime. Compared
> to ZGC with adaptive thread placement, the default ZGC leads to more frequent GC activities and
> longer GC time. The reason is that worker threads and GC threads share the same set of CPUs,
> leading to severe contention. The longest latencies are due to the ZGC stalls when there is not
> enough memory on the JVM heap for allocation before the GC is finished. Separating ZGC threads
> and placing them onto CPUs that do not host worker threads of Cassandra can render smaller and
> more stable latency. By placing GC threads and worker threads onto different CPUs, GC runs faster
> without CPU contention which avoids the ZGC allocation stall. As Cassandra employs the
> log-structured merge-tree, the latency of Update operations is relatively more fluctuating due to
> write amplification in which one write operation triggers multiple times of physical writes to the
> disk.

图 1(a) 展示了运行时 Cassandra 的 Read 与 Update 操作的延迟。与采用自适应线程放置的 ZGC 相比，默认的 ZGC 导致更频繁的 GC 活动与更长的 GC 时间。原因在于工作线程与 GC 线程共享同一组 CPU，导致了严重的争用。最长的那些延迟是由 ZGC 停顿引起的：在 GC 完成之前，JVM 堆上没有足够的内存可供分配。将 ZGC 线程分离出来，并把它们放到不承载 Cassandra 工作线程的 CPU 上，可以获得更小且更稳定的延迟。通过将 GC 线程与工作线程放到不同的 CPU 上，GC 在无 CPU 争用的情况下运行得更快，从而避免了 ZGC 的分配停顿。由于 Cassandra 采用日志结构合并树（log-structured merge-tree），Update 操作的延迟波动相对更大，这是由写放大造成的——一次写操作会触发多次对磁盘的物理写入。

**图 2：GC、CPU 利用率与尾延迟之间的相关性（Fig. 2: Correlation of GC, CPU utilization, and tail latency）**

> We further illustrate the correlation of GC triggering with the CPU utilization and latency of
> Cassandra. We use YCSB workload A that consists of 50% of Read and 50% of Update operations. We
> set the number of requests to 4,000 operations per second. Figure 2 shows the request latency and
> CPU utilization at runtime. The average request latency during the runtime is about 0.35 ms. A
> full GC is triggered around the 14th second and lasts about 2.4 seconds (the red line). During the
> GC process, the CPU usage spikes to nearly 100%, more than 30% higher than the usual. Meanwhile,
> the latency also increases significantly. There are two peaks of latency occurring after the GC
> start point and before the GC end point due to STWs. The first peak of latency is around 119 ms.
> After the first STW, it is the concurrent marking phase of GC. During this period, the average
> latency jumps to around 3 ms, which is about 9x larger than the average latency. The second STW is
> less severe, causing the request latency to 16 ms. Overall, though concurrent GC reduces the STW
> overhead, the CPU interference during the concurrent marking phase leads to severe degradation of
> LC service latency.

我们进一步说明 GC 触发与 Cassandra 的 CPU 利用率及延迟之间的相关性。我们使用 YCSB 的 workload A，它由 50% 的 Read 操作与 50% 的 Update 操作组成。我们将请求速率设为每秒 4,000 次操作。图 2 展示了运行时的请求延迟与 CPU 利用率。运行期间的平均请求延迟约为 0.35 ms。在第 14 秒左右触发了一次 full GC，持续约 2.4 秒（红线）。在 GC 过程中，CPU 使用率飙升至接近 100%，比平时高出 30% 以上。与此同时，延迟也显著上升。由于 STW，在 GC 起点之后与 GC 终点之前出现了两个延迟峰值。第一个延迟峰值约为 119 ms。第一次 STW 之后是 GC 的并发标记阶段。在此期间，平均延迟跃升至约 3 ms，约为平均延迟的 9 倍。第二次 STW 的影响较轻，使请求延迟达到 16 ms。总体而言，尽管并发 GC 降低了 STW 开销，但并发标记阶段的 CPU 干扰仍导致 LC 服务延迟严重劣化。

### 2.2 SMT/超线程干扰（SMT/Hyper-Threading Interference）

> SMT has been widely enabled in modern multi-core processors. It improves the system throughput by
> scheduling instructions from multiple hardware threads (each is called a logical CPU) in one
> physical core, which can concurrently utilize different execution units on the core. In this
> paper, we target Intel's implementation of SMT, which is named Hyper-Threading (HT). We use CPU to
> represent logical CPU and core to represent physical core. HT implements a 2-way hardware threads
> on a physical core. Two CPUs on a core share the same pipeline and L1, L2 caches.

SMT 已在现代多核处理器中被广泛启用。它通过在一个物理核上调度来自多个硬件线程（每个硬件线程称为一个逻辑 CPU）的指令来提升系统吞吐量，从而可以并发地利用该核上的不同执行单元。在本文中，我们针对的是 Intel 对 SMT 的实现，即超线程（Hyper-Threading, HT）。我们用 CPU 表示逻辑 CPU，用核（core）表示物理核。HT 在一个物理核上实现 2 路硬件线程。一个核上的两个 CPU 共享同一条流水线以及 L1、L2 缓存。

> While HT technique improves the resource efficiency, it also incurs non-trivial overhead to the
> both hardware threads on the core for two reasons. 1) The core is unaware of the hardware threads
> from which instructions come. There is not a mechanism that can prioritize a thread with a better
> opportunity for occupying the pipeline. When two threads run on the two CPUs of the same core,
> each thread executes instructions slower compared to that one thread runs exclusively on the core.
> The contention of execution units on the core delays the execution of both threads. 2) Contention
> also occurs in memory hierarchy. Two CPUs of the core share resources, especially the memory
> hierarchy, i.e., L1, L2 caches. As more cache misses lead to more accesses to the next level of
> the memory hierarchy, the execution of instructions is prolonged which increases the execution
> time of batch jobs or LC services [29].

尽管 HT 技术提升了资源效率，但它也会给该核上的两个硬件线程带来不可忽视的开销，原因有二。1）核并不知道指令来自哪个硬件线程。目前并不存在一种机制，能够优先照顾某个更有机会占用流水线的线程。当两个线程运行在同一个核的两个 CPU 上时，相较于一个线程独占该核运行的情形，每个线程执行指令的速度都会变慢。核上执行单元的争用会延迟两个线程的执行。2）存储层次上同样存在争用。一个核的两个 CPU 共享资源，尤其是存储层次，即 L1、L2 缓存。由于更多的缓存缺失会导致更多对存储层次下一级的访问，指令的执行被拖长，从而增加了批处理作业或 LC 服务的执行时间 [29]。

**图 3：Cassandra 全部请求的延迟 CDF（单位：μs）（Fig. 3: The latency CDF of all requests (μs) for Cassandra）**

> We use a case study to show the impact of HT on latency of LC services in two scenarios. HT is
> enabled: worker threads of Cassandra and batch threads of KMeans run on two CPUs of the same core,
> respectively. HT is disabled: worker threads of Cassandra run on one CPU while the other CPU of
> the core is idle. Figure 3 shows the latency of Read and Update operations of Cassandra with HT
> enabled and disabled. We observe that about 5% of operations are significantly delayed when HT is
> enabled. The latency impact on LC services is due to the two identified causes, i.e., contention
> of the execution units and contention in the memory hierarchy.

我们用一个案例研究来展示 HT 对 LC 服务延迟的影响，涵盖两种场景。启用 HT：Cassandra 的工作线程与 KMeans 的批处理线程分别运行在同一个核的两个 CPU 上。禁用 HT：Cassandra 的工作线程运行在一个 CPU 上，而该核的另一个 CPU 处于空闲状态。图 3 展示了启用与禁用 HT 时 Cassandra 的 Read 和 Update 操作的延迟。我们观察到，在启用 HT 时，约 5% 的操作被显著延迟。HT 对 LC 服务延迟的影响源于前述两个已识别的原因，即执行单元的争用与存储层次中的争用。

> Summary. We summarize the identified issues as below. 1) Garbage Collection is a CPU intense
> operation. Thus, it should be adaptively triggered based on the CPU resource at runtime.
> 2) Although concurrent GC reduces the stall of applications, it leads to more CPU interference as
> it runs in parallel with worker threads of the same process. 3) Assigning GC threads and batch
> threads on the CPUs of the same core severely degrades the tail latency of LC services due to
> CPU contention and HT interference.

小结。我们将所识别的问题总结如下。1）垃圾回收是一种 CPU 密集型操作。因此，它应当依据运行时的 CPU 资源情况自适应地触发。2）尽管并发 GC 减少了应用的停顿，但由于它与同一进程的工作线程并行运行，会带来更多的 CPU 干扰。3）将 GC 线程与批处理线程分配到同一个核的多个 CPU 上，会因 CPU 争用与 HT 干扰而严重劣化 LC 服务的尾延迟。


## 3 iGC 设计（iGC Design）

> We design iGC, a middleware that improves concurrent GC performance in multi-tenant systems and reduces the latency of LC services by addressing the identified issues. iGC centers on two key techniques. 1) Adaptive GC triggering. iGC triggers GC adaptively based on the available CPU resource at runtime to speeds up GC procedure and reduce its CPU contention. It monitors the CPU usage of servers and dynamically adjusts the time interval between GC operations based on the heap usage of the JVM. 2) Dynamic CPU allocation and thread placement. iGC deploys a simple yet effective priority-based dynamic CPU allocation and thread placement strategy to reduce the interference of job co-location and improve the cache performance by leveraging HT.

我们设计了 iGC，一种中间件，它通过解决上述已识别的问题，提升多租户系统中并发 GC 的性能，并降低延迟关键（latency critical, LC）服务的延迟。iGC 围绕两项关键技术展开。1）自适应 GC 触发。iGC 在运行时根据可用的 CPU 资源自适应地触发 GC，以加速 GC 过程并减少其 CPU 争用。它监控服务器的 CPU 使用率，并根据 JVM 的堆使用情况动态调整 GC 操作之间的时间间隔。2）动态 CPU 分配与线程放置。iGC 部署了一种简单而有效的、基于优先级的动态 CPU 分配与线程放置策略，以减少作业混部带来的干扰，并借助超线程（HT）提升缓存性能。

**图 4：iGC 的架构与五种 CPU 及线程放置场景（Fig. 4: The architecture of iGC with five CPU and thread placement scenarios）**

图中标注：Monitor（监控器）、Long Term Controller（长期控制器）、Short Term Controller（短期控制器）、Trigger GC（触发 GC）、JMX、CPU Usage Limit（CPU 使用限制）、CPU Reschedule Threads（CPU 重调度线程）、Control Flow（控制流）、Data Flow（数据流）、GC Thread（GC 线程）、Worker Thread（工作线程）、Batch Thread（批处理线程）、CPU、Core（核心）；编号 1–5 对应五个 JVM / 核心 / CPU 的放置场景。

> Figure 4 shows the architecture of iGC. iGC consists of three units: 1) Monitor that obtains server resource information, i.e., heap size of the target application, CPU utilization for each core on the server. It uses samples to calculate the speed at which the heap size grows. 2) Short term controller that triggers GC based on the information from the other two units. It also limits the resources used by the batch threads when necessary. 3) Long term controller that decides the parameters related to GC triggering, such as thresholds of heap size, CPU utilization and time interval of GC, and importantly CPU allocation and thread placement. A JVM is the runtime for a LC service. It contains worker threads and GC threads. In addition, there are batch threads for batch jobs. There are five thread placement scenarios on the two CPUs of a core, i.e., worker and worker, batch and batch, worker and GC, worker and batch with GC, and batch and batch with GC. In case of virtualized cloud environments, each virtual machine (VM) hosts one iGC that makes GC triggering and CPU scheduling decisions for the hosted applications. Each VM can access multiple cores on the server. Note that scenario 3 only occurs when resources are plenty, otherwise, it will be transformed to scenario 4 or 5 for higher resource utilization.

图 4 展示了 iGC 的架构。iGC 由三个单元组成：1）Monitor（监控器），用于获取服务器资源信息，即目标应用的堆大小、服务器上每个核心的 CPU 利用率。它利用采样来计算堆大小的增长速度。2）Short term controller（短期控制器），根据另外两个单元提供的信息触发 GC。它还会在必要时限制批处理线程所使用的资源。3）Long term controller（长期控制器），决定与 GC 触发相关的参数，例如堆大小阈值、CPU 利用率阈值和 GC 的时间间隔，以及十分重要的 CPU 分配与线程放置。一个 JVM 是某个 LC 服务的运行时，其中包含工作线程和 GC 线程。此外，还存在用于批处理作业的批处理线程。在一个核心的两个 CPU 上存在五种线程放置场景，即 worker and worker、batch and batch、worker and GC、worker and batch with GC、batch and batch with GC。在虚拟化云环境中，每台虚拟机（VM）承载一个 iGC，为所承载的应用做出 GC 触发与 CPU 调度决策。每台 VM 可以访问服务器上的多个核心。注意，场景 3 仅在资源充足时出现，否则它会被转换为场景 4 或 5，以获得更高的资源利用率。

### 3.1 监控器（Monitor）

> The monitor periodically collects the resource utilization information including CPU usage and heap usage of a target JVM. The CPU usage is obtained from the file system in Linux OS. The heap usage of the JVM is obtained through the Java Management Extensions (JMX) framework introduced since Java 1.5 [15]. JMX is an out-of-the-box management tool for monitoring and managing JVM. The monitor also tracks the allocation of objects on the Java heap and computes the allocation rate. The obtained information is sent to the short-term controller and the long-term controller.

监控器周期性地采集目标 JVM 的资源利用率信息，包括 CPU 使用率和堆使用情况。CPU 使用率从 Linux 操作系统的文件系统中获取。JVM 的堆使用情况通过自 Java 1.5 起引入的 Java Management Extensions（JMX）框架获取 [15]。JMX 是一种开箱即用的管理工具，用于监控和管理 JVM。监控器还会跟踪 Java 堆上对象的分配情况并计算分配速率。所获取的信息被发送给短期控制器和长期控制器。

### 3.2 短期控制器（Short Term Controller）

> The controller manages the adaptive GC triggering procedure. The logic is shown in Algorithm 1. It triggers GC based on CPU usage, heap usage, and interval set by Monitor and the long term controller. It limits the CPU usage of batch threads on the allocated CPU where GC threads are running (scenarios 4 and 5 in Figure 4). One naive way is to remove the allocated CPU from the affinity list of batch threads. Here, we choose to set low priority in CPU usage for batch threads so they can be resumed when GC is finished.

该控制器管理自适应 GC 触发流程。其逻辑如算法 1 所示。它根据 CPU 使用率、堆使用情况，以及由 Monitor 和长期控制器设定的时间间隔来触发 GC。它会限制运行 GC 线程的已分配 CPU 上批处理线程的 CPU 使用量（图 4 中的场景 4 和 5）。一种朴素的做法是把该已分配 CPU 从批处理线程的亲和性列表中移除。此处我们选择为批处理线程设置较低的 CPU 使用优先级，以便在 GC 结束后它们能够恢复运行。

#### 3.2.1 如何触发 GC（How to trigger GC）

> iGC decides how to trigger GC at runtime. Instead of modifying the JVM directly, we access JVM information and control GC via JMX. JMX obtains the status of Java heap and triggers GC via call system.gc() from outside of the JVM. Algorithm 1 shows the operations when GC is triggered. By default, GC threads may share a CPU with batch threads. If the JVM is started with root privilege, GC is triggered directly as the OS scheduler will treat it with high priority. If not, GC is triggered if cpu_usage of the CPU where GC threads run is smaller than cpu_threshold, which is empirically set to 70% to reduce the CPU contention between GC and batch threads. Note that with iGC, the GC CPU usage on each assigned CPU is around 30%. To speed up GC operations, we set GC threads with a high priority on the CPUs where GC is triggered. If the CPU resource is not enough, iGC limits the CPU usage of batch threads sharing the same CPU with GC threads by cgroup [11].

iGC 在运行时决定如何触发 GC。我们并不直接修改 JVM，而是通过 JMX 访问 JVM 信息并控制 GC。JMX 获取 Java 堆的状态，并通过从 JVM 外部调用 system.gc() 来触发 GC。算法 1 给出了 GC 被触发时的操作。默认情况下，GC 线程可能与批处理线程共享同一个 CPU。如果 JVM 以 root 权限启动，则直接触发 GC，因为操作系统调度器会以高优先级对待它。否则，只有当运行 GC 线程的 CPU 的 cpu_usage 小于 cpu_threshold 时才触发 GC；该阈值根据经验设为 70%，以减少 GC 与批处理线程之间的 CPU 争用。注意，在使用 iGC 时，每个被分配的 CPU 上 GC 的 CPU 使用率约为 30%。为加速 GC 操作，我们在触发 GC 的 CPU 上为 GC 线程设置高优先级。如果 CPU 资源不足，iGC 通过 cgroup [11] 限制与 GC 线程共享同一 CPU 的批处理线程的 CPU 使用量。

**算法 1：触发 GC 的流程（Algorithm 1: The procedure of triggering GC）**

```text
Algorithm 1 The procedure of triggering GC.
1: 𝑐𝑝𝑢_𝑡ℎ𝑟𝑒𝑠ℎ𝑜𝑙𝑑: the CPU usage threshold;
2: 𝑔𝑐_𝑡ℎ𝑟𝑒𝑎𝑑𝑠: the list of GC threads;
3: 𝑏𝑎𝑡𝑐ℎ_𝑡ℎ𝑟𝑒𝑎𝑑𝑠: the list of batch threads sharing CPU with GC threads;
4: 𝑚𝑜𝑑𝑖𝑓 𝑦_𝑏𝑎𝑡𝑐ℎ: a flag to set low priority for batch threads;
5: function Trigger_GC(𝑐𝑝𝑢_𝑢𝑠𝑎𝑔𝑒)
6:   if isRunningWithRootPrivilege() then
7:     do GC()
8:     return
9:   end if
10:  if 𝑐𝑝𝑢_𝑢𝑠𝑎𝑔𝑒 < 𝑐𝑝𝑢_𝑡ℎ𝑟𝑒𝑠ℎ𝑜𝑙𝑑 then
11:    limitCpuUsage(𝑏𝑎𝑡𝑐ℎ_𝑡ℎ𝑟𝑒𝑎𝑑𝑠);
12:    𝑚𝑜𝑑𝑖𝑓 𝑦_𝑏𝑎𝑡𝑐ℎ = 1; //set low priority to batch threads
13:  end if
14:  prioritizeThreads(𝑔𝑐_𝑡ℎ𝑟𝑒𝑎𝑑𝑠);
15:  do GC()
16:  if 𝑚𝑜𝑑𝑖𝑓 𝑦_𝑏𝑎𝑡𝑐ℎ == 1 then
17:    resumeCpuUsage(𝑏𝑎𝑡𝑐ℎ_𝑡ℎ𝑟𝑒𝑎𝑑𝑠);
18:  end if
19:  deprioritizeThreads(𝑔𝑐_𝑡ℎ𝑟𝑒𝑎𝑑𝑠);
20:  return
21: end function
```

算法 1 变量与流程说明：cpu_threshold 为 CPU 使用率阈值；gc_threads 为 GC 线程列表；batch_threads 为与 GC 线程共享 CPU 的批处理线程列表；modify_batch 为是否将批处理线程设为低优先级的标志。具体流程为：若进程以 root 权限运行，则直接执行 GC 并返回；否则，当 cpu_usage 小于 cpu_threshold 时，限制批处理线程的 CPU 使用量并置 modify_batch = 1（为批处理线程设置低优先级）；随后提升 GC 线程的优先级、执行 GC，若 modify_batch == 1 则恢复批处理线程的 CPU 使用量；最后把 GC 线程降回原优先级并返回。

#### 3.2.2 何时触发 GC（When to trigger GC）

> There are two conditions for the timing of triggering GC, 1) There is enough CPU resource so it proactively triggers GC, and 2) the heap usage increases beyond a threshold such that GC must be triggered to avoid the allocation stall. iGC keeps monitoring the heap usage of the JVM and the CPU usage of the server. It periodically checks whether GC should be triggered by a dynamic time interval I and two thresholds. Threshold A is a threshold for the heap usage, below which GC is not triggered as there is enough heap space for allocation. iGC tries to trigger GC when heap_usage is larger than threshold A. In this case, if heap_usage is not larger than threshold B, it triggers GC if cpu_usage is smaller than cpu_threshold. If heap_usage is larger than threshold B, GC is triggered regardless of the value of cpu_usage. Using two thresholds reduces the interference to batch threads. To ensure iGC is properly working, the default concurrent GC is disabled. Note when the heap size reaches a very high level between iGC samplings, the JVM can still trigger GC to avoid the stall of LC services.

触发 GC 的时机有两个条件：1）有足够的 CPU 资源，因此它主动触发 GC；2）堆使用量增长超过某一阈值，此时必须触发 GC 以避免分配停顿。iGC 持续监控 JVM 的堆使用量和服务器的 CPU 使用率。它通过一个动态时间间隔 I 和两个阈值周期性地检查是否应当触发 GC。阈值 A 是堆使用量的阈值，低于该阈值时不触发 GC，因为仍有足够的堆空间用于分配。当 heap_usage 大于阈值 A 时，iGC 尝试触发 GC。在这种情况下，如果 heap_usage 不大于阈值 B，则在 cpu_usage 小于 cpu_threshold 时触发 GC。如果 heap_usage 大于阈值 B，则无论 cpu_usage 取值如何都会触发 GC。使用两个阈值可以减少对批处理线程的干扰。为确保 iGC 正常工作，默认的并发 GC 被禁用。注意，当堆大小在两次 iGC 采样之间达到很高的水平时，JVM 仍可自行触发 GC，以避免 LC 服务的停顿。

**表 1：Cassandra 的 JVM 中一个典型的线程列表（Table 1: A typical thread list from the JVM of Cassandra）**

| Label 标签 | Worker Thread & JVM Priority 工作线程与 JVM 优先级 | GC Thread GC 线程 |
| --- | --- | --- |
| (5)RMI TCP Accept | 5 | ZGC: |
| (5)MemtableFlushWriter | 5 | ZDriector |
| (5)MemtableReclaimMemory | 5 | ZDriver |
| (5)SecondaryIndexManagement | 5 | ZStat |
| (5)ScheduledTasks | 5 | ZWorker#0∼n |
| (5)NonPeriodicTasks | 5 | Sweeper thread |
| (5)logback#1∼n | 5 | G1GC: |
| (5)RuntimeWorker#0∼n | 5 | GCThread#0∼n |
| (9)Signal Dispatcher | 9 | G1 Conc#0∼n |
| (9)Attach Listener | 9 | G1 Refine#0∼n |
| (9)C1CompilerThread | 9 | G1 RemSet Sampling |
| (9)C2CompilerThread | 9 | |
| (9)Service Thread | 9 | |
| (9)VM Thread | 9 | |
| (9)VM Periodic Task Thread | 9 | |
| (10) Reference Handler | 10 | |
| (10) Process Reaper | 10 | |
| (10) epollEventLoopGroup | 10 | |
| JVM Priority JVM 优先级 | 5 ∼10 | 10 |
| OS Priority 操作系统优先级 | 0 | 0 |

### 3.3 长期控制器（Long Term Controller）

> This unit has two key functions, dynamic CPU allocation and thread placement and adaptive monitoring interval. The former addresses non-trivial CPU contentions between GC threads and worker threads and between batch threads and worker threads. It also leverages HT for cache performance improvement. The latter controls parameters used in GC, including the heap usage threshold, the CPU usage threshold, and the number of GC threads.

该单元有两个关键功能：动态 CPU 分配与线程放置，以及自适应监控间隔。前者解决 GC 线程与工作线程之间、以及批处理线程与工作线程之间不可忽视的 CPU 争用问题，同时还借助 HT 来提升缓存性能。后者控制 GC 中所使用的参数，包括堆使用阈值、CPU 使用率阈值以及 GC 线程的数量。

#### 3.3.1 基于优先级的动态线程放置（Priority-based Dynamic Thread Placement）

> There is not a thread scheduler in JVM itself. A JVM sets different priority levels to Java threads and creates an OS-level thread for each Java thread. The Linux OS schedules Java threads based on their priority.

JVM 本身并没有线程调度器。JVM 为 Java 线程设置不同的优先级，并为每个 Java 线程创建一个操作系统级线程。Linux 操作系统依据优先级来调度这些 Java 线程。

> **JVM Thread Priority.** All GC threads are initialized when a JVM starts runtime. We use command jps to get process ID and command jstack to capture the thread dump of the JVM and obtain thread labels and IDs. Table 1 shows the label and priority level of all GC threads and representative worker threads in the JVM of Cassandra. Note that ZGC and G1 GC use different labels for GC threads. We can see that most of the worker threads have JVM priority levels from 5 to 9, except for the last three threads that have the highest priority level 10. All GC threads have the highest JVM priority level 10. The last row of the table shows that all JVM threads are assigned the priority level 0 by the OS when the JVM is initiated at user space. As a result, for the JVM running in user space, GC threads and worker threads are treated equally by the Linux OS scheduler and the use of round-robin time-slicing based scheduling. When the JVM is initiated with root privilege, the priority levels of the threads are converted to the OS-level priority levels by the JVM and the OS schedules the threads based on their priority.

**JVM Thread Priority（JVM 线程优先级）。** 所有 GC 线程都在 JVM 启动运行时被初始化。我们使用命令 jps 获取进程 ID，使用命令 jstack 抓取 JVM 的线程转储，从而获得线程标签与 ID。表 1 给出了 Cassandra 的 JVM 中所有 GC 线程和代表性工作线程的标签与优先级。注意 ZGC 与 G1 GC 对 GC 线程使用不同的标签。可以看到，大多数工作线程的 JVM 优先级为 5 到 9，只有最后三个线程具有最高优先级 10。所有 GC 线程都具有最高的 JVM 优先级 10。表的最后一行表明，当 JVM 在用户空间启动时，操作系统为所有 JVM 线程分配优先级 0。因此，对于运行在用户空间的 JVM，GC 线程与工作线程被 Linux 操作系统调度器同等对待，并采用基于时间片轮转的调度方式。当 JVM 以 root 权限启动时，线程的优先级会被 JVM 转换为操作系统级优先级，操作系统据此调度线程。

> Under both scenarios, the OS is incapable of avoiding or mitigating the interference between the GC threads and worker threads by its scheduler itself. If GC threads are assigned with a higher priority level than those of worker threads, worker threads will be preempted or reduced time slice size when GC is triggered, which prolongs the request latency. If GC threads are assigned with a lower priority, it may lead to expensive STW problems or even OutOfMemory errors.

在上述两种情形下，操作系统都无法仅依靠其调度器来避免或缓解 GC 线程与工作线程之间的干扰。如果 GC 线程被赋予比工作线程更高的优先级，那么在 GC 被触发时工作线程会被抢占或时间片被缩短，从而延长请求延迟。如果 GC 线程被赋予较低的优先级，则可能导致代价高昂的 STW（全停顿）问题，甚至引发 OutOfMemory 错误。

> **iGC Dynamic Thread Placement Strategy.** To avoid or mitigate the interference due to concurrent GC and job co-location but also increase the cache performance, iGC places three types of threads on each core with five scenarios in order, which are illustrated at the left side of Figure 4. 1) It assigns both CPUs of one core to worker threads, which avoids the interference from batch threads and makes use of the memory hierarchy of the core. 2) It assigns both CPUs of one core to batch threads, which avoids the impact on worker threads. 3) When there are enough CPUs, worker threads and GC threads are assigned to the two CPUs of one core, respectively. 4) When CPUs are not enough, GC threads share the CPU with batch threads on one CPU, while worker threads use the other CPU. 5) In the worst scenario, GC threads share one CPU with batch threads, while other batch threads use the other CPU of the same core.

**iGC Dynamic Thread Placement Strategy（iGC 动态线程放置策略）。** 为了避免或缓解并发 GC 与作业混部所带来的干扰，同时提升缓存性能，iGC 按顺序在每个核心上放置三类线程，共五种场景，如图 4 左侧所示。1）将一个核心的两个 CPU 都分配给工作线程，从而避免来自批处理线程的干扰，并利用该核心的存储层次。2）将一个核心的两个 CPU 都分配给批处理线程，从而避免对工作线程造成影响。3）当 CPU 充足时，将工作线程与 GC 线程分别分配到同一个核心的两个 CPU 上。4）当 CPU 不足时，GC 线程在一个 CPU 上与批处理线程共享该 CPU，而工作线程使用另一个 CPU。5）在最差的情况下，GC 线程与批处理线程共享一个 CPU，而其他批处理线程使用同一核心的另一个 CPU。

> The CPU and thread placement is dynamic. Firstly, iGC places worker threads and batch threads into different cores (scenarios 1 and 2). If the CPU usage of the LC service is higher than 90% during a sampling interval, one more core is assigned to the LC service. If there are more batch jobs and the CPU usage is heavy on batch threads, iGC co-locates GC threads with batch threads on one CPU (scenarios 4 and 5) by orchestrating with the Linux OS scheduler. We discuss three possible co-location scenarios at the CPU level: 1) Worker threads and GC threads, 2) batch threads and GC threads, and 3) Worker threads and batch threads.

CPU 与线程的放置是动态的。首先，iGC 将工作线程与批处理线程放入不同的核心（场景 1 和 2）。如果在一个采样间隔内 LC 服务的 CPU 使用率高于 90%，则为该 LC 服务再分配一个核心。如果批处理作业更多、且批处理线程上的 CPU 使用率很高，iGC 通过与 Linux 操作系统调度器协同，把 GC 线程与批处理线程混部到同一个 CPU 上（场景 4 和 5）。我们在 CPU 层面讨论三种可能的混部情形：1）工作线程与 GC 线程；2）批处理线程与 GC 线程；3）工作线程与批处理线程。

> **Worker & GC** Worker threads may compete with GC threads when they are placed on the same CPU. However, Linux OS does not distinguish the two types of threads at runtime. To address the problem, we collect information of Java threads from the JVM as each thread has its own label. Each Java thread in the JVM has its corresponding system thread in Linux. Those system threads share one process identification (PID). With the obtained labels, iGC distinguishes the two types of threads and pins them onto different CPUs, which avoids the contention between the worker threads and GC threads.

**Worker & GC（工作线程与 GC 线程）。** 当工作线程与 GC 线程被放置在同一 CPU 上时，它们可能相互竞争。然而，Linux 操作系统在运行时并不区分这两类线程。为解决该问题，我们从 JVM 收集 Java 线程的信息，因为每个线程都有自己的标签。JVM 中的每个 Java 线程在 Linux 中都有与之对应的系统线程，这些系统线程共享同一个进程标识（PID）。借助所获得的标签，iGC 区分这两类线程并将它们固定到不同的 CPU 上，从而避免工作线程与 GC 线程之间的争用。

> **Batch & GC** Though batch threads compete resources with GC threads, GC is not in the critical path of a LC service as the interference on GC threads from batch threads will not impact the latency of the LC service. GC threads are only active for a short interval. As long as it finishes before the heap size reaches the limit, there is not allocation stall. As a result, GC threads can share CPU resources with batch threads. By starting JVMs with root privilege, iGC co-locates GC threads with low-priority batch threads on one CPU for high utilization. It is the Linux OS scheduler that schedules the threads based on their priority. Alternatively, if JVMs are started at user space, iGC prioritizes GC threads by the use of nice command [27] to reduce the interference between batch threads and GC threads.

**Batch & GC（批处理线程与 GC 线程）。** 尽管批处理线程会与 GC 线程竞争资源，但 GC 并不处于 LC 服务的关键路径上，因为批处理线程对 GC 线程的干扰不会影响 LC 服务的延迟。GC 线程只在一个很短的间隔内活跃。只要它在堆大小达到上限之前完成，就不会出现分配停顿。因此，GC 线程可以与批处理线程共享 CPU 资源。通过以 root 权限启动 JVM，iGC 将 GC 线程与低优先级的批处理线程混部到同一个 CPU 上，以获得高利用率，此时由 Linux 操作系统调度器依据优先级来调度线程。另一种做法是，如果 JVM 在用户空间启动，iGC 使用 nice 命令 [27] 提高 GC 线程的优先级，以减少批处理线程与 GC 线程之间的干扰。

> **Worker & Batch** Batch threads compete CPU and memory hierarchy resources with worker threads. Batch threads should not be placed in CPUs that execute worker threads.

**Worker & Batch（工作线程与批处理线程）。** 批处理线程会与工作线程竞争 CPU 与存储层次资源，因此批处理线程不应被放置在执行工作线程的 CPU 上。

#### 3.3.2 利用 HT（Making use of HT）

> Two factors are considered when making use of the HT technique, the impact on the memory hierarchy and on the instruction throughput. Sharing the same physical core leads to contention on the core-level caches, i.e., L1 data cache, L1 code cache, and L2 cache. Each process owns one virtual memory space. The data stored in caches can be reused by threads from the same process. An efficient way that takes advantage of the HT technique is to place worker threads on one CPU and GC threads on the other CPU of the same physical core. When worker threads and GC threads of the LC service share two different CPUs on the same core, they can share the cached data. It reduces the access to the lower levels of the memory hierarchy, i.e., LLC cache and the main memory. In return, the fast data fetching reduces the stall in the instruction pipeline and increases the instruction throughput as well.

在利用 HT 技术时需要考虑两个因素：对存储层次的影响以及对指令吞吐量的影响。共享同一个物理核心会导致对核心级缓存的争用，即 L1 数据缓存、L1 代码缓存和 L2 缓存。每个进程拥有一个虚拟内存空间。缓存中存储的数据可以被同一进程的线程复用。利用 HT 技术的一种高效方式是把工作线程放置在一个 CPU 上，而把 GC 线程放置在同一个物理核心的另一个 CPU 上。当 LC 服务的工作线程与 GC 线程共享同一核心上的两个不同 CPU 时，它们可以共享缓存数据。这减少了对存储层次中更低层级（即 LLC 缓存和主存）的访问。反过来，更快的数据获取减少了指令流水线中的停顿，同时也提高了指令吞吐量。

#### 3.3.3 自适应监控间隔（Adaptive monitoring interval）

> iGC uses an adaptive monitoring interval (𝐼) that is initially set to 5 seconds, and then dynamically changed at runtime according to the heap allocation growth. It aims to reduce the possibility of triggering the default GC while reducing the monitoring overhead. Algorithm 2 shows the procedure of the adaptive monitoring interval. The interval of obtaining the resource usage and heap usage varies in different configurations. The frequency of GC is also dynamically changed based on the allocation rate of new objects in the Java heap. The allocation rate is calculated by the number of new objects allocated in Java heap per second. iGC keeps sampling the allocation rate of new objects and computes the new value of the interval.

iGC 使用一个自适应监控间隔（𝐼），其初始值设为 5 秒，随后在运行时根据堆分配的增长而动态变化。其目标是在降低监控开销的同时，减少触发默认 GC 的可能性。算法 2 给出了自适应监控间隔的流程。获取资源使用率和堆使用情况的时间间隔在不同配置下有所不同。GC 的频率也会根据 Java 堆中新对象的分配速率动态变化。分配速率由每秒在 Java 堆中分配的新对象数量计算得出。iGC 持续采样新对象的分配速率，并计算出新的间隔值。

> In Algorithm 2, parameter u is the coefficient that determines the weight of the new rate and the current rate in the new monitoring interval. The formula at line 6 is to get the new interval for triggering GC. The new interval is calculated based on the current interval, the current allocation rate, and the new allocation rate. The allocation rate is volatile in Java heap. Since it changes frequently, we choose to give a higher weight for the new allocation rate. To reduce the fluctuation from bursty heap allocations, the value of u is empirically assigned to 0.2. As a result, the interval of triggering GC can adaptively change at runtime.

在算法 2 中，参数 u 是决定新速率与当前速率在新监控间隔中所占权重的系数。第 6 行的公式用于得到触发 GC 的新间隔。新间隔基于当前间隔、当前分配速率和新分配速率计算得出。Java 堆中的分配速率具有波动性。由于它变化频繁，我们选择为新分配速率赋予更高的权重。为减少突发堆分配带来的波动，u 的取值根据经验设为 0.2。因此，触发 GC 的间隔可以在运行时自适应地变化。

**算法 2：自适应监控间隔（Algorithm 2: Adaptive monitoring interval）**

```text
Algorithm 2 Adaptive monitoring interval.
1: 𝐼: the time interval of triggering GC;
2: 𝑐𝑢𝑟_𝑟𝑎𝑡𝑒: the current allocation rate of new objects;
3: 𝑢: the coefficient (0<𝑢<1);
4: function Test_GC
5:   𝑛𝑒𝑤_𝑟𝑎𝑡𝑒 = getCurrentAllocationRate()
6:   𝐼 = 𝑢∗𝐼 + (1-𝑢)∗I∗(𝑛𝑒𝑤_𝑟𝑎𝑡𝑒 - 𝑐𝑢𝑟_𝑟𝑎𝑡𝑒)/𝑐𝑢𝑟_𝑟𝑎𝑡𝑒;
7:   adjustGcThreadNumber(𝑐𝑢𝑟_𝑟𝑎𝑡𝑒, 𝑛𝑒𝑤_𝑟𝑎𝑡𝑒);
8:   adjustHeapUsageThreshold(𝑐𝑢𝑟_𝑟𝑎𝑡𝑒, 𝑛𝑒𝑤_𝑟𝑎𝑡𝑒);
9:   𝑐𝑢𝑟_𝑟𝑎𝑡𝑒 = 𝑛𝑒𝑤_𝑟𝑎𝑡𝑒;
10:  return (𝑛)
11: end function
```

算法 2 变量与流程说明：I 为触发 GC 的时间间隔；cur_rate 为新对象的当前分配速率；u 为系数（0<u<1）。函数 Test_GC 先获取当前分配速率并赋给 new_rate，再按第 6 行的公式更新间隔 I（新间隔由当前间隔、当前分配速率与新分配速率共同决定），随后依据新旧分配速率调整 GC 线程数量与堆使用阈值，最后用 new_rate 更新 cur_rate 并返回。

## 4 实现（Implementation）

> iGC, implemented as a lightweight middleware between JVM and Linux OS, is written in Bash and C++ with around 1,000 lines of code. The implementation tackles several challenges. Firstly, its monitor needs to collect information from the JVM, i.e., different thread types, heap usage, object allocation rate, but also from the Linux OS, i.e., CPU usage for each core. Secondly, iGC needs a suitable interface for controlling JVM parameters at runtime. Last but not least, iGC needs to be compatible with the two state-of-the-art concurrent GC mechanisms ZGC and G1GC. iGC project is open source at https://github.com/lotusknight/iGC.

iGC 被实现为 JVM 与 Linux 操作系统之间的一个轻量级中间件，使用 Bash 和 C++ 编写，代码量约为 1,000 行。该实现需要应对若干挑战。首先，其监控器既需要从 JVM 收集信息，即不同的线程类型、堆使用情况、对象分配速率，也需要从 Linux 操作系统收集信息，即每个核心的 CPU 使用率。其次，iGC 需要一个合适的接口来在运行时控制 JVM 参数。最后但同样重要的是，iGC 需要与两种最先进的并发 GC 机制 ZGC 和 G1GC 兼容。iGC 项目已在 https://github.com/lotusknight/iGC 上开源。

> iGC daemon process runs with the root privilege to access the information of resource usages and control resource scheduling on servers. It obtains CPU usage information from the Linux file /proc/stat. It samples the CPU usage in each second.

iGC 守护进程以 root 权限运行，以便访问资源使用信息并控制服务器上的资源调度。它从 Linux 文件 /proc/stat 获取 CPU 使用率信息，并且每秒对 CPU 使用率采样一次。

> JMX facilitates programmers to monitor and manage JVM. iGC uses JMX to access the Java heap usage and trigger GC from outside of the JVM. With JMX, iGC can be deployed for all Java applications without modifying the source code of applications. When employing JMX, we assign JMX port number for each Java application when starting the JVM.

JMX 便于程序员监控和管理 JVM。iGC 使用 JMX 从 JVM 外部访问 Java 堆使用情况并触发 GC。借助 JMX，iGC 可以部署到所有 Java 应用之上，而无需修改应用的源代码。在采用 JMX 时，我们在启动 JVM 时为每个 Java 应用分配 JMX 端口号。

> iGC has interfaces to ZGC [16] and G1 GC [22]. Note that iGC and JVM won't trigger GC threads at the same time as iGC is a middleware that triggers GC through JVM GC interface. Once iGC has triggered GC, it will change a status flag in JVM so that no more GC will be triggered at the same.

iGC 具有面向 ZGC [16] 和 G1 GC [22] 的接口。注意，iGC 与 JVM 不会同时触发 GC 线程，因为 iGC 是通过 JVM 的 GC 接口来触发 GC 的中间件。一旦 iGC 触发了 GC，它就会修改 JVM 中的一个状态标志，使得在此期间不会再触发其他 GC。

> iGC uses scripts to obtain the PIDs of the JVMs for LC services and batch jobs. It uses command jstack to capture the JVM thread dump and obtain thread labels and thread IDs. Threads both related to GC and worker are recognized as worker threads. With the labels and thread IDs, iGC distinguishes the GC threads and places them onto CPUs according to the priority-based dynamic thread placement strategy. iGC sets the thread CPU affinity by using taskset command. We use Linux cgroup to adjust the CPU limits for GC threads and batch threads which share the same CPUs. iGC can adapt to JVMs in both user space and root privilege. If JVMs are started at user space, command nice() is used to set GC threads with higher priority when they are co-located with batch threads to reduce the CPU interference. If JVMs are started with root privilege, iGC dynamic thread placement avoids or mitigates the interference.

iGC 使用脚本获取 LC 服务和批处理作业所对应 JVM 的 PID。它使用命令 jstack 抓取 JVM 线程转储，并获得线程标签与线程 ID。同时与 GC 和工作线程相关的线程都被识别为工作线程。借助这些标签和线程 ID，iGC 区分出 GC 线程，并按照基于优先级的动态线程放置策略将它们放置到相应 CPU 上。iGC 使用 taskset 命令设置线程的 CPU 亲和性。我们使用 Linux cgroup 来调整共享相同 CPU 的 GC 线程和批处理线程的 CPU 限额。iGC 可以同时适配用户空间和 root 权限下的 JVM。如果 JVM 在用户空间启动，当 GC 线程与批处理线程混部时，使用 nice() 命令将 GC 线程设置为更高优先级，以减少 CPU 干扰。如果 JVM 以 root 权限启动，则 iGC 的动态线程放置策略会避免或缓解这些干扰。


## 5 评测（Evaluation）

### 5.1 评测设置（Evaluation Setup）

> We use two servers in the evaluation. Each server has two Intel Xeon E5-2630 v3
> @ 2.40GHz CPUs 16 cores (32 hyper threads in total, L1 cache 320KB (data),
> 320KB (code), L2 cache 2.5MB, Haswell architecture with eight execution units),
> 32 GB DRAM, and 1TB HDD.

我们在评测中使用两台服务器。每台服务器配备两颗 Intel Xeon E5-2630 v3 @ 2.40GHz CPU、16 核（共 32 个超线程，L1 缓存 320KB（数据）、320KB（指令），L2 缓存 2.5MB，Haswell 架构，含八个执行单元）、32 GB DRAM 以及 1TB HDD。

> LC services. We use three NoSQL databases as LC services, Cassandra-4.0-beta2,
> HBase-2.2.7, and Solr-8.8.2. Cassandra, a wide-column store, is a popular NoSQL
> database management system that is designed to handle large amount of data
> across servers. HBase runs on top of HDFS [48], providing Google
> Bigtable-like capability for Hadoop. Solr is a specialized database designed
> for read only (scan is a sequential of reads). They are commonly used as LC
> services for performance evaluation. Three databases are running with JDK 11.
> The heap size of each database is set at 8GB.

LC 服务。我们使用三个 NoSQL 数据库作为 LC 服务：Cassandra-4.0-beta2、HBase-2.2.7 和 Solr-8.8.2。Cassandra 是一种宽列（wide-column）存储，是流行的 NoSQL 数据库管理系统，旨在跨服务器处理海量数据。HBase 运行在 HDFS [48] 之上，为 Hadoop 提供类似 Google Bigtable 的能力。Solr 是专为只读场景设计的专用数据库（scan 是一系列顺序读）。它们通常被用作 LC 服务以进行性能评测。三个数据库均运行在 JDK 11 上。每个数据库的堆大小设置为 8GB。

> We use YCSB [12] as the benchmark to send requests to the LC services. YCSB is
> an open-source specification and program suite, which is specifically used to
> compare relative performance of NoSQL database management systems. It offers
> workloads containing different operations including read, update, insert, and
> scan. Table 2 shows five types of workloads and the percentage of operations
> in each workload used in evaluation. As Solr is mainly used as a search engine,
> we only use read and scan workloads (Workloads C and S) for evaluation. Unless
> mentioned, a YCSB Client sends operations in best effort per second in our
> experiments.

我们使用 YCSB [12] 作为基准测试，向 LC 服务发送请求。YCSB 是一套开源规范与程序套件，专门用于比较 NoSQL 数据库管理系统的相对性能。它提供的负载包含 read、update、insert 和 scan 等不同操作。表 2 给出了评测中使用的五类负载以及各负载中操作所占的百分比。由于 Solr 主要用作搜索引擎，我们仅使用读与扫描负载（负载 C 和 S）进行评测。除特别说明外，在我们的实验中 YCSB 客户端以尽力而为的方式每秒发送操作。

**表 2：各负载中操作所占的百分比（Table 2: The Percentage of Operations in Workloads.）**

| Operation 操作 | READ 读 | UPDATE 更新 | INSERT 插入 | SCAN 扫描 |
| --- | --- | --- | --- | --- |
| Workload A 负载 A | 50% | 50% | 0 | 0 |
| Workload B 负载 B | 95% | 5% | 0 | 0 |
| Workload C 负载 C | 100% | 0 | 0 | 0 |
| Workload D 负载 D | 95% | 0 | 5% | 0 |
| Workload S 负载 S | 0 | 0 | 0 | 100% |

> Batch jobs. We use Yarn in Hadoop-2.9.2 [49] as the scheduler for batch jobs.
> Spark-2.1.0 runs KMeans workload from HiBench-6.0 [30] as the batch jobs. Batch
> jobs runs with JDK 8. One server works as the master of Yarn and the client of
> LC services. The other serves as the multi-tenant system where LC services and
> best-effort batch jobs are co-located, where iGC is installed. All experiments
> were conducted under co-location, unless those marked Alone.

批处理作业。我们使用 Hadoop-2.9.2 [49] 中的 Yarn 作为批处理作业的调度器。Spark-2.1.0 运行 HiBench-6.0 [30] 中的 KMeans 负载作为批处理作业。批处理作业运行在 JDK 8 上。一台服务器作为 Yarn 的主节点以及 LC 服务的客户端，另一台则作为多租户系统，LC 服务与尽力而为（best-effort）型批处理作业在该系统上混部，iGC 也安装于此。除标记为 Alone 的实验外，所有实验均在混部条件下进行。

> Approaches. We primarily use ZGC as the concurrent GC in iGC performance
> evaluation. ZGC is the latest concurrent GC, which is of low latency. We also
> use the other popular concurrent GC, G1GC, in the evaluation.

方法。在 iGC 的性能评测中，我们主要使用 ZGC 作为并发 GC。ZGC 是最新的并发 GC，具有低延迟特性。我们在评测中也使用了另一种流行的并发 GC——G1GC。

> Metrics. We run three NoSQL databases with different workloads to evaluate the
> latency of LC requests and the throughput of LC service and batch jobs. Then,
> we use cache miss and the number of retired instructions as metrics to
> illustrate the benefit of taking advantage of memory hierarchy in iGC. We then
> test the performance of iGC under different pressure levels. Next, we present
> the latency of each operation in Cassandra. Finally, we discuss the overhead of
> iGC.

指标。我们让三个 NoSQL 数据库运行不同的负载，以评测 LC 请求的延迟以及 LC 服务和批处理作业的吞吐量。随后，我们使用缓存缺失和退休指令数作为指标，说明 iGC 利用存储层次所带来的收益。接着，我们在不同的压力水平下测试 iGC 的性能。然后，我们给出 Cassandra 中每种操作的延迟。最后，我们讨论 iGC 的开销。

### 5.2 延迟与吞吐量（Latency and Throughput）

> We evaluate the latency of LC services as well as the throughput of both LC
> services and batch jobs using the three NoSQL databases. Experiments are
> conducted in three configurations: 1) LC services running alone using ZGC
> (Alone w/ ZGC), 2) LC services co-located with batch jobs using ZGC (Co /w
> ZGC), and 3) LC services co-located with batch jobs using iGC (Co w/ iGC).
> Under each configuration, we run the experiment for one hour and measure the
> average values.

我们使用三个 NoSQL 数据库评测 LC 服务的延迟以及 LC 服务和批处理作业的吞吐量。实验在三种配置下进行：1）LC 服务单独运行并使用 ZGC（Alone w/ ZGC）；2）LC 服务与批处理作业混部并使用 ZGC（Co /w ZGC）；3）LC 服务与批处理作业混部并使用 iGC（Co w/ iGC）。在每种配置下，我们运行实验一小时并测量平均值。

#### 5.2.1 延迟降低（Latency Reduction）

> The tail latency CDF of all requests of LC services Cassandra, HBase and Solr
> are shown in Figure 5, Figure 6 and Figure 7, respectively. Figure 5 shows that
> under all three configurations, Alone with ZGC shows the best performance for
> all Cassandra workloads as there is no interference. Compared to the
> co-location with ZGC, co-location with iGC shows significantly reduction of the
> tail latency in the all three workloads, especially for workload A. The reason
> is that iGC places GC threads and worker threads in different CPUs, which
> mitigates the interference of concurrent GC on worker threads of LC services.

LC 服务 Cassandra、HBase 和 Solr 所有请求的尾延迟 CDF 分别如图 5、图 6 和图 7 所示。图 5 表明，在三种配置下，由于不存在干扰，Alone with ZGC 在所有 Cassandra 负载中都表现最佳。与使用 ZGC 的混部相比，使用 iGC 的混部在全部三种负载下都显著降低了尾延迟，负载 A 尤为明显。原因在于 iGC 将 GC 线程与工作线程放置在不同的 CPU 上，从而缓解了并发 GC 对 LC 服务工作线程的干扰。

> Under co-location, most of the latency due to iGC and ZGC are concentrated
> around three values: 500𝜇𝑠 (low latency), 4000𝜇𝑠 (medium latency), and
> 8000𝜇𝑠 (high latency), which form a stair-like shape. The reason is that there
> is a mechanism in Cassandra that maintains cache layers to speed up requests.
> iGC reduces the access to the next layer as it incurs less contention to memory
> hierarchy. As a result, iGC renders 6% (p96-p90) more requests that have low
> latency and 2% (p98-p96) more requests that have medium latency for workload A,
> in compared with ZGC.

在混部情况下，iGC 和 ZGC 造成的延迟大多集中在三个数值附近：500𝜇𝑠（低延迟）、4000𝜇𝑠（中等延迟）和 8000𝜇𝑠（高延迟），形成阶梯状。原因在于 Cassandra 中存在一种维护缓存层次的机制，用以加速请求。由于 iGC 对存储层次的争用更小，它减少了向下一层的访问。因此，与 ZGC 相比，iGC 在负载 A 下使低延迟请求多了 6%（p96-p90），中等延迟请求多了 2%（p98-p96）。

**表 3：完成的批处理作业数量（Table 3: The Number of Finished Batch Jobs.）**

| | Cassandra | HBase | Solr |
| --- | --- | --- | --- |
| Co w/ ZGC | 189 | 187 | 189 |
| Co w/ iGC | 185 | 182 | 184 |
| Standalone 单独运行 | 203 | | |

> Figure 6 shows the experimental results for HBase workloads. iGC renders the
> lowest latency for requests almost at all percentiles. In HBase, CPU contention
> is very severe. By placing GC threads and worker threads in different CPUs, iGC
> is able to mitigate the interference while Alone with ZGC does not. Note that
> the interference from batch jobs results in longer p99 latency under iGC in
> comparison with Alone with ZGC. Importantly, iGC significantly outperforms
> co-location with ZGC.

图 6 给出了 HBase 负载的实验结果。iGC 在几乎所有百分位上都为请求带来最低的延迟。在 HBase 中，CPU 争用非常严重。通过将 GC 线程与工作线程放置在不同的 CPU 上，iGC 能够缓解干扰，而 Alone with ZGC 则不能。需要注意的是，与 Alone with ZGC 相比，批处理作业带来的干扰使 iGC 下的 p99 延迟更长。重要的是，iGC 显著优于使用 ZGC 的混部。

> Figure 7 shows the experimental results for Solr workloads. iGC renders the
> best performance under workload C. The alleviation of CPU interference on
> worker threads leads to smaller latency for most requests. The SCAN operation
> sequentially reads data, which leads to higher cache miss rate. As a result,
> iGC renders results similar to that due to Alone with ZGC in workload S as it
> loses the advantage of memory hierarchy.

图 7 给出了 Solr 负载的实验结果。iGC 在负载 C 下表现最佳。对工作线程所受 CPU 干扰的缓解使大多数请求的延迟更小。SCAN 操作顺序读取数据，导致更高的缓存缺失率。因此，由于失去了存储层次方面的优势，iGC 在负载 S 下的结果与 Alone with ZGC 相似。

> Figure 8 shows the reduction of the average latency and latency at four
> specific percentiles by iGC in comparison with ZGC for the three LC services
> under co-location. Compared with the default ZGC, iGC can reduce the p95
> latency by 83% (Workload A), 37% (Workload A), and 22% (Workload S) for
> Cassandra, HBase and Solr, respectively.

图 8 给出了混部条件下三种 LC 服务中，iGC 相对于 ZGC 在平均延迟以及四个特定百分位延迟上的降低幅度。与默认的 ZGC 相比，iGC 可将 Cassandra、HBase 和 Solr 的 p95 延迟分别降低 83%（负载 A）、37%（负载 A）和 22%（负载 S）。

#### 5.2.2 系统吞吐量（System Throughput）

> We measure the throughput of LC services in terms of the number of operations
> per second. Figure 9 shows the throughput for each LC service with different
> workloads by iGC and ZGC under co-location. iGC shows non-trivial throughput
> improvement for all eight workloads, because the separation of GC threads and
> worker threads significantly reduces the CPU interference on worker threads.
> Further, it limits the resource usage of batch threads when GC threads are
> co-located with batch threads and GC is triggered.

我们以每秒操作数来衡量 LC 服务的吞吐量。图 9 给出了混部条件下 iGC 与 ZGC 下各 LC 服务在不同负载上的吞吐量。iGC 在全部八个负载上都带来了可观的吞吐量提升，因为 GC 线程与工作线程的分离显著降低了工作线程所受的 CPU 干扰。此外，当 GC 线程与批处理线程混部且 GC 被触发时，它限制了批处理线程的资源使用。

> To measure the throughput of batch jobs, we count the number of the finished
> batch jobs. Table 3 shows the number of the finished batch jobs in one hour due
> to iGC and ZGC. As the baseline, the Standalone entry shows the number of batch
> jobs running alone on the server without the resource limit. The throughput of
> batch jobs due to iGC under co-location with Cassandra, HBase and Solr is only
> reduced by 2.2%, 2.6% and 2.7%, respectively. The reduction is due to two
> reasons. First, iGC separates GC threads and worker threads of LC services onto
> different CPUs, which uses more CPU resource in exchange for lower latency by
> mitigating the CPU resource contention. Second, iGC dynamically triggers GC
> with an adaptive time interval, which causes additional CPU resource but
> reduces the interference of GC on batch jobs. The impact of iGC on the
> throughput of batch jobs is trivial, because the STW time interval is small in
> concurrent GC. Under job co-location, the latency and throughput of LC services
> are the dominant performance metrics.

为了衡量批处理作业的吞吐量，我们统计已完成的批处理作业数量。表 3 给出了一小时内 iGC 和 ZGC 下完成的批处理作业数量。作为基线，Standalone 一栏给出了批处理作业在服务器上单独运行且不受资源限制时的作业数量。在与 Cassandra、HBase 和 Solr 混部时，iGC 下批处理作业的吞吐量分别仅下降 2.2%、2.6% 和 2.7%。这一下降有两个原因。第一，iGC 将 LC 服务的 GC 线程与工作线程分离到不同的 CPU 上，通过缓解 CPU 资源争用，以更多的 CPU 资源换取更低的延迟。第二，iGC 以自适应的时间间隔动态触发 GC，这会带来额外的 CPU 资源开销，但减少了 GC 对批处理作业的干扰。iGC 对批处理作业吞吐量的影响微乎其微，因为在并发 GC 中 STW 的时间间隔很短。在作业混部场景下，LC 服务的延迟和吞吐量才是主导性的性能指标。

> Due to the space limit, we use Cassandra as the representative LC service in
> the following evaluation.

受篇幅所限，在后续评测中我们使用 Cassandra 作为代表性的 LC 服务。

**图 5：三种 Cassandra 负载下所有请求的尾延迟 CDF（Fig. 5: The tail latency CDF of all requests of three Cassandra workloads.）**

- 子图：(a) Workload A、(b) Workload B、(c) Workload D
- 横轴：Latency (us)，刻度 0、2000、4000、6000、8000、10000；纵轴：CDF，刻度 0.80、0.85、0.90、0.95、0.99
- 图例：Alone w/ ZGC、Co w/ ZGC、Co w/ iGC

**图 6：三种 HBase 负载下所有请求的尾延迟 CDF（Fig. 6: The tail latency CDF of all requests of three HBase workloads.）**

- 子图：(a) Workload A、(b) Workload B、(c) Workload D
- 横轴：Latency (us)，刻度 0、500、1000、1500、2000；纵轴：CDF，刻度 0.80、0.85、0.90、0.95、0.99
- 图例：Alone w/ ZGC、Co w/ ZGC、Co w/ iGC

### 5.3 HT 对缓存性能的影响（HT Impact on Cache Performance）

**表 4：一分钟内各监控事件的数量（Table 4: The Number of Each Monitor Events in One Minute.）**

| Monitor Event 监控事件 | Description 描述 | Event No. 事件编号 | Co w/ ZGC | Co w/ iGC |
| --- | --- | --- | --- | --- |
| L1D_PEND_MISS.PENDING | L1D misses every cycle. | 0148h | 130 ∗ 10⁹ | 57.4 ∗ 10⁹ |
| L2_RQSTS.MISS | Requests that missed L2. | 3F24h | 154 ∗ 10⁹ | 70.0 ∗ 10⁹ |
| L1D.REPALCEMENT | Lines evicted by new data lines. | 0151h | 129 ∗ 10⁹ | 51.5 ∗ 10⁹ |
| INST_RETIRED.ANY_P | Instructions at retirement. | 00C0h | 44.2 ∗ 10⁹ | 114 ∗ 10⁹ |
| UOPS_RETIRED.ALL | Micro-ops retired. | 01C2h | 59.6 ∗ 10⁹ | 130 ∗ 10⁹ |
| UOPS_RETIRED.RETIRE_SLOTS | Retired slots used in each cycle. | 02C2h | 48.8 ∗ 10⁹ | 109 ∗ 10⁹ |

> We use two dimensions (the cache miss and the instruction throughput), three
> cache metrics, and three types of retired instructions, to evaluate the impact
> of HT on cache performance under co-location with iGC and ZGC. We use Cassandra
> workload A in the evaluation. We obtain the counter values from the hardware
> monitor events of the Intel processors. We compare the number of cache misses
> and the number of retired instructions on a CPU where worker threads run.
> Table 4 gives six monitoring events related to cache performance [14].

我们使用两个维度（缓存缺失和指令吞吐量）、三项缓存指标以及三类退休指令，来评测在 iGC 和 ZGC 混部下 HT 对缓存性能的影响。评测中使用 Cassandra 负载 A。我们从 Intel 处理器的硬件监控事件中获取计数器值。我们在工作线程所运行的 CPU 上比较缓存缺失数与退休指令数。表 4 给出了与缓存性能相关的六个监控事件 [14]。

> The experiment runs under two different configurations: 1) Worker threads run on
> one CPU while GC threads run on the other CPU of the same core (co-location w/
> iGC), and 2) Worker threads run on one CPU while batch threads run on the other
> CPU of the same core (co-location w/ ZGC).

实验在两种不同的配置下运行：1）工作线程运行在一个 CPU 上，而 GC 线程运行在同一核心的另一个 CPU 上（与 iGC 混部）；2）工作线程运行在一个 CPU 上，而批处理线程运行在同一核心的另一个 CPU 上（与 ZGC 混部）。

#### 5.3.1 缓存缺失与替换（Cache Miss and Replacement）

> In Table 4, the first three entries show the cache miss of the Cassandra
> workload due to iGC. We count the operations for one minute for each event. We
> chose a 1-minute time interval as it is long enough to include 10 cycles of GC
> for analysis. Three metrics are 1) the number of L1 data cache misses per
> cycle, 2) the number of requests that missed L2 cache, and 3) the number of
> cache lines in L1 cache which are evicted by bringing in new line to L1 cache.
> Compared with the co-location with ZGC, iGC reduces the three metrics by 55.7%,
> 54.4%, and 59.9%, respectively.

在表 4 中，前三项给出了 iGC 下 Cassandra 负载的缓存缺失情况。我们对每个事件统计一分钟内的操作数。我们选择 1 分钟的时间间隔，因为它足够长，可以涵盖用于分析的 10 个 GC 周期。三项指标分别是：1）每个周期的 L1 数据缓存缺失数；2）未命中 L2 缓存的请求数；3）因向 L1 缓存引入新行而被驱逐的 L1 缓存行数。与使用 ZGC 的混部相比，iGC 将这三项指标分别降低了 55.7%、54.4% 和 59.9%。

> With HT enabled, GC threads and worker threads are assigned to the two CPUs on
> the same core. Each core shares the same L1 cache and L2 cache. With iGC, the
> L1 cache miss and L2 cache miss decrease dramatically. We also use the L1
> replacement as the metric. It shows that fewer cache lines are replaced by new
> cache lines in L1 cache. Compared with the co-location with ZGC, iGC shows less
> contention in memory hierarchy. Threads of LC services on both CPUs of the same
> core are from the same JVM. They share the same memory space such as Java heap,
> so the cache data can be reused by both threads and the cache misses are
> reduced. Additionally, it reduces the replacement of cache lines by batch jobs
> threads, which also reduces the cache miss.

在启用 HT 的情况下，GC 线程和工作线程被分配到同一核心的两个 CPU 上。每个核心共享相同的 L1 缓存和 L2 缓存。使用 iGC 时，L1 缓存缺失和 L2 缓存缺失都大幅下降。我们还使用 L1 替换作为指标，它表明 L1 缓存中被新缓存行替换的缓存行更少。与使用 ZGC 的混部相比，iGC 在存储层次上的争用更少。同一核心两个 CPU 上 LC 服务的线程来自同一个 JVM，它们共享 Java 堆等相同的内存空间，因此缓存数据可以被两个线程复用，缓存缺失随之减少。此外，它还减少了批处理作业线程对缓存行的替换，这同样降低了缓存缺失。

#### 5.3.2 退休指令数（Retired Instructions）

> The last three entries in Table 4 give the number of retired CPU instructions
> for LC services due to iGC. We count the operations for one minute for each
> event. Three metrics are 1) the number of all retired instructions, 2) the
> number of retired micro-operations (uops), and 3) the number of retired slots
> (a.k.a. execution units) used in each clock cycle. There are eight slots in
> Haswell architecture that can be used in each cycle, which means eight
> micro-operations or eight instructions could be retired in each cycle. Compared
> with ZGC, iGC improves the three metrics by 2.56X, 2.18X, and 2.23X,
> respectively.

表 4 中后三项给出了 iGC 下 LC 服务退休的 CPU 指令数。我们对每个事件统计一分钟内的操作数。三项指标分别是：1）全部退休指令数；2）退休微操作（uops）数；3）每个时钟周期使用的退休槽位数（又称执行单元）。Haswell 架构中每个周期有八个可用槽位，意味着每个周期可以退休八个微操作或八条指令。与 ZGC 相比，iGC 将这三项指标分别提升了 2.56X、2.18X 和 2.23X。

**图 7：两种 Solr 负载下所有请求的尾延迟 CDF（Fig. 7: The tail latency CDF of all requests of two Solr workloads.）**

- 子图：(a) Workload C、(b) Workload S
- 横轴：Latency (us)，子图 (a) 刻度 300、400、500、600、700、800、900、1000，子图 (b) 刻度 0、10000、20000、30000、40000、50000、60000、70000、80000、90000；纵轴：CDF，刻度 0.50、0.90、0.95、0.99
- 图例：Alone w/ ZGC、Co w/ ZGC、Co w/ iGC

> Each instruction consists of several micro-operations. An instruction is
> retired when it is finished on the CPU pipeline. Since iGC separates GC threads
> and worker threads onto different CPUs, the interference between GC threads and
> worker threads is avoided. As a result, the utilization of the retired slots per
> cycle is higher, so are the number of retired instructions and uops. iGC
> significantly increases the throughput of LC services.

每条指令由若干微操作组成。指令在 CPU 流水线上执行完成时即被退休。由于 iGC 将 GC 线程与工作线程分离到不同的 CPU 上，GC 线程与工作线程之间的干扰得以避免。因此，每个周期的退休槽位利用率更高，退休指令数和微操作数也随之提高。iGC 显著提升了 LC 服务的吞吐量。


### 5.4 压力测试与可扩展性（Stress Test and Scalability）

> We now study the performance of iGC under different client request levels. We
> run Cassandra Workload A with different numbers of YCSB client threads to
> evaluate the latency reduction due to iGC in comparison with ZGC under
> co-location.

我们现在研究 iGC 在不同客户端请求水平下的性能。我们以不同数量的 YCSB 客户端线程运行 Cassandra 负载 A，以评测混部条件下 iGC 相对于 ZGC 的延迟降低效果。

> Figure 10 shows the CDF of all requests of the tail latency when the number of
> client threads is set to 1, 2, 4, 8, 16, and 32 for iGC and ZGC. We can see that
> iGC obtains marginal improvement for the p90 and p95 tail latency under 1 and 2
> threads. As the number of threads increases, the latency reduction due to iGC
> improves significantly. The reason is that there are contention among client
> threads, which increases the interference for CPU resources. With iGC, GC
> threads are assigned to different CPUs, which avoids the CPU contention with
> worker threads. As a result, iGC significantly alleviates the degradation of the
> tail latency. When there are 32 client threads, the latency reductions for p90,
> p95, and p99 due to iGC are 64%, 87%, and 83%, respectively. The results
> demonstrate that iGC has better scalability than ZGC.

图 10 给出了客户端线程数分别设为 1、2、4、8、16 和 32 时，iGC 与 ZGC 下所有请求尾延迟的 CDF。可以看到，在 1 个和 2 个线程时，iGC 对 p90 和 p95 尾延迟的改善很有限。随着线程数增加，iGC 带来的延迟降低显著提升。原因在于客户端线程之间存在争用，加大了对 CPU 资源的干扰。使用 iGC 时，GC 线程被分配到不同的 CPU 上，避免了与工作线程的 CPU 争用。因此，iGC 显著缓解了尾延迟的恶化。当有 32 个客户端线程时，iGC 对 p90、p95 和 p99 的延迟降低分别为 64%、87% 和 83%。这些结果表明 iGC 比 ZGC 具有更好的可扩展性。

> To further demonstrate the scalability of iGC, we obtain the performance of iGC
> under different request per second (RPS) levels. We run Cassandra Workload A
> with different RPS, ranging from 500 to 5,000 with 500 as a step. Figure 11
> shows the p95 tail latency due to iGC and ZGC. As the number of RPS increases,
> the p95 tail latency also increases. Compared to ZGC, iGC renders smaller but
> also more stable tail latency. It shows similar p95 latency from 500 to 3,500
> RPS. On the other hand, the p95 latency due to ZGC deteriorates dramatically
> after the RPS reaches 2,500. The results again demonstrate that iGC has good
> scalability.

为进一步展示 iGC 的可扩展性，我们获取了 iGC 在不同每秒请求数（request per second, RPS）水平下的性能。我们以不同的 RPS 运行 Cassandra 负载 A，范围从 500 到 5,000，步长为 500。图 11 给出了 iGC 和 ZGC 下的 p95 尾延迟。随着 RPS 增加，p95 尾延迟也随之增加。与 ZGC 相比，iGC 的尾延迟更小，也更稳定。从 500 到 3,500 RPS，它表现出相近的 p95 延迟。另一方面，ZGC 的 p95 延迟在 RPS 达到 2,500 之后急剧恶化。这些结果再次表明 iGC 具有良好的可扩展性。

**表 5：每种操作的平均延迟（𝜇𝑠）（Table 5: The average latency (𝜇𝑠) of each operation.）**

| Operations 操作 | Co W/ ZGC | Co W/ iGC |
| --- | --- | --- |
| READ | 1,169 | 860 |
| UPDATE | 1,010 | 682 |
| INSERT | 1,052 | 741 |
| SCAN | 31,124 | 35,700 |

**表 6：iGC 基于 G1 GC 时的吞吐量（Table 6: The Throughput by iGC upon G1 GC.）**

| Workload 负载 | Cassandra (op/sec) A | Cassandra (op/sec) B | Cassandra (op/sec) D | Batch (num/h) 批处理作业（个/小时） |
| --- | --- | --- | --- | --- |
| Co w/ G1GC | 1231 | 849 | 1240 | 187 |
| Co w/ iGC | 1783 | 1178 | 1711 | 183 |

### 5.5 各操作的微观视角（Microscopic View of Each Operation）

> To evaluate the impact of iGC on each operation in LC services, we run four
> Cassandra operations (READ, UPDATE, INSERT, and SCAN) individually. Figure 12
> shows the tail latency CDF of all requests of the four operations. Table 5 gives
> the average latency of each operation due to co-location with ZGC and iGC. iGC
> reduces the average latency of three operations READ, UPDATE, and INSERT by
> 28%, 32%, and 30%, respectively. For SCAN operation, the latency due to iGC is
> slightly longer than that due to ZGC. The reason is that a SCAN operation
> contains several sequential READ operations, which read data from the start to
> the end in a range. When Cassandra runs SCAN as the only operation, the
> allocation rate of new objects in Java heap is very high, which leads to
> consecutive GC. In such a case, iGC cannot adaptively trigger GC as GC keeps
> running. Thus, iGC leads to longer latency for SCAN. However, SCAN is not a
> common operation in NoSQL requests. There is a rare chance that SCAN operations
> execute one after another. As a result, iGC obtains the overall better
> performance for LC services.

为了评测 iGC 对 LC 服务中每种操作的影响，我们分别运行四种 Cassandra 操作（READ、UPDATE、INSERT 和 SCAN）。图 12 给出了这四种操作所有请求的尾延迟 CDF。表 5 给出了与 ZGC 和 iGC 混部时每种操作的平均延迟。iGC 将 READ、UPDATE 和 INSERT 三种操作的平均延迟分别降低了 28%、32% 和 30%。对于 SCAN 操作，iGC 下的延迟略长于 ZGC 下的延迟。原因是一次 SCAN 操作包含若干顺序 READ 操作，它们在一个范围内从头到尾读取数据。当 Cassandra 仅运行 SCAN 这一种操作时，Java 堆中新对象的分配速率非常高，导致连续的 GC。在这种情况下，由于 GC 持续运行，iGC 无法自适应地触发 GC。因此，iGC 导致 SCAN 的延迟更长。然而，SCAN 并不是 NoSQL 请求中的常见操作，SCAN 操作连续执行的概率很低。因此，iGC 总体上为 LC 服务带来了更好的性能。

### 5.6 G1 GC

> In the previous experiments, we compare iGC with the default ZGC. Here, we
> experiment iGC upon G1 GC, another representative concurrent GC that is the
> default GC on JDK 9. We use Cassandra with workloads A, B, and D.

在之前的实验中，我们将 iGC 与默认的 ZGC 进行了比较。这里我们在 G1 GC 之上实验 iGC，G1 GC 是另一种有代表性的并发 GC，也是 JDK 9 上的默认 GC。我们使用 Cassandra 并运行负载 A、B 和 D。

> Table 6 shows the throughput of Cassandra workloads per second and that of
> batch jobs in one hour. iGC improves Cassandra throughput by 1.45X, 1.38X and
> 1.37X for workload A, B, and D, respectively. Its impact on the throughput of
> batch jobs is trivial. Note that the latency and throughput of LC services are
> the primary metrics for job co-location.

表 6 给出了 Cassandra 各负载每秒的吞吐量以及一小时内批处理作业的吞吐量。iGC 对负载 A、B 和 D 分别将 Cassandra 吞吐量提升了 1.45X、1.38X 和 1.37X。它对批处理作业吞吐量的影响微乎其微。需要注意的是，LC 服务的延迟和吞吐量才是作业混部场景下的主要指标。

> Figure 13 shows that compared to the default G1 GC under co-location, iGC
> reduces the p95 tail latency of Cassandra by 78%, 68%, and 72% for workload A,
> B, and D, respectively. We also notice that iGC reduces the p80 latency quite
> significantly, which is not the case in comparison of iGC with ZGC (as shown in
> Figure 5). There is a mechanism in G1 GC that shortens the pause time for each
> GC cycle. Because of that, GC is only triggered on part of the heap and G1 GC
> reallocates less memory in each GC cycle. As a result, GC is triggered more
> frequently and the total GC time is prolonged. Longer GC time leads to more CPU
> resource contention and affects more operations of LC services. Thus, iGC is
> able to reduce the latency for low percentiles as well.

图 13 表明，与混部条件下默认的 G1 GC 相比，iGC 将 Cassandra 在负载 A、B 和 D 下的 p95 尾延迟分别降低了 78%、68% 和 72%。我们还注意到，iGC 相当显著地降低了 p80 延迟，而在将 iGC 与 ZGC 比较时并非如此（如图 5 所示）。G1 GC 中有一种机制会缩短每个 GC 周期的停顿时间。因此，GC 只对堆的一部分触发，且 G1 GC 在每个 GC 周期中重新分配的内存更少。结果是 GC 触发更加频繁，总的 GC 时间被拉长。更长的 GC 时间导致更多的 CPU 资源争用，并影响 LC 服务更多的操作。因此，iGC 也能降低较低百分位上的延迟。

**图 8：iGC 下三种 LC 服务在不同负载上的延迟降低幅度（Fig. 8: Latency reduction of three LC services with different workloads in iGC.）**

- 子图：(a) Cassandra、(b) HBase、(c) Solr
- 横轴：avg.、p75、p90、p95、p99；纵轴：Reduction (%)，子图 (a)(b) 刻度 0、10、20、30、40、50、60、70、80、90，子图 (c) 刻度 0、10、20、30、40、50、60、70、80
- 图例：子图 (a)(b) 为 Workload A、Workload B、Workload D；子图 (c) 为 Workload C、Workload S

**图 9：混部 LC 服务的吞吐量（Fig. 9: The throughput of co-located LC services.）**

- 横轴分组：Cassandra（A、B、D）、HBase（A、B、D）、Solr（C、S）；纵轴：Througput (ops/sec)，刻度 0、500、1000、1500、2000、2500、3000
- 图例：Co w/ ZGC、Co w/ iGC
- 图中数值标注：2834

**图 10：不同客户端线程数下 iGC 与 ZGC 所有请求的尾延迟 CDF（Fig. 10: The tail latency CDF of all requests due to iGC and ZGC under different number of client threads.）**

- 左图横轴：ZGC Latency (us)，刻度 0、1000、2000、3000、4000、5000、6000；右图横轴：iGC Latency (us)，刻度相同
- 纵轴：CDF，刻度 0.90、0.95、0.99
- 图例：1、2、4、8、16、32

**图 11：不同 RPS 下的 p95 延迟（Fig. 11: The p95 latency under different RPS.）**

- 横轴：RPS，刻度 1000、2000、3000、4000、5000；纵轴：P95 Latency (us)，刻度 0、1000、2000、3000、4000、5000
- 图例：iGC、ZGC

**图 12：微观视角：四种 Cassandra 操作所有请求（𝑚𝑠）的尾延迟 CDF（Fig. 12: Microscopic view: The tail latency CDF of all requests (𝑚𝑠) for four Cassandra operations.）**

- 子图：READ、UPDATE、INSERT、SCAN
- 横轴：READ、UPDATE、INSERT 子图刻度 0、2、4、6、8、10、12、14，SCAN 子图刻度 0、50、100、150、200；纵轴：CDF，READ、UPDATE、INSERT 子图刻度 0.80、0.85、0.90、0.95、0.99，SCAN 子图刻度 0.50、0.90、0.95、0.99
- 图例：Co w/ ZGC、Co w/ iGC

> Overall, GC demonstrates significant improvement upon both ZGC and G1 GC.

总体而言，GC 在 ZGC 和 G1 GC 之上都展现出显著改进。

### 5.7 讨论（Discussions）

> iGC Overhead. iGC separates GC threads from worker threads and place them onto
> different CPUs. Compared with ZGC or G1 GC, iGC uses one more CPU to run GC
> threads. Note that overhead of GC is limited at most one CPU in
> non-work-conserving manner regardless of the cluster size. As GC threads are
> not in the critical path of LC services, GC threads can share CPUs with threads
> of batch jobs. Additionally, GC threads only consume CPU resource when GC is
> triggered. In our evaluation, the CPU utilization of GC threads is around 30%
> of a CPU when GC is running. As long as iGC avoids allocation stall, its
> overhead is less than the capacity of one logical CPU.

iGC 的开销。iGC 将 GC 线程与工作线程分离，并把它们放置到不同的 CPU 上。与 ZGC 或 G1 GC 相比，iGC 多用一个 CPU 来运行 GC 线程。需要注意的是，无论集群规模如何，GC 的开销都以非工作保留（non-work-conserving）的方式被限制在至多一个 CPU 之内。由于 GC 线程不在 LC 服务的关键路径上，GC 线程可以与批处理作业的线程共享 CPU。此外，GC 线程仅在 GC 被触发时才消耗 CPU 资源。在我们的评测中，GC 运行期间 GC 线程的 CPU 利用率约为一个 CPU 的 30%。只要 iGC 避免了分配停顿（allocation stall），其开销就小于一个逻辑 CPU 的容量。

> iGC Downward Compatibility. Recently, JDK changes the naming convention so that
> there are many new versions (V11-V17). Currently, JDK 11 is used in production
> systems. Cassandra and Hadoop support JDK 11 runtimes. G1 GC is the
> state-of-the-art mechanism for JDKs. Thus, iGC uses the newest-compatible Java
> version and supports downward compatibility.

iGC 的向下兼容性。近来 JDK 改变了版本命名约定，出现了许多新版本（V11-V17）。目前生产系统中使用 JDK 11。Cassandra 和 Hadoop 支持 JDK 11 运行时。G1 GC 是面向 JDK 的最先进机制。因此，iGC 使用最新的兼容 Java 版本并支持向下兼容。

**图 13：iGC 基于 G1 GC 时三种 Cassandra 负载下所有请求的尾延迟 CDF（Fig. 13: The tail latency CDF of all requests of three Cassandra workloads due to iGC upon G1 GC.）**

- 子图：(a) Workload A、(b) Workload B、(c) Workload D
- 横轴：Latency (us)，刻度 0、2000、4000、6000、8000、10000；纵轴：CDF，刻度 0.80、0.85、0.90、0.95、0.99
- 图例：Co w/ G1GC、Co w/ iGC


## 6 相关工作（Related Work）

> LC service Optimization. Many studies focus on improving the QoS of LC services from the
> perspective of reducing tail latency [4, 20, 24, 46, 55]. For example, PerfIso [31] uses
> native CPU isolation to achieve CPU sharing between LC services and best-effort batch
> jobs. FlatStore [10] is a key-value store that uses logs in persistent memory for serving
> data requests. Caladan [24] proposes a new CPU scheduler in the Linux kernel that improves
> QoS of LC services, especially the tail latency. Perséphone [20] is a novel kernel-bypass
> OS scheduler designed to minimize the tail latency for applications executing at the
> microsecond scale.

LC 服务优化。许多研究从降低尾延迟的角度出发，致力于提升 LC 服务的服务质量（QoS）[4, 20, 24, 46, 55]。例如，PerfIso [31] 利用原生 CPU 隔离来实现 LC 服务与尽力而为（best-effort）批处理作业之间的 CPU 共享。FlatStore [10] 是一种键值存储，它使用持久内存中的日志来提供数据请求服务。Caladan [24] 在 Linux 内核中提出了一种新的 CPU 调度器，用于提升 LC 服务的 QoS，尤其是尾延迟。Perséphone [20] 是一种新颖的旁路内核（kernel-bypass）操作系统调度器，旨在将微秒级应用的尾延迟降至最低。

> GC Optimization. There are studies on JVM optimization [5, 22, 28, 28, 38, 40, 43, 44, 52,
> 54]. For example, Facade [44] implements a compiler framework, which automatically stores
> data to native storage. ROLP [5] profiles object lifetime and uses the profiled
> information for efficient GC. Recently, Semeru [51] provides a distributed GC that
> offloads object tracing to memory servers so that tracing is performed closer to data in
> memory-disaggregated systems. Other studies focus on reducing the overhead of object
> serialization and caching intermediate data in memory. Deca [37] transforms application
> code to access native storage directly for performance improvement. Skyway [42] and
> FlashByte [54] propose to transfer objects from heap to native storage, which alleviates
> the serialization overhead. Gerenuk [41] is a compiler and runtime that aims to enable a
> JVM-based data-parallel system to achieve near native efficiency.

GC 优化。已有关于 JVM 优化的研究 [5, 22, 28, 28, 38, 40, 43, 44, 52, 54]。例如，Facade [44] 实现了一个编译器框架，可自动将数据存储到原生存储中。ROLP [5] 对对象生命周期进行剖析（profile），并利用剖析所得的信息实现高效的 GC。近来，Semeru [51] 提供了一种分布式 GC，将对象追踪卸载到内存服务器，从而在内存解耦（memory-disaggregated）系统中让追踪在离数据更近的地方执行。另一些研究则聚焦于降低对象序列化的开销，以及在内存中缓存中间数据。Deca [37] 对应用代码进行变换，使其直接访问原生存储以提升性能。Skyway [42] 和 FlashByte [54] 提出将对象从堆转移到原生存储，从而缓解序列化开销。Gerenuk [41] 是一个编译器和运行时，旨在使基于 JVM 的数据并行系统达到接近原生的效率。

> Resource Management in Multi-tenant Systems. Many studies [8–10, 23, 26, 32] focus on job
> scheduling and memory management in multi-tenant systems. For instance, HipsterCo [45] is
> a task scheduler for LC services running on multi-core systems. BIG-C [9] is a preemptive,
> low latency datacenter scheduling framework via lightweight containers. iTask [23] and
> UAkiller [7] forcibly suspend tasks and performs node-level memory management to
> prioritize specific tasks. Hermes [46] offers fast memory allocation and proactive
> reclamation for reducing the latency of LC services under memory pressure. It targets
> applications written in C. iGC is an adaptive CPU allocation and thread placement approach
> that leverages on concurrent GC at runtime to tackle performance interference and reduce
> the tail latency of LC services. It targets Java applications that require GC.

多租户系统中的资源管理。许多研究 [8–10, 23, 26, 32] 关注多租户系统中的作业调度与内存管理。例如，HipsterCo [45] 是一种面向运行在多核系统上的 LC 服务的任务调度器。BIG-C [9] 是一个借助轻量级容器实现的抢占式、低延迟数据中心调度框架。iTask [23] 和 UAkiller [7] 强制挂起任务，并执行节点级内存管理，以对特定任务进行优先调度。Hermes [46] 提供快速内存分配与主动回收，以降低内存压力下 LC 服务的延迟。它的目标应用是使用 C 语言编写的应用。iGC 是一种自适应的 CPU 分配与线程放置方法，它在运行时利用并发 GC 来应对性能干扰并降低 LC 服务的尾延迟。它面向需要 GC 的 Java 应用。

> Performance Interference and Isolation. Prior work has tackled performance interference in
> cloud systems in several ways. Some approaches disallow interactive services from sharing
> resources with other applications to avoid interference [36, 47, 50]. Some other
> approaches try to avoid co-scheduling applications that are likely to interfere with each
> other [18, 39]. The third category of research focus on eliminating interference
> altogether, by partitioning and isolating resources among co-scheduled services [19,
> 34–36, 46]. Parties [6] is a representative QoS-aware resource manager that enables a
> number of interactive LC services to share a physical node without QoS violations. iGC
> tackles the interference between GC threads and the worker threads of LC services, as well
> as the interference due to HT.

性能干扰与隔离。已有工作从多个方面着手解决云系统中的性能干扰问题。一些方法禁止交互式服务与其他应用共享资源，以避免干扰 [36, 47, 50]。另一些方法则试图避免将可能相互干扰的应用共同调度 [18, 39]。第三类研究侧重于彻底消除干扰，即通过在共同调度的服务之间划分并隔离资源 [19, 34–36, 46]。Parties [6] 是一个具有代表性的 QoS 感知资源管理器，它使多个交互式 LC 服务能够共享同一台物理节点而不违反 QoS。iGC 所应对的是 GC 线程与 LC 服务工作线程之间的干扰，以及由 HT 引起的干扰。

> SMT Optimization. Several studies [21, 33, 53] focus on optimizing application performance
> running on SMT servers. Deng [21] proposes a symbiotic scheduling approach to solve the
> resource contention between virtual machines on SMT processors. vsmt-io [53] proposes to
> improve I/O performance and efficiency on SMT processors in virtualized clouds. iGC
> proposes to place different kinds of threads onto different CPUs while taking advantage of
> memory hierarchy.

SMT 优化。若干研究 [21, 33, 53] 关注优化运行在 SMT 服务器上的应用性能。Deng [21] 提出一种共生调度（symbiotic scheduling）方法，以化解 SMT 处理器上虚拟机之间的资源争用。vsmt-io [53] 提出提升虚拟化云中 SMT 处理器上的 I/O 性能与效率。iGC 提出将不同种类的线程放置到不同的 CPU 上，同时充分利用存储层次。

## 7 结论（Conclusion）

> iGC is motivated by the performance degradation due to CPU interference between concurrent
> GC threads and worker threads of LC services. Scheduling GC threads and worker threads
> onto separate processing units opens up the opportunity for performance improvement of LC
> services. Furthermore, iGC tackles HT interference by placing worker threads and batch
> threads onto different cores, but also takes advantage of HT capability by prioritizing the
> placement of worker threads and GC threads in the same cores for high cache utilization.
> iGC is designed and implemented as a ready-to-use middleware for JVM-based applications in
> Linux environments. Experimental results show that iGC significantly improves the
> performance of two state-of-the-art concurrent GC mechanisms (ZGC and G1 GC) for LC
> services and the throughput in multi-tenant systems.

iGC 的动机源于并发 GC 线程与 LC 服务工作线程之间的 CPU 干扰所导致的性能退化。将 GC 线程与工作线程调度到不同的处理单元上，为提升 LC 服务的性能创造了机会。此外，iGC 通过将工作线程与批处理线程放置到不同核心上来应对 HT 干扰，同时也利用 HT 的能力，优先将工作线程与 GC 线程放置在同一核心中，以获得高缓存利用率。iGC 被设计与实现为一个即插即用的中间件，面向 Linux 环境中基于 JVM 的应用。实验结果表明，iGC 显著提升了两种最先进的并发 GC 机制（ZGC 和 G1 GC）面向 LC 服务的性能，以及多租户系统中的吞吐量。

> In future, we plan to examine iGC performance on different types of processors and extend
> it for other OSes.

未来，我们计划考察 iGC 在不同类型处理器上的性能，并将其扩展至其他操作系统。

## 8 致谢（Acknowledgment）

> This research was supported in part by U.S. NSF grant CCF-1816850.

本研究部分得到了美国 NSF 资助项目 CCF-1816850 的支持。

## 参考文献（References）

> 说明：以下参考文献列表保留英文原文，未作翻译。

[1] Apache. 2022. Apache Cassandra™. Retrieved Feb. 17, 2022 from https://cassandra.apache.org/

[2] Apache. 2022. Apache HBase™. Retrieved May 18, 2022 from https://hbase.apache.org/

[3] Apache. 2022. Apache Solr™. Retrieved May 12, 2022 from https://solr.apache.org/

[4] Daniel S Berger, Benjamin Berg, Timothy Zhu, Siddhartha Sen, and Mor Harchol-Balter. 2018. RobinHood: Tail Latency Aware Caching–Dynamic Reallocation from Cache-Rich to Cache-Poor. In 13th USENIX Symposium on Operating Systems Design and Implementation (OSDI 18). Oct. 8–10, 2018, CARLSBAD, CA, USA. USENIX Association, Carlsbad, CA, 195–212. https://doi.org/10.5555/3291168.3291183

[5] Rodrigo Bruno, Duarte Patricio, José Simão, Luis Veiga, and Paulo Ferreira. 2019. Runtime object lifetime profiler for latency sensitive big data applications. In Proceedings of the Fourteenth EuroSys Conference 2019. March 25-28, 2019, Dresden Germany. ACM, New York, NY, USA, 1–16. https://doi.org/10.1145/3302424.3303988

[6] Shuang Chen, Christina Delimitrou, and José F Martínez. 2019. Parties: QoS-aware resource partitioning for multiple interactive services. In Proceedings of the Twenty-Fourth International Conference on Architectural Support for Programming Languages and Operating Systems. April 13-17, 2019, Providence, Rhode Island, USA. ACM, New York, NY, USA, 107–120. https://doi.org/10.1145/3297858.3304005

[7] Wei Chen, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2019. OS-augmented oversubscription of opportunistic memory with a user-assisted oom killer. In Proceedings of the 20th International Middleware Conference. Dec. 9-13, 2019, Davis CA USA. ACM, New York, NY, USA, 28–40. https://doi.org/10.1145/3361525.3361534

[8] Wei Chen, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2019. Pufferfish: Container-driven elastic memory management for data-intensive applications. In Proceedings of the ACM Symposium on Cloud Computing. Nov. 20-23, 2019, Santa Cruz CA USA. ACM, New York, NY, USA, 259–271. https://doi.org/10.1145/3357223.3362730

[9] Wei Chen, Jia Rao, and Xiaobo Zhou. 2017. Preemptive, low latency datacenter scheduling via lightweight virtualization. In 2017 USENIX Annual Technical Conference (USENIX ATC 17). July 12–14, 2017, Santa Clara, CA. USENIX Association, Santa Clara, CA, 251–263. https://doi.org/10.5555/3154690.3154714

[10] Youmin Chen, Youyou Lu, Fan Yang, Qing Wang, Yang Wang, and Jiwu Shu. 2020. Flatstore: An efficient log-structured key-value storage engine for persistent memory. In Proceedings of the Twenty-Fifth International Conference on Architectural Support for Programming Languages and Operating Systems. March 16-20, 2020, Lausanne, Switzerland. ACM, New York, NY, USA, 1077–1091. https://doi.org/10.1145/3373376.3378515

[11] The Linux Kernel Development Community. 2012. Control Group v2. Retrieved Oct. 2015 from https://www.kernel.org/doc/html/v4.18/admin-guide/cgroup-v2.html

[12] Brian F Cooper, Adam Silberstein, Erwin Tam, Raghu Ramakrishnan, and Russell Sears. 2010. Benchmarking cloud serving systems with YCSB. In Proceedings of the 1st ACM symposium on Cloud computing. June 10-11, 2010, Indianapolis Indiana USA. ACM, New York, NY, USA, 143–154. https://doi.org/10.1145/1807128.1807152

[13] Oracle Corperation. 2016. Java Garbage Collection Stop The World Event. Retrieved May 20, 2016 from https://www.oracle.com/technetwork/tutorials/tutorials-1873457.html

[14] Intel Corporation. 2017. Intel® 64 and IA32 Architectures Performance Monitoring Events. Retrieved May 18, 2022 from https://software.intel.com/content/www/us/en/develop/download/intel-64-and-ia32-architectures-performance-monitoring-events.html

[15] Oracle Corporation. 2022. Java Management Extensions. Retrieved May 18, 2022 from https://docs.oracle.com/javase/8/docs/technotes/guides/management/agent.html

[16] Oracle Corporation. 2022. The Z Garbage Collector. Retrieved May 18, 2022 from https://openjdk.java.net/projects/zgc/

[17] Jeffrey Dean and Luiz André Barroso. 2013. The tail at scale. Commun. ACM 56, 2 (2013), 74–80. https://doi.org/10.1145/2408776.2408794

[18] Christina Delimitrou and Christos Kozyrakis. 2013. Paragon: QoS-aware scheduling for heterogeneous datacenters. ACM SIGPLAN Notices 48, 4 (2013), 77–88. https://doi.org/10.1145/2451116.2451125

[19] Christina Delimitrou and Christos Kozyrakis. 2017. Bolt: I know what you did last summer... in the cloud. ACM SIGARCH Computer Architecture News 45, 1 (2017), 599–613. https://doi.org/10.1145/3093337.3037703

[20] Henri Maxime Demoulin, Joshua Fried, Isaac Pedisich, Marios Kogias, Boon Thau Loo, Linh Thi Xuan Phan, and Irene Zhang. 2021. When Idling is Ideal: Optimizing Tail-Latency for Heavy-Tailed Datacenter Workloads with Perséphone. In Proceedings of the ACM SIGOPS 28th Symposium on Operating Systems Principles. Oct. 26-29, 2021, Virtual Event Germany. ACM, New York, NY, USA, 621–637. https://doi.org/10.1145/3477132.3483571

[21] Kefeng Deng, Kaijun Ren, and Junqiang Song. 2012. Symbiotic scheduling for virtual machines on smt processors. In 2012 Second International Conference on Cloud and Green Computing. Nov. 1-3, 2012, Xiangtan China. IEEE, 145–152. https://doi.org/10.1109/CGC.2012.54

[22] David Detlefs, Christine Flood, Steve Heller, and Tony Printezis. 2004. Garbage-first garbage collection. In Proceedings of the 4th international symposium on Memory management. Oct. 24-25, 2004, Vancouver BC Canada. ACM, New York, NY, USA, 37–48. https://doi.org/10.1145/1029873.1029879

[23] Lu Fang, Khanh Nguyen, Guoqing Xu, Brian Demsky, and Shan Lu. 2015. Interruptible tasks: Treating memory pressure as interrupts for highly scalable data-parallel programs. In Proceedings of the 25th Symposium on Operating Systems Principles. Oct. 4-7, 2015, Monterey California. ACM, New York, NY, USA, 394–409. https://doi.org/10.1145/2815400.2815407

[24] Joshua Fried, Zhenyuan Ruan, Amy Ousterhout, and Adam Belay. 2020. Caladan: Mitigating interference at microsecond timescales. In 14th USENIX Symposium on Operating Systems Design and Implementation (OSDI 20). Nov. 4-6, 2020, virtual. USENIX Association, 281–297. https://doi.org/10.5555/3488766.3488782

[25] Yu Gan and Christina Delimitrou. 2018. The architectural implications of cloud microservices. IEEE Computer Architecture Letters 17, 2 (2018), 155–158. https://doi.org/10.1109/LCA.2018.2839189

[26] Eran Gilad, Edward Bortnikov, Anastasia Braginsky, Yonatan Gottesman, Eshcar Hillel, Idit Keidar, Nurit Moscovici, and Rana Shahout. 2020. EvenDB: optimizing key-value storage for spatial locality. In Proceedings of the Fifteenth European Conference on Computer Systems. April 27-30, 2020, Heraklion Greece. ACM, New York, NY, USA, 1–16. https://doi.org/10.1145/3342195.3387523

[27] Jambit GmbH. 2021. nice - change process priority. Retrieved Aug. 27, 2021 from https://man7.org/linux/man-pages/man2/nice.2.html

[28] Ionel Gog, Jana Giceva, Malte Schwarzkopf, Kapil Vaswani, Dimitrios Vytiniotis, Ganesan Ramalingam, Manuel Costa, Derek G Murray, Steven Hand, and Michael Isard. 2015. Broom: Sweeping out garbage collection from big data systems. In 15th Workshop on Hot Topics in Operating Systems (HotOS XV). May 18-20, 2015, Kartause Ittingen, Switzerland. USENIX Association, USA. https://doi.org/10.5555/2831090.2831092

[29] Wessam M Hassanein, Layali K Rashid, and Moustafa A Hammad. 2008. Analyzing the effects of hyperthreading on the performance of data management systems. International Journal of Parallel Programming 36, 2 (2008), 206–225. https://doi.org/10.1007/s10766-007-0066-x

[30] Intel-bigdata. 2020. HiBench Suite. Retrieved Jan. 11, 2022 from https://github.com/Intel-bigdata/HiBench

[31] Călin Iorgulescu, Reza Azimi, Youngjin Kwon, Sameh Elnikety, Manoj Syamala, Vivek Narasayya, Herodotos Herodotou, Paulo Tomita, Alex Chen, Jack Zhang, et al. 2018. PerfIso: Performance Isolation for Commercial Latency-Sensitive Services. In 2018 USENIX Annual Technical Conference (USENIX ATC 18). July 11–13, 2018, Boston, MA, USA. USENIX Association, USA, 519–532. https://doi.org/10.5555/3277355.3277406

[32] Călin Iorgulescu, Florin Dinu, Aunn Raza, Wajih Ul Hassan, and Willy Zwaenepoel. 2017. Don't cry over spilled records: Memory elasticity of data-parallel applications and its application to cluster scheduling. In 2017 USENIX Annual Technical Conference (USENIX ATC 17). July 12–14, 2017, Santa Clara, CA. USENIX Association, Santa Clara, CA, 97–109. https://doi.org/10.5555/3154690.3154700

[33] Weiwei Jia, Jianchen Shan, Tsz On Li, Xiaowei Shang, Heming Cui, and Xiaoning Ding. 2020. vSMT-IO: Improving I/O Performance and Efficiency on SMT Processors in Virtualized Clouds. In 2020 USENIX Annual Technical Conference (USENIX ATC 20). July 15–17, 2020, virtual. USENIX Association, USA, 449–463. https://doi.org/10.5555/3489146.3489176

[34] Harshad Kasture, Davide B Bartolini, Nathan Beckmann, and Daniel Sanchez. 2015. Rubik: Fast analytical power management for latency-critical systems. In 2015 48th Annual IEEE/ACM International Symposium on Microarchitecture (MICRO). Dec. 5-12, 2015, Waikiki, HI, USA. IEEE, 598–610. https://doi.org/10.1145/2830772.2830797

[35] Palden Lama, Shaoqi Wang, Xiaobo Zhou, and Dazhao Cheng. 2018. Performance isolation of data-intensive scale-out applications in a multi-tenant cloud. In 2018 IEEE International Parallel and Distributed Processing Symposium (IPDPS). May 21–25, 2018, Vancouver, British Columbia CANADA. IEEE, 85–94. https://doi.org/10.1109/IPDPS.2018.00019

[36] David Lo, Liqun Cheng, Rama Govindaraju, Parthasarathy Ranganathan, and Christos Kozyrakis. 2015. Heracles: Improving resource efficiency at scale. In Proceedings of the 42nd Annual International Symposium on Computer Architecture. June 13-17, 2015, Portland Oregon. 450–462. https://doi.org/10.1145/2749469.2749475

[37] Lu Lu, Xuanhua Shi, Yongluan Zhou, Xiong Zhang, Hai Jin, Cheng Pei, Ligang He, and Yuanzhen Geng. 2016. Lifetime-Based Memory Management for Distributed Data Processing Systems. Proc. VLDB Endow. 9, 12 (2016). https://doi.org/10.14778/2994509.2994513

[38] Martin Maas, Tim Harris, Krste Asanović, and John Kubiatowicz. 2015. Trash day: Coordinating garbage collection in distributed systems. In 15th Workshop on Hot Topics in Operating Systems (HotOS XV). May 18-20, 2015, Kartause Ittingen, Switzerland. USENIX Association, USA. https://doi.org/10.5555/2831090.2831091

[39] Jason Mars and Lingjia Tang. 2013. Whare-map: Heterogeneity in "homogeneous" warehouse-scale computers. In Proceedings of the 40th Annual International Symposium on Computer Architecture. June 23-27, 2013, Tel-Aviv Israel, Vol. 41. ACM, New York, NY, USA, 619–630. https://doi.org/10.1145/2508148.2485975

[40] Kris Mok. 2011. JVM@Taobao. Retrieved Oct. 24, 2011 from https://www.slideshare.net/RednaxelaFX/jvm-taobao

[41] Christian Navasca, Cheng Cai, Khanh Nguyen, Brian Demsky, Shan Lu, Miryung Kim, and Guoqing Harry Xu. 2019. Gerenuk: thin computation over big native data using speculative program transformation. In Proceedings of the 27th ACM Symposium on Operating Systems Principles. Oct. 27-30, 2019, Huntsville Ontario Canada. ACM, New York, NY, USA, 538–553. https://doi.org/10.1145/3341301.3359643

[42] Khanh Nguyen, Lu Fang, Christian Navasca, Guoqing Xu, Brian Demsky, and Shan Lu. 2018. Skyway: Connecting managed heaps in distributed big data systems. ACM SIGPLAN Notices 53, 2 (2018), 56–69. https://doi.org/10.1145/3173162.3173200

[43] Khanh Nguyen, Lu Fang, Guoqing Xu, Brian Demsky, Shan Lu, Sanazsadat Alamian, and Onur Mutlu. 2016. Yak: A High-Performance Big-Data-Friendly Garbage Collector. In 12th USENIX Symposium on Operating Systems Design and Implementation (OSDI 16). Nov. 2–4, 2016, SAVANNAH, GA, USA. USENIX Association, USA, 349–365. https://doi.org/10.5555/3026877.3026905

[44] Khanh Nguyen, Kai Wang, Yingyi Bu, Lu Fang, Jianfei Hu, and Guoqing Xu. 2015. Facade: A compiler and runtime for (almost) object-bounded big data applications. ACM SIGARCH Computer Architecture News 43, 1 (2015), 675–690. https://doi.org/10.1145/2775054.2694345

[45] Rajiv Nishtala, Paul Carpenter, Vinicius Petrucci, and Xavier Martorell. 2017. Hipster: Hybrid task manager for latency-critical cloud workloads. In 2017 IEEE International Symposium on High Performance Computer Architecture (HPCA). Feb. 4-8 2017, Austin, Texas. IEEE, 409–420. https://doi.org/10.1109/HPCA.2017.13

[46] Aidi Pi, Junxian Zhao, Shaoqi Wang, and Xiaobo Zhou. 2021. Memory at your service: fast memory allocation for latency-critical services. In Proceedings of the 22nd International Middleware Conference. Dec. 6-10, 2021, Québec city Canada. ACM, New York, NY, USA, 185–197. https://doi.org/10.1145/3464298.3493394

[47] Malte Schwarzkopf, Andy Konwinski, Michael Abd-El-Malek, and John Wilkes. 2013. Omega: Flexible, Scalable Schedulers for Large Compute Clusters. In Proceedings of the 8th ACM European Conference on Computer Systems. April 15-17, 2013, Prague Czech Republic. ACM, New York, NY, USA, 351–364. https://doi.org/10.1145/2465351.2465386

[48] Konstantin Shvachko, Hairong Kuang, Sanjay Radia, and Robert Chansler. 2010. The hadoop distributed file system. In 2010 IEEE 26th symposium on mass storage systems and technologies (MSST). May 3-7, 2010, Lake Tahoe, Nevada, USA. IEEE, 1–10. https://doi.org/10.1109/MSST.2010.5496972

[49] Vinod Kumar Vavilapalli, Arun C Murthy, Chris Douglas, Sharad Agarwal, Mahadev Konar, Robert Evans, Thomas Graves, Jason Lowe, Hitesh Shah, Siddharth Seth, et al. 2013. Apache hadoop yarn: Yet another resource negotiator. In Proceedings of the 4th annual Symposium on Cloud Computing. Oct. 1-3, 2013, Santa Clara California. ACM, New York, NY, USA, 1–16. https://doi.org/10.1145/2523616.2523633

[50] Abhishek Verma, Luis Pedrosa, Madhukar Korupolu, David Oppenheimer, Eric Tune, and John Wilkes. 2015. Large-Scale Cluster Management at Google with Borg. In Proceedings of the Tenth European Conference on Computer Systems. April 21-24, 2015, Bordeaux France. Association for Computing Machinery, New York, NY, USA, 1–17. https://doi.org/10.1145/2741948.2741964

[51] Chenxi Wang, Haoran Ma, Shi Liu, Yuanqi Li, Zhenyuan Ruan, Khanh Nguyen, Michael D Bond, Ravi Netravali, Miryung Kim, and Guoqing Harry Xu. 2020. Semeru: A Memory-Disaggregated Managed Runtime. In 14th USENIX Symposium on Operating Systems Design and Implementation (OSDI 20). Nov. 4-6, 2020, virtual. USENIX Association, USA, 261–280. https://doi.org/10.5555/3488766.3488781

[52] Jingjing Wang and Magdalena Balazinska. 2017. Elastic memory management for cloud data analytics. In 2017 USENIX Annual Technical Conference (USENIX ATC 17). July 12–14, 2017, Santa Clara, CA. USENIX Association, USA, 745–758. https://doi.org/10.5555/3154690.3154760

[53] Xi Yang, Stephen M Blackburn, and Kathryn S McKinley. 2016. Elfen Scheduling: Fine-Grain Principled Borrowing from Latency-Critical Workloads Using Simultaneous Multithreading. In 2016 USENIX Annual Technical Conference (USENIX ATC 16). July 12-14, 2017, Santa Clara, CA. USENIX Association, USA, 309–322. https://doi.org/10.5555/3026959.3026988

[54] Junxian Zhao, Aidi Pi, Shaoqi Wang, and Xiaobo Zhou. 2021. Flashbyte: Improving memory efficiency with lightweight native storage. In 2021 IEEE/ACM 21st International Symposium on Cluster, Cloud and Internet Computing (CCGrid). May 10-13, 2021, Melbourne, Australia. IEEE, 61–70. https://doi.org/10.1109/CCGrid51090.2021.00016

[55] Timothy Zhu, Michael A Kozuch, and Mor Harchol-Balter. 2017. Workload-compactor: Reducing datacenter cost while providing tail latency slo guarantees. In Proceedings of the 2017 Symposium on Cloud Computing. Sept. 24-27, 2017, Santa Clara California. ACM, New York, NY, USA, 598–610. https://doi.org/10.1145/3127479.3132245


