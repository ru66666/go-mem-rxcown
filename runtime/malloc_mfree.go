package runtime

import (
	"internal/runtime/math"
	"unsafe"
)

type memOptFrame struct {
	id   uintptr
	base int // data 在大数组中的起始位置
	size int // data 的大小
}

type memOptFrames struct {
	frames []memOptFrame
	data   []uintptr // 所有 frame 的数据存放在一起
}

type tfHandler struct {
	ptr      unsafe.Pointer
	sz       uintptr
	catogery uint8
	// 0: object-noscan
	// 1: slice-noscan
	// 2: map-noscan
	// 3: object-scan
	// 4: slice-scan
	// 5: map
}

type tfHandlers []tfHandler

type allocOptInfo struct {
	optNumPerHandler map[int]uint

	totalNewObjNum    uint
	toFreeNewObjNum   uint
	trueFreeNewObjNum uint

	totalSliceNum    uint
	toFreeSliceNum   uint
	trueFreeSliceNum uint

	totalMapNum    uint
	toFreeMapNum   uint
	trueFreeMapNum uint

	totalGrowSliceNum    uint
	toFreeGrowSliceNum   uint
	trueFreeGrowSliceNum uint
}

// Push 往栈里推入一个新的 frame
// frameSize 表示数据区大小，id 是帧标识，datas 是填充的数据
func (s *memOptFrames) push(id uintptr, index int, data unsafe.Pointer) {
	// grow stack
	if cap(s.data) < len(s.data)+index+1 {
		newCap := cap(s.data)*2 + index + 1
		if newCap < 64 {
			newCap = 64
		}
		newData := make([]uintptr, len(s.data), newCap)
		copy(newData, s.data)
		s.data = newData
	}

	if len(s.frames) > 0 {
		top := &s.frames[len(s.frames)-1]
		if top.id == id {
			if top.size <= index {
				s.data = s.data[:top.base+index+1]
				top.size = index + 1
			}
			s.data[top.base+index] = uintptr(data)
			return
		}
	}

	nextBase := len(s.data)
	s.frames = append(s.frames, memOptFrame{
		id:   id,
		base: nextBase,
		size: index + 1,
	})
	s.data = s.data[:nextBase+index+1]
	s.data[nextBase+index] = uintptr(data)
}

// Pop 弹出最顶层 frame
func (s *memOptFrames) pop() {
	if len(s.frames) == 0 {
		panic("pop from empty stack")
	}
	top := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	for i := top.base; i < len(s.data); i++ {
		s.data[i] = 0
	}
	s.data = s.data[:top.base]
}

// GetData 获取顶层 frame 中指定 idx 的数据
// 先验证 id 是否一致
func (s *memOptFrames) getdata(id uintptr, idx int) unsafe.Pointer {
	if len(s.frames) == 0 {
		return nil
	}
	if s.frames[len(s.frames)-1].id != id {
		return nil
	}
	if idx < 0 || idx >= s.frames[len(s.frames)-1].size {
		return nil
	}
	// if debug.mfreeall > 1 {
	// 	println("getdata:", id, idx, s.data[s.frames[len(s.frames)-1].base+idx])
	// }
	return unsafe.Pointer(s.data[s.frames[len(s.frames)-1].base+idx])
}

func Get_memopt_data(id uintptr, index int) uintptr {
	result := getg().gls_frames.getdata(id, index)
	if result != nil {
		return (uintptr)(unsafe.Pointer(result))
	}
	return 0
}

func Push_memopt_data(index int, optHandler uintptr) {
	// if debug.mfreeall > 1 {
	getg().gls_frames.push(getcallerfp(), index, unsafe.Pointer(optHandler))
}

func Pop_memopt_data() {
	getg().gls_frames.pop()
}

func recv_memopt_data(index int) unsafe.Pointer {
	fp := getcallerfp()
	if fp != 0 {
		fp = *(*uintptr)(unsafe.Pointer(fp))
	}
	return get_memopt_data(fp, index)
}

func get_memopt_data(id uintptr, index int) unsafe.Pointer {
	result := getg().gls_frames.getdata(id, index)
	if result != nil {
		return result
	}
	return nil
}

