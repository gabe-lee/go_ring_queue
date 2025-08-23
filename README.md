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
  - 32 byte struct on 64bit machines (not including dynamic memory)
  - Fuzz tested against a standard slice using standard library methods for exactly equal state and return values
  - Implements `io.Reader`, `io.Writer`, `io.Closer`
  - No dependencies
  - MIT License
## Cons
  - Uses `uint32` fields for read index and write index
    - Maximum of 4294967295 values