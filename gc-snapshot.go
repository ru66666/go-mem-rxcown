// Copyright (c) Huawei Technologies Co., Ltd. 2012-2019. All rights reserved.
// package pool: mempool 快照统计（统一结构 + 模式开关版）
//
// 完整版与聚合版共用一次采样（sampleSnapshot），只打印当前值/累计值，无 diff：
// 用户可自行对比前后两次日志（malloc/free 都是单调累计，直接相减即可）。
//
//   - 完整版（DumpModeFull）：逐 pool 分级明细（Pool -> Arena -> Span）+ ArenaOB + 聚合段
//   - 聚合版（DumpModeAgg）：仅聚合段，输出量固定，适合上万 pool
//
// 聚合段（两种模式都打印）：
//   [GC.Heap]      Go 堆当前值（HeapAlloc / HeapInuse / HeapObjects / NumGC）
//   [GC.Mem]       系统级内存当前值（Sys / RSS）
//   [GC.Total]     pool 总量当前值 + mallocSpan/freeSpan 累计
//   [GC.Span]      每个 objectSize 一行（mallocSpan 或 freeSpan 不为 0 才打印）
//   [GC.Pool.Top]  释放量最大的前 3 个 pool 的 name
//
// 采样语义：
//   before = 上一次采样（tick 或上一次 GC 回调）
//   after  = 本次采样（reason=gc 时即 GC 之后）
//   值全部为当前值/累计值，与前后差值由用户对比日志得出。

package pool

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

/* ============================ 快照数据结构 ============================ */

// DumpMode 打印模式开关：完整版（明细+聚合）或聚合版
type DumpMode int

const (
	DumpModeFull DumpMode = iota // 完整版：逐 pool 分级明细 + 聚合段
	DumpModeAgg                  // 聚合版：仅聚合段
)

// modeName 返回模式名（打印在头部）
func modeName(m DumpMode) string {
	if m == DumpModeAgg {
		return "aggregate"
	}
	return "full"
}

// poolStatic Pool 层不变项
type poolStatic struct {
	name string
	kind string
}

// poolDynamic Pool 层会变项
type poolDynamic struct {
	allocated uint64
	freed     uint64
	inUsed    int64
	usage     int
}

// arenaStatic Arena 层不变项（配置）
type arenaStatic struct {
	objectSize uint32 // 每对象字节数
	spanSize   uint8  // 每 span 槽位数
}

// arenaDynamic Arena 层会变项
type arenaDynamic struct {
	spanNum      uint32 // 当前活跃 span 数
	spanAllocNum uint64 // 累计分配 span 次数
	spanFreedNum uint64 // 累计整块释放 span 次数
	objRetainNum uint64 // 累计"对象已释放但 span 未空"的滞留次数
}

// spanDynamic Span 层会变项（仅统计当前活跃 span）
type spanDynamic struct {
	inUsedObjNum uint64 // 在用对象总数
	objAllocNum  uint64 // 活跃 span 累计分配对象数
	objFreeNum   uint64 // 活跃 span 累计释放对象数
}

// arenaSnap 单个 arena 的快照
type arenaSnap struct {
	static  arenaStatic
	dynamic arenaDynamic
	span    spanDynamic
}

// arenaOBDynamic ArenaOB 层会变项（无配置不变项）
type arenaOBDynamic struct {
	inUsed        int64  // 当前在用字节数
	objAllocNum   uint64 // 累计分配对象数
	objFreeNum    uint64 // 累计释放对象数
	objFreedBytes uint64 // 累计释放字节数
	objectsCap    uint64 // objects 数组峰值容量
	idxPoolCap    uint64 // idxPool 数组峰值容量
}

// poolSnap 整个 pool 的一次快照（完整明细 + 聚合量）
type poolSnap struct {
	isMPool bool
	static  poolStatic
	dynamic poolDynamic
	arenas  []arenaSnap
	arenaOB arenaOBDynamic
	agg     poolAgg // 聚合量（采集后由 aggregatePool 填充）
}