func push_memopt_data(index int, optHandler unsafe.Pointer) {
	// if debug.mfreeall > 1 {
	// 	println("push_memopt_data:", index, optHandler)
	// }
	getg().gls_frames.push(getcallerfp(), index, optHandler)
}

func pop_memopt_data() {
	getg().gls_frames.pop()
}

func intToString(num int) string {
	if num == 0 {
		return "0"
	}

	// 处理负数
	isNegative := num < 0
	if isNegative {
		num = -num
	}

	// 计算数字位数
	temp := num
	length := 0
	for temp > 0 {
		temp /= 10
		length++
	}

	if isNegative {
		length++ // 为负号预留位置
	}

	// 创建字节数组并填充数字
	result := make([]byte, length)
	i := length - 1

	for num > 0 {
		result[i] = byte('0' + num%10)
		num /= 10
		i--
	}

	// 添加负号（如果需要）
	if isNegative {
		result[0] = '-'
	}

	return string(result)
}

// i >= 3
func printCallStack(i int) {
	pcs := make([]uintptr, 1)
	n := Callers(i+3, pcs) // 直接获取第3层调用者（跳过runtime.Callers、printCallStack和直接调用者）

	if n == 0 {
		println("No caller found")
		return
	}

	frames := CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	println(getg(), ",", frame.Function, ", ", frame.File+":"+intToString(frame.Line))
}

func tofree_newobject(typ *_type, handler unsafe.Pointer, index int) unsafe.Pointer {
	result := newobject(typ)

	tofree := (*tfHandlers)(handler)
	if handler == nil {
		fp := getcallerfp()
		if fp != 0 {
			fp = *(*uintptr)(unsafe.Pointer(fp))
		}
		tofree = (*tfHandlers)(get_memopt_data(fp, index))
	}
	if tofree != nil {
		if typ.Pointers() {
			*tofree = append(*tofree, tfHandler{result, typ.Size_, 3})
		} else {
			*tofree = append(*tofree, tfHandler{result, typ.Size_, 0})
		}
		if debug.mfreeall > 0 {
			println("tofree_newobject:", result)
			printCallStack(0)
		}
	}

	if MfStatsEnable {
		mp := acquirem()
		mpid := mp.p.ptr().id
		mfstats[mpid].ceaAllocRecord.toFreeNewObjNum += 1
		if tofree != nil {
			mfstats[mpid].ceaAllocRecord.trueFreeNewObjNum += 1
		}
		releasem(mp)
	}
	return result
}

func tofree_makeslice(et *_type, length, cap int, handler unsafe.Pointer, index int) unsafe.Pointer {
	result := makeslice(et, length, cap)

	tofree := (*tfHandlers)(handler)
	if handler == nil {
		fp := getcallerfp()
		if fp != 0 {
			fp = *(*uintptr)(unsafe.Pointer(fp))
		}
		tofree = (*tfHandlers)(get_memopt_data(fp, index))
	}
	if tofree != nil {
		memSize, _ := math.MulUintptr(et.Size_, uintptr(cap))
		if et.Pointers() {
			*tofree = append(*tofree, tfHandler{result, memSize, 4})
		} else {
			*tofree = append(*tofree, tfHandler{result, memSize, 1})
		}
		if debug.mfreeall > 0 {
			println("tofree_makeslice:", result)
			printCallStack(0)
		}
	}

	if MfStatsEnable {
		mp := acquirem()
		mpid := mp.p.ptr().id
		mfstats[mpid].ceaAllocRecord.toFreeSliceNum += 1
		if tofree != nil {
			mfstats[mpid].ceaAllocRecord.trueFreeSliceNum += 1
		}
		releasem(mp)
	}
	return result
}

