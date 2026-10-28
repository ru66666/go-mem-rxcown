// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// mpool 提前回收扩展（10.23/10.27）：对外导出入口。
//
// 组合方案（双机制，全档位覆盖）：
//   - 小对象（<32KiB-864）：freegc 显式复用链（malloc_freegc.go，设计
//     freegc.md 的 free list，"主动排队复用"）；
//   - 大对象（>32KiB）：mfree 两步清扫（malloc_mfree.go，页回 mheap +
//     记账下调 + scavenger 唤醒）。
//
// 随工具链重编默认生效，无需 GOEXPERIMENT/构建标签/环境变量；
// GODEBUG=eagerfreegc=0 可运行时关闭（两个机制共用该开关）。
//
// mpool 通过 //go:linkname 绑定本文件的 EagerFreeObj（见
// mempool/cpu-mem-coopt/instr.go 的 s4EagerFreeObj），签名固定为
// uintptr 进/uintptr 出，不引入指针类型（避免与 write barrier 冲突）。
//
// 语义约束（调用方必须满足，见 malloc_freegc.go / malloc_mfree.go）：
//   - objP 是堆对象的最后一个且唯一的引用（对象逻辑已死）；
//   - 对象无 specials（SetFinalizer/AddCleanup）；
//   - 调用发生在 alloc-safe 位置（内部 acquirem）。
//
// 返回值：
//   0 成功——内存已归还 allocator（小对象=链上可复用；大对象=页已回 mheap），
//          调用方此后不得再触碰该内存；
//   非 0 失败——对象状态未被修改，调用方退回原 GC 路径即可。

package runtime

import "unsafe"

// EagerFreeResult 提前回收结果码（与 mfree 的 Mfree* 码共用数值域）。
const (
	EagerFreeSuccess     = uintptr(MfreeSuccess)           // 成功：内存已归还
	EagerFreeFreegc      = uintptr(7)                      // 小对象：freegc 显式复用链
	EagerFreeMfreeSmall  = uintptr(8)                      // 小对象：mfree 位图翻转
	EagerFreeMfreeLarge  = uintptr(9)                      // 大对象：mfree 两步清扫
	EagerFreeGcRunning   = uintptr(MfreeGcRunning)         // GC 运行中，本次跳过
	EagerFreeNotInMcache = uintptr(MfreeNotInMcache)       // 小对象不在当前 mcache
	EagerFreeFailedLock  = uintptr(MfreeFailedToLockLarge) // 大对象 sweep 所有权竞争
	EagerFreeSkipped     = uintptr(6)                      // 通用失败：未生效，退回原路径
)

// EagerFreeObj 提前回收一个逻辑已死的堆对象（mpool 10.23/10.27 扩展入口）。
//
// 按尺寸自动路由：
//   - size < 32KiB-mallocHeaderSize：小对象，freegc 显式复用链；
//   - size >= 32KiB：大对象，mfree 两步清扫。
//
// 参数 size 必须等于该对象在 mallocgc 时的"用户请求尺寸"（对 mpool 而言
// 就是 span mem 的总字节数，即 (objectSize+reservedLen)*spanSize 档位值，
// 而不是 Go size class 取整后的值——freegc 内部会自行做 size class 换算）。
//
// push linkname 标记：允许 mpool 通过 pull-linkname 绑定本符号
// （Go 1.23+ 链接器 checklinkname 要求 def 侧有标记，见
// cmd/link/internal/loader checkLinkname 的 IsLinkname 放行规则）。
//
//	//go:linkname EagerFreeObj
//
// mpool 侧绑定：
//
//	//go:linkname s4EagerFreeObj runtime.EagerFreeObj
//	func s4EagerFreeObj(objP uintptr, size uintptr) uintptr
//
//go:linkname EagerFreeObj
func EagerFreeObj(objP, size uintptr) uintptr {
	if objP == 0 {
		return EagerFreeSkipped
	}
	if size >= maxTinySize && size < eagerFreegcSizeLimit-eagerFreegcHeaderSize {
		// 小对象：优先 freegc 显式复用链（主动 O(1) 复用）
		if freegc(unsafe.Pointer(objP), size, true) {
			return EagerFreeFreegc
		}
		// 10.28: freegc 未生效（例如 span 不在当前 mcache）→ 退级到
		// MfreeObj 位图翻转（mfree 小对象路径）：对 objIdx < freeindex 的
		// 槽位做 allocCache 移位回退 + allocCount--，槽位同样立即公共化。
		// MfreeObj 仍要求 span 属于当前 P 的 mcache，并且对象位于当前
		// 64 位 allocCache 窗口；它的优势是按指针反查真实 span/class，
		// 不会因调用者传入尺寸与 runtime size class 的取整差异而误判。
		code := MfreeObj(objP, 0, 0)
		if code == EagerFreeSuccess {
			return EagerFreeMfreeSmall
		}
		return code
	}
	// 大对象：mfree 两步清扫（页回 mheap）。
	// hint=0、cat=0 与 runtime 内部 mfree_obj() 的调用方式一致（noscan）。
	code := MfreeObj(objP, 0, 0)
	if code == EagerFreeSuccess {
		return EagerFreeMfreeLarge
	}
	return code
}

// EagerFreeEnabled 供调用方探测本 runtime 是否编译了提前回收扩展。
func EagerFreeEnabled() bool {
	return eagerFreegcEnabled
}
