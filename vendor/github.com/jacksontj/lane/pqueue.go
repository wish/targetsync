package lane

import (
	"fmt"
	"sync"
)

// PQType represents a priority queue ordering kind (see MAXPQ and MINPQ)
type PQType int

const (
	MAXPQ PQType = iota
	MINPQ
)

type Item struct {
	value    interface{}
	priority int64
	index    uint
}

// PQueue is a heap priority queue data structure implementation.
// It can be whether max or min ordered and it is synchronized
// and is safe for concurrent operations.
type PQueue struct {
	sync.RWMutex
	items      []*Item
	elemsCount uint
	comparator func(int64, int64) bool
}

func newItem(value interface{}, priority int64) *Item {
	return &Item{
		value:    value,
		priority: priority,
	}
}

func (i *Item) String() string {
	return fmt.Sprintf("<item value:%s priority:%d index:%d>", i.value, i.priority, i.index)
}

// NewPQueue creates a new priority queue with the provided pqtype
// ordering type
func NewPQueue(pqType PQType) *PQueue {
	var cmp func(int64, int64) bool

	if pqType == MAXPQ {
		cmp = max
	} else {
		cmp = min
	}

	items := make([]*Item, 1)
	items[0] = nil // Heap queue first element should always be nil

	return &PQueue{
		items:      items,
		elemsCount: 0,
		comparator: cmp,
	}
}

// Push the value item into the priority queue with provided priority.
func (pq *PQueue) Push(value interface{}, priority int64) *Item {
	item := newItem(value, priority)

	pq.Lock()
	item.index = pq.elemsCount + 1
	pq.items = append(pq.items, item)
	pq.elemsCount += 1
	pq.swim(pq.size())
	pq.Unlock()
	return item
}

// Pop and returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *PQueue) Pop() (interface{}, int64) {
	pq.Lock()
	defer pq.Unlock()

	if pq.size() < 1 {
		return nil, 0
	}

	var max *Item = pq.items[1]

	pq.exch(1, pq.size())
	pq.items = pq.items[0:pq.size()]
	pq.elemsCount -= 1
	pq.sink(1)

	return max.value, max.priority
}

// Head returns the highest/lowest priority item (depending on whether
// you're using a MINPQ or MAXPQ) from the priority queue
func (pq *PQueue) Head() (interface{}, int64) {
	pq.RLock()
	defer pq.RUnlock()

	if pq.size() < 1 {
		return nil, 0
	}

	headValue := pq.items[1].value
	headPriority := pq.items[1].priority

	return headValue, headPriority
}

// Size returns the elements present in the priority queue count
func (pq *PQueue) Size() uint {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size()
}

// Check queue is empty
func (pq *PQueue) Empty() bool {
	pq.RLock()
	defer pq.RUnlock()
	return pq.size() == 0
}

func (pq *PQueue) Remove(i *Item) bool {
	pq.Lock()
	defer pq.Unlock()

	if pq.size() < 1 {
		return false
	}

	if pq.size() > i.index+1 {
		pq.items = append(pq.items[:i.index], pq.items[i.index+1:len(pq.items)]...)
	} else {
		pq.items = pq.items[:i.index]
	}
	pq.elemsCount -= 1
	return true
}

func (pq *PQueue) size() uint {
	return pq.elemsCount
}

func max(i, j int64) bool {
	return i < j
}

func min(i, j int64) bool {
	return i > j
}

func (pq *PQueue) less(i, j uint) bool {
	return pq.comparator(pq.items[i].priority, pq.items[j].priority)
}

func (pq *PQueue) exch(i, j uint) {
	var tmpItem *Item = pq.items[i]

	pq.items[i] = pq.items[j]
	pq.items[i].index = i
	pq.items[j] = tmpItem
	pq.items[j].index = j
}

func (pq *PQueue) swim(k uint) {
	for k > 1 && pq.less(k/2, k) {
		pq.exch(k/2, k)
		k = k / 2
	}

}

func (pq *PQueue) sink(k uint) {
	for 2*k <= pq.size() {
		var j uint = 2 * k

		if j < pq.size() && pq.less(j, j+1) {
			j++
		}

		if !pq.less(k, j) {
			break
		}

		pq.exch(k, j)
		k = j
	}
}
