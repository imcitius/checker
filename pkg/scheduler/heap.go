package scheduler

import (
	"time"

	"github.com/imcitius/checker/pkg/models"
)

// CheckItem represents an item in the priority queue
type CheckItem struct {
	CheckDef models.CheckDefinition
	NextRun  time.Time
	Index    int // The index of the item in the heap.
}

// CheckHeap implements a min-heap of CheckItems based on NextRun time
type CheckHeap []*CheckItem

func (h CheckHeap) Len() int { return len(h) }

func (h CheckHeap) Less(i, j int) bool {
	return h[i].NextRun.Before(h[j].NextRun)
}

func (h CheckHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *CheckHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*CheckItem)
	item.Index = n
	*h = append(*h, item)
}

func (h *CheckHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// Peek returns the earliest item without removing it
func (h *CheckHeap) Peek() *CheckItem {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}
