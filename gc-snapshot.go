// Copyright (c) Huawei Technologies Co., Ltd. 2012-2019. All rights reserved.
// package pool: mempool 快照 + 差值统计（精简版）

package pool

import (
	"fmt"
	"sync"
	"time"
)

// arenaSnap 单个 arena 的一次快照
// 只保留不可推导的关键统计：字节类（FreedBytes 等）可由数量 × 对象大小推导，不再收集。
type arenaSnap struct {
	objectSize    uint32
	spanNum       uint32
	spanAllocNum  uint64
	spanFreedNum  uint64
	spanFreedObjs uint64
	objRetainNum  uint64
	spanSummary   spanSummary
}

// spanSummary 当前活跃 span 的汇总（替代逐 span 明细，避免日志膨胀）
type spanSummary struct {
	spanNum      uint32 // 活跃 span 数
	inUsedObjNum uint64 // 在用对象总数
	totalSlotNum uint64 // 总槽位数 = spanNum × spanSize
	freedSpanNum uint32 // 有释放历史（ObjFree>0）的 span 数
}

// arenaOBSnap arenaOB 的一次快照
type arenaOBSnap struct {
	inUsed      int64
	objAllocNum uint64
	objFreeNum  uint64
	objectsCap  uint64
	idxPoolCap  uint64
}

// poolSnap 整个 pool 的一次快照
type poolSnap struct {
	ts              time.Time
	arenasBeforegc  []arenaSnap //释放前释放后
	arenasAftergc   []arenaSnap //释放后
	arenaOBBeforegc *arenaOBSnap
	arenaOBAftergc  *arenaOBSnap
}

var (
	snapMu   sync.Mutex
	lastSnap = make(map[string]*poolSnap) // pool name -> 上次快照（仅 MPool）
)

// poolKind 返回 pool 的类型标签，用于日志区分
//
// 已知接口直接给出友好名；未知的退化为具体类型名（如 *pool.oPool）。
// 如果后续新增了 OPool / SPool 等接口，按需补 case 即可。
func poolKind(p Pool) string {
	switch p.(type) {
	case MPool:
		return "MPool"
		// case OPool:
		// return "OPool"
		// case SPool:
		// return "SPool"
	}
	return fmt.Sprintf("%T", p)
}

// DumpAllPoolsGCStats 遍历所有 pool，打印精简统计（manual 触发，无 reason 标记）
func DumpAllPoolsGCStats(prefix string) {
	DumpAllPoolsGCStatsWithReason(prefix, "manual")
}

