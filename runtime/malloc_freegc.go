// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// mpool 提前回收扩展（10.23/10.27，移植自 freegc 提案 issue 74299 / CL 673695，
// 设计文档 cpu-pool/freegc.md，参考实现 cpu-pool/mempool/cpu-mem-coopt/malloc.go，
// 适配 go1.24.0→1.24.2 runtime）。
//
// 本文件实现小对象（<32KiB）的 freegc：把逻辑已死的 noscan 堆对象记入当前 P
// 的 mcache 显式复用链，之后的 mallocgcSmallNoscan 从链上 O(1) 弹出立即复用，
// 无需等待 GC 标记/清扫，也无需位图扫描（与 mfree 位图翻转的"被动等待发现"
// 相比，这是 freegc.md 设计的"主动排队复用"）。
//
// 开关：GODEBUG=eagerfreegc 控制，默认开启（见 runtime1.go parsedebugvars）。
// 关闭时 freegc 直接 no-op，mallocgcSmallNoscan 走原路径，运行时行为与
// 未改动版本完全一致。
//
// 安全约束（与上游 freegc 相同，freegc.md:298-328）：
//   - ptr 必须是对象的最后一个且唯一的引用（"logically dead"）；
//   - 对象必须无 specials（SetFinalizer/AddCleanup）；
//   - 16B <= size <= 32KiB - mallocHeaderSize，否则 no-op；
//   - noscan 对象专用（带指针对象返回 false 退化，不 throw）。

package runtime

import "unsafe"

// eagerFreegcEnabled 由 GODEBUG=eagerfreegc 控制，在 parsedebugvars 中解析。
var eagerFreegcEnabled bool

// eagerFreegcSizeLimit 复用链支持的最大用户尺寸。
// 与上游一致：>32KiB 的对象走 mallocgcLarge（整 mspan），无槽位可挂，
// 由 mfree 两步清扫处理（malloc_mfree.go）。
const eagerFreegcSizeLimit = 32 << 10

// eagerFreegcHeaderSize malloc header 大小（mbitmap.go 定义）。
const eagerFreegcHeaderSize = mallocHeaderSize

// reusableFreegcSize reports whether size is currently supported by the
// freegc reuse list.
func reusableFreegcSize(size uintptr) bool {
	// >= maxTinySize: tiny allocator 拥有 <16B 对象，不参与（与上游一致）。
	// <= eagerFreegcSizeLimit-eagerFreegcHeaderSize: 上限之上走 large 路径。
	return size >= maxTinySize && size <= eagerFreegcSizeLimit-eagerFreegcHeaderSize
}

// freegc records that a heap object is reusable and available for immediate
// reuse in a subsequent mallocgcSmallNoscan allocation, without needing to
// wait for the GC cycle to progress.
//
// 必须在 alloc-safe 位置调用（本函数 acquirem，同 mallocgc）。
// 返回 true 表示已成功挂入复用链（对象所有权移交 runtime）；
// 返回 false 表示未生效（GODEBUG 关闭/尺寸超限/带指针/不在当前 mcache），
// 调用方应退回原 GC 路径，对象状态未被修改。
func freegc(ptr unsafe.Pointer, size uintptr, noscan bool) bool {
	if !eagerFreegcEnabled || !reusableFreegcSize(size) {
		return false
	}
	if !noscan {
		// 上游 throw；这里温和退化（返回 false），便于调用方统一走
		// "失败退回原路径"逻辑。
		return false
	}

	if ptr == nil {
		throw("freegc nil")
	}

	// Set mp.mallocing to keep from being preempted by GC.
	// Otherwise, the GC could flush our mcache or otherwise cause problems.
	mp := acquirem()
	if mp.mallocing != 0 {
		throw("freegc deadlock")
	}
	if mp.gsignal == getg() {
		throw("freegc during signal")
	}
	mp.mallocing = 1

	if mp.curg != nil && mp.curg.stack.lo <= uintptr(ptr) && uintptr(ptr) < mp.curg.stack.hi {
		// Points into our stack, so free is a no-op.
		mp.mallocing = 0
		releasem(mp)
		return false
	}

	if debug.clobberfree != 0 {
		clobberfree(ptr, size)
	}

	// We first check if p is still in our per-P cache.
	c := getMCache(mp)
	if c == nil {
		throw("freegc called without a P or outside bootstrapping")
	}

	v := uintptr(ptr)

	// 与官方 freegc 一致：size 是调用者提供的用户尺寸，用它计算复用链
	// 的 span class；这里不要求对象属于当前 P 的 mcache。对象可能是在
	// 另一个 P 上分配的，仍然可以先挂到当前 P 的显式复用链。真正复用
	// 时再通过 spanOf 找回对象所属 span。
	var sizeclass uint8
	if size <= smallSizeMax-8 {
		sizeclass = size_to_class8[divRoundUp(size, smallSizeDiv)]
	} else {
		sizeclass = size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]
	}
	spc := makeSpanClass(sizeclass, true)

	// 挂入当前 P 的显式复用链（freegc.md 设计的 free list）。
	// addReusableNoscan 在 mcache.go 中实现：链表节点直接使用对象自身
	// 内存（gclink），零额外分配。
	c.addReusableNoscan(spc, v)

	// For stats, we leave allocCount alone, roughly pretending to the rest
	// of the system that this potential reuse never happened.（与上游一致）

	mp.mallocing = 0
	releasem(mp)

	return true
}