func tofree_makeslice64(et *_type, len64, cap64 int64, handler unsafe.Pointer, index int) unsafe.Pointer {
	result := makeslice64(et, len64, cap64)

	tofree := (*tfHandlers)(handler)
	if handler == nil {
		fp := getcallerfp()
		if fp != 0 {
			fp = *(*uintptr)(unsafe.Pointer(fp))
		}
		tofree = (*tfHandlers)(get_memopt_data(fp, index))
	}
	if tofree != nil {
		memSize, _ := math.MulUintptr(et.Size_, uintptr(cap64))
		if et.Pointers() {
			*tofree = append(*tofree, tfHandler{result, memSize, 4})
		} else {
			*tofree = append(*tofree, tfHandler{result, memSize, 1})
		}
		if debug.mfreeall > 0 {
			println("tofree_makeslice64:", result)
			printCallStack(0)
		}
	}

	if MfStatsEnable {
		mp := acquirem()
		mpid := mp.p.ptr().id
		mfstats[mpid].ceaAllocRecord.toFreeSliceNum += 1
		if tofree != nil {
			mfstats[mpid].ceaAllocRecord.trueFreeSliceNum += 1
		}
		releasem(mp)
	}
	return result
}

func tofree_growslice(oldPtr unsafe.Pointer, newLen, oldCap, num int, et *_type, handler unsafe.Pointer, index int) slice {
	result := growslice(oldPtr, newLen, oldCap, num, et)

	tofree := (*tfHandlers)(handler)
	if handler == nil {
		fp := getcallerfp()
		if fp != 0 {
			fp = *(*uintptr)(unsafe.Pointer(fp))
		}
		tofree = (*tfHandlers)(get_memopt_data(fp, index))
	}
	if tofree != nil {
		memSize := et.Size_ * uintptr(result.cap)
		if et.Pointers() {
			*tofree = append(*tofree, tfHandler{result.array, memSize, 4})
		} else {
			*tofree = append(*tofree, tfHandler{result.array, memSize, 1})
		}
		if debug.mfreeall > 0 {
			println("tofree_growslice:", result.array)
			printCallStack(0)
		}
	}

	if MfStatsEnable {
		mp := acquirem()
		mpid := mp.p.ptr().id
		mfstats[mpid].ceaAllocRecord.toFreeGrowSliceNum += 1
		if tofree != nil {
			mfstats[mpid].ceaAllocRecord.trueFreeGrowSliceNum += 1
		}
		releasem(mp)
	}

	return result
}

//go:nosplit
func mfree_all(handler unsafe.Pointer) {
	ptrs := (*tfHandlers)(handler)

	// 10.27: 提前回收始终生效；mfreeall 仅控制调试打印
	if debug.mfreeall > 0 && len(*ptrs) > 0 {
		printCallStack(0)
	}

	if MfStatsEnable {
		mp := acquirem()
		mpid := mp.p.ptr().id
		if _, ok := mfstats[mpid].ceaAllocRecord.optNumPerHandler[len(*ptrs)]; ok {
			mfstats[mpid].ceaAllocRecord.optNumPerHandler[len(*ptrs)] += 1
		} else {
			mfstats[mpid].ceaAllocRecord.optNumPerHandler[len(*ptrs)] = 1
		}
		releasem(mp)
	}

	for _, tfh := range *ptrs {
		switch tfh.catogery {
		case 0:
			fallthrough
		case 1:
			fallthrough
		case 3:
			fallthrough
		case 4:
			cat := tfh.catogery
			if tfh.catogery > 3 {
				cat -= 3
			}
			MfreeObj(uintptr(tfh.ptr), tfh.sz, int(cat))

		default:
			panic("mfree_all: unexpected catogery")
		}
	}
	// 10.27: 提前回收始终生效；mfreeall 仅控制调试打印
	if debug.mfreeall > 0 {
		println("")
	}

}

func mfree_obj(ptr uintptr) {
	if ptr != 0 {
		MfreeObj(ptr, 0, 0)
	}
}

func mfree_slice(ptr uintptr) {
	if ptr != 0 {
		MfreeObj(ptr, 0, 1)
	}
}

