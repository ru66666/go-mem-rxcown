// Copyright (c) Huawei Technologies Co., Ltd. 2012-2019. All rights reserved.
// package pool: mpool 统计插桩实现
//
// mpool-old.go 中只插入 s4Xxx 函数调用（越少侵入越好），全部实现集中在本文件：
//   - 写路径：span / arena / arenaOB 的计数更新与重置
//   - 读路径：SpanStats / ArenaOBStats 统计接口及采集函数（与插桩版 mpool.go 效果一致）

package pool

import (
	"sync/atomic"
	"unsafe" // 10.23: 需要 &s.mem[0] 地址；//go:linkname 绑定 runtime.EagerFreeObj 亦依赖
)

/* ============================ 通用辅助 ============================ */

// s4MaxUint64 返回较大的 uint64 值（统计辅助函数）
func s4MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

/* ============================ freeSpan 复用链（方案B，10.21新增；方案A 已淘汰移除） ============================ */

// 方案B：freeSpan 复用链（span 缓存）
//
//	背景：整 span 回收时 clear() 把 mem 大切片丢给 GC（free 潮时一次上万块），
//	      随后的 malloc 潮又重新 make 同样大小的切片，GC 双重负担。
//	方案：整 span 回收时不丢 mem，入 arena 级复用链；malloc 新建分支优先从链里弹
//	      一个现成 span（免 make、免逐槽位 magic 初始化）。
//	容量：按字节预算限制（s4SpanCacheBytes），超预算的 span 退回 clear() 丢给 GC，
//	      保证缓存不放大内存占用（缓存水位 ≈ 最近一次 free 潮峰值，而非累计）。
//
//	日志实测（1020.log）：稳态 free 潮每快照增量 mean≈1067、max=8701 个 span，
//	加权平均 span 大小 17.1KB；潮集中在少数 arena，128KB/arena 预算可容纳
//	32B 档 ~186 个或 4KB 档 ~1 个，足够覆盖单 arena 潮峰，全局水位远小于收益。
const s4SpanCacheBytes = 128 * 1024

// s4SpanCacheOn 复用链开关（true=启用缓存）
var s4SpanCacheOn = true

// s4SpanCacheStats 复用链统计：hit=malloc 命中复用链, evict=入链时因预算被丢弃, entry=当前在链个数
var s4SpanCacheStats struct {
	hit   uint64
	evict uint64
	entry uint64
}

// s4SpanRecycle 整 span 回收入复用链：保留 mem/objects 底层切片，仅重置游标。
// 预算已满时返回 false（调用方走原 clear() 路径丢给 GC）。
func s4SpanRecycle(list *[]span, s *span, spanBytes int) bool {
	if !s4SpanCacheOn {
		return false
	}
	// 字节预算：链内已有 span 总字节 + 本 span 不超预算
	if (len(*list)+1)*spanBytes > s4SpanCacheBytes {
		atomic.AddUint64(&s4SpanCacheStats.evict, 1)
		return false
	}
	s.objectNum = 0
	s4SpanInit(s)             // 10.21新增：入链前重置代际计数，复用后的 span 统计从零开始（否则混住度翻新率会跨代累计失真）
	s.objects = s.objects[:0] // 游标归零，底层数组保留
	*list = append(*list, *s) // 结构体值拷贝入链，mem 指针随行
	atomic.AddUint64(&s4SpanCacheStats.entry, 1)
	s.clear() // 原位置清引用，避免与链中副本双重持有
	return true
}

// s4SpanReuse 从复用链弹出一个现成 span 到 s（mem 直接可用，免 make）。
// 返回 true 表示命中；false 表示链空，调用方走 initSpan 正常新建。
// 复用后重新铺满槽位空闲序（O(spanSize) 的 uint8 赋值，远小于 make+magic 初始化）。
func s4SpanReuse(list *[]span, s *span) bool {
	n := len(*list)
	if n == 0 || !s4SpanCacheOn {
		return false
	}
	cached := (*list)[n-1]
	*list = (*list)[:n-1]
	atomic.AddUint64(&s4SpanCacheStats.entry, ^uint64(0))

	// 接管缓存的 mem，恢复槽位空闲序（与 initSpan 一致）
	s.mem = cached.mem
	s.objects = make([]uint8, s.spanSize, s.spanSize)
	for i := uint8(0); i < s.spanSize; i++ {
		s.objects[i] = i
	}
	s.objectNum = 0
	atomic.AddUint64(&s4SpanCacheStats.hit, 1)
	return true
}

