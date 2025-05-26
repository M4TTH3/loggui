package storage

import (
	"container/list"
	"sync"
)

type Hashable interface {
	Hash() uint64
}

// FixedHashSet is an ordered set with a max capacity. It is an alternative
// to RingBuffer, offering fast element search, however, lacks operations
// like Seek.
type FixedHashSet[T Hashable] struct {
	capacity int
	keys     *list.List
	hashMap  map[uint64]struct {
		item *T
		node *list.Element
	}

	mutex sync.RWMutex
}

func NewFixedHashSet[T Hashable](capacity int) *FixedHashSet[T] {
	if capacity <= 0 {
		capacity = 1 // Ensure at least one item can be stored
	}
	return &FixedHashSet[T]{
		capacity: capacity,
		hashMap: make(map[uint64]struct {
			item *T
			node *list.Element
		}),
		keys: list.New(),
	}
}

func (h *FixedHashSet[T]) Size() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.keys.Len()
}

// Get retrieves an item from the set by its hash. If the item does not
// exist, it returns nil.
func (h *FixedHashSet[T]) Get(hash uint64) *SetElement[T] {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if i, ok := h.hashMap[hash]; ok {
		return newSetElement[T](i.node, h)
	}

	return nil
}

// Add adds an item to the set. If the item already exists, it returns
// the existing item.
func (h *FixedHashSet[T]) Add(item T) *SetElement[T] {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	hash := item.Hash()
	if prev, exists := h.hashMap[hash]; exists {
		return newSetElement[T](prev.node, h) // Item already exists
	}

	if h.keys.Len() >= h.capacity {
		// Remove the oldest item
		if oldest := h.keys.Back(); oldest != nil {
			h.keys.Remove(oldest)
			delete(h.hashMap, oldest.Value.(uint64))
		}
	}

	node := h.keys.PushFront(hash)
	h.hashMap[hash] = struct {
		item *T
		node *list.Element
	}{&item, node}

	return newSetElement[T](node, h)
}

func (h *FixedHashSet[T]) Remove(item T) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	hash := item.Hash()
	if i, ok := h.hashMap[hash]; !ok {
		return false
	} else {
		h.keys.Remove(i.node)
	}

	delete(h.hashMap, hash)

	return true
}

type SetElement[T Hashable] struct {
	item *T
	node *list.Element
	set  *FixedHashSet[T]
}

func newSetElement[T Hashable](node *list.Element, set *FixedHashSet[T]) *SetElement[T] {
	return &SetElement[T]{
		item: set.hashMap[node.Value.(uint64)].item,
		node: node,
		set:  set,
	}
}

func (e *SetElement[T]) Item() *T {
	return e.item
}

func (e *SetElement[T]) Next() *SetElement[T] {
	e.set.mutex.RLock()
	defer e.set.mutex.RUnlock()
	if e.node == nil || e.node.Next() == nil {
		return nil
	}
	return newSetElement[T](e.node.Next(), e.set)
}