//go:nosplit
func MfreeObj(objP uintptr, hint uintptr, cat int) uintptr {
	// if debug.mfreeall == 0 {
	// 	return 0
	// }

	mp := acquirem()
	// // Avoid freeing objects inside MfreeObj
	// if mp.mallocing == 1 {
	// 	return 0
	// }
	mp.mallocing = 1

	printLog := debug.mfreetrace > 0
	fuzz := 0 // 0 true free, 1 fill zero, 2 inverse bits, 3 do nothing

	var deallocatedSize uintptr = 0
	resultCode := MfreeSuccess

	for true {
		if gcphase != _GCoff {
			if printLog {
				println("mfree gcphase != _GCoff")
			}
			resultCode = MfreeGcRunning
			break
		}

		if printLog {
			println("MfreeObj() called")
		}

		span := spanOfHeap(objP)
		if span == nil {
			if printLog {
				println("    on stack")
			}
			resultCode = MfreeOnStack
			break
		}

		spc := span.spanclass
		szc := spc.sizeclass()
		if printLog {
			println("    span", span, "size class", szc, "allocCount", span.allocCount)
		}
		if spc == tinySpanClass {
			if printLog {
				println("    skip tiny")
			}
			resultCode = MfreeTiny
			break
		}

		if span.specials != nil {
			if printLog {
				println("    have special")
			}
			resultCode = MfreeHaveSpecial
			break
		}

		elemsize := span.elemsize

		if szc != 0 {
			// Normal size: decrase bumping pointer

			c := getMCache(mp)
			spanFromC := c.alloc[spc]
			if span != spanFromC {
				if printLog {
					println("    not current m's mcache")
				}
				resultCode = MfreeNotInMcache
				break
			}

			freeindex := uintptr(span.freeindex)
			// Beginning of currently cached bits
			freeindexBase := freeindex &^ uintptr(64-1)
			objIdx := span.objIndex(objP)
			if printLog {
				println("    freeindex", freeindex, "objIdx", objIdx)
				println("    before allocCache", unsafe.Pointer(uintptr(span.allocCache)))
			}

			if !(objIdx >= freeindexBase && objIdx < freeindexBase+64) {
				if printLog {
					println("    not in allocCache")
				}
				resultCode = MfreeNotInAllocCache
				break
			}

			if objIdx >= freeindex {
				if printLog {
					println("    objIdx >= freeindex")
				}
				objOffset := objIdx - freeindex
				if span.allocCache&(1<<objOffset) != 0 {
					if printLog {
						println("    already mfreed")
					}
					resultCode = MfreeAlreadyMfreed
					break
				}
				if fuzz == 0 {
					span.allocCache |= (1 << objOffset)
					span.allocCount--
					span.needzero = 1
					// if debug.mfreeall > 0 {
					// 	ab := span.allocBitsForIndex(objIdx)
					// 	gmb := span.markBitsForIndex(objIdx)
					// 	println("Target object:", unsafe.Pointer(objP), ", gcMarkBits.isMarked:", gmb.isMarked(), ", allocCache:", hex(span.allocCache), ", allocBits.isMarked:", ab.isMarked(), ", span:", span, ", freeIndex:", span.freeindex, ", objectIndex:", objIdx, ", freeIndexForScan:", span.freeIndexForScan)
					// }
				} else if fuzz == 1 {
					for i := uintptr(0); i < elemsize; i++ {
						bytep := (*uint8)(unsafe.Pointer(span.base() + objIdx*elemsize + i))
						*bytep = uint8(0)
					}
				} else if fuzz == 2 {
					for i := uintptr(0); i < elemsize; i++ {
						bytep := (*uint8)(unsafe.Pointer(span.base() + objIdx*elemsize + i))
						*bytep = uint8(0xff - *bytep)
					}
				}
				if printLog {
					println("    after: freeIndex", span.freeindex, "allocCount", span.allocCount)
				}
			} else {
				negObjOffset := freeindex - objIdx
				if fuzz == 0 {
					span.allocCache <<= negObjOffset
					span.allocCache |= 1
					span.freeindex = uint16(objIdx)
					span.freeIndexForScan = span.freeindex
					span.allocCount--
					span.needzero = 1
					// if debug.mfreeall > 0 {
					// 	ab := span.allocBitsForIndex(objIdx)
					// 	gmb := span.markBitsForIndex(objIdx)
					// 	println("Target object:", unsafe.Pointer(objP), ", gcMarkBits.isMarked:", gmb.isMarked(), ", allocCache:", hex(span.allocCache), ", allocBits.isMarked:", ab.isMarked(), ", span:", span, ", freeIndex:", span.freeindex, ", objectIndex:", objIdx, ", freeIndexForScan:", span.freeIndexForScan)
					// }
				} else if fuzz == 1 {
					for i := uintptr(0); i < elemsize; i++ {
						bytep := (*uint8)(unsafe.Pointer(span.base() + objIdx*elemsize + i))
						*bytep = uint8(0)
					}
				} else if fuzz == 2 {
					for i := uintptr(0); i < elemsize; i++ {
						bytep := (*uint8)(unsafe.Pointer(span.base() + objIdx*elemsize + i))
						*bytep = uint8(0xff - *bytep)
					}
				}
				if printLog {
					println("    after: freeIndex", span.freeindex, "allocCount", span.allocCount)
				}
			}

			if printLog {
				println("    freed", elemsize, "bytes normal")
				println("    after allocCache", unsafe.Pointer(uintptr(span.allocCache)))
			}
		} else {
			if fuzz == 0 {
				// allocLarge: do a two-step sweep
				_g_ := getg()
				_g_.m.locks++
				sl := sweep.active.begin()
				// sl.blockCompletion()
				// spanLocked := sweepLocked{span}
				// 10.28 fix: begin() 在 sweep 队列已排空时返回 invalid
				// locker，此时不得调用 end()（activeSweep 计数会减到负数，
				// throw "mismatched begin/end of activeSweep"）。
				// 对齐 sweepone 的防护模式：仅在 sl.valid 时操作计数。
				if sl.valid {
					if spanLocked, ok := sl.tryAcquireMfreeLarge(span); ok {
						if span.state.get() != mSpanInUse {
							if printLog {
								println("    large span not in use")
							}
							sweep.active.end(sl)
							_g_.m.locks--
							resultCode = MfreeLargeNotInUse
							break
						}
						span.state.set(mSpanTwoStepSweep)
						spanLocked.sweep(false)
						if printLog {
							println("    freed", elemsize, "bytes large")
						}
						sweep.active.end(sl)
						_g_.m.locks--
					} else {
						if printLog {
							println("    failed to lock large span")
						}
						sweep.active.end(sl)
						_g_.m.locks--
						resultCode = MfreeFailedToLockLarge
						break
					}
				} else {
					if printLog {
						println("    sweep queue drained, defer to normal sweep")
					}
					_g_.m.locks--
					resultCode = MfreeFailedToLockLarge
					break
				}
			} else if fuzz == 1 {
				for i := uintptr(0); i < elemsize; i++ {
					bytep := (*uint8)(unsafe.Pointer(span.base() + i))
					*bytep = uint8(0)
				}
			} else if fuzz == 2 {
				for i := uintptr(0); i < elemsize; i++ {
					bytep := (*uint8)(unsafe.Pointer(span.base() + i))
					*bytep = uint8(0xff - *bytep)
				}
			}
		}

		if gcphase != _GCoff {
			println("mfree interrupted by gc")
			break
		}

		deallocatedSize = elemsize
		break
	}

	mp.mallocing = 0

	mid := mp.p.ptr().id

	programPoint := ""
	category := ""

	if HistogramEnable {
		head := ProgramPointCacheHead[mid]
		tail := ProgramPointCacheTail[mid]
		i := head
		for i < tail && ProgramPointCache[mid][i%PPCSIZE].addr != hint {
			i++
		}

		if i != tail {
			programPoint = ProgramPointCache[mid][i%PPCSIZE].programPoint
			category = ProgramPointCache[mid][i%PPCSIZE].category
		} else {
			buf := StackTraceBufferCache[mid][:]
			n := Stack(buf, false)
			if n < len(buf) {
				buf = buf[:n]
			}
			programPoint, category = getProgramPointFromStackTrace(string(buf))
			if programPoint == "NOOO" {
				print("Caching ", mid)
			}
			buf = nil
			ProgramPointCacheHead[mid]++
			ProgramPointCache[mid][tail%PPCSIZE].addr = hint
			ProgramPointCache[mid][tail%PPCSIZE].programPoint = programPoint
			ProgramPointCache[mid][tail%PPCSIZE].category = category
			ProgramPointCacheTail[mid]++
		}
	}

	if printLog {
		buf := StackTraceBufferCache[mid][:]
		var stackTrace string
		// Print stack trace
		n := Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
		}
		stackTrace = string(buf)
		// Don't fucking do heap allocation in mfree()!!!
		// You can't free yourself!!!
		if printLog {
			print(stackTrace)
		}
		buf = nil
	}

	if MfStatsEnable {
		mfstats[mid].MfreeCount++
		mfstats[mid].MfreeBytes += uint64(deallocatedSize)
		mfstats[mid].MfreeBytesCat[cat] += uint64(deallocatedSize)
		if HistogramEnable {

			histogramLen := HistogramLen[mid]

			// programPoint, category = getProgramPointFromStackTrace(stackTrace)
			// programPoint := "programPointprogramPointprogramPointprogramPointprogramPoint"
			// if histogramLenLocal < 100 {
			// 	programPoint += string(histogramLenLocal)
			// } else {
			// 	programPoint += "99"
			// }
			// category := "category"
			i := uint64(0)
			for i < histogramLen && Histogram[mid][i].programPoint != string(programPoint) {
				i++
			}

			// Assign term-by-term to avoid memory copy, which may causs nonsplit stack overflow
			if i == histogramLen {
				HistogramLen[mid]++
				Histogram[mid][i].programPoint = programPoint
				Histogram[mid][i].category = category
				Histogram[mid][i].tries = 1
				for j := 0; j < MfreeResultCodeEnd; j++ {
					Histogram[mid][i].results[j] = 0
				}
				Histogram[mid][i].results[resultCode]++
				Histogram[mid][i].size = uint64(deallocatedSize)
				// print("HistogramLen ", HistogramLen, "\n")
				// Release write lock
			} else {
				Histogram[mid][i].tries++
				Histogram[mid][i].results[resultCode]++
				Histogram[mid][i].size += uint64(deallocatedSize)
			}

		}
	}

	releasem(mp)

	return deallocatedSize
}

