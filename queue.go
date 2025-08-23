package go_ring_queue

import (
	"io"
)

// Holds a queue of values that are inserted at the end and removed
// (returned) from the front. Uses a [Ring Buffer](https://en.wikipedia.org/wiki/Circular_buffer)
// to efficiently recycle empty space without resizing the queue
// unless absolutely neccessary
type RingQueue[T any] struct {
	data []T
	ridx uint32
	widx uint32
}

// Create a new `RingQueue[T]` with capacity for at least
// `initCapacity` items
func New[T any](initCapacity uint32) RingQueue[T] {
	return RingQueue[T]{
		data: make([]T, 0, initCapacity),
		ridx: 0,
		widx: 0,
	}
}

// Return the current length of the queue
func (q RingQueue[T]) Len() int {
	return len(q.data)
}

// Return the current capacity of the queue
func (q RingQueue[T]) Cap() int {
	return cap(q.data)
}

// Reset the queue to a state with no data, but retain
// the existing memory capacity
func (q *RingQueue[T]) Clear() {
	q.data = q.data[:0]
	q.ridx = 0
	q.widx = 0
}

// Fully deinitialize the queue, releasing the memory pointer
// for the garbage collecter if no other references to it exist
func (q *RingQueue[T]) Release() {
	q.data = nil
	q.ridx = 0
	q.widx = 0
}

// Return the base underlying slice that is holding
// the queue. The current queue data may not be aligned
// to the beginning of the slice
func (q RingQueue[T]) RawSlice() []T {
	return q.data[:q.Cap()]
}

// Returns a new `RingQueue[T]` with a copy of the current data
func (q RingQueue[T]) Clone() RingQueue[T] {
	newSlice := make([]T, q.Len())
	data := q.GetDataSlices()
	n := copy(newSlice, data[0])
	copy(newSlice[n:], data[1])
	return RingQueue[T]{
		data: newSlice,
		widx: uint32(q.Len()),
		ridx: 0,
	}
}

// Return the data slices in-place holding all the current data in the queue.
//
// Returns 2 slices, in logical order, such that result[0] -> result[1]
// is in the same order that would be expected if using a normal
// slice/list
func (q RingQueue[T]) GetDataSlices() [2][]T {
	slice := q.RawSlice()
	len_end := int(q.ridx) + q.Len()
	c1_end := min(q.Cap(), len_end)
	overflow := q.Len() - (c1_end - int(q.ridx))
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
	slice := q.RawSlice()
	free := q.Cap() - q.Len()
	free_end := int(q.widx) + free
	c1_end := min(q.Cap(), free_end)
	overflow := free - (c1_end - int(q.widx))
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
func (q *RingQueue[T]) InreaseWriteIndex(n int) {
	q.EnsureFreeSpace(n)
	q.widx += uint32(n)
	q.widx %= uint32(q.Cap())
	q.data = q.data[:q.Len()+n]
}

// Explicitly increase the read index of the queue by n places,
// without returning any values or increaseing beyond the write index.
//
// Returns the actual number of places the read index was moved.
//
// This is intended for discarding unwanted values,
// or for use AFTER using `GetDataSlices()`
// to manually read data from the beginning of the queue.
func (q *RingQueue[T]) InreaseReadIndex(n int) (nActual int) {
	nActual = min(q.Len(), n)
	q.ridx += uint32(nActual)
	q.ridx %= uint32(q.Cap())
	q.data = q.data[:q.Len()-nActual]
	return
}

// Ensure the queue has space for at least n more items,
// resizing if neccessary
func (q *RingQueue[T]) EnsureFreeSpace(n int) {
	if q.Cap()-q.Len() < n {
		newSlice := append(([]T)(nil), make([]T, q.Len()+n)...)
		oldData := q.GetDataSlices()
		n := copy(newSlice, oldData[0])
		copy(newSlice[n:], oldData[1])
		q.data = newSlice
		q.ridx = 0
		q.widx = uint32(q.Len())
	}
}

// Append one val to the end of the queue, resizing
// if neccessary
func (q *RingQueue[T]) Queue(val T) {
	q.EnsureFreeSpace(1)
	slice := q.RawSlice()
	slice[q.widx] = val
	q.widx += 1
	q.widx %= uint32(q.Cap())
	q.data = q.data[:q.Len()+1]
}

// Append all vals to the end of the queue, resizing
// if neccessary
func (q *RingQueue[T]) QueueMany(vals ...T) {
	n := len(vals)
	q.EnsureFreeSpace(n)
	frees := q.GetFreeSlices()
	nn := copy(frees[0], vals)
	copy(frees[1], vals[nn:])
	q.widx += uint32(n)
	q.widx %= uint32(q.Cap())
	q.data = q.data[:q.Len()+n]
}

// Remove and return the first value at the front of the queue,
// and a `bool` indicating whether any value existed
// to return
func (q *RingQueue[T]) Dequeue() (val T, ok bool) {
	ok = q.Len() > 0
	if !ok {
		return
	}
	slice := q.RawSlice()
	val = slice[q.ridx]
	q.ridx += 1
	q.ridx %= uint32(q.Cap())
	q.data = q.data[:q.Len()-1]
	return
}

// Remove and return up to `n` vals from the front of the queue in a new slice
//
// If the queue has fewer than `n` items, the length of `vals`
// will be the previous length of the queue, and the queue
// will now be empty
func (q *RingQueue[T]) DequeueMany(n int) (vals []T) {
	datas := q.GetDataSlices()
	vals = make([]T, n)
	nn := copy(vals, datas[0])
	nn += copy(vals[nn:], datas[1])
	q.ridx += uint32(nn)
	q.ridx %= uint32(cap(q.data))
	q.data = q.data[:q.Len()-nn]
	vals = vals[:nn]
	return
}

// Remove and copy up to `n` vals from the front of the queue into
// the provided destination slice, returning the number of values actually
// dequeued
//
// `nCopied = min(n, len(dest), queue.Len())`
func (q *RingQueue[T]) DequeueManyInto(dest []T, n int) (nCopied int) {
	datas := q.GetDataSlices()
	nCopied = copy(dest[:n], datas[0])
	nCopied += copy(dest[nCopied:n], datas[1])
	q.ridx += uint32(nCopied)
	q.ridx %= uint32(cap(q.data))
	q.data = q.data[:q.Len()-nCopied]
	return
}

// Read is an implementation of io.Reader, genericized across all types
//
// Always returns error `io.EOF` if `n < len(p)`
func (q *RingQueue[T]) Read(p []T) (n int, err error) {
	n = int(q.DequeueManyInto(p, len(p)))
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