// heapSnap GC 直接作用的部分（Go 堆）+ 系统级内存，用于对照 pool 计数器
type heapSnap struct {
	numGC       uint32
	heapAlloc   uint64
	heapInuse   uint64
	heapObjects uint64
	sys         uint64 // Go 向 OS 申请的虚拟内存总量（MemStats.Sys）
	rss         uint64 // 进程常驻内存（/proc/self/statm，非 linux 为 0）
}

// spanAggItem 按 objectSize 聚合的 span 分配/释放量（聚合段 [GC.Span] 用）
type spanAggItem struct {
	objectSize   uint32
	spanAllocNum uint64
	spanFreedNum uint64
}

// sampleSnapshot 统一采样结构：完整版与聚合版共用一次采集
type sampleSnapshot struct {
	ts        time.Time
	mode      DumpMode
	heap      heapSnap
	poolCount int
	pools     map[string]*poolSnap    // 完整明细（每 pool 含聚合量 agg）
	total     poolAgg                 // 全部 pool 聚合
	spanAgg   map[uint32]*spanAggItem // 按 objectSize 聚合的 span 分配/释放
}

var (
	snapMu     sync.Mutex
	lastSample *sampleSnapshot // 上一次采样（完整版/聚合版共用同一条差值链）
)

/* ============================ 采集 ============================ */

func readHeapSnap() heapSnap {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return heapSnap{
		numGC:       ms.NumGC,
		heapAlloc:   ms.HeapAlloc,
		heapInuse:   ms.HeapInuse,
		heapObjects: ms.HeapObjects,
		sys:         ms.Sys,
		rss:         readRSS(),
	}
}

// readRSS 读取进程常驻内存（字节）；仅支持 linux /proc，失败返回 0
func readRSS() uint64 {
	b, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return pages * uint64(os.Getpagesize())
}

// spanSummary 活跃 span 的汇总（只保留不可推导的原始值）
type spanSummary struct {
	inUsedObjNum uint64
	objAllocNum  uint64
	objFreeNum   uint64
}

func summarizeSpans(spans []SpanStats) spanSummary {
	var sum spanSummary
	for _, sp := range spans {
		sum.inUsedObjNum += uint64(sp.ObjectNum())
		sum.objAllocNum += sp.ObjAllocNum()
		sum.objFreeNum += sp.ObjFreeNum()
	}
	return sum
}

// capturePool 采集一个 pool 在当前时刻的完整分层快照
func capturePool(p Pool, now time.Time) *poolSnap {
	s := &poolSnap{
		static: poolStatic{name: p.Name(), kind: poolKind(p)},
	}

	if stat, ok := p.(Statistic); ok {
		s.dynamic.allocated = uint64(stat.Allocated())
		s.dynamic.freed = uint64(stat.Freed())
		s.dynamic.inUsed = stat.InUsed()
		s.dynamic.usage = int(stat.Usage())
	}

	mp, ok := p.(MPool)
	if !ok {
		return s
	}
	s.isMPool = true

	stats := mp.ArenaStats()
	s.arenas = make([]arenaSnap, len(stats))
	for i, as := range stats {
		ss := summarizeSpans(as.ActiveSpans())
		s.arenas[i] = arenaSnap{
			static: arenaStatic{
				objectSize: as.ObjectSize(),
				spanSize:   as.SpanSize(),
			},
			dynamic: arenaDynamic{
				spanNum:      as.SpanNum(),
				spanAllocNum: as.SpanAllocNum(),
				spanFreedNum: as.SpanFreedNum(),
				objRetainNum: as.ObjRetainNum(),
			},
			span: spanDynamic{
				inUsedObjNum: ss.inUsedObjNum,
				objAllocNum:  ss.objAllocNum,
				objFreeNum:   ss.objFreeNum,
			},
		}
	}

	ob := mp.OBStats()
	s.arenaOB = arenaOBDynamic{
		inUsed:        ob.InUsed(),
		objAllocNum:   ob.ObjAllocNum(),
		objFreeNum:    ob.ObjFreeNum(),
		objFreedBytes: ob.ObjFreedBytes(),
		objectsCap:    ob.ObjectsCap(),
		idxPoolCap:    ob.IdxPoolCap(),
	}

	return s
}

