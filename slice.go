package utils

import "sync"

// ConcurrentSlice type that can be safely shared between goroutines.
type ConcurrentSlice struct {
	sync.RWMutex
	items []interface{}
}

// ConcurrentSliceItem contains the index/value pair of an item in a
// concurrent slice.
type ConcurrentSliceItem struct {
	Index int
	Value interface{}
}

// NewConcurrentSlice creates a new concurrent slice.
func NewConcurrentSlice() *ConcurrentSlice {
	cs := &ConcurrentSlice{
		items: make([]interface{}, 0),
	}

	return cs
}

// Append adds an item to the concurrent slice.
func (cs *ConcurrentSlice) Append(item interface{}) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = append(cs.items, item)
}

// get an index
func (cs *ConcurrentSlice) Get(index int) (item interface{}) {
	cs.RLock()
	defer cs.RUnlock()
	if isset(cs.items, index) {
		return cs.items[index]
	}
	return nil
}

func isset(arr []interface{}, index int) bool {
	return (len(arr) > index)
}

// Iter iterates over the items in the concurrent slice.
// Each item is sent over a channel, so that
// we can iterate over the slice using the builin range keyword.
func (cs *ConcurrentSlice) Iter() <-chan ConcurrentSliceItem {
	c := make(chan ConcurrentSliceItem)
	f := func() {
		cs.RLock()
		defer cs.RUnlock()
		for index, value := range cs.items {
			c <- ConcurrentSliceItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}
