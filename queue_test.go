package go_ring_queue

import (
	"slices"
	"strings"
	"testing"
)

func FuzzQueue(f *testing.F) {
	const (
		ACTION_QUEUE_ONE byte = iota
		ACTION_QUEUE_MANY
		ACTION_DEQUEUE_ONE
		ACTION_DEQUEUE_MANY
	)
	var has2More = func(fuzzInput *[]byte) bool {
		return len(*fuzzInput) > 1
	}
	var has1More = func(fuzzInput *[]byte) bool {
		return len(*fuzzInput) > 0
	}
	var getAction = func(fuzzInput *[]byte, i *int) byte {
		val := (*fuzzInput)[0]
		*fuzzInput = (*fuzzInput)[1:]
		*i += 1
		return val % 4
	}
	var getCount = func(fuzzInput *[]byte, i *int) byte {
		val := (*fuzzInput)[0]
		*fuzzInput = (*fuzzInput)[1:]
		*i += 1
		return val % 16
	}
	var getOneVal = func(fuzzInput *[]byte, i *int) byte {
		val := (*fuzzInput)[0]
		*fuzzInput = (*fuzzInput)[1:]
		*i += 1
		return val
	}
	var getManyVals = func(fuzzInput *[]byte, i *int, n byte) []byte {
		nn := min(int(n), len(*fuzzInput))
		vals := (*fuzzInput)[0:nn]
		*fuzzInput = (*fuzzInput)[nn:]
		*i += nn
		return vals
	}
	var queueOne = func(queue *RingQueue[byte], list *[]byte, val byte) {
		queue.Queue(val)
		*list = append(*list, val)
	}
	var queueMany = func(queue *RingQueue[byte], list *[]byte, vals []byte) {
		queue.QueueMany(vals...)
		*list = append(*list, vals...)
	}
	var dequeueOne = func(queue *RingQueue[byte], list *[]byte) (valQ, valL byte, okQ, okL bool) {
		valQ, okQ = queue.Dequeue()
		okL = len(*list) > 0
		if okL {
			valL = (*list)[0]
			*list = slices.Delete(*list, 0, 1)
		}
		return
	}
	var dequeueMany = func(queue *RingQueue[byte], list *[]byte, n byte) (valsQ, valsL []byte) {
		qnn := min(len(queue.data), int(n))
		valsQ = queue.DequeueMany(qnn)
		lnn := min(len(*list), int(n))
		valsL = make([]byte, lnn)
		copy(valsL, (*list)[:lnn])
		*list = slices.Delete(*list, 0, lnn)
		return
	}
	var sameState = func(queue RingQueue[byte], list []byte) bool {
		if len(list) != len(queue.data) {
			return false
		}
		qdata := queue.GetDataSlices()
		lenC1 := len(qdata[0])
		if !slices.Equal(qdata[0], list[:lenC1]) {
			return false
		}
		return slices.Equal(qdata[1], list[lenC1:])
	}
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	f.Add([]byte{36, 164, 232, 39, 162, 63, 33, 72, 161, 107, 237, 17, 218, 19, 106, 23, 134, 188, 83, 152})
	f.Add([]byte{94, 40, 218, 244, 100, 100, 162, 41, 37, 68, 12, 242, 208, 86, 123, 246, 180, 50, 82, 220})
	f.Add([]byte{235, 53, 55, 190, 202, 178, 84, 57, 76, 92, 131, 66, 210, 93, 138, 179, 241, 58, 67, 196})
	f.Fuzz(func(t *testing.T, a []byte) {
		if len(a) < 3 {
			return
		}
		a = a[:min(64, len(a))]
		aa := a
		i := 0
		list := make([]byte, 0, 10)
		queue := New[byte](0)
		for has1More(&aa) {
			ac := getAction(&aa, &i)
			switch ac {
			case ACTION_QUEUE_ONE:
				if !has1More(&aa) {
					return
				}
				val := getOneVal(&aa, &i)
				queueOne(&queue, &list, val)
				if !sameState(queue, list) {
					qdata := queue.GetDataSlices()
					t.Errorf("\ncase failed: RingQueue[byte].Queue():\nEXP: %v\nGOT: %v%v\nCASE: % 3v\nPOS:  %s^\n", list, qdata[0], qdata[1], a, strings.Repeat(" ", i*4))
					return
				}
			case ACTION_QUEUE_MANY:
				if !has2More(&aa) {
					return
				}
				count := getCount(&aa, &i)
				vals := getManyVals(&aa, &i, count)
				queueMany(&queue, &list, vals)
				if !sameState(queue, list) {
					qdata := queue.GetDataSlices()
					t.Errorf("\ncase failed: RingQueue[byte].QueueMany():\nEXP: %v\nGOT: %v%v\nCASE: % 3v\nPOS:  %s^\n", list, qdata[0], qdata[1], a, strings.Repeat(" ", i*4))
					return
				}
			case ACTION_DEQUEUE_ONE:
				if len(list) == 0 {
					continue
				}
				valQ, valL, okQ, okL := dequeueOne(&queue, &list)
				if valQ != valL || okQ != okL {
					t.Errorf("\ncase failed: RingQueue[byte].Dequeue():\nEXP: %d, %v\nGOT: %d, %v\nCASE: % 3v\nPOS:  %s^\n", valL, okL, valQ, okQ, a, strings.Repeat(" ", i*4))
					return
				}
			case ACTION_DEQUEUE_MANY:
				if !has1More(&aa) {
					return
				}
				count := getCount(&aa, &i)
				count = byte(min(int(count), len(list)))
				if count == 0 {
					continue
				}
				valsQ, valsL := dequeueMany(&queue, &list, count)
				if !slices.Equal(valsQ, valsL) {
					t.Errorf("\ncase failed: RingQueue[byte].DequeueMany():\nEXP: %v\nGOT: %v\nCASE: % 3v\nPOS:  %s^\n", valsL, valsQ, a, strings.Repeat(" ", i*4))
					return
				}
			}
		}
	})
}
