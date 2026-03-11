package scheduler

import (
	"container/heap"
	"testing"
	"time"

	"checker/internal/models"
)

func TestCheckHeap(t *testing.T) {
	h := &CheckHeap{}
	heap.Init(h)

	now := time.Now()

	// Push items in random order
	item1 := &CheckItem{
		CheckDef: models.CheckDefinition{UUID: "1", Name: "First"},
		NextRun:  now.Add(10 * time.Minute),
	}
	item2 := &CheckItem{
		CheckDef: models.CheckDefinition{UUID: "2", Name: "Second"},
		NextRun:  now.Add(1 * time.Minute),
	}
	item3 := &CheckItem{
		CheckDef: models.CheckDefinition{UUID: "3", Name: "Third"},
		NextRun:  now.Add(5 * time.Minute),
	}

	heap.Push(h, item1)
	heap.Push(h, item2)
	heap.Push(h, item3)

	if h.Len() != 3 {
		t.Errorf("Expected len 3, got %d", h.Len())
	}

	// Pop items, expect them in order of NextRun (earliest first)

	// Expect item2 (1 min)
	p1 := heap.Pop(h).(*CheckItem)
	if p1.CheckDef.UUID != "2" {
		t.Errorf("Expected UUID 2, got %s", p1.CheckDef.UUID)
	}

	// Expect item3 (5 min)
	p2 := heap.Pop(h).(*CheckItem)
	if p2.CheckDef.UUID != "3" {
		t.Errorf("Expected UUID 3, got %s", p2.CheckDef.UUID)
	}

	// Expect item1 (10 min)
	p3 := heap.Pop(h).(*CheckItem)
	if p3.CheckDef.UUID != "1" {
		t.Errorf("Expected UUID 1, got %s", p3.CheckDef.UUID)
	}
}

func TestCheckHeap_Peek(t *testing.T) {
	h := &CheckHeap{}
	heap.Init(h)

	if h.Peek() != nil {
		t.Error("Expected nil peek on empty heap")
	}

	item := &CheckItem{
		NextRun: time.Now(),
	}
	heap.Push(h, item)

	if h.Peek() != item {
		t.Error("Peek did not return expected item")
	}
}