// captureSample 一次全量采集：完整明细 + 聚合量（完整版/聚合版共用）
func captureSample(prefix string, now time.Time, mode DumpMode) *sampleSnapshot {
	pools := GetPools(prefix)
	s := &sampleSnapshot{
		ts:        now,
		mode:      mode,
		pools:     make(map[string]*poolSnap, len(pools)),
		spanAgg:   make(map[uint32]*spanAggItem),
		poolCount: len(pools),
	}
	for _, p := range pools {
		ps := capturePool(p, now)
		ps.agg = aggregatePool(ps)
		s.pools[ps.static.name] = ps
		s.total.add(ps.agg)
		for i := range ps.arenas {
			c := &ps.arenas[i]
			item, ok := s.spanAgg[c.static.objectSize]
			if !ok {
				item = &spanAggItem{objectSize: c.static.objectSize}
				s.spanAgg[c.static.objectSize] = item
			}
			item.spanAllocNum += c.dynamic.spanAllocNum
			item.spanFreedNum += c.dynamic.spanFreedNum
		}
	}
	s.heap = readHeapSnap()
	return s
}

/* ============================ 聚合量 ============================ */

// gcTopPools 聚合段固定打印占比最大的前 3 个 pool
const gcTopPools = 3

// poolAgg 单个 pool 的聚合量
type poolAgg struct {
	kind        string
	spanNum     uint64 // 活跃 span 数
	inUsedObj   uint64 // 在用对象数
	totalBytes  uint64 // arena span 提交字节 + arenaOB 在用字节
	inUsedBytes uint64 // 在用对象字节
	emptyBytes  uint64 // 空槽滞留字节
	obBytes     uint64 // arenaOB 在用字节

	inUsed int64 // pool Statistic.InUsed（字节）

	spanAllocNum       uint64 // 累计新建 span 数
	spanFreedNum       uint64 // 累计整块释放 span 数
	arenaReleasedBytes uint64 // 累计 arena 释放字节 = spanFreedNum*spanSize*slotBytes
	obFreedBytes       uint64 // 累计 arenaOB 释放字节
}

func (a *poolAgg) add(b poolAgg) {
	a.spanNum += b.spanNum
	a.inUsedObj += b.inUsedObj
	a.totalBytes += b.totalBytes
	a.inUsedBytes += b.inUsedBytes
	a.emptyBytes += b.emptyBytes
	a.obBytes += b.obBytes
	a.inUsed += b.inUsed
	a.spanAllocNum += b.spanAllocNum
	a.spanFreedNum += b.spanFreedNum
	a.arenaReleasedBytes += b.arenaReleasedBytes
	a.obFreedBytes += b.obFreedBytes
}

func (a *poolAgg) releasedBytes() uint64 {
	return a.arenaReleasedBytes + a.obFreedBytes
}

// aggregatePool 把一个 pool 快照压成聚合量
func aggregatePool(s *poolSnap) poolAgg {
	a := poolAgg{kind: s.static.kind, inUsed: s.dynamic.inUsed}
	if !s.isMPool {
		return a
	}
	for i := range s.arenas {
		c := &s.arenas[i]
		slotBytes := uint64(c.static.objectSize) + uint64(reservedLen)
		totalSlots := uint64(c.dynamic.spanNum) * uint64(c.static.spanSize)
		a.spanNum += uint64(c.dynamic.spanNum)
		a.inUsedObj += c.span.inUsedObjNum
		a.totalBytes += totalSlots * slotBytes
		a.inUsedBytes += c.span.inUsedObjNum * slotBytes
		if totalSlots > c.span.inUsedObjNum {
			a.emptyBytes += (totalSlots - c.span.inUsedObjNum) * slotBytes
		}
		a.spanAllocNum += c.dynamic.spanAllocNum
		a.spanFreedNum += c.dynamic.spanFreedNum
		a.arenaReleasedBytes += c.dynamic.spanFreedNum * uint64(c.static.spanSize) * slotBytes
	}
	if s.arenaOB.inUsed > 0 {
		a.obBytes = uint64(s.arenaOB.inUsed)
		a.totalBytes += a.obBytes
	}
	a.obFreedBytes = s.arenaOB.objFreedBytes
	return a
}