// s4SpanCacheSnapshot 返回复用链统计快照（gc-snapshot 聚合段用）
func s4SpanCacheSnapshot() (hit, evict, entry uint64) {
	return atomic.LoadUint64(&s4SpanCacheStats.hit),
		atomic.LoadUint64(&s4SpanCacheStats.evict),
		atomic.LoadUint64(&s4SpanCacheStats.entry)
}

// s4SpanCacheDrop 复用链被整体清空（arena release）时校正 entry 计数
func s4SpanCacheDrop(n int) {
	for i := 0; i < n; i++ {
		atomic.AddUint64(&s4SpanCacheStats.entry, ^uint64(0))
	}
}

/* ============================ 提前回收（10.23/10.27，双机制组合方案） ============================ */
//
// 工具链（go1.24.2，含 mfree 内置 + freegc 复用链补丁）导出：
//   - runtime.FreegcEx(objP, size)：小对象（<32KiB）→ freegc 显式复用链，
//     下次同 P 同 spanClass 的 malloc O(1) 弹出复用（主动排队复用，
//     对齐 freegc.md 的 free list 设计）。
//   - runtime.MfreeObj(objP, 0, 0)：大对象（>32KiB）→ 两步清扫，页立即回
//     mheap + heapLive/heapInUse 记账下调 + scavenger 唤醒（RSS 立降）。
//
// 无需任何构建参数/环境变量/GODEBUG——工具链默认行为即生效
//（GODEBUG=eagerfreegc=0 可运行时关闭，两机制共用）。
//
// 级联（mpool.go free 分支）：
//   L1 s4SpanEagerFree（按尺寸路由到上述两机制）→ 立即回收（理想路径）
//   L2 失败 → freeSpans 复用链（128KB/arena 预算）→ 震荡场景兜底
//   L3 链满 → clear() 丢 GC（原路径）
//
// 安全性：两机制所有失败路径保证 span 状态未被修改，返回非 Success 时
// 本层照旧走 L2/L3，无额外风险；成功路径下 mem 所有权移交 runtime，
// 此后不得再触碰（本层置 nil）。

// s4EagerFreeOn 提前回收开关（true=启用；一键关闭避免多余调用）
var s4EagerFreeOn = true

// s4EagerFreeStats 提前回收统计（[GC.EagerFree] 行）
var s4EagerFreeStats struct {
	ok              uint64 // 成功次数（mem 已归还 runtime）
	bytes           uint64 // 成功归还字节
	freegc          uint64 // freegc 显式复用链成功次数
	mfreeSmall      uint64 // mfree 小对象位图翻转成功次数
	mfreeLarge      uint64 // mfree 大对象两步清扫成功次数
	freegcBytes     uint64
	mfreeSmallBytes uint64
	mfreeLargeBytes uint64
	failGC          uint64 // MfreeGcRunning / MfreeFailedToLockLarge（本次跳过，退 L2/L3）
	failSkip        uint64 // 其他失败码（不在 mcache/specials/GODEBUG 关等）
}

// s4EagerFreeObj 提前回收统一入口（10.23/10.27）。
// runtime 侧由 export_eargc.go 的 EagerFreeObj 按尺寸路由到 freegc 链（小）
// 或 MfreeObj 两步清扫（大）；签名 uintptr,uintptr → uintptr，linkname 绑定。
//
//go:linkname s4EagerFreeObj runtime.EagerFreeObj
func s4EagerFreeObj(objP uintptr, size uintptr) uintptr

// s4EagerFreeResultCode 导出层结果码（与 runtime/export_eargc.go 常量对齐）
const (
	s4EagerFreeSuccess     = 0 // 成功：内存已归还
	s4EagerFreeGcRunning   = 1 // GC 运行中
	s4EagerFreeNotInMcache = 2 // 小对象不在当前 P 的 mcache（freegc/mfree 位图路径共用此码）
	s4EagerFreeFailedLock  = 3 // 大对象 sweep 所有权竞争
	s4EagerFreeSkipped     = 6 // 通用失败（GODEBUG 关等）
	s4EagerFreeFreegc      = 7
	s4EagerFreeMfreeSmall  = 8
	s4EagerFreeMfreeLarge  = 9
)

