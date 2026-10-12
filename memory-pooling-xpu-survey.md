# 池化内存管理与 CPU/XPU 协同：文献、工具与研究组

**整理日期：** 2026-10-12  
**覆盖范围：** CXL/远端或分解式内存池；CPU-XPU/GPU 统一内存与协同管理；内存分配器、运行时、设备管理与观测工具；Go 语言相关库、运行时及研究/工程生态

> 本文是面向系统软件和研究原型的定向梳理，不是穷尽式系统综述。条目按与主题的关系分为直接相关、邻近支撑和背景资料。

## 先读结论

- “池化内存”至少有三类：CXL/远端主机内存池、GPU/XPU 显存池、Go 进程内对象/缓冲区复用。它们分属不同层级，不能仅凭名称视为同一种技术。
- CPU/XPU 协同的核心问题包括分配和回收、页放置与迁移、地址空间/可见性、带宽竞争、QoS、碎片和生命周期。
- Go 生态目前更适合写控制面和应用接入层；底层设备分配一般通过原生 runtime 和 cgo 实现。此次定向检索未发现成熟的 Go 专用 CXL+XPU 通用内存池或相应专门研究组。

## 术语与边界

- **memory_pooling：** 将分散或专用的内存容量按需汇集、分配与回收；本资料主要指 CXL/远端内存池，也涵盖 GPU/加速器显存池。
- **cpu_xpu_coordination：** CPU、GPU 或其他加速器在地址空间、内存放置、迁移、分配、带宽/QoS 与生命周期上的协同。
- **scope_warning：** CXL 内存池、GPU 显存分配器、统一虚拟内存和 Go sync.Pool 是不同层次的机制；相似的“pool”名称不代表可互换。

## 推荐阅读顺序

