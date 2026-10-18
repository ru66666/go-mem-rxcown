// Copyright (c) Huawei Technologies Co., Ltd. 2012-2019. All rights reserved.
// package pool: Provides data pooling services based on sync.Pool

package pool

import (
	"math"
	"reflect"
	"sync/atomic"
	"unsafe"

	"code.huawei.com/5gcore/base/data/ddf/common/derror"
	"code.huawei.com/5gcore/base/data/ddf/common/dlog"
)

/*
span memory format:
| 1 byte | 1 byte | 2 bytes | 4 bytes | 4 bytes | data len | 4 bytes |
| arena idx | object idx | data len | span idx | header magic | data | tail magic |
*/

/*
out of bounds arena memory format:
| 1 byte | 3 bytes | 4 bytes | 4 bytes | data len | 4 bytes |
| arena idx | 0 | object idx | header magic | data | tail magic |
*/

const (
	arenaIdxOffset    = 0
	spanIdxOffset     = 4
	objectIdxOffsetOB = 4
	objectIdxOffset   = 1
	dataLenOffset     = 2
	// length of memory object header
	headerLen = 8
	// head magic
	headerMagicLen = 2
	// tail magic
	tailMagicLen = 2
	// magic number
	magicNum = 0x5a5a
	// reserved len
	reservedLen = headerLen + headerMagicLen + tailMagicLen
	// data offset
	dataOffset = headerLen + headerMagicLen
	// default memory object num in each span
	DefaultSpanSize = 16
	// max memory object num in each span
	MaxSpanSize = math.MaxUint8
	// default max object in memory pool
	MaxObjectSize = 32 * 1024
	// upper limit of max object in memory pool
	MaxObjectSizeUpperLimit = 64 * 1024
	// default object size diff between adjacent arena
	DefaultArenaInterval = 512
	// minimum object size diff between adjacent arena
	MinArenaInterval = 64
	// invalid arena index
	ArenaOutOfBounds = 0xff
)

const (
	invalidSpanIdx = 0xffffffff
	sentinel       = 0
)

const (
	DxSpanHeaderDamaged    = "span header damaged"
	DxSpanTailDamaged      = "span tail damaged"
	DxArenaOBHeaderDamaged = "arena ob header damaged"
	DxArenaOBTailDamaged   = "arena ob tail damaged"
)

// memory pool interface
type MPool interface {
	Name() string
	Statistic
	New(name string) (MPool, error)
	Release()
	Close()
	Malloc(size uint32) uintptr
	Free(ptr uintptr)
	Size() uint64
	ArenaStats() []ArenaStats
	OBStats() ArenaOBStats

	Dx() string
}

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

type arenaOBStats struct {
	poolName      string
	inUsed        int64
	objAllocNum   uint64
	objFreeNum    uint64
	objFreedBytes uint64
	objectsCap    uint64
	idxPoolCap    uint64
}

// arena stats interface
type ArenaStats interface {
	PoolName() string
	ObjectSize() uint32
	SpanSize() uint8
	SpanNum() uint32
	Usage() int // 0-100 arena usage means percentage of memory objects in use
	SpanAllocNum() uint64
	SpanFreedNum() uint64
	SpanFreedBytes() uint64
	SpanFreedObjs() uint64
	ObjRetainNum() uint64
	ActiveSpans() []SpanStats
}

type arenaStats struct {
	poolName       string
	objectSize     uint32
	spanSize       uint8
	spanNum        uint32
	usage          int // 0-100
	spanAllocNum   uint64
	spanFreedNum   uint64
	spanFreedBytes uint64
	spanFreedObjs  uint64
	objRetainNum   uint64
	activeSpans    []SpanStats
}

/*
@title
Idx
@description
Get span index of span stats
@PARAM
@return
uint32: Span index of span stats
@note
*/
func (s *spanStats) Idx() uint32 {
	return s.idx
}

/*
@title
ObjectNum
@description
Get object num of span stats
@PARAM
@return
uint8: Object num of span stats
@note
*/
func (s *spanStats) ObjectNum() uint8 {
	return s.objectNum
}

/*
@title
SpanSize
@description
Get span size of span stats
@PARAM
@return
uint8: Span size of span stats
@note
*/
func (s *spanStats) SpanSize() uint8 {
	return s.spanSize
}

/*
@title
ObjAllocNum
@description
Get accumulated malloc object num of span stats
@PARAM
@return
uint64: Accumulated malloc object num of span stats
@note
*/
func (s *spanStats) ObjAllocNum() uint64 {
	return s.objAllocNum
}

/*
@title
ObjFreeNum
@description
Get accumulated free object num of span stats
@PARAM
@return
uint64: Accumulated free object num of span stats
@note
*/
func (s *spanStats) ObjFreeNum() uint64 {
	return s.objFreeNum
}

/*
@title
PoolName
@description
Get pool name of arenaOB stats
@PARAM
@return
string: Pool name of arenaOB stats
@note
*/
func (o *arenaOBStats) PoolName() string {
	return o.poolName
}

/*
@title
InUsed
@description
Get inuse object num of arenaOB stats
@PARAM
@return
int64: Inuse object num of arenaOB stats
@note
*/
func (o *arenaOBStats) InUsed() int64 {
	return o.inUsed
}

/*
@title
ObjAllocNum
@description
Get accumulated malloc object num of arenaOB stats
@PARAM
@return
uint64: Accumulated malloc object num of arenaOB stats
@note
*/
func (o *arenaOBStats) ObjAllocNum() uint64 {
	return o.objAllocNum
}

/*
@title
ObjFreeNum
@description
Get accumulated free object num of arenaOB stats
@PARAM
@return
uint64: Accumulated free object num of arenaOB stats
@note
*/
func (o *arenaOBStats) ObjFreeNum() uint64 {
	return o.objFreeNum
}

/*
@title
ObjFreedBytes
@description
Get accumulated freed bytes of arenaOB stats
@PARAM
@return
uint64: Accumulated freed bytes of arenaOB stats
@note
*/
func (o *arenaOBStats) ObjFreedBytes() uint64 {
	return o.objFreedBytes
}

/*
@title
ObjectsCap
@description
Get objects array peak capacity of arenaOB stats
@PARAM
@return
uint64: Objects array peak capacity of arenaOB stats
@note
*/
func (o *arenaOBStats) ObjectsCap() uint64 {
	return o.objectsCap
}

/*
@title
IdxPoolCap
@description
Get idxPool array peak capacity of arenaOB stats
@PARAM
@return
uint64: IdxPool array peak capacity of arenaOB stats
@note
*/
func (o *arenaOBStats) IdxPoolCap() uint64 {
	return o.idxPoolCap
}

/*
@title
PoolName
@description
Get pool name of arena stats
@PARAM
@return
string: Pool name of arena stats
@note
*/
func (a *arenaStats) PoolName() string {
	return a.poolName
}

/*
@title
ObjectSize
@description
Get object size of arena stats
@PARAM
@return
uint32: Object size of arena stats
@note
*/
func (a *arenaStats) ObjectSize() uint32 {
	return a.objectSize
}

/*
@title
SpanSize
@description
Get span size of arena stats
@PARAM
@return
uint8: Span size of arena stats
@note
*/
func (a *arenaStats) SpanSize() uint8 {
	return a.spanSize
}

/*
@title
SpanNum
@description
Get span num of arena stats
@PARAM
@return
uint32: Span num of arena stats
@note
*/
func (a *arenaStats) SpanNum() uint32 {
	return a.spanNum
}

/*
@title
Usage
@description
Get usage of arena stats
@PARAM
@return
int: Usage of arena stats
@note
*/
func (a *arenaStats) Usage() int {
	return a.usage
}

/*
@title
SpanAllocNum
@description
Get accumulated allocated span num of arena stats
@PARAM
@return
uint64: Accumulated allocated span num of arena stats
@note
*/
func (a *arenaStats) SpanAllocNum() uint64 {
	return a.spanAllocNum
}

/*
@title
SpanFreedNum
@description
Get accumulated freed span num of arena stats
@PARAM
@return
uint64: Accumulated freed span num of arena stats
@note
*/
func (a *arenaStats) SpanFreedNum() uint64 {
	return a.spanFreedNum
}

/*
@title
SpanFreedBytes
@description
Get accumulated freed span bytes of arena stats
@PARAM
@return
uint64: Accumulated freed span bytes of arena stats
@note
*/
func (a *arenaStats) SpanFreedBytes() uint64 {
	return a.spanFreedBytes
}

/*
@title
SpanFreedObjs
@description
Get accumulated freed span object num of arena stats
@PARAM
@return
uint64: Accumulated freed span object num of arena stats
@note
*/
func (a *arenaStats) SpanFreedObjs() uint64 {
	return a.spanFreedObjs
}

/*
@title
ObjRetainNum
@description
Get accumulated freed-but-retained object num of arena stats
@PARAM
@return
uint64: Accumulated freed-but-retained object num of arena stats
@note
*/
func (a *arenaStats) ObjRetainNum() uint64 {
	return a.objRetainNum
}

/*
@title
ActiveSpans
@description
Get active span stats list of arena stats
@PARAM
@return
[]SpanStats: Active span stats list of arena stats
@note
*/
func (a *arenaStats) ActiveSpans() []SpanStats {
	return a.activeSpans
}

/*
@title

newArenaStats

@description

# Create new ArenaStats instance

@PARAM

poolName string: Pool name

objectSize uint32: Object size

spanNum uint32: Span num

spanSize uint8: Span size

inUseNum int64: Inuse num

@return

ArenaStats: ArenaStats instance

@note
*/
func newArenaStats(poolName string, objectSize, spanNum uint32, spanSize uint8, inUseNum int64,
	spanAllocNum, spanFreedNum, spanFreedBytes, spanFreedObjs, objRetainNum uint64, activeSpans []SpanStats) ArenaStats {
	state := &arenaStats{
		poolName:       poolName,
		objectSize:     objectSize,
		spanSize:       spanSize,
		spanNum:        spanNum,
		spanAllocNum:   spanAllocNum,
		spanFreedNum:   spanFreedNum,
		spanFreedBytes: spanFreedBytes,
		spanFreedObjs:  spanFreedObjs,
		objRetainNum:   objRetainNum,
		activeSpans:    activeSpans,
	}

	if spanNum == 0 {
		state.usage = fullUsage
	} else {
		totalNum := int(spanSize) * int(spanNum)
		state.usage = int(inUseNum) * fullUsage / totalNum
	}

	return state
}

/*
@title
newArenaOBStats
@description
Create new ArenaOBStats instance
@PARAM
poolName string: Pool name
inUsed int64: Inuse object num
objAllocNum uint64: Accumulated malloc object num
objFreeNum uint64: Accumulated free object num
objFreedBytes uint64: Accumulated freed bytes
objectsCap uint64: Objects array peak capacity
idxPoolCap uint64: IdxPool array peak capacity
@return
ArenaOBStats: ArenaOBStats instance
@note
*/
func newArenaOBStats(poolName string, inUsed int64,
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

type span struct {
	pre         uint32
	next        uint32
	idx         uint32
	objectSize  uint32
	spanSize    uint8
	objectNum   uint8
	objAllocNum uint64 // 累计 malloc 对象数（统计用）
	objFreeNum  uint64 // 累计 free 对象数（统计用）
	mem         []byte
	objects     []uint8
}

/*
@title
initSpan
@description
Initialize span instance
@PARAM
idx uint32: Span index
spanSize uint8: Span size
objectSize uint32: Object size
@return
@note
*/
func (s *span) initSpan(idx uint32, spanSize uint8, objectSize uint32) {
	s.idx = idx
	s.spanSize = spanSize
	s.objectSize = objectSize
	s.objAllocNum = 0
	s.objFreeNum = 0
	memSize := (objectSize + reservedLen) * uint32(spanSize)
	if memSize > math.MaxInt32 {
		dlog.DebugEF(nil, "mem size %d out of range", memSize)
		return
	}
	s.mem = make([]byte, memSize, memSize)
	s.objects = make([]uint8, spanSize, spanSize)
	ptr := uintptr(unsafe.Pointer(&s.mem[0]))
	for i := uint8(0); i < spanSize; i++ {
		s.objects[i] = i
		setPtrObjectIdx(ptr, i)
		setMagic(ptr + uintptr(headerLen))
		setMagic(ptr + uintptr(s.objectSize+dataOffset))
		ptr += uintptr(s.objectSize + reservedLen)
	}
}

/*
@title

getObjectIdx

@description

# Get available object index in span

@PARAM

@return

uint8: Available object index in span

@note
*/
func (s *span) getObjectIdx() uint8 {
	offset := s.spanSize - s.objectNum - 1
	idx := s.objects[offset]
	s.objects = s.objects[:offset]

	return idx
}

/*
@title
getObjectIdxByPtr
@description
Get object index from memory address
@PARAM
ptr uintptr: Memory block address
@return
uint8: Object index
@note
*/
func getObjectIdxByPtr(ptr uintptr) uint8 {
	return *(*uint8)(unsafe.Pointer(ptr + objectIdxOffset))
}

/*
@title
setPtrObjectIdx
@description
Set object index to memory block header
@PARAM
ptr uintptr: Memory block address
idx uint8: Object index
@return
@note
*/
func setPtrObjectIdx(ptr uintptr, idx uint8) {
	*(*uint8)(unsafe.Pointer(ptr + objectIdxOffset)) = idx
}

/*
@title
setMagic
@description
Set object magic number
@PARAM
ptr uintptr: Memory block address
@return
@note
*/
func setMagic(ptr uintptr) {
	*(*uint16)(unsafe.Pointer(ptr)) = magicNum
}

/*
@title
checkMagic
@description
Check object magic number
@PARAM
ptr uintptr: Memory block address
@return
@note
*/
func checkMagic(ptr uintptr) bool {
	return *(*uint16)(unsafe.Pointer(ptr)) == magicNum
}

/*
@title

malloc

@description

# Malloc memory block from span

@PARAM

@return

uintptr: Memory block address

@note
*/
func (s span) malloc() uintptr {
	idx := s.getObjectIdx()
	ptr := uintptr(unsafe.Pointer(&s.mem[0])) + uintptr(uint32(idx)(s.objectSize+reservedLen))
	s.objectNum++
	atomic.AddUint64(&s.objAllocNum, 1)

	return ptr
}

/*
@title

free

@description

# Free memory block to span

@PARAM

ptr uintptr: Memory block address

@return

bool: If span is empty after free

@note
*/
func (s *span) free(ptr uintptr) bool {
	idx := getObjectIdxByPtr(ptr)
	resetObject(ptr+dataOffset, s.objectSize)
	s.objects = append(s.objects, idx)
	s.objectNum--
	atomic.AddUint64(&s.objFreeNum, 1)

	return s.objectNum == 0
}

/*
@title
full
@description
If span is full
@PARAM
@return
bool: If span is full
@note
*/
func (s *span) full() bool {
	return len(s.objects) == 0
}

/*
@title
size
@description
Object num in span
@PARAM
@return
uint8: Object num in span
@note
*/
func (s *span) size() uint8 {
	return s.objectNum
}

/*
@title
clear
@description
Clear span
@PARAM
@return
@note
*/
func (s *span) clear() {
	s.pre = invalidSpanIdx
	s.next = invalidSpanIdx
	s.objectNum = 0
	s.objAllocNum = 0
	s.objFreeNum = 0
	s.mem = nil
	s.objects = nil
}

/*
@title
Pre
@description
Get pre span by chain
@PARAM
a *arena: Arena to which span belongs
@return
*span: Pre span
@note
*/
func (s *span) Pre(a *arena) *span {
	return &a.spans[s.pre]
}

/*
@title
Next
@description
Get next span by chain
@PARAM
a *arena: Arena to which span belongs
@return
*span: Next span
@note
*/
func (s *span) Next(a *arena) *span {
	return &a.spans[s.next]
}

/*
@title

# Dx

@description

mem span diagnosis

@PARAM

@return

string: mem span diagnosis report

@note
*/
func (s *span) Dx() string {
	o := ""

	if len(s.mem) == 0 {
		return o
	}

	ptr := uintptr(unsafe.Pointer(&s.mem[0]))
	for i := uint8(0); i < s.spanSize; i++ {
		if !checkMagic(ptr + uintptr(headerLen)) {
			o += DxSpanHeaderDamaged + "\n"
			break
		}
		if !checkMagic(ptr + uintptr(s.objectSize+dataOffset)) {
			o += DxSpanTailDamaged + "\n"
			break
		}
		ptr += uintptr(s.objectSize + reservedLen)
	}

	return o
}

type arenaOB struct { // arena out of bounds
	statistic
	objects       [][]byte
	idxPool       []uint32
	objAllocNum   uint64 // 累计 malloc 对象数（统计用）
	objFreeNum    uint64 // 累计 free 对象数（统计用）
	objFreedBytes uint64 // 累计 free 对象字节数（统计用）
	objectsCap    uint64 // objects 数组峰值容量（统计用）
	idxPoolCap    uint64 // idxPool 数组峰值容量（统计用）
}

/*
@title

malloc

@description

# Malloc memory block from arenaOB

@PARAM

size uint32: Memory block size

@return

uintptr: Memory block address

@note
*/
func (o *arenaOB) malloc(size uint32) uintptr {
	s := make([]byte, size+reservedLen, size+reservedLen)
	ptr := uintptr(unsafe.Pointer(&s[0]))
	idx := o.getObjectIdx()
	setPtrOBObjectIdx(ptr, idx)
	setMagic(ptr + headerLen)
	setMagic(ptr + dataOffset + uintptr(size))
	o.objects[idx] = s
	o.inc(int64(cap(s)), int64(cap(s)))
	atomic.AddUint64(&o.objAllocNum, 1)
	atomic.StoreUint64(&o.objectsCap, maxUint64(o.objectsCap, uint64(cap(o.objects))))
	atomic.StoreUint64(&o.idxPoolCap, maxUint64(o.idxPoolCap, uint64(cap(o.idxPool))))

	return ptr
}

/*
@title

free

@description

# Free memory block to arenaOB

@PARAM

ptr uintptr: Memory block address

@return

int64: Memory block size

@note
*/
func (o *arenaOB) free(ptr uintptr) int64 {
	idx := getOBObjectIdxByPtr(ptr)
	size := int64(cap(o.objects[idx]))
	o.dec(size, size)
	o.objects[idx] = nil
	o.idxPool = append(o.idxPool, idx)
	atomic.AddUint64(&o.objFreeNum, 1)
	atomic.AddUint64(&o.objFreedBytes, uint64(size))
	atomic.StoreUint64(&o.idxPoolCap, maxUint64(o.idxPoolCap, uint64(cap(o.idxPool))))

	return size - reservedLen
}

/*
@title

getObjectIdx

@description

# Get available object index in arenaOB

@PARAM

@return

uint32: Available object index in arenaOB

@note
*/
func (o *arenaOB) getObjectIdx() uint32 {
	if len(o.idxPool) == 0 {
		o.objects = append(o.objects, nil)
		return uint32(len(o.objects) - 1)
	}

	idx := o.idxPool[len(o.idxPool)-1]
	o.idxPool = o.idxPool[:len(o.idxPool)-1]

	return idx
}

/*
@title
release
@description
Release arenaOB
@PARAM
@return
@note
*/
func (o *arenaOB) release() {
	o.objects = nil
	o.idxPool = nil
	atomic.StoreUint64(&o.objAllocNum, 0)
	atomic.StoreUint64(&o.objFreeNum, 0)
	atomic.StoreUint64(&o.objFreedBytes, 0)
	atomic.StoreUint64(&o.objectsCap, 0)
	atomic.StoreUint64(&o.idxPoolCap, 0)
	o.statistic.clear()
}

/*
@title

# Dx

@description

mem arena diagnosis

@PARAM

@return

string: mem arena diagnosis report

@note
*/
func (o *arenaOB) Dx() string {
	ou := ""

	if len(o.objects) == 0 {
		return ou
	}

	for i := 0; i < len(o.objects); i++ {
		if o.objects[i] == nil {
			continue
		}
		ptr := uintptr(unsafe.Pointer(&o.objects[i][0]))
		if !checkMagic(ptr + uintptr(headerLen)) {
			ou += DxArenaOBHeaderDamaged + "\n"
			break
		}
		if !checkMagic(ptr + uintptr(len(o.objects[i])-tailMagicLen)) {
			ou += DxArenaOBTailDamaged + "\n"
			break
		}
	}

	return ou
}

/*
@title
size
@description
Get inuse size of arenaOB
@PARAM
@return
uint64: Inuse size of arenaOB
@note
*/
func (o *arenaOB) size() uint64 {
	return uint64(atomic.LoadInt64(&o.inUseSize))
}

type arena struct {
	statistic
	idx            uint8
	spanSize       uint8
	objectSize     uint32
	spanNum        uint32
	spanAllocNum   uint64 // 累计新建 span 数（统计用）
	spanFreedNum   uint64 // 累计释放 span 数（统计用）
	spanFreedBytes uint64 // 累计释放 span 字节数（统计用）
	spanFreedObjs  uint64 // 累计释放 span 对象数（统计用）
	objRetainNum   uint64 // 累计释放但 span 未空导致滞留的对象数（统计用）
	spans          []span
	idxPool        []uint32
}

/*
@title
initArena
@description
Initialize arena instance
@PARAM
idx uint8: Arena index
spanSize uint8: Span size
objectSize uint32: Object size
@return
@note
*/
func (a *arena) initArena(idx, spanSize uint8, objectSize uint32) {
	a.idx = idx
	a.spanSize = spanSize
	a.objectSize = objectSize
	a.initSpans()
}

/*
@title
size
@description
Get total size of arena
@PARAM
@return
uint64: Total size of arena
@note
*/
func (a *arena) size() uint64 {
	return uint64(a.objectSize+reservedLen) * uint64(a.spanSize) * uint64(atomic.LoadUint32(&a.spanNum))
}

/*
@title
release
@description
Release arena
@PARAM
@return
@note
*/
func (a *arena) release() {
	atomic.StoreUint32(&a.spanNum, 0)
	atomic.StoreUint64(&a.spanAllocNum, 0)
	atomic.StoreUint64(&a.spanFreedNum, 0)
	atomic.StoreUint64(&a.spanFreedBytes, 0)
	atomic.StoreUint64(&a.spanFreedObjs, 0)
	atomic.StoreUint64(&a.objRetainNum, 0)
	a.initSpans()
	a.idxPool = nil
	a.statistic.clear()
}

/*
@title
initSpans
@description
Initialize spans in arena
@PARAM
@return
@note
*/
func (a *arena) initSpans() {
	a.spans = make([]span, 1, 1)
	a.spans[0].idx = sentinel
	a.sentinel().pre = sentinel
	a.sentinel().next = sentinel
}

/*
@title
ObjectSize
@description
Get object size of arena
@PARAM
@return
uint32: Object size of arena
@note
*/
func (a *arena) ObjectSize() uint32 {
	return a.objectSize
}

/*
@title
SpanNum
@description
Get span num of arena
@PARAM
@return
uint32: Span num of arena
@note
*/
func (a *arena) SpanNum() uint32 {
	return atomic.LoadUint32(&a.spanNum)
}

/*
@title

getSpanIdx

@description

# Get available span index in arena

@PARAM

@return

uint32: Available span index in arena

@note
*/
func (a *arena) getSpanIdx() uint32 {
	if len(a.idxPool) == 0 {
		a.spans = append(a.spans, span{})
		return uint32(len(a.spans) - 1)
	}

	idx := a.idxPool[len(a.idxPool)-1]
	a.idxPool = a.idxPool[:len(a.idxPool)-1]

	return idx
}

/*
@title
toHead
@description
Move span to chain head
@PARAM
s *span: Span to move
@return
@note
*/
func (a *arena) toHead(s *span) {
	s.next = a.sentinel().next
	s.pre = sentinel
	a.head().pre = s.idx
	a.sentinel().next = s.idx
}

/*
@title
toTail
@description
Move span to chain tail
@PARAM
s *span: Span to move
@return
@note
*/
func (a *arena) toTail(s *span) {
	s.Next(a).pre = s.pre
	s.Pre(a).next = s.next
	s.next = sentinel
	s.pre = a.sentinel().pre
	a.tail().next = s.idx
	a.sentinel().pre = s.idx
}

/*
@title
remove
@description
Remove span from arena
@PARAM
s *span: Span to remove
@return
@note
*/
func (a *arena) remove(s *span) {
	s.Next(a).pre = s.pre
	s.Pre(a).next = s.next
}

/*
@title
sentinel
@description
Get sentinel span in arena
@PARAM
@return
*span: Sentinel span
@note
*/
func (a *arena) sentinel() *span {
	return &a.spans[sentinel]
}

/*
@title
head
@description
Get head span in arena
@PARAM
@return
*span: Head span
@note
*/
func (a *arena) head() *span {
	return &a.spans[a.spans[sentinel].next]
}

/*
@title
tail
@description
Get tail span in arena
@PARAM
@return
*span: Tail span
@note
*/
func (a *arena) tail() *span {
	return &a.spans[a.spans[sentinel].pre]
}

/*
@title

malloc

@description

# Malloc memory block from arena

@PARAM

size uint32: Memory block size

@return

uintptr: Memory block address

@note
*/
func (a *arena) malloc(size uint32) uintptr {
	a.inc(int64(a.objectSize), int64(size))

	s := a.head()
	if atomic.LoadUint32(&a.spanNum) == 0 || s.full() { // no span or all spans full, new one
		idx := a.getSpanIdx()
		s = &a.spans[idx]
		s.initSpan(idx, a.spanSize, a.objectSize)
		atomic.AddUint32(&a.spanNum, 1)
		atomic.AddUint64(&a.spanAllocNum, 1)
		a.toHead(s)
	}

	ptr := s.malloc()
	if s.full() && !s.Next(a).full() {
		a.toTail(s)
	}
	setPtrSpanIdx(ptr, s.idx)

	return ptr
}

/*
@title
move
@description
Move span as sorting rules
@PARAM
s *span: Span to move
@return
@note
*/
func (a *arena) move(s *span) {
	if s.pre != sentinel && s.Pre(a).full() {
		a.moveForward(s)
	} else {
		a.moveBackward(s)
	}
}

/*
@title
moveForward
@description
Move span forward as sorting rules
@PARAM
s *span: Span to move
@return
@note
*/
func (a *arena) moveForward(s *span) {
	a.remove(s)
	a.toHead(s)
}

/*
@title

moveBackward

@description

# Move span backward as sorting rules

@PARAM

s *span: Span to move

@return

@note
*/
func (a *arena) moveBackward(s *span) {
	var target *span
	n := s
	for {
		n = n.Next(a)
		if n.idx == sentinel || n.full() || s.size() >= n.size() {
			break
		}
		target = n
	}

	if target != nil {
		a.swap(s, target)
	}
}

/*
@title
swap
@description
Swap two spans
@PARAM
x *span: Span to swap
y *span: Span to swap
@return
@note
*/
func (a *arena) swap(x, y *span) {
	// adjacent swap
	if y.idx == x.Next(a).idx {
		x.Pre(a).next = y.idx
		y.pre = x.pre
		y.Next(a).pre = x.idx
		x.pre = y.idx
		x.next = y.next
		y.next = x.idx
	} else { // non-adjacent swap
		x.Pre(a).next = y.idx
		x.Next(a).pre = y.idx
		y.Next(a).pre = x.idx
		y.Pre(a).next = x.idx
		x.pre, y.pre = y.pre, x.pre
		x.next, y.next = y.next, x.next
	}
}

/*
@title

free

@description

# Free memory block to arena

@PARAM

ptr uintptr: Memory block address

@return

@note
*/
func (a *arena) free(ptr uintptr) {
	a.dec(int64(a.objectSize), int64(getPtrDataLen(ptr)))

	idx := getSpanIdxByPtr(ptr)
	s := &a.spans[idx]

	if s.free(ptr) {
		a.remove(s)
		atomic.AddUint64(&a.spanFreedNum, 1)
		atomic.AddUint64(&a.spanFreedBytes, uint64(a.objectSize+reservedLen)*uint64(a.spanSize))
		atomic.AddUint64(&a.spanFreedObjs, uint64(a.spanSize))
		s.clear()
		a.idxPool = append(a.idxPool, idx)
		atomic.AddUint32(&a.spanNum, ^uint32(0))
	} else {
		atomic.AddUint64(&a.objRetainNum, 1)
		if atomic.LoadUint32(&a.spanNum) > 1 {
			a.move(s)
		}
	}
}

/*
@title

# Dx

@description

mem arena diagnosis

@PARAM

@return

string: mem arena diagnosis report

@note
*/
func (a *arena) Dx() string {
	o := ""

	for _, s := range a.spans {
		o += s.Dx()
	}

	return o
}

/*
@title

activeSpans

@description

# Get active span stats in arena by chain order

@PARAM

@return

[]SpanStats: Active span stats in arena

@note
*/
func (a *arena) activeSpans() []SpanStats {
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

type memPool struct {
	statistic
	name      string
	arenaProp ArenaProp
	arenas    []arena
	arenaOB   arenaOB
}

/*
@title
Name
@description
Get pool name
@PARAM
@return
string: Pool name
@note
*/
func (m *memPool) Name() string {
	return m.name
}

/*
@title

# Malloc

@description

# Malloc memory block

@PARAM

size uint32: Memory block size

@return

uintptr: Memory block address

@note
*/
func (m *memPool) Malloc(size uint32) uintptr {
	if size == 0 {
		return 0
	}

	var ptr uintptr

	idx := m.arenaProp.GetArenaIdx(size)
	if idx == ArenaOutOfBounds {
		ptr = m.arenaOB.malloc(size)
		m.inc(int64(size), int64(size))
	} else {
		a := &m.arenas[idx]
		ptr = a.malloc(size)
		m.inc(int64(a.ObjectSize()), int64(size))
		setPtrDataLen(ptr, uint16(size))
	}
	setPtrArenaIdx(ptr, idx)

	return ptr + dataOffset
}

/*
@title

# Free

@description

# Free memory block

@PARAM

ptr uintptr: Memory block address

@return

@note
*/
func (m *memPool) Free(ptr uintptr) {
	ptrO := ptr - dataOffset
	idx := getArenaIdxByPtr(ptrO)
	if idx == ArenaOutOfBounds {
		size := m.arenaOB.free(ptrO)
		m.dec(size, size)
		return
	}

	a := &m.arenas[idx]
	m.dec(int64(a.ObjectSize()), int64(getPtrDataLen(ptrO)))
	a.free(ptrO)
}

/*
@title
Release
@description
Release all allocated memory blocks
@PARAM
@return
@note
*/
func (m *memPool) Release() {
	for i := range m.arenas {
		m.arenas[i].release()
	}
	m.arenaOB.release()
	m.statistic.clear()
}

/*
@title
Close
@description
Close mem pool
@PARAM
@return
@note
*/
func (m *memPool) Close() {
	deletePool(m.name)
}

/*
@title

# NewMPool

@description

# Create new MPool instance

@PARAM

name string: Pool name

arenaProp ArenaProp: Arena Property

@return

MPool: MPool instance

error: Any errors that may occur in this process

@note
*/
func NewMPool(name string, arenaProp ArenaProp) (MPool, error) {
	m := &memPool{
		name: name,
	}

	m.arenaProp = arenaProp
	arenaItems := arenaProp.GetArenaItems()
	if len(arenaItems) > ArenaOutOfBounds {
		return nil, derror.ErrArenaItemsNumOutOfRange
	}

	m.arenas = make([]arena, len(arenaItems), len(arenaItems))
	for i := range arenaItems {
		m.arenas[i].initArena(uint8(i), arenaItems[i].SpanSize, arenaItems[i].ObjectSize)
	}
	addPool(name, m)

	return m, nil
}

/*
@title
New
@description
Create new MPool instance
@PARAM
name string: Pool name
@return
MPool: MPool instance
error: Any errors that may occur in this process
@note
*/
func (m *memPool) New(name string) (MPool, error) {
	return NewMPool(name, m.arenaProp)
}

/*
@title

# Size

@description

# Get total size of mem pool

@PARAM

@return

uint64: Total size of mem pool

@note
*/
func (m *memPool) Size() uint64 {
	var totalSize uint64
	for i := range m.arenas {
		totalSize += m.arenas[i].size()
	}
	totalSize += m.arenaOB.size()

	return totalSize
}

/*
@title

# ArenaStats

@description

# Get all ArenaStats in mem pool

@PARAM

@return

[]ArenaStats: All ArenaStats in mem pool

@note
*/
func (m *memPool) ArenaStats() []ArenaStats {
	var states []ArenaStats
	for i := range m.arenas {
		a := &m.arenas[i]
		states = append(states, newArenaStats(m.name, a.objectSize, atomic.LoadUint32(&a.spanNum), a.spanSize, a.InUsed(),
			atomic.LoadUint64(&a.spanAllocNum), atomic.LoadUint64(&a.spanFreedNum),
			atomic.LoadUint64(&a.spanFreedBytes), atomic.LoadUint64(&a.spanFreedObjs),
			atomic.LoadUint64(&a.objRetainNum), a.activeSpans()))
	}

	return states
}

/*
@title
OBStats
@description
Get arenaOB stats of mem pool
@PARAM
@return
ArenaOBStats: ArenaOB stats of mem pool
@note
*/
func (m *memPool) OBStats() ArenaOBStats {
	o := &m.arenaOB
	return newArenaOBStats(m.name, o.InUsed(),
		atomic.LoadUint64(&o.objAllocNum), atomic.LoadUint64(&o.objFreeNum),
		atomic.LoadUint64(&o.objFreedBytes), atomic.LoadUint64(&o.objectsCap),
		atomic.LoadUint64(&o.idxPoolCap))
}

/*
@title

# Dx

@description

mem pool diagnosis

@PARAM

@return

string: mem pool diagnosis report

@note
*/
func (m *memPool) Dx() string {
	o := ""

	o += m.arenaOB.Dx()

	for _, a := range m.arenas {
		o += a.Dx()
	}

	if o != "" {
		o = "mem pool diagnosis: " + m.name + "\n" + o
	}

	return o
}

/*
@title

# ToBytes

@description

# Transfer memory block address to byte slice

@PARAM

size int: Byte slice size

ptr uintptr: Memory block address

@return

[]byte: Byte slice

@note
*/
func ToBytes(size int, ptr uintptr) []byte {
	var byteSlice []byte
	s := (*reflect.SliceHeader)(unsafe.Pointer(&byteSlice))
	s.Data, s.Len, s.Cap = ptr, size, size

	return byteSlice
}

/*
@title
getArenaIdxByPtr
@description
Get arena index value from memory block address
@PARAM
ptr uintptr: Memory block address
@return
uint8: Arena index
@note
*/
func getArenaIdxByPtr(ptr uintptr) uint8 {
	return *(*uint8)(unsafe.Pointer(ptr + arenaIdxOffset))
}

/*
@title
setPtrArenaIdx
@description
Set arena index value to memory block header
@PARAM
ptr uintptr: Memory block address
idx uint8: Arena index
@return
@note
*/
func setPtrArenaIdx(ptr uintptr, idx uint8) {
	*(*uint8)(unsafe.Pointer(ptr + arenaIdxOffset)) = idx
}

/*
@title
getPtrDataLen
@description
Get data length value from memory block address
@PARAM
ptr uintptr: Memory block address
@return
uint16: Data length
@note
*/
func getPtrDataLen(ptr uintptr) uint16 {
	return *(*uint16)(unsafe.Pointer(ptr + dataLenOffset))
}

/*
@title
setPtrDataLen
@description
Set data length value to memory block header
@PARAM
ptr uintptr: Memory block address
len uint16: Data length
@return
@note
*/
func setPtrDataLen(ptr uintptr, len uint16) {
	*(*uint16)(unsafe.Pointer(ptr + dataLenOffset)) = len
}

/*
@title
getSpanIdxByPtr
@description
Get span index value from memory block address
@PARAM
ptr uintptr: Memory block address
@return
uint32: Span index
@note
*/
func getSpanIdxByPtr(ptr uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(ptr + spanIdxOffset))
}

/*
@title
setPtrSpanIdx
@description
Set span index value to memory block header
@PARAM
ptr uintptr: Memory block address
idx uint32: Span index
@return
@note
*/
func setPtrSpanIdx(ptr uintptr, idx uint32) {
	*(*uint32)(unsafe.Pointer(ptr + spanIdxOffset)) = idx
}

/*
@title
getOBObjectIdxByPtr
@description
Get object index value from memory block address
@PARAM
ptr uintptr: Memory block address
@return
uint32: Object index
@note
*/
func getOBObjectIdxByPtr(ptr uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(ptr + objectIdxOffsetOB))
}

/*
@title
setPtrOBObjectIdx
@description
Set object index value to memory block header
@PARAM
ptr uintptr: Memory block address
idx uint32: Object index
@return
@note
*/
func setPtrOBObjectIdx(ptr uintptr, idx uint32) {
	*(*uint32)(unsafe.Pointer(ptr + objectIdxOffsetOB)) = idx
}

var emptyObject = make([]byte, math.MaxUint16, math.MaxUint16)

// maxUint64 returns the larger of two uint64 values (统计辅助函数)
func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

/*
@title
resetObject
@description
Reset memory block
@PARAM
object uintptr: Memory block address
size uint32: Memory block size
@return
@note
*/
func resetObject(object uintptr, size uint32) {
	var byteSlice []byte
	s := (*reflect.SliceHeader)(unsafe.Pointer(&byteSlice))
	s.Data, s.Len, s.Cap = object, int(size), int(size)
	copy(byteSlice, emptyObject)
}

/*
when implementing this interface, it is necessary to ensure that the returned index value does not cross the
boundary, and the size of the memory object managed in corresponding arena is no less than entry parameter size,
also a enumeration unit test is needed to verify the correctness of the function.
*/
type ArenaProp interface {
	GetArenaIdx(size uint32) uint8
	GetArenaItems() []ArenaItem
}