// DumpAllPoolsGCStatsWithReason 遍历所有 pool，打印精简统计：
// - Pool 行：池名 / 类型 / 累计分配、释放 / 在用 / 占用率
// - Arena 行：objectSize / spanNum / 累计 span 分配、释放 / 累计整块释放对象数 / 滞留对象数（带差值）
// - Span 汇总行：活跃 span 数 / 在用对象 / 总槽位 / 有释放历史的 span 数（替代逐 span 明细）
// - ArenaOB 行：与普通 arena 分开统计
//
// reason 标记本次采样来源（如 "tick" / "gc" / "manual"），打印在头部，便于区分采样线。
// prefix: 传给 GetPools 的前缀过滤，空串表示遍历全部
func DumpAllPoolsGCStatsWithReason(prefix, reason string) {
	snapMu.Lock()
	defer snapMu.Unlock()

	now := time.Now()
	fmt.Printf("[%s] memPool stats dump (reason=%s), time: %s\n", prefix, reason, now.Format(time.RFC3339))
	fmt.Printf("=== memPool Statistics ===\n")

	for _, p := range GetPools(prefix) {
		kind := poolKind(p)

		// 顶层统计：所有 pool 都打（comma-ok 避免 panic）
		if stat, ok := p.(Statistic); ok {
			fmt.Printf("Pool: %s [%s], Allocated: %d, Freed: %d, InUsed: %d, Usage: %d%%\n",
				p.Name(), kind, stat.Allocated(), stat.Freed(), stat.InUsed(), stat.Usage())
		} else {
			fmt.Printf("Pool: %s [%s]\n", p.Name(), kind)
		}

		// Arena 详情：只有 MPool 才有
		mp, ok := p.(MPool)
		if !ok {
			continue
		}

		// 抓当前 arena 快照
		stats := mp.ArenaStats()
		cur := &poolSnap{
			ts:     now,
			arenas: make([]arenaSnap, len(stats)),
		}
		for i, as := range stats {
			cur.arenas[i] = arenaSnap{
				objectSize:    as.ObjectSize(),
				spanNum:       as.SpanNum(),
				spanAllocNum:  as.SpanAllocNum(),
				spanFreedNum:  as.SpanFreedNum(),
				spanFreedObjs: as.SpanFreedObjs(),
				objRetainNum:  as.ObjRetainNum(),
				spanSummary:   summarizeSpans(as.ActiveSpans()),
			}
		}

		// 抓当前 arenaOB 快照（与普通 arena 分开统计）
		ob := mp.OBStats()
		cur.arenaOB = &arenaOBSnap{
			inUsed:      ob.InUsed(),
			objAllocNum: ob.ObjAllocNum(),
			objFreeNum:  ob.ObjFreeNum(),
			objectsCap:  ob.ObjectsCap(),
			idxPoolCap:  ob.IdxPoolCap(),
		}

		prev := lastSnap[mp.Name()]
		hasPrev := prev != nil && len(prev.arenas) == len(cur.arenas)

		for i := range cur.arenas {
			c := &cur.arenas[i]
			ss := &c.spanSummary

			// ===== Arena 行 =====
			if !hasPrev {
				fmt.Printf("  [Arena] ObjSize: %d, SpanNum: %d, SpanAlloc: %d, SpanFreed: %d, FreedObjs: %d, ObjRetain: %d\n",
					c.objectSize, c.spanNum, c.spanAllocNum, c.spanFreedNum,
					c.spanFreedObjs, c.objRetainNum)
			} else {
				pv := &prev.arenas[i]
				fmt.Printf("  [Arena] ObjSize: %d, SpanNum: %d(%+d), SpanAlloc: %d(%+d), SpanFreed: %d(%+d), FreedObjs: %d(%+d), ObjRetain: %d(%+d)\n",
					c.objectSize,
					c.spanNum, int64(c.spanNum)-int64(pv.spanNum),
					c.spanAllocNum, int64(c.spanAllocNum)-int64(pv.spanAllocNum),
					c.spanFreedNum, int64(c.spanFreedNum)-int64(pv.spanFreedNum),
					c.spanFreedObjs, int64(c.spanFreedObjs)-int64(pv.spanFreedObjs),
					c.objRetainNum, int64(c.objRetainNum)-int64(pv.objRetainNum),
				)
			}

			// ===== Span 汇总行 =====
			fmt.Printf("  [Spans] Active: %d, InUsed: %d, Slots: %d, FreedSpans: %d\n",
				ss.spanNum, ss.inUsedObjNum, ss.totalSlotNum, ss.freedSpanNum)
		}

		// ===== ArenaOB 行 =====（与普通 Arena 分开统计）
		prevOB := (*arenaOBSnap)(nil)
		if prev != nil {
			prevOB = prev.arenaOB
		}
		if prevOB == nil {
			fmt.Printf("  [ArenaOB] InUsed: %d, ObjAlloc: %d, ObjFree: %d, ObjectsCap: %d, IdxPoolCap: %d\n",
				cur.arenaOB.inUsed, cur.arenaOB.objAllocNum, cur.arenaOB.objFreeNum,
				cur.arenaOB.objectsCap, cur.arenaOB.idxPoolCap)
		} else {
			fmt.Printf("  [ArenaOB] InUsed: %d(%+d), ObjAlloc: %d(%+d), ObjFree: %d(%+d), ObjectsCap: %d, IdxPoolCap: %d\n",
				cur.arenaOB.inUsed, cur.arenaOB.inUsed-prevOB.inUsed,
				cur.arenaOB.objAllocNum, int64(cur.arenaOB.objAllocNum)-int64(prevOB.objAllocNum),
				cur.arenaOB.objFreeNum, int64(cur.arenaOB.objFreeNum)-int64(prevOB.objFreeNum),
				cur.arenaOB.objectsCap, cur.arenaOB.idxPoolCap)
		}

		lastSnap[mp.Name()] = cur
	}

	fmt.Printf("===========================\n")
}

// summarizeSpans 将活跃 span 列表汇总成一行描述，替代逐 span 明细。
func summarizeSpans(spans []SpanStats) spanSummary {
	var sum spanSummary
	for _, sp := range spans {
		sum.spanNum++
		sum.inUsedObjNum += uint64(sp.ObjectNum())
		sum.totalSlotNum += uint64(sp.SpanSize())
		if sp.ObjFreeNum() > 0 {
			sum.freedSpanNum++
		}
	}
	return sum
}

// StartPoolStatsTicker 启动定时采样：每 interval 调一次 DumpAllPoolsGCStatsWithReason。
// 与 GC 回调采样共用同一 lastSnap 差值链，两条采样线之间也能互相算差值。
//
// interval: 采样间隔（如 5s）
// prefix: 传给 GetPools 的前缀过滤，空串表示遍历全部
// reason: 采样来源标记，打印在头部（建议与 GC 回调的 reason 区分，如 "tick"）
// stopCh: 关闭信号，用于优雅退出
func StartPoolStatsTicker(interval time.Duration, prefix, reason string, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				DumpAllPoolsGCStatsWithReason(prefix, reason)
			case <-stopCh:
				return
			}
		}
	}()
}