// absI64 返回 int64 绝对值
func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// fmtBytes 把字节数格式化为自适应单位（B/KB/MB/GB）
func fmtBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%dB", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	val := float64(v) / float64(div)
	suffix := "KMGT"[exp]
	return fmt.Sprintf("%.2f%cB", val, suffix)
}

/* ============================ 入口（模式开关） ============================ */

// DumpAllPoolsGCStats 遍历所有 pool，打印分层统计（manual 触发，完整版）
func DumpAllPoolsGCStats(prefix string) {
	dumpAllPools(prefix, "manual", DumpModeFull)
}

// DumpAllPoolsGCStatsWithReason 完整版：逐 pool 分级明细（Pool -> Arena -> Span）+ ArenaOB + 聚合段。
// reason: 采样来源（tick / gc / manual），打印在头部便于区分采样线。
func DumpAllPoolsGCStatsWithReason(prefix, reason string) {
	dumpAllPools(prefix, reason, DumpModeFull)
}

// DumpAllPoolsGCStatsAggWithReason 聚合版：仅聚合段（系统级内存、总量、
// mallocSpan/freeSpan 按 objectSize、Top3 pool 名字），输出量固定，适合上万 pool。
func DumpAllPoolsGCStatsAggWithReason(prefix, reason string) {
	dumpAllPools(prefix, reason, DumpModeAgg)
}

// RecordGCPoolStats 旧接口：每次 GC 后调用聚合版（Top3）。
// topN 已不再使用（聚合段固定 Top3），仅为兼容保留参数。
func RecordGCPoolStats(prefix string, topN int) {
	_ = topN
	DumpAllPoolsGCStatsAggWithReason(prefix, "gc")
}

// dumpAllPools 统一采样并按模式打印。
// 完整版：聚合段在开头，明细在后；分块流式写出（每 flushBlockPools 个 pool flush 一次），
// 跳过空 arena；空闲 pool 只打一行名字+标空。
func dumpAllPools(prefix, reason string, mode DumpMode) {
	snapMu.Lock()
	defer snapMu.Unlock()

	now := time.Now()
	cur := captureSample(prefix, now, mode)

	w := bufio.NewWriterSize(os.Stdout, 64*1024)
	defer w.Flush()

	fmt.Fprintf(w, "[%s] memPool stats dump (reason=%s, mode=%s), time: %s\n",
		prefix, reason, modeName(mode), now.Format(time.RFC3339))

	// 聚合段（完整版与聚合版都打印，完整版放在开头）
	printAgg(w, prefix, cur)

	if mode == DumpModeFull {
		// 逐 pool 明细（按名字排序，输出稳定），空闲 pool 只打一行
		names := make([]string, 0, len(cur.pools))
		for name := range cur.pools {
			names = append(names, name)
		}
		sort.Strings(names)

		printed, skipped := 0, 0
		for _, name := range names {
			ps := cur.pools[name]
			if poolIdle(ps) {
				skipped++
				fmt.Fprintf(w, "Pool: %s [%s] (idle)\n", ps.static.name, ps.static.kind)
				continue
			}
			printPool(w, ps)
			printed++
			// 分块 flush：内存占用 O(1)，每块之间整 pool 边界
			if printed%flushBlockPools == 0 {
				w.Flush()
			}
		}
		fmt.Fprintf(w, "=== full detail: printed %d, idle %d, total %d pools ===\n",
			printed, skipped, len(names))
	}

	w.Flush()

	lastSample = cur
}