// s4SpanEagerFree 对整 span 的 mem 执行提前回收。
// 返回 true 表示成功（mem 所有权已移交 runtime，调用方不得再 clear/触碰 mem）；
// 返回 false 表示未生效（调用方走 L2 复用链 / L3 clear）。
func s4SpanEagerFree(s *span, spanBytes int) bool {
	if !s4EagerFreeOn || s.mem == nil {
		return false
	}
	code := s4EagerFreeObj(uintptr(unsafe.Pointer(&s.mem[0])), uintptr(spanBytes))
	switch code {
	case s4EagerFreeSuccess, s4EagerFreeFreegc, s4EagerFreeMfreeSmall, s4EagerFreeMfreeLarge:
		atomic.AddUint64(&s4EagerFreeStats.ok, 1)
		atomic.AddUint64(&s4EagerFreeStats.bytes, uint64(spanBytes))
		switch code {
		case s4EagerFreeFreegc:
			atomic.AddUint64(&s4EagerFreeStats.freegc, 1)
			atomic.AddUint64(&s4EagerFreeStats.freegcBytes, uint64(spanBytes))
		case s4EagerFreeMfreeSmall:
			atomic.AddUint64(&s4EagerFreeStats.mfreeSmall, 1)
			atomic.AddUint64(&s4EagerFreeStats.mfreeSmallBytes, uint64(spanBytes))
		case s4EagerFreeMfreeLarge:
			atomic.AddUint64(&s4EagerFreeStats.mfreeLarge, 1)
			atomic.AddUint64(&s4EagerFreeStats.mfreeLargeBytes, uint64(spanBytes))
		}
		// 所有权已移交 runtime：置 nil，防止后续 clear/复用链双重持有
		s.mem = nil
		s.objects = nil
		return true
	case s4EagerFreeGcRunning, s4EagerFreeFailedLock, s4EagerFreeNotInMcache:
		atomic.AddUint64(&s4EagerFreeStats.failGC, 1)
	default: // s4EagerFreeSkipped 等
		atomic.AddUint64(&s4EagerFreeStats.failSkip, 1)
	}
	return false
}

// s4EagerFreeSnapshot 返回提前回收统计快照（gc-snapshot 聚合段用）
func s4EagerFreeSnapshot() (ok, bytes, freegc, freegcBytes, mfreeSmall, mfreeSmallBytes, mfreeLarge, mfreeLargeBytes, failGC, failSkip uint64) {
	return atomic.LoadUint64(&s4EagerFreeStats.ok),
		atomic.LoadUint64(&s4EagerFreeStats.bytes),
		atomic.LoadUint64(&s4EagerFreeStats.freegc),
		atomic.LoadUint64(&s4EagerFreeStats.freegcBytes),
		atomic.LoadUint64(&s4EagerFreeStats.mfreeSmall),
		atomic.LoadUint64(&s4EagerFreeStats.mfreeSmallBytes),
		atomic.LoadUint64(&s4EagerFreeStats.mfreeLarge),
		atomic.LoadUint64(&s4EagerFreeStats.mfreeLargeBytes),
		atomic.LoadUint64(&s4EagerFreeStats.failGC),
		atomic.LoadUint64(&s4EagerFreeStats.failSkip)
}

/* ============================ span 统计（写路径） ============================ */

// s4SpanInit 重置 span 统计计数（initSpan 时调用）
func s4SpanInit(s *span) {
	s.objAllocNum = 0
	s.objFreeNum = 0
}

// s4SpanMalloc span 每分配一个对象，累计分配数 +1（malloc 时调用）
// 10.23 note: 原混住度 refill 检测已随 [GC.Mix] 统计一并移除，仅保留纯计数
func s4SpanMalloc(s *span) {
	atomic.AddUint64(&s.objAllocNum, 1)
}

// s4SpanFree span 每释放一个对象，累计释放数 +1（free 时调用）
func s4SpanFree(s *span) {
	atomic.AddUint64(&s.objFreeNum, 1)
}

// s4SpanClear 重置 span 统计计数（clear 时调用）
func s4SpanClear(s *span) {
	s.objAllocNum = 0
	s.objFreeNum = 0
}

/* ============================ arenaOB 统计（写路径） ============================ */

// s4ArenaOBMalloc arenaOB 分配对象时更新累计分配数与峰值容量（malloc 时调用）
func s4ArenaOBMalloc(o *arenaOB) {
	atomic.AddUint64(&o.objAllocNum, 1)
	atomic.StoreUint64(&o.objectsCap, s4MaxUint64(o.objectsCap, uint64(cap(o.objects))))
	atomic.StoreUint64(&o.idxPoolCap, s4MaxUint64(o.idxPoolCap, uint64(cap(o.idxPool))))
}

// s4ArenaOBFree arenaOB 释放对象时更新累计释放数、释放字节与峰值容量（free 时调用，size 为块总大小）
func s4ArenaOBFree(o *arenaOB, size int64) {
	atomic.AddUint64(&o.objFreeNum, 1)
	atomic.AddUint64(&o.objFreedBytes, uint64(size))
	atomic.StoreUint64(&o.idxPoolCap, s4MaxUint64(o.idxPoolCap, uint64(cap(o.idxPool))))
}