// nextReusableFreegc returns the next reusable object and, when the write
// barrier is enabled, its actual span. The object may have been freed by a
// different P, so the caller's current span is not necessarily the right one.
//
// 由 mallocgcSmallNoscan（malloc.go）在分配侧调用；见该处接入说明。
//
//go:nosplit
func (c *mcache) nextReusableFreegc(s *mspan, spc spanClass) (gclinkptr, *mspan) {
	v := c.reusableFreegc[spc]
	if v == 0 {
		return 0, s
	}
	c.reusableFreegc[spc] = v.ptr().next
	if !writeBarrier.enabled {
		return v, nil
	}
	if s != nil && s.base() <= uintptr(v) && uintptr(v) < s.limit {
		return v, s
	}
	span := spanOf(uintptr(v))
	if span == nil {
		throw("nextReusableFreegc: nil span for pointer in reusable object free list")
	}
	return v, span
}

// hasReusableFreegc reports whether the reuse list for spc is non-empty.
//
//go:nosplit
func (c *mcache) hasReusableFreegc(spc spanClass) bool {
	return c.reusableFreegc[spc] != 0
}

// mallocgcSmallNoscanFreegcReuse pops a previously freed noscan object from
// the reuse list and prepares it for reuse. Must only be called if
// hasReusableFreegc returned true. 见 malloc.go 接入点。
//
//go:nosplit
func mallocgcSmallNoscanFreegcReuse(c *mcache, span *mspan, spc spanClass, size uintptr, needzero bool) unsafe.Pointer {
	v, span := c.nextReusableFreegc(span, spc)

	x := unsafe.Pointer(v)

	// 补偿 GC assist 记账：mallocgc 分配新对象时会扣 assist 信用，
	// 复用对象不是新分配，这里把信用加回，避免记账失真
	//（与上游 mallocgcSmallNoscanReuse 相同的处理）。
	freegcAssistCompensate(size)

	if needzero {
		// 复用对象可能含旧数据；官方复用路径按调用者的 needzero
		// 直接清零，不依赖原 span 的 needzero 标志。
		memclrNoHeapPointers(x, size)
	}

	// Allocate black during GC（与上游一致）：
	// 复用对象在标记期间被分配时需置黑，避免保守扫描误判。
	if writeBarrier.enabled {
		gcmarknewobject(span, uintptr(x))
	}
	return x
}

// freegcAssistCompensate 补偿 GC assist 记账：mallocgc 分配新对象时会
// deductAssistCredit(size)，复用对象不是新分配，这里把信用加回。
// deductAssistCredit 内部为 gcAssistBytes -= int64(size)，
// uintptr 的二进制补码负数经 int64 转换后即为负值，实现"加回"。
//
//go:nosplit
func freegcAssistCompensate(size uintptr) {
	deductAssistCredit(-size)
}