- [Memory Pooling With CXL](https://doi.org/10.1109/mm.2023.3237491)（2023，IEEE Micro）：介绍 CXL 内存池架构与设计空间，适合作为 CXL pooling 的入门概览。
- [Pond: CXL-Based Memory Pooling Systems for Cloud Platforms](https://doi.org/10.1145/3575693.3578835)（2023，ASPLOS）：代表性的 CXL 云内存池系统工作，研究池化容量如何服务云平台工作负载及系统软件如何管理资源。适合作为架构与动机起点。
- [A Case Against CXL Memory Pooling](https://doi.org/10.1145/3626111.3628195)（2023，HotNets）：从成本、性能和实际部署假设方面质疑 CXL pooling 的收益。建议与支持 pooling 的系统论文对读，检查工作负载、带宽、扩展和利用率假设。
- [TPP: Transparent Page Placement for CXL-Enabled Tiered-Memory](https://doi.org/10.1145/3582016.3582063)（2023，ASPLOS）：通过透明页放置管理 CXL 分层内存，重点在页迁移和冷热页识别。虽是 tiering 而非所有定义下的共享池，但和 CPU 侧池化内存管理高度相关。
- [Clio: A Hardware-Software Co-Designed Disaggregated Memory System](https://doi.org/10.1145/3503222.3507762)（2022，ASPLOS）：硬件/软件协同设计的分解式内存系统，提供 CXL 之外的远端池化内存架构参照。
- [An Adaptive Framework for Oversubscription Management in CPU-GPU Unified Memory](https://doi.org/10.23919/DATE51398.2021.9473982)（2021，DATE）：面向 CPU-GPU Unified Memory 超额订阅的自适应管理，关注容量压力和迁移行为。
- [GMLake: Efficient and Transparent GPU Memory Defragmentation for Large-scale DNN Training with Virtual Memory Stitching](https://doi.org/10.1145/3620665.3640423)（2024，ASPLOS）：利用虚拟内存拼接缓解大规模 DNN 训练中的 GPU 内存碎片问题，是 XPU 显存管理与透明分配的代表工作。
- [Host-Based Allocators for Device Memory](https://arxiv.org/abs/2405.07079)（2024，arXiv）：研究由主机端管理设备内存的分配器设计，和统一的 CPU/XPU 分配接口直接相关。预印本，需核对最新状态。
- [StarPU: A Unified Platform for Task Scheduling on Heterogeneous Multicore Architectures](https://doi.org/10.1002/cpe.1631)（2010，Concurrency and Computation: Practice and Experience）：异构任务运行时，支持 CPU/GPU 上任务调度与数据管理。不是远端内存池，但可作为 CPU/XPU 生命周期和数据移动运行时的参考。
- [Profile-driven Memory Bandwidth Management for Accelerators and CPUs in QoS-enabled Platforms](https://doi.org/10.1007/s11241-022-09382-x)（2022，Real-Time Systems）：直接关注 QoS 平台上 CPU 与加速器的内存带宽管理，和池化容量管理的性能隔离问题相关。

## 文献

| ID | 年份 / 类型 | 论文 | 主题关联 | 简介 |
|---|---|---|---|---|
| L01 | 2023 / conference paper | [Pond: CXL-Based Memory Pooling Systems for Cloud Platforms](https://doi.org/10.1145/3575693.3578835) | direct | 代表性的 CXL 云内存池系统工作，研究池化容量如何服务云平台工作负载及系统软件如何管理资源。适合作为架构与动机起点。 |
| L02 | 2023 / journal/magazine article | [Memory Pooling With CXL](https://doi.org/10.1109/mm.2023.3237491) | direct | 介绍 CXL 内存池架构与设计空间，适合作为 CXL pooling 的入门概览。 |
| L03 | 2023 / workshop paper | [A Case Against CXL Memory Pooling](https://doi.org/10.1145/3626111.3628195) | direct | 从成本、性能和实际部署假设方面质疑 CXL pooling 的收益。建议与支持 pooling 的系统论文对读，检查工作负载、带宽、扩展和利用率假设。 |
| L04 | 2023 / journal/magazine article | [Dynamic Capacity Service for Improving CXL Pooled Memory Efficiency](https://doi.org/10.1109/mm.2023.3237756) | direct | 聚焦池化内存容量的动态服务与利用率，适合研究容量分配策略和资源管理接口。 |
| L05 | 2022 / conference paper | [Evaluating Emerging CXL-enabled Memory Pooling for HPC Systems](https://doi.org/10.1109/MCHPC56545.2022.00007) | direct | 从 HPC 系统角度评估 CXL 内存池，适合作为性能评估指标和 HPC 场景的参考。 |
| L06 | 2023 / preprint | [CXLMemUring: A Hardware Software Co-design Paradigm for Asynchronous and Flexible Parallel CXL Memory Pool Access](https://arxiv.org/abs/2309.04011) | direct | 探索基于 io_uring 的异步、并行 CXL 池访问路径，适合关注数据面延迟、并发和异步 API 的读者。预印本，阅读时应核对后续版本。 |
| L07 | 2023 / conference paper | [TPP: Transparent Page Placement for CXL-Enabled Tiered-Memory](https://doi.org/10.1145/3582016.3582063) | direct | 通过透明页放置管理 CXL 分层内存，重点在页迁移和冷热页识别。虽是 tiering 而非所有定义下的共享池，但和 CPU 侧池化内存管理高度相关。 |
| L08 | 2022 / conference paper | [Clio: A Hardware-Software Co-Designed Disaggregated Memory System](https://doi.org/10.1145/3503222.3507762) | direct | 硬件/软件协同设计的分解式内存系统，提供 CXL 之外的远端池化内存架构参照。 |
| L09 | 2019 / journal/magazine article | [A Disaggregated Memory System for Deep Learning](https://doi.org/10.1109/MM.2019.2929165) | direct | 较早研究深度学习系统中的分解式内存，适合了解加速器工作负载与远端容量的结合动机。 |
| L10 | 2024 / conference paper | [DRackSim: Simulating CXL-enabled Large-Scale Disaggregated Memory Systems](https://doi.org/10.1145/3615979.3656059) | direct | CXL 分解式内存系统模拟器，适合在硬件不可用时探索大规模拓扑与策略。 |
| L11 | 2019 / conference paper | [Nimble Page Management for Tiered Memory Systems](https://doi.org/10.1145/3297858.3304024) | adjacent | 研究分层内存页管理，可作为 CXL tiering 策略与页迁移设计的基础参考。 |
| L12 | 2021 / conference paper | [An Adaptive Framework for Oversubscription Management in CPU-GPU Unified Memory](https://doi.org/10.23919/DATE51398.2021.9473982) | direct | 面向 CPU-GPU Unified Memory 超额订阅的自适应管理，关注容量压力和迁移行为。 |
| L13 | 2017 / conference paper | [Analyzing Memory Management Methods on Integrated CPU-GPU Systems](https://doi.org/10.1145/3092255.3092256) | direct | 对集成 CPU-GPU 系统的内存管理方法进行分析，适合作为统一/分离内存行为对照。 |
| L14 | 2016 / journal article | [Building Heterogeneous Unified Virtual Memories (UVMs) without the Overhead](https://doi.org/10.1145/2889488) | direct | 研究异构统一虚拟内存机制及其开销，适合关注 CPU/XPU 地址空间与访问代价的读者。 |
| L15 | 2020 / conference paper | [Efficient GPU Memory Management for Nonlinear DNNs](https://doi.org/10.1145/3369583.3392684) | adjacent | 优化非线性 DNN 工作负载的 GPU 内存管理，可辅助理解显存复用、峰值和分配开销。 |
| L16 | 2024 / conference paper | [GMLake: Efficient and Transparent GPU Memory Defragmentation for Large-scale DNN Training with Virtual Memory Stitching](https://doi.org/10.1145/3620665.3640423) | direct | 利用虚拟内存拼接缓解大规模 DNN 训练中的 GPU 内存碎片问题，是 XPU 显存管理与透明分配的代表工作。 |
| L17 | 2024 / preprint | [Host-Based Allocators for Device Memory](https://arxiv.org/abs/2405.07079) | direct | 研究由主机端管理设备内存的分配器设计，和统一的 CPU/XPU 分配接口直接相关。预印本，需核对最新状态。 |
| L18 | 2018 / preprint | [SuperNeurons: Dynamic GPU Memory Management for Training Deep Neural Networks](https://arxiv.org/abs/1801.04380) | adjacent | 讨论训练阶段的动态 GPU 内存管理和复用策略，可作为显存池及生命周期优化的背景。 |
| L19 | 2019 / survey | [Survey on Memory Management Techniques in Heterogeneous Computing Systems](https://doi.org/10.1049/iet-cdt.2019.0092) | background | 异构计算内存管理综述，可用于扩展关键词、分类和早期文献。 |
| L20 | 2015 / survey | [A Survey of CPU-GPU Heterogeneous Computing Techniques](https://doi.org/10.1145/2788396) | background | CPU-GPU 异构计算综述，范围比内存管理更广，适合补充系统背景。 |
| L21 | 2010 / journal article | [StarPU: A Unified Platform for Task Scheduling on Heterogeneous Multicore Architectures](https://doi.org/10.1002/cpe.1631) | adjacent | 异构任务运行时，支持 CPU/GPU 上任务调度与数据管理。不是远端内存池，但可作为 CPU/XPU 生命周期和数据移动运行时的参考。 |
| L22 | 2020 / conference paper | [E-WarP: A System-wide Framework for Memory Bandwidth Profiling and Management](https://doi.org/10.1109/RTSS49844.2020.00039) | adjacent | 系统级内存带宽剖析和管理框架，可借鉴 CPU 与加速器共享带宽下的 QoS 观测和控制。 |
| L23 | 2022 / journal article | [Profile-driven Memory Bandwidth Management for Accelerators and CPUs in QoS-enabled Platforms](https://doi.org/10.1007/s11241-022-09382-x) | direct | 直接关注 QoS 平台上 CPU 与加速器的内存带宽管理，和池化容量管理的性能隔离问题相关。 |
| L24 | 2018 / journal article | [GFlink: An In-Memory Computing Architecture on Heterogeneous CPU-GPU Clusters for Big Data](https://doi.org/10.1109/TPDS.2018.2794343) | adjacent | CPU-GPU 集群上的内存计算架构，适合作为数据处理和内存驻留工作负载的系统级参考。 |
| L25 | 2012 / conference paper | [TAP: A TLP-aware Cache Management Policy for a CPU-GPU Heterogeneous Architecture](https://doi.org/10.1109/HPCA.2012.6168947) | background | CPU-GPU 异构架构缓存管理的早期工作，非池化内存，但可帮助理解共享内存层级和竞争。 |
| L26 | 2018 / conference/journal record | [Devirtualizing Memory in Heterogeneous Systems](https://doi.org/10.1145/3296957.3173194) | adjacent | 探讨异构系统中的内存虚拟化设计；可用于分析虚拟地址、物理放置与设备可见性之间的接口问题。 |
| L27 | 2015 / conference paper | [Unified Virtual Memory Support for Deep CNN Accelerator on SoC FPGA](https://doi.org/10.1007/978-3-319-27119-4_5) | adjacent | SoC FPGA CNN 加速器的统一虚拟内存支持，属于 CPU 与专用加速器协同管理的早期案例。 |
| L28 | 2019 / master thesis | [GoA: Actors with Locally Managed Memory for Go](https://hdl.handle.net/10012/14430) | adjacent | 研究 Go actor 的局部内存管理与回收语义，适合了解 Go 运行时和对象所有权的研究路径；不涉及 CXL 或 GPU 显存池。 |

## 工具与项目

关联度说明：`direct` = 直接管理 CXL/XPU 内存或 CPU-XPU 协同；`adjacent` = 异构运行时或设备分配支撑；`support` = 主机对象池、拓扑、观测或编排工具。

### OS / CXL memory

- **Linux CXL subsystem**（C/kernel；direct）：Linux CXL 驱动、设备和内存区域支持文档；用于确认 Type-3 内存设备、区域和主机端接入路径。具体能力依赖内核版本、固件和硬件。 [主页](https://docs.kernel.org/driver-api/cxl/)

### CXL administration

- **ndctl / cxl-cli**（C；direct）：Linux 上管理和检查 NVDIMM/CXL 相关设备的用户态工具集合；用于枚举设备、区域和配置。按发行版、驱动与设备代际核对命令支持。 [主页](https://github.com/pmem/ndctl)

### OS / placement

- **Linux memory tiering / NUMA policies**（Linux kernel；direct）：NUMA 内存策略与节点放置基础；可将 CXL 暴露的内存节点纳入放置、绑定和迁移实验。 [主页](https://docs.kernel.org/admin-guide/mm/numa_memory_policy.html)

### OS / memory telemetry

- **DAMON**（Linux kernel；support）：Linux 数据访问监测框架，可用于分析工作集热度并探索回收、迁移与分层策略。 [主页](https://docs.kernel.org/mm/damon/index.html)

### NUMA placement

- **numactl / libnuma**（C；support）：查看 NUMA 拓扑并设置 CPU/内存亲和性；适合作为 DRAM 与 CXL 节点基线工具。 [主页](https://github.com/numactl/numactl)

### topology discovery

- **hwloc**（C；support）：发现 CPU、NUMA、PCIe 与设备拓扑，为内存池和 XPU 放置策略提供拓扑输入。 [主页](https://www.open-mpi.org/projects/hwloc/)

### CXL simulator

- **DRackSim**（research simulator；direct）：CXL 大规模分解式内存系统模拟研究；建议先确认代码是否公开及其模型假设，再用于复现实验。 [主页](https://doi.org/10.1145/3615979.3656059)

### heterogeneous allocator

- **Umpire**（C++；direct）：LLNL 的多资源内存管理器，面向 NUMA 和 GPU 资源，可提供 allocator/resource 组合和策略。适合做 CPU/XPU 分配管理原型。 [主页](https://github.com/LLNL/Umpire) · [文档](https://umpire.readthedocs.io/en/develop/)

### GPU allocator

- **RAPIDS RMM**（C++ / Python；direct）：NVIDIA GPU 内存分配与资源管理库，提供设备内存资源和池化分配能力；主要面向 CUDA，不是 CXL 主机内存池管理器。 [主页](https://github.com/rapidsai/rmm) · [文档](https://docs.nvidia.com/rmm/latest/)
- **Vulkan Memory Allocator (VMA)**（C++；adjacent）：Vulkan GPU 显存分配库，支持子分配和预算查询；其“memory pool”主要是 Vulkan device memory 子分配，不是 CPU 可共享池。 [主页](https://github.com/GPUOpen-LibrariesAndSDKs/VulkanMemoryAllocator)

### GPU allocator API

- **CUDA memory pools / cudaMallocAsync**（CUDA C++；direct）：CUDA stream-ordered 分配和 memory pool API，可管理设备内存复用及访问属性；适用于 CUDA 设备侧显存池。 [主页](https://docs.nvidia.com/cuda/cuda-programming-guide/04-special-topics/stream-ordered-memory-allocation.html)
- **HIP memory management / memory pools**（HIP C++；direct）：AMD ROCm/HIP 的设备内存分配与内存池接口，便于构建 AMD GPU 侧分配器。 [主页](https://rocm.docs.amd.com/projects/HIP/en/latest/how-to/hip_runtime_api/memory_management.html)

### CPU-GPU memory

- **CUDA Unified Memory**（CUDA C++；direct）：CUDA managed memory 与迁移/预取机制，用于 CPU-GPU 共享虚拟地址空间工作负载；它不等同于 CXL 共享内存池。 [主页](https://docs.nvidia.com/cuda/cuda-programming-guide/04-special-topics/unified-memory.html)

### CPU-XPU memory API

- **SYCL Unified Shared Memory (USM)**（C++ / SYCL；direct）：SYCL 的 host/device/shared allocation 模型，提供异构共享或显式设备分配接口。不同设备对 shared USM 的支持和性能有差异。 [主页](https://registry.khronos.org/SYCL/specs/sycl-2020/html/sycl-2020.html)

### heterogeneous runtime

- **StarPU**（C；direct）：CPU/GPU 等异构设备任务调度与数据管理运行时，适合作为数据生命周期、传输和异构任务协同的参考。 [主页](https://starpu.gitlabpages.inria.fr/)

### heterogeneous programming / memory spaces

- **Kokkos**（C++；support）：性能可移植编程模型，提供多种执行空间和内存空间抽象；适合验证应用层 CPU/XPU 数据结构与分配方式。 [主页](https://github.com/kokkos/kokkos)

### host allocator / analytics

- **Apache Arrow MemoryPool**（C++ (多语言绑定)；adjacent）：Arrow 的主机端缓冲区分配抽象，适合列式数据的 CPU 端内存管理；默认并不管理 GPU 或 CXL 设备。 [主页](https://arrow.apache.org/docs/cpp/memory.html)

### host allocator baseline

- **jemalloc**（C；support）：通用主机内存分配器，可作为 CPU 主机分配与碎片行为基线；不是 XPU/CXL 池管理器。 [主页](https://github.com/jemalloc/jemalloc)
- **mimalloc**（C；support）：通用高性能主机分配器，适合作为 CPU 堆分配器对照组。 [主页](https://github.com/microsoft/mimalloc)

### Go host object reuse

- **Go sync.Pool**（Go standard library；support）：降低短生命周期临时对象分配压力的标准库对象复用设施；GC 可清空内容，不能用于要求确定性生命周期的 GPU/CXL 资源池。 [主页](https://pkg.go.dev/sync#Pool)

### Go runtime control

- **Go runtime/debug.SetMemoryLimit**（Go standard library；support）：设置 Go runtime 的软内存目标以影响 GC 行为；不是硬上限，也不直接限制 C 分配、GPU 显存或 CXL 设备内存。 [主页](https://pkg.go.dev/runtime/debug#SetMemoryLimit)

### Go native interop

- **Go cgo**（Go / C；support）：连接 Go 控制面与 CUDA/HIP/oneAPI、Umpire、RMM 等原生数据面的一种入口。需明确释放责任、异步操作生命周期及 cgo 指针规则。 [主页](https://pkg.go.dev/cmd/cgo)
- **Go runtime.Pinner**（Go standard library；support）：在受支持的 cgo 场景中固定 Go 对象的 GC 地址；不提供 OS 页锁定或 DMA 所需的 pinned host allocation。 [主页](https://pkg.go.dev/runtime#Pinner)

### Go host buffer pool

- **Go bytebufferpool**（Go；support）：复用 Go 字节缓冲区，适合网络和临时 CPU buffer；不提供 pinned host memory 或设备分配。 [主页](https://github.com/valyala/bytebufferpool)
- **libp2p go-buffer-pool**（Go；support）：轻量 Go 字节切片复用工具；只解决进程内 host buffer 分配。 [主页](https://github.com/libp2p/go-buffer-pool)

### Go object/buffer pool

- **gobwas/pool**（Go；support）：Go pool helper，适合作为 host 对象/缓冲区复用组件。 [主页](https://github.com/gobwas/pool)

### Go generic resource pool

- **go-commons-pool**（Go；support）：通用对象资源池，提供池化生命周期管理思路；资源是否可复用及何时释放仍需应用层实现。 [主页](https://github.com/jolestar/go-commons-pool)

### Go ML framework

- **Gorgonia**（Go；adjacent）：Go 语言机器学习/计算图生态，可作为 Go 侧 XPU 工作负载入口；后端兼容性、设备支持和维护状态应按具体版本验证。 [主页](https://github.com/gorgonia/gorgonia)

### Go CUDA bindings

- **gorgonia/cu**（Go / cgo / CUDA；adjacent）：Gorgonia 生态的 CUDA 互操作项目；可供 Go 原型连接原生 CUDA，但不属于通用显存池管理器。 [主页](https://github.com/gorgonia/cu)

### Go tensor library

- **gorgonia/tensor**（Go；adjacent）：Go 张量数据结构项目，面向应用层数据表示；需要另接可用的设备后端和显式设备内存管理。 [主页](https://github.com/gorgonia/tensor)

### Go XPU bindings

- **go-opencl**（Go / cgo / OpenCL；adjacent）：OpenCL Go bindings，可作为跨厂商设备接口原型入口；发布版本、系统 OpenCL runtime 与硬件兼容性需确认。 [主页](https://github.com/jgillich/go-opencl)

### Go inference binding

- **onnxruntime_go**（Go / cgo / ONNX Runtime；adjacent）：Go 对 ONNX Runtime 的封装；能否使用 CUDA/其他加速执行提供程序取决于所链接的 ONNX Runtime 构建，不负责池化内存策略。 [主页](https://github.com/yalue/onnxruntime_go)

### Go inference framework

- **Hugot**（Go；adjacent）：Go 中的 ONNX transformer pipeline 项目；可作为推理控制面/应用层，不应视为底层 XPU allocator。 [主页](https://github.com/knights-analytics/hugot)

### Go XPU application

- **MuMax3**（Go / CUDA；adjacent）：Go 编写的 GPU 加速微磁模拟软件，展示 Go 应用与 GPU 后端结合的成熟案例；目标是科学计算，不是通用内存池。 [主页](https://github.com/mumax/3)

### Go device telemetry

- **NVIDIA go-nvml**（Go bindings / NVML；support）：Go 绑定 NVIDIA 管理库，可用于设备发现和遥测/控制面；不是显存 allocator。 [主页](https://github.com/NVIDIA/go-nvml)

### Go device orchestration

- **NVIDIA Kubernetes device plugin**（Go；support）：Kubernetes 中暴露和分配 GPU 设备资源；粒度主要是设备及资源声明，不负责在应用间池化显存。 [主页](https://github.com/NVIDIA/k8s-device-plugin)
- **AMD ROCm Kubernetes device plugin**（Go；support）：Kubernetes 对 ROCm GPU 的设备资源发现与暴露；不等于 GPU 内存或 CXL 内存池。 [主页](https://github.com/ROCm/k8s-device-plugin)
- **Intel device plugins for Kubernetes**（Go；support）：Kubernetes 上 Intel 加速器等设备插件集合，可支持 XPU 资源发现和调度集成；内存策略仍需另外实现。 [主页](https://github.com/intel/intel-device-plugins-for-kubernetes)

## 研究组与团队

以下按公开论文作者单位、项目主页和组织主页归纳；这是研究线索而非对当前组织结构的保证。Go 项目/社区另列在工具部分，不把项目误称为学术实验室。

| 团队 | 机构 | 关注方向与证据 | 关联 |
|---|---|---|---|
| [UC San Diego — Storage Systems / SysNet 相关团队](https://cseweb.ucsd.edu/~yiying/) | University of California San Diego | 分解式/远端内存系统、存储系统与系统软件协同。 证据：Clio: A Hardware-Software Co-Designed Disaggregated Memory System (ASPLOS 2022) 作者单位包含 UC San Diego。 以论文作者及公开个人/团队页面为线索；具体课题组名称和成员会随时间调整。 | direct / L08 |
| [University of Michigan — Systems Lab / Mosharaf Chowdhury 团队](https://systems.engin.umich.edu/) | University of Michigan | 云与集群系统、内存分层、CXL tiering 和数据中心资源管理。 证据：TPP: Transparent Page Placement for CXL-Enabled Tiered-Memory (ASPLOS 2023) 作者包含 Michigan 团队成员。 官方页面可能对自动化访问有限；论文 DOI 是可验证的研究线索。 | direct / L07 |
| [KAIST — 内存系统与架构研究团队](https://cs.kaist.ac.kr/) | Korea Advanced Institute of Science and Technology | CXL 内存池、内存系统和存储/内存架构。 证据：Memory Pooling With CXL (IEEE Micro 2023) 作者单位记录包含 KAIST。 这里按论文单位归纳研究团队方向，不将单篇论文误称为固定的实验室名称。 | direct / L02 |
| [上海交通大学 / 上海期智研究院 GMLake 作者团队](https://doi.org/10.1145/3620665.3640423) | Shanghai Jiao Tong University / Shanghai Qi Zhi Institute | GPU 内存管理、虚拟内存拼接和显存碎片管理。证据：GMLake (ASPLOS 2024) 作者来自上海交通大学及上海期智研究院。按论文作者单位归纳，未将作者团队直接归属到某个特定实验室。 | direct / L16 |
| [ETH Zürich — SAFARI Research Group](https://safari.ethz.ch/) | ETH Zürich | 内存系统、内存层次、数据移动和高性能计算架构；与池化/分层内存问题邻近。 证据：公开研究方向覆盖 memory systems；本次清单未将其归为某一 CXL pooling 核心论文的唯一来源。 作为内存系统方向的相关研究组推荐，读者应从当前主页确认在研项目。 | adjacent / — |
| [Inria Bordeaux / StarPU 项目团队](https://starpu.gitlabpages.inria.fr/) | Inria and University of Bordeaux | 异构 CPU/GPU 运行时、任务调度和数据管理。 证据：StarPU 论文及官方项目主页；与设备间数据生命周期管理直接相关。 更偏异构运行时与数据移动，不是 CXL 内存池研究组。 | adjacent / L21 |
| [Lawrence Livermore National Laboratory — Umpire 团队](https://computing.llnl.gov/projects/umpire) | Lawrence Livermore National Laboratory | 面向 HPC 的 NUMA/GPU 内存资源和分配器管理。 证据：Umpire 官方项目说明称其用于 NUMA 与 GPU 架构的内存管理。 以开源软件团队和应用研发团队为主，适合工具协作与原型参考。 | direct / — |
| [Microsoft / Azure 云系统研究与工程团队](https://www.microsoft.com/en-us/research/publication/pond-cxl-based-memory-pooling-systems-for-cloud-platforms/) | Microsoft Research and Azure | 云平台、资源池化和 CXL-based memory pooling。 证据：Pond: CXL-Based Memory Pooling Systems for Cloud Platforms (ASPLOS 2023) 包含 Microsoft 与云平台作者。 按公开论文作者单位归纳，不代表当前组织编制或团队名称。 | direct / L01 |
| [Ghent University — DyNaMat / MuMax3 团队](https://www.ugent.be/we/solidstatesciences/dynamat/en) | Ghent University | Go 与 CUDA 结合的 GPU 科学计算软件，涉及设备侧数据生命周期和 CPU-GPU 协同。证据：MuMax3 项目主页明确写明由 Ghent University 的 DyNaMat 团队开发维护；项目源码构建依赖 Go 与 CUDA。是 Go+GPU 应用研究组，不是 CXL 池化内存或通用显存分配器研究组。 | adjacent / — |
| [University of Waterloo — GoA 研究线索](https://hdl.handle.net/10012/14430) | University of Waterloo | Go actor 的局部内存管理与对象所有权。证据：2019 年学位论文 GoA: Actors with Locally Managed Memory for Go 收录于 UWSpace。仅能确认论文及机构，不能据此认定存在仍在运行的同名专门课题组。 | adjacent / L28 |

### Go 生态判断

本次定向检索未找到成熟的 Go 专用 CXL memory-pooling + CPU/XPU 统一管理研究组或通用底层 allocator。Go 的优势目前更像控制面、服务编排和应用接入；底层 pinned/device memory 分配通常由 Linux、CUDA/HIP/SYCL、C/C++ runtime 负责。

相关入口：[Gorgonia](https://github.com/gorgonia/gorgonia)、[gorgonia/cu](https://github.com/gorgonia/cu)、[go-opencl](https://github.com/jgillich/go-opencl)、[onnxruntime_go](https://github.com/yalue/onnxruntime_go)、[MuMax3](https://github.com/mumax/3)、[NVIDIA go-nvml](https://github.com/NVIDIA/go-nvml)、[NVIDIA Kubernetes device plugin](https://github.com/NVIDIA/k8s-device-plugin)、[AMD ROCm Kubernetes device plugin](https://github.com/ROCm/k8s-device-plugin)

- Go sync.Pool 管理的是可丢弃的临时 Go 对象；它不保证资源保留，也没有设备分配、页锁定或 CXL NUMA placement 语义。
- Go runtime/debug.SetMemoryLimit 主要调节 Go runtime 的软内存目标；cgo 分配、GPU 显存和设备内存池需单独记账。
- runtime.Pinner 用于固定 Go 对象以满足特定 cgo 指针生命周期要求，不等价于操作系统 mlock，也不等价于 CUDA/HIP pinned host allocation。
- 可行原型常采用 Go 控制面 + 原生 C/C++ 数据面：Go 管理配额、队列、设备遥测和策略；原生 runtime 执行 GPU/CXL 分配与迁移。需要明确所有权、异步完成前的生命周期和释放路径。

## 可落地的研究切入点

- 测量 CPU DRAM、CXL 内存、GPU HBM/显存及 GPU 可访问 host memory 的延迟/带宽差异；区分本地访问、DMA、页迁移和远端访问。
- 设计跨层配额与 placement 策略：在 CPU 热页、GPU 工作集、CXL 容量之间结合访问热度、QoS、容量压力和迁移开销。
- 评估 fragmentation、异步分配、跨进程隔离、公平性和尾延迟；不要只报告平均吞吐或池容量利用率。
- 在 Go 原型中先把设备内存/池资源建模为显式句柄和确定性 Close/Release 生命周期，再接入 cgo 原生 allocator；避免把 GPU 资源放进可被 GC 丢弃的 sync.Pool。
- 做端到端对照：原生 C++ runtime、Go 控制面+cgo 数据面、纯 host object pool；覆盖分配吞吐、CPU 开销、XPU 利用率、迁移字节数、故障/回收和多租户 QoS。

## 来源和方法

论文标题、作者、年份与 DOI 通过学术索引元数据交叉核对；工具说明优先依据官方项目和文档页面。项目维护状态、硬件支持矩阵和文档 URL 可能更新，复现实验前请检查所用版本。

- [OpenAlex works index](https://openalex.org/)：发现及交叉核对论文标题、作者、年份、DOI、出版物。
- [Crossref DOI metadata](https://www.crossref.org/)：DOI 元数据交叉核验入口。
- [Linux kernel CXL documentation](https://docs.kernel.org/driver-api/cxl/)：核对 CXL/Linux 内核工具链与设备模型。
- [Go package and runtime documentation](https://pkg.go.dev/)：核对 Go 标准库与 Go 包用途。
- [GitHub projects and official manuals](https://github.com/)：核对项目主页、代码语言和功能定位；活跃度会变化。

## 机器可读数据

同目录的 `memory-pooling-xpu-survey.json` 保存与本文一致的结构化条目。
