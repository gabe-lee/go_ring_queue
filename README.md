# go_ring_queue
Golang ring queue implementation, for fast and memory efficient queue/dequeue operations

## Installation
Run the following command in your project:
```
go get github.com/gabe-lee/go_ring_queue@latest
```
Import where needed:
```golang
import "github.com/gabe-lee/go_ring_queue"
```

## Why might you want this?
  - Fast, memory efficient queue data structure using a [Ring Buffer (https://en.wikipedia.org/wiki/Circular_buffer)](https://en.wikipedia.org/wiki/Circular_buffer)
  - Operations are O(1) (single item) or O(N) (many item) in every case **_except_** when the queue has completely run out of underlying memory for a Queue, in which case it is O(M+N) (where M = current len, N = new count)
  - Around ~200 lines of code (not including comments)
  - 24 byte struct on 64bit machines (not including dynamic memory)
  - Fuzz tested against a standard slice using standard library methods for exactly equal state and return values
  - Implements `io.Reader`, `io.Writer`, `io.Closer`
  - No dependencies
  - MIT License
## Cons
  - Uses `uint32` fields for len, cap, read index, and write index
    - Maximum of 4294967295 values
  - Cannot safely hold pointers in queue (without keeping some other memory reference to the same item in scope)
    - The ring queue uses a scalar pointer to the root memory offset `*T` to save space. As a result, the golang GC is unlikely to infer that it represents a slice, and if that slice holds pointers, it may not know that those pointers are still in scope and may free them if no other references to the values exist