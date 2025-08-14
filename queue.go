package go_ring_queue

import "unsafe"

type RingQueue[T any] struct {
	ptr  *T
	len  uint32
	cap  uint32
	ridx uint32
	widx uint32
}

func New[T any](initCapacity uint32) RingQueue[T] {
	slice := append(([]T)(nil), make([]T, initCapacity)...)
	cap := uint32(cap(slice))
	ptr := unsafe.SliceData(slice)
	return RingQueue[T]{
		ptr:  ptr,
		len:  0,
		cap:  cap,
		ridx: 0,
		widx: 0,
	}
}

func (q *RingQueue[T]) GetDataSlices() [2][]T {
	slice := unsafe.Slice(q.ptr, q.cap)
	len_end := q.ridx + q.len
	c1_end := min(q.cap, len_end)
	overflow := q.len - (c1_end - q.ridx)
	c1 := slice[q.ridx:c1_end]
	c2 := slice[0:overflow]
	return [2][]T{c1, c2}
}

func (q *RingQueue[T]) GetFreeSlices() [2][]T {
	slice := unsafe.Slice(q.ptr, q.cap)
	free := q.cap - q.len
	free_end := q.widx + free
	c1_end := min(q.cap, free_end)
	overflow := free - (c1_end - q.widx)
	c1 := slice[q.widx:c1_end]
	c2 := slice[0:overflow]
	return [2][]T{c1, c2}
}

func (q *RingQueue[T]) EnsureFreeSpace(n uint32) {
	if q.cap-q.len < n {
		newSlice := append(([]T)(nil), make([]T, q.len+n)...)
		newCap := uint32(cap(newSlice))
		newPtr := unsafe.SliceData(newSlice)
		oldData := q.GetDataSlices()
		n := copy(newSlice, oldData[0])
		copy(newSlice[n:], oldData[1])
		q.ptr = newPtr
		q.cap = newCap
		q.ridx = 0
		q.widx = q.len
	}
}

func (q *RingQueue[T]) Queue(val T) {
	q.EnsureFreeSpace(1)
	slice := unsafe.Slice(q.ptr, q.cap)
	slice[q.widx] = val
	q.widx += 1
	q.widx %= q.cap
	q.len += 1
}

func (q *RingQueue[T]) QueueMany(vals ...T) {
	n := uint32(len(vals))
	q.EnsureFreeSpace(n)
	frees := q.GetFreeSlices()
	nn := copy(frees[0], vals)
	copy(frees[1], vals[nn:])
	q.widx += n
	q.widx %= q.cap
	q.len += n
}

func (q *RingQueue[T]) Dequeue() (val T, ok bool) {
	ok = q.len > 0
	if !ok {
		return
	}
	slice := unsafe.Slice(q.ptr, q.cap)
	val = slice[q.ridx]
	q.ridx += 1
	q.ridx %= q.cap
	q.len -= 1
	return
}

func (q *RingQueue[T]) DequeueMany(n uint32) (vals []T) {
	datas := q.GetDataSlices()
	vals = make([]T, n)
	nn := copy(vals, datas[0])
	nn += copy(vals[nn:], datas[1])
	q.ridx += uint32(nn)
	q.ridx %= q.cap
	q.len -= uint32(nn)
	vals = vals[:nn]
	return
}