// s4ArenaOBRelease 重置 arenaOB 统计计数（release 时调用）
func s4ArenaOBRelease(o *arenaOB) {
	atomic.StoreUint64(&o.objAllocNum, 0)
	atomic.StoreUint64(&o.objFreeNum, 0)
	atomic.StoreUint64(&o.objFreedBytes, 0)
	atomic.StoreUint64(&o.objectsCap, 0)
	atomic.StoreUint64(&o.idxPoolCap, 0)
}

/* ============================ arena 统计（写路径） ============================ */

// s4ArenaSpanAlloc arena 新建 span 时累计分配数 +1（malloc 新建 span 分支调用）
func s4ArenaSpanAlloc(a *arena) {
	atomic.AddUint64(&a.spanAllocNum, 1)
}

// s4ArenaSpanFree arena 整块释放 span 时更新累计计数（free 释放 span 分支调用）
func s4ArenaSpanFree(a *arena) {
	atomic.AddUint64(&a.spanFreedNum, 1)
	atomic.AddUint64(&a.spanFreedBytes, uint64(a.objectSize+reservedLen)*uint64(a.spanSize))
	atomic.AddUint64(&a.spanFreedObjs, uint64(a.spanSize))
}

// s4ArenaObjRetain 对象已释放但 span 未空时滞留计数 +1（free 非空分支调用）
func s4ArenaObjRetain(a *arena) {
	atomic.AddUint64(&a.objRetainNum, 1)
}

// s4ArenaRelease 重置 arena 统计计数（release 时调用）
func s4ArenaRelease(a *arena) {
	atomic.StoreUint64(&a.spanAllocNum, 0)
	atomic.StoreUint64(&a.spanFreedNum, 0)
	atomic.StoreUint64(&a.spanFreedBytes, 0)
	atomic.StoreUint64(&a.spanFreedObjs, 0)
	atomic.StoreUint64(&a.objRetainNum, 0)
}

/* ============================ 统计读取接口 ============================ */

// span stats interface
type SpanStats interface {
	Idx() uint32
	ObjectNum() uint8
	SpanSize() uint8
	ObjAllocNum() uint64
	ObjFreeNum() uint64
}

// span stats struct
type spanStats struct {
	idx         uint32
	objectNum   uint8
	spanSize    uint8
	objAllocNum uint64
	objFreeNum  uint64
}

// Idx Get span index of span stats
func (s *spanStats) Idx() uint32 {
	return s.idx
}

// ObjectNum Get object num of span stats
func (s *spanStats) ObjectNum() uint8 {
	return s.objectNum
}

// SpanSize Get span size of span stats
func (s *spanStats) SpanSize() uint8 {
	return s.spanSize
}

// ObjAllocNum Get accumulated malloc object num of span stats
func (s *spanStats) ObjAllocNum() uint64 {
	return s.objAllocNum
}

// ObjFreeNum Get accumulated free object num of span stats
func (s *spanStats) ObjFreeNum() uint64 {
	return s.objFreeNum
}

// arenaOB stats interface
type ArenaOBStats interface {
	PoolName() string
	InUsed() int64
	ObjAllocNum() uint64
	ObjFreeNum() uint64
	ObjFreedBytes() uint64
	ObjectsCap() uint64
	IdxPoolCap() uint64
}

// arenaOB stats struct
type arenaOBStats struct {
	poolName      string
	inUsed        int64
	objAllocNum   uint64
	objFreeNum    uint64
	objFreedBytes uint64
	objectsCap    uint64
	idxPoolCap    uint64
}

// PoolName Get pool name of arenaOB stats
func (o *arenaOBStats) PoolName() string {
	return o.poolName
}

// InUsed Get inuse object num of arenaOB stats
func (o *arenaOBStats) InUsed() int64 {
	return o.inUsed
}

// ObjAllocNum Get accumulated malloc object num of arenaOB stats
func (o *arenaOBStats) ObjAllocNum() uint64 {
	return o.objAllocNum
}

// ObjFreeNum Get accumulated free object num of arenaOB stats
func (o *arenaOBStats) ObjFreeNum() uint64 {
	return o.objFreeNum
}

// ObjFreedBytes Get accumulated freed bytes of arenaOB stats
func (o *arenaOBStats) ObjFreedBytes() uint64 {
	return o.objFreedBytes
}

// ObjectsCap Get objects array peak capacity of arenaOB stats
func (o *arenaOBStats) ObjectsCap() uint64 {
	return o.objectsCap
}