// flushBlockPools 完整版每打印这么多 pool 明细 flush 一次，控制缓冲内存占用
const flushBlockPools = 100

// poolIdle 判断 pool 是否空闲：无任何活跃 span、arenaOB 在用、累计 span 分配
func poolIdle(s *poolSnap) bool {
	if !s.isMPool {
		return false
	}
	if s.arenaOB.inUsed != 0 {
		return false
	}
	for i := range s.arenas {
		c := &s.arenas[i]
		if c.dynamic.spanNum != 0 || c.dynamic.spanAllocNum != 0 {
			return false
		}
	}
	return true
}

/* ============================ 打印：逐 pool 明细（完整版） ============================ */

// printPool 写入一个 pool 的分级快照（当前值/累计值，无 diff）：
// Pool -> Arena（只打活跃 span 的 arena）-> Span（一行汇总）+ ArenaOB
func printPool(w *bufio.Writer, cur *poolSnap) {
	fmt.Fprintf(w, "Pool: %s [%s], Allocated: %d, Freed: %d, InUsed: %d, Usage: %d%%\n",
		cur.static.name, cur.static.kind,
		cur.dynamic.allocated, cur.dynamic.freed, cur.dynamic.inUsed, cur.dynamic.usage)

	if !cur.isMPool {
		return
	}

	for i := range cur.arenas {
		c := &cur.arenas[i]
		if c.dynamic.spanNum == 0 { // 空 arena 跳过：信息在聚合段 [GC.Span] 里
			continue
		}
		slotBytes := uint64(c.static.objectSize) + uint64(reservedLen)
		totalSlots := uint64(c.dynamic.spanNum) * uint64(c.static.spanSize)
		fmt.Fprintf(w, "  [Arena] ObjectSize: %d, SpanNum: %d, SpanAlloc: %d, SpanFreed: %d, FreedBytes: %s, FreedObjs: %d\n",
			c.static.objectSize, c.dynamic.spanNum,
			c.dynamic.spanAllocNum, c.dynamic.spanFreedNum,
			fmtBytes(c.dynamic.spanFreedNum*uint64(c.static.spanSize)*slotBytes),
			c.dynamic.spanFreedNum*uint64(c.static.spanSize))
		fmt.Fprintf(w, "    [Span] TotalSlots: %d, InUsed: %d\n", totalSlots, c.span.inUsedObjNum)
	}

	fmt.Fprintf(w, "  [ArenaOB] InUsed: %d, ObjAlloc: %d, ObjFree: %d, FreedBytes: %d, ObjectsCap: %d, IdxPoolCap: %d\n",
		cur.arenaOB.inUsed, cur.arenaOB.objAllocNum, cur.arenaOB.objFreeNum,
		cur.arenaOB.objFreedBytes, cur.arenaOB.objectsCap, cur.arenaOB.idxPoolCap)
}

/* ============================ 聚合段（两种模式共用） ============================ */

