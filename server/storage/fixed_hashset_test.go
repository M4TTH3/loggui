package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type testHashable struct {
	id  uint64
	val string
}

func (t testHashable) Hash() uint64 { return t.id }

func TestFixedHashSet_Basic(t *testing.T) {
	hs := NewFixedHashSet[testHashable](3)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}
	c := testHashable{id: 3, val: "c"}
	hs.Add(a)
	hs.Add(b)
	hs.Add(c)

	assert.Equal(t, 3, hs.Size())
	assert.NotNil(t, hs.Get(1))
	assert.NotNil(t, hs.Get(2))
	assert.NotNil(t, hs.Get(3))

	// Adding a duplicate should not increase size
	el := hs.Add(a)
	assert.Equal(t, 3, hs.Size())
	assert.Equal(t, a.id, el.Item().id)

	// Adding a new item should evict the oldest (a)
	d := testHashable{id: 4, val: "d"}
	hs.Add(d)
	assert.Equal(t, 3, hs.Size())
	assert.Nil(t, hs.Get(1))
	assert.NotNil(t, hs.Get(2))
	assert.NotNil(t, hs.Get(3))
	assert.NotNil(t, hs.Get(4))
}

func TestFixedHashSet_Remove(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}
	hs.Add(a)
	hs.Add(b)
	assert.Equal(t, 2, hs.Size())
	assert.True(t, hs.Remove(a))
	assert.Equal(t, 1, hs.Size())
	assert.False(t, hs.Remove(a))
	assert.True(t, hs.Remove(b))
	assert.Equal(t, 0, hs.Size())
}

func TestFixedHashSet_Iteration(t *testing.T) {
	hs := NewFixedHashSet[testHashable](3)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}
	c := testHashable{id: 3, val: "c"}
	hs.Add(a)
	hs.Add(b)
	hs.Add(c)

	el := hs.Get(3) // Most recent
	assert.NotNil(t, el)
	assert.Equal(t, uint64(3), el.Item().id)
	el = el.Next()
	assert.NotNil(t, el)
	assert.Equal(t, uint64(2), el.Item().id)
	el = el.Next()
	assert.NotNil(t, el)
	assert.Equal(t, uint64(1), el.Item().id)
	el = el.Next()
	assert.Nil(t, el)
}

func TestFixedHashSet_Add_DuplicateHash(t *testing.T) {
	hs := NewFixedHashSet[testHashable](3)

	a := testHashable{id: 1, val: "a"}
	ax := testHashable{id: 1, val: "ax"} // Same hash as a, different value

	el1 := hs.Add(a)
	assert.Equal(t, a.id, el1.Item().id)
	assert.Equal(t, a.val, el1.Item().val)

	el2 := hs.Add(ax)
	// Should return the previous item (a), not add ax
	assert.Equal(t, a.id, el2.Item().id)
	assert.Equal(t, a.val, el2.Item().val)
	assert.Equal(t, 1, hs.Size())
}

func TestFixedHashSet_Add_EvictOldest(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}
	c := testHashable{id: 3, val: "c"}

	hs.Add(a)
	hs.Add(b)
	assert.Equal(t, 2, hs.Size())

	// Add c, should evict a (the oldest)
	hs.Add(c)
	assert.Equal(t, 2, hs.Size())
	assert.Nil(t, hs.Get(1), "Oldest item should be evicted and Get should return nil")
	assert.NotNil(t, hs.Get(2))
	assert.NotNil(t, hs.Get(3))
}

func TestFixedHashSet_ZeroCapacity(t *testing.T) {
	hs := NewFixedHashSet[testHashable](0)

	a := testHashable{id: 1, val: "a"}

	// Should still be able to add one item
	hs.Add(a)
	assert.Equal(t, 1, hs.Size())
	assert.NotNil(t, hs.Get(1))
}

func TestFixedHashSet_NegativeCapacity(t *testing.T) {
	hs := NewFixedHashSet[testHashable](-5)

	a := testHashable{id: 1, val: "a"}

	// Should still be able to add one item
	hs.Add(a)
	assert.Equal(t, 1, hs.Size())
	assert.NotNil(t, hs.Get(1))
}

func TestFixedHashSet_AddRemoveAdd(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}

	hs.Add(a)
	hs.Add(b)
	assert.Equal(t, 2, hs.Size())
	assert.True(t, hs.Remove(a))
	assert.Equal(t, 1, hs.Size())

	// Add a again, should not evict b
	hs.Add(a)
	assert.Equal(t, 2, hs.Size())
	assert.NotNil(t, hs.Get(1))
	assert.NotNil(t, hs.Get(2))
}

func TestFixedHashSet_RemoveNonexistent(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	a := testHashable{id: 1, val: "a"}
	b := testHashable{id: 2, val: "b"}

	// Remove before adding
	assert.False(t, hs.Remove(a))

	hs.Add(a)
	assert.True(t, hs.Remove(a))
	assert.False(t, hs.Remove(a))

	// Remove something never added
	assert.False(t, hs.Remove(b))
}

func TestFixedHashSet_Iteration_Empty(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	assert.Nil(t, hs.Get(1))
}

func TestFixedHashSet_Iteration_SingleItem(t *testing.T) {
	hs := NewFixedHashSet[testHashable](2)

	a := testHashable{id: 1, val: "a"}

	hs.Add(a)
	el := hs.Get(1)
	assert.NotNil(t, el)
	assert.Equal(t, uint64(1), el.Item().id)
	assert.Nil(t, el.Next())
}