// IdxPoolCap Get idxPool array peak capacity of arenaOB stats
func (o *arenaOBStats) IdxPoolCap() uint64 {
	return o.idxPoolCap
}

// SpanAllocNum Get accumulated allocated span num of arena stats
func (a *arenaStats) SpanAllocNum() uint64 {
	return a.spanAllocNum
}

// SpanFreedNum Get accumulated freed span num of arena stats
func (a *arenaStats) SpanFreedNum() uint64 {
	return a.spanFreedNum
}

// SpanFreedBytes Get accumulated freed span bytes of arena stats
func (a *arenaStats) SpanFreedBytes() uint64 {
	return a.spanFreedBytes
}

// SpanFreedObjs Get accumulated freed span object num of arena stats
func (a *arenaStats) SpanFreedObjs() uint64 {
	return a.spanFreedObjs
}

// ObjRetainNum Get accumulated freed-but-retained object num of arena stats
func (a *arenaStats) ObjRetainNum() uint64 {
	return a.objRetainNum
}

// ActiveSpans Get active span stats list of arena stats
func (a *arenaStats) ActiveSpans() []SpanStats {
	return a.activeSpans
}

/* ============================ 统计读取（memPool 调用） ============================ */

// s4ActiveSpans 按链表顺序收集 arena 内活跃 span 的统计快照
func s4ActiveSpans(a *arena) []SpanStats {
	if atomic.LoadUint32(&a.spanNum) == 0 {
		return nil
	}

	spans := make([]SpanStats, 0, a.spanNum)
	for s := a.head(); s.idx != sentinel; s = s.Next(a) {
		spans = append(spans, &spanStats{
			idx:         s.idx,
			objectNum:   s.objectNum,
			spanSize:    s.spanSize,
			objAllocNum: atomic.LoadUint64(&s.objAllocNum),
			objFreeNum:  atomic.LoadUint64(&s.objFreeNum),
		})
	}

	return spans
}

// s4NewArenaStats 在原 newArenaStats 基础上补充插桩统计字段，构造完整 ArenaStats
func s4NewArenaStats(poolName string, objectSize, spanNum uint32, spanSize uint8, inUseNum int64,
	spanAllocNum, spanFreedNum, spanFreedBytes, spanFreedObjs, objRetainNum uint64, activeSpans []SpanStats) ArenaStats {
	state := newArenaStats(poolName, objectSize, spanNum, spanSize, inUseNum)
	if s, ok := state.(*arenaStats); ok {
		s.spanAllocNum = spanAllocNum
		s.spanFreedNum = spanFreedNum
		s.spanFreedBytes = spanFreedBytes
		s.spanFreedObjs = spanFreedObjs
		s.objRetainNum = objRetainNum
		s.activeSpans = activeSpans
	}

	return state
}

// s4NewArenaOBStats 构造 ArenaOBStats 实例
func s4NewArenaOBStats(poolName string, inUsed int64,
	objAllocNum, objFreeNum, objFreedBytes, objectsCap, idxPoolCap uint64) ArenaOBStats {
	return &arenaOBStats{
		poolName:      poolName,
		inUsed:        inUsed,
		objAllocNum:   objAllocNum,
		objFreeNum:    objFreeNum,
		objFreedBytes: objFreedBytes,
		objectsCap:    objectsCap,
		idxPoolCap:    idxPoolCap,
	}
}

// s4PoolArenaStats 采集 memPool 全部 arena 的插桩统计（memPool.ArenaStats 调用）
func s4PoolArenaStats(m *memPool) []ArenaStats {
	var states []ArenaStats
	for i := range m.arenas {
		a := &m.arenas[i]
		states = append(states, s4NewArenaStats(m.name, a.objectSize, atomic.LoadUint32(&a.spanNum), a.spanSize, a.InUsed(),
			atomic.LoadUint64(&a.spanAllocNum), atomic.LoadUint64(&a.spanFreedNum),
			atomic.LoadUint64(&a.spanFreedBytes), atomic.LoadUint64(&a.spanFreedObjs),
			atomic.LoadUint64(&a.objRetainNum), s4ActiveSpans(a)))
	}

	return states
}

// s4PoolOBStats 采集 memPool arenaOB 的插桩统计（memPool.OBStats 调用）
func s4PoolOBStats(m *memPool) ArenaOBStats {
	o := &m.arenaOB
	return s4NewArenaOBStats(m.name, o.InUsed(),
		atomic.LoadUint64(&o.objAllocNum), atomic.LoadUint64(&o.objFreeNum),
		atomic.LoadUint64(&o.objFreedBytes), atomic.LoadUint64(&o.objectsCap),
		atomic.LoadUint64(&o.idxPoolCap))
}