//go:nosplit
func MfreeSlice(v uintptr) uintptr {
	// v 应该是指向切片描述符 (slice header) 的指针的数值
	// 转换为 *[]byte 以便访问切片内容
	slicePtr := (*[]byte)(unsafe.Pointer(v))

	// ========== 以下为原有逻辑（从 *[]byte 版本复制并微调）==========
	// 10.27: 提前回收始终生效（不受 debug.mfreeslice/mfreeall 控制），
	// 与 MfreeObj 一致；如需临时禁用整个 mfree 机制，注释掉调用方
	//（mpool/instr.go 的 s4SpanEagerFree 分支）即可。

	printLog := debug.mfreetrace > 0

	if printLog {
		println("MfreeSlice() called")
	}

	// 获取切片内部表示（依赖运行时 slice 结构）
	s := (*slice)(unsafe.Pointer(slicePtr))
	objP := uintptr(s.array)

	// 将原切片置为 nil（通过传入的指针修改原变量）
	*slicePtr = nil

	if printLog {
		println("    slice, len", s.len, "cap", s.cap, "array", unsafe.Pointer(objP))
	}

	// 调用底层回收函数（第三个参数为 1，表示释放一个对象）
	return MfreeObj(objP, v, 1)
}

func getFirstLineFromString(s string) string { // Not including the '\n'
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[0:i]
		}
	}
	return s
}

func getProgramPointFromStackTrace(stk string) (pp string, category string) {
	category = "mapgrow"
	// origstk := stk
	for stk != "" {
		line := getFirstLineFromString(stk)
		if len(line) < 2 {
			stk = stk[len(line)+1:]
			continue
		}
		if stringIncludes(line, "MfreeSlice") {
			category = "slicedie"
		}
		if stringIncludes(line, "MfreeMap") {
			category = "mapdie"
		}
		if stringIncludes(line, "MfreeMapGrow") {
			category = "mapgrow"
		}
		// if len(line) < 2 {
		// 	print(origstk)
		// 	return "NOOO", "NOOO"
		// }
		if line[0] == '\t' && line[1] == '/' {
			if !stringIncludes(line, "malloc.go") {
				return line[1:], category
			}
		}
		stk = stk[len(line)+1:]
	}
	return "", ""
}

func stringIncludes(s string, p string) bool {
	for i := 0; i < len(s)-len(p)+1; i++ {
		matching := true
		for j := 0; j < len(p); j++ {
			if s[i+j] != p[j] {
				matching = false
				break
			}
		}
		if matching {
			return true
		}
	}
	return false
}
