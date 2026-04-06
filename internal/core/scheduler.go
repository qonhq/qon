package core

import (
	"container/heap"
	"sync"
)

type priorityScheduler struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  taskHeap
	closed bool
}

func newPriorityScheduler() *priorityScheduler {
	ps := &priorityScheduler{}
	ps.cond = sync.NewCond(&ps.mu)
	heap.Init(&ps.queue)
	return ps
}

func (ps *priorityScheduler) Submit(task RequestTask) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	heap.Push(&ps.queue, task)
	ps.cond.Signal()
}

func (ps *priorityScheduler) Next() (RequestTask, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for ps.queue.Len() == 0 && !ps.closed {
		ps.cond.Wait()
	}
	if ps.closed {
		return RequestTask{}, false
	}
	task := heap.Pop(&ps.queue).(RequestTask)
	return task, true
}

func (ps *priorityScheduler) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.closed = true
	ps.cond.Broadcast()
}

type taskHeap []RequestTask

func (h taskHeap) Len() int { return len(h) }
func (h taskHeap) Less(i, j int) bool {
	return h[i].Request.Priority > h[j].Request.Priority
}
func (h taskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *taskHeap) Push(x any)   { *h = append(*h, x.(RequestTask)) }
func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
