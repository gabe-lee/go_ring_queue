package go_ring_queue

import (
	"io"
	"unsafe"
)

// Holds a queue of values that are inserted at the end and removed
// (returned) from the front. Uses a [Ring Buffer](https://en.wikipedia.org/wiki/Circular_buffer)
// to efficiently recycle empty space without resizing the queue
// unless absolutely neccessary
type RingQueue[T any] struct {
	ptr  *T
	len  uint32
	cap  uint32
	ridx uint32
	widx uint32
}

// Create a new `RingQueue[T]` with capacity for at least
// `initCapacity` items
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

// Return the current length of the queue
func (q RingQueue[T]) Len() int {
	return int(q.len)
}

// Return the current capacity of the queue
func (q RingQueue[T]) Cap() int {
	return int(q.cap)
}

// Reset the queue to a state with no data, but retain
// the existing memory capacity
func (q *RingQueue[T]) Clear() {
	q.len = 0
	q.ridx = 0
	q.widx = 0
}

// Fully deinitialize the queue, releasing the memory pointer
// for the garbage collecter if no other references to it exist
func (q *RingQueue[T]) Release() {
	q.ptr = nil
	q.cap = 0
	q.len = 0
	q.ridx = 0
	q.widx = 0
}

// Return the base underlying slice that is holding
// the queue. The current queue data may not be aligned
// to the beginning of the slice
func (q RingQueue[T]) RawSlice() []T {
	return unsafe.Slice(q.ptr, q.cap)
}

// Returns a new `RingQueue[T]` with a copy of the current data
func (q RingQueue[T]) Clone() RingQueue[T] {
	newQueue := New[T](q.cap)
	slice := newQueue.RawSlice()
	data := q.GetDataSlices()
	n := copy(slice, data[0])
	copy(slice[n:], data[1])
	newQueue.len = q.len
	newQueue.ridx = 0
	newQueue.widx = q.widx
	return newQueue
}

// Return the data slices in-place holding all the current data in the queue.
//
// Returns 2 slices, in logical order, such that result[0] -> result[1]
// is in the same order that would be expected if using a normal
// slice/list
func (q RingQueue[T]) GetDataSlices() [2][]T {
	slice := unsafe.Slice(q.ptr, q.cap)
	len_end := q.ridx + q.len
	c1_end := min(q.cap, len_end)
	overflow := q.len - (c1_end - q.ridx)
	c1 := slice[q.ridx:c1_end]
	c2 := slice[0:overflow]
	return [2][]T{c1, c2}
}

// Return the free slices in-place holding all the current free space in the queue.
//
// Returns 2 slices, in logical order, such that result[0] -> result[1]
// is in the same order that would be expected if using a normal
// slice/list
func (q RingQueue[T]) GetFreeSlices() [2][]T {
	slice := unsafe.Slice(q.ptr, q.cap)
	free := q.cap - q.len
	free_end := q.widx + free
	c1_end := min(q.cap, free_end)
	overflow := free - (c1_end - q.widx)
	c1 := slice[q.widx:c1_end]
	c2 := slice[0:overflow]
	return [2][]T{c1, c2}
}

// Explicitly increase the write index of the queue by n places,
// without writing any values
//
// This may resize the queue if needed. The values in the queued spaces
// are undefined. This is intended for use AFTER using `GetFreeSlices()`
// to manually write data into the beginning of the free area,
// or BEFORE using `GetDataSlices()` to manually write the new data to the end
// of the extended data area
func (q *RingQueue[T]) InreaseWriteIndex(n uint32) {
	q.EnsureFreeSpace(n)
	q.widx += n
	q.widx %= q.cap
	q.len += n
}

// Explicitly increase the read index of the queue by n places,
// without returning any values or increaseing beyond the write index.
//
// Returns the actual number of places the read index was moved.
//
// This is intended for discarding unwanted values,
// or for use AFTER using `GetDataSlices()`
// to manually read data from the beginning of the queue.
func (q *RingQueue[T]) InreaseReadIndex(n uint32) (nActual uint32) {
	nActual = min(q.len, n)
	q.ridx += nActual
	q.ridx %= q.cap
	q.len -= nActual
	return
}

// Ensure the queue has space for at least n more items,
// resizing if neccessary
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

// Append one val to the end of the queue, resizing
// if neccessary
func (q *RingQueue[T]) Queue(val T) {
	q.EnsureFreeSpace(1)
	slice := unsafe.Slice(q.ptr, q.cap)
	slice[q.widx] = val
	q.widx += 1
	q.widx %= q.cap
	q.len += 1
}

// Append all vals to the end of the queue, resizing
// if neccessary
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

// Remove and return the first value at the front of the queue,
// and a `bool` indicating whether any value existed
// to return
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

// Remove and return up to `n` vals from the front of the queue in a new slice
//
// If the queue has fewer than `n` items, the length of `vals`
// will be the previous length of the queue, and the queue
// will now be empty
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

// Remove and copy up to `n` vals from the front of the queue into
// the provided destination slice, returning the number of values actually
// dequeued
//
// `nCopied = min(n, len(dest), queue.Len())`
func (q *RingQueue[T]) DequeueManyInto(dest []T, n uint32) (nCopied uint32) {
	datas := q.GetDataSlices()
	nCopied = uint32(copy(dest[:n], datas[0]))
	nCopied += uint32(copy(dest[nCopied:n], datas[1]))
	q.ridx += uint32(nCopied)
	q.ridx %= q.cap
	q.len -= uint32(nCopied)
	return
}

// Read is an implementation of io.Reader, genericized across all types
//
// Always returns error `io.EOF` if `n < len(p)`
func (q *RingQueue[T]) Read(p []T) (n int, err error) {
	n = int(q.DequeueManyInto(p, uint32(len(p))))
	if n != len(p) {
		err = io.EOF
	}
	return
}

// Write is an implementation of io.Writer, genericized across all types
//
// Returned error is always `nil`
func (q *RingQueue[T]) Write(p []T) (n int, err error) {
	q.QueueMany(p...)
	return len(p), nil
}

// Close is an implementation of io.Closer
//
// This is an alias for queue.Release(), which fully deinitializes the
// queue and releases the memory pointer. Always returns `nil`
func (q *RingQueue[T]) Close() error {
	q.Release()
	return nil
}

var _ io.Writer = (*RingQueue[byte])(nil)
var _ io.Reader = (*RingQueue[byte])(nil)
var _ io.Closer = (*RingQueue[byte])(nil)