// printAgg 写入聚合段（全当前值，无 diff）：
//
//	[GC.Heap]      Go 堆当前值
//	[GC.Mem]       系统级内存当前值（Sys / RSS）
//	[GC.Total]     pool 总量当前值 + mallocSpan/freeSpan 累计
//	[GC.Span]      每个 objectSize 一行（mallocSpan 或 freeSpan 不为 0 才打印）
//	[GC.Pool.Top]  释放量最大的前 3 个 pool 的 name（按累计 releasedBytes 排序）
func printAgg(b *bufio.Writer, prefix string, cur *sampleSnapshot) {
	b.WriteString(fmt.Sprintf("[%s] ===== GC aggregate =====\n", prefix))

	h := cur.heap
	b.WriteString(fmt.Sprintf("[GC.Heap] HeapAlloc: %d, HeapInuse: %d, HeapObjects: %d, NumGC: %d\n",
		h.heapAlloc, h.heapInuse, h.heapObjects, h.numGC))
	b.WriteString(fmt.Sprintf("[GC.Mem] Sys: %d, RSS: %d\n", h.sys, h.rss))
	b.WriteString(fmt.Sprintf("[GC.Total] Pools: %d, PoolBytes: %d (inUse %d + empty %d + OB %d), InUsed: %d, mallocSpan: %d, freeSpan: %d\n",
		cur.poolCount, cur.total.totalBytes, cur.total.inUsedBytes, cur.total.emptyBytes,
		cur.total.obBytes, cur.total.inUsed, cur.total.spanAllocNum, cur.total.spanFreedNum))

	// [GC.Span]：每个 objectSize 一行，mallocSpan 或 freeSpan 不为 0 才打印，按 size 升序
	sizes := make([]uint32, 0, len(cur.spanAgg))
	for size, item := range cur.spanAgg {
		if item.spanAllocNum != 0 || item.spanFreedNum != 0 {
			sizes = append(sizes, size)
		}
	}
	sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })
	for _, size := range sizes {
		item := cur.spanAgg[size]
		b.WriteString(fmt.Sprintf("[GC.Span] objectsize=%d, mallocSpan=%d, freeSpan=%d\n",
			size, item.spanAllocNum, item.spanFreedNum))
	}

	// [s4改] [GC.Frag]：free 时刻 span 空槽率分布（碎片度量），4 桶：<25%/<50%/<75%/<100%
	frag := s4FragSnapshot()
	b.WriteString(fmt.Sprintf("[GC.Frag] empty<25%%: %d, <50%%: %d, <75%%: %d, <100%%: %d\n",
		frag[0], frag[1], frag[2], frag[3]))

	// [s4改] [GC.SpanCache]：freeSpan 复用链效果（命中免 make/免 GC 回收）
	ch, ev, en := s4SpanCacheSnapshot()
	b.WriteString(fmt.Sprintf("[GC.SpanCache] hit: %d, evict: %d, cached: %d\n", ch, ev, en))

	// [GC.Pool.Top]：按累计释放字节排前 3 的 pool 名字
	type topPool struct {
		name  string
		bytes uint64
	}
	tops := make([]topPool, 0, len(cur.pools))
	for name, ps := range cur.pools {
		tops = append(tops, topPool{name: name, bytes: ps.agg.releasedBytes()})
	}
	sort.Slice(tops, func(i, j int) bool {
		if tops[i].bytes != tops[j].bytes {
			return tops[i].bytes > tops[j].bytes
		}
		return tops[i].name < tops[j].name
	})
	topN := gcTopPools
	if topN > len(tops) {
		topN = len(tops)
	}
	topNames := make([]string, 0, topN)
	for i := 0; i < topN; i++ {
		topNames = append(topNames, tops[i].name)
	}
	b.WriteString(fmt.Sprintf("[GC.Pool.Top] %s\n", strings.Join(topNames, ", ")))

	b.WriteString("===============================\n")
}

/* ============================ 辅助 ============================ */

// poolKind 返回 pool 的类型标签，用于日志区分
func poolKind(p Pool) string {
	switch p.(type) {
	case MPool:
		return "MPool"
	}
	return fmt.Sprintf("%T", p)
}

// StartPoolStatsTicker 启动定时采样（完整版）：每 interval 调一次。
// 与 GC 回调采样共用同一条采样链。
func StartPoolStatsTicker(interval time.Duration, prefix, reason string, stopCh <-chan struct{}) {
	StartPoolStatsTickerMode(interval, prefix, reason, stopCh, DumpModeFull)
}

// StartPoolStatsTickerMode 启动定时采样，mode 决定完整版/聚合版
func StartPoolStatsTickerMode(interval time.Duration, prefix, reason string, stopCh <-chan struct{}, mode DumpMode) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dumpAllPools(prefix, reason, mode)
			case <-stopCh:
				return
			}
		}
	}()
}

// unused 占位：保留 lastSample 以便后续如需恢复差值对比（当前不打印 diff）
var _ = lastSample