// ArenaItem
type ArenaItem struct {
	Level      int
	SpanSize   uint8
	ObjectSize uint32
}

/*
@title
NewArenaItem
@description
Create new ArenaItem instance
@PARAM
level int: Arena level
spanSize uint8: Span size
objectSize uint32: Object size
@return
ArenaItem: ArenaItem instance
@note
*/
func NewArenaItem(level int, spanSize uint8, objectSize uint32) ArenaItem {
	return ArenaItem{Level: level, SpanSize: spanSize, ObjectSize: objectSize}
}

type fixedArenaProp struct {
	interval uint32
	maxSize  uint32
	items    []ArenaItem
}

/*
@title

# GetArenaIdx

@description

# Get arena index by malloc size

@PARAM

size uint32: Malloc size

@return

uint8: Arena index

@note
*/
func (f *fixedArenaProp) GetArenaIdx(size uint32) uint8 {
	if size > f.maxSize {
		return ArenaOutOfBounds
	}

	idx := size / f.interval
	if size%f.interval == 0 {
		idx--
	}

	return uint8(idx)
}

/*
@title
GetArenaItems
@description
Get arena items
@PARAM
@return
[]ArenaItem: Arena items
@note
*/
func (f *fixedArenaProp) GetArenaItems() []ArenaItem {
	return f.items
}

/*
@title

# NewFixedArenaProp

@description

# Create new ArenaProp instance which has fixed arena interval

@PARAM

interval uint32: Size between neighboring arena

maxSize uint32: Max arena size

spanSize uint8: Span size

@return

ArenaProp: ArenaProp instance

@note
*/
func NewFixedArenaProp(interval, maxSize uint32, spanSize uint8) ArenaProp {
	var items []ArenaItem
	level := 0
	currentMax := uint32(0)
	for size := interval; size <= maxSize; size += interval {
		items = append(items, NewArenaItem(level, spanSize, size))
		currentMax = size
		level++
	}

	return &fixedArenaProp{interval: interval, maxSize: currentMax, items: items}
}

/*
@title
NewMPoolDefault
@description
Create new MPool instance by default arena property
@PARAM
name string: Pool name
@return
MPool: MPool instance
error: Any errors that may occur in this process
@note
*/
func NewMPoolDefault(name string) (MPool, error) {
	return NewMPool(name, NewFixedArenaProp(DefaultArenaInterval, MaxObjectSize, DefaultSpanSize))
}
