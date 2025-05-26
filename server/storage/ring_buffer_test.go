package storage

import (
	"context"
	"github.com/stretchr/testify/assert"
	"math"
	"slices"
	"sync"
	"testing"
)

func TestRingBuffer_Write(t *testing.T) {
	buffer := NewRingBuffer[Log](3)

	// Test Write with valid logs
	logs := []*Log{
		{Message: "Log 1"},
		{Message: "Log 2"},
		{Message: "Log 3"},
		{Message: "Log 4"}, // Overwrites the first log
	}

	for _, log := range logs {
		buffer.Write(log)
	}

	expectedMessages := []string{"Log 4", "Log 2", "Log 3"}
	for i, expected := range expectedMessages {
		assert.Equal(t, expected, buffer.data[i].Message)
	}

	// Test Write with nil log
	assert.Panics(t, func() {
		buffer.Write(nil)
	})
}

func TestRingBuffer_Write_ZeroCapacity(t *testing.T) {
	assert.Panics(t, func() {
		_ = NewRingBuffer[Log](0)
	})
}

func TestRingBuffer_Write_OverwriteWithNil(t *testing.T) {
	buffer := NewRingBuffer[int](2)
	a := 1
	b := 2
	buffer.Write(&a)
	buffer.Write(&b)
	// Overwrite with nil (should panic)
	assert.Panics(t, func() {
		buffer.Write(nil)
	})
}

func TestRingBuffer_Write_And_Overwrite_Int(t *testing.T) {
	buffer := NewRingBuffer[int](2)
	a := 1
	b := 2
	c := 3
	buffer.Write(&a)
	buffer.Write(&b)
	// Overwrite a with c
	buffer.Write(&c)
	el := buffer.Element()
	assert.NotNil(t, el)
	assert.Equal(t, &c, el.Value())
	el = el.Next(0)
	assert.NotNil(t, el)
	assert.Equal(t, &b, el.Value())
	el = el.Next(0)
	assert.Nil(t, el)
}

func TestRingBuffer_Element_Empty(t *testing.T) {
	buffer := NewRingBuffer[int](3)
	assert.Nil(t, buffer.Element())
}

func TestRingBuffer_Element_Empty_Int(t *testing.T) {
	buffer := NewRingBuffer[int](3)
	assert.Nil(t, buffer.Element())
}

func TestRingBuffer_ElementAndListener_ContextCancel(t *testing.T) {
	buffer := NewRingBuffer[int](2)
	ctx, cancel := context.WithCancel(context.Background())
	el, ch := buffer.ElementAndListener(ctx)
	assert.Nil(t, el)
	cancel()
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after context cancel")
}

type testSafeLog struct {
	id      int
	cleanup bool
}

// Implement SafeElement
func (l *testSafeLog) Cleanup() { l.cleanup = true }

func TestRingBuffer_Write_SafeElement(t *testing.T) {
	buffer := NewRingBuffer[testSafeLog](2)

	log1 := &testSafeLog{id: 1}
	log2 := &testSafeLog{id: 2}
	log3 := &testSafeLog{id: 3}

	buffer.Write(log1)
	buffer.Write(log2)
	// Overwrite log1, should call Cleanup on log1
	buffer.Write(log3)

	assert.True(t, log1.cleanup, "Cleanup should be called on log1 when overwritten")
	assert.False(t, log2.cleanup, "Cleanup should not be called on log2 yet")
	assert.False(t, log3.cleanup, "Cleanup should not be called on log3")

	// Overwrite log2, should call Cleanup on log2
	log4 := &testSafeLog{id: 4}
	buffer.Write(log4)
	assert.True(t, log2.cleanup, "Cleanup should be called on log2 when overwritten")
	assert.False(t, log3.cleanup, "Cleanup should not be called on log3")
	assert.False(t, log4.cleanup, "Cleanup should not be called on log4")
}

func TestRingBuffer_Write_SafeElement_Nil(t *testing.T) {
	buffer := NewRingBuffer[testSafeLog](2)
	// Write nil, should panic
	assert.Panics(t, func() {
		buffer.Write(nil)
	})
}

func TestRingBuffer_Write_SafeElement_CleanupOnOverwrite(t *testing.T) {
	buffer := NewRingBuffer[testSafeLog](2)
	log1 := &testSafeLog{id: 1}
	log2 := &testSafeLog{id: 2}
	log3 := &testSafeLog{id: 3}
	buffer.Write(log1)
	buffer.Write(log2)
	buffer.Write(log3)
	assert.True(t, log1.cleanup, "Cleanup should be called on log1 when overwritten")
}

func TestRingBuffer_Get(t *testing.T) {
	buffer := NewRingBuffer[int](3)

	// Write items to the buffer
	items := []int{1, 2, 3}
	for _, log := range items {
		buffer.Write(&log)
	}

	// Create a el and read items
	el := buffer.Element()
	slices.Reverse(items)
	for _, expected := range items {
		assert.NotNil(t, el)
		assert.Equal(t, &expected, el.Value())

		el = el.Next(0)
	}

	// Last one should read out of bounds
	assert.Nil(t, el)
}

func TestRingBuffer_Get_Overwritten(t *testing.T) {
	buffer := NewRingBuffer[int](3)
	el := buffer.Element()
	assert.Nil(t, el)

	items := []int{1, 2, 3, 4}

	for _, item := range items[:2] {
		buffer.Write(&item)
	}

	el = buffer.Element()
	assert.NotNil(t, el)
	assert.Equal(t, &items[1], el.Value())

	for _, item := range items[2:] {
		buffer.Write(&item)
	}

	el = el.Next(0)
	assert.Nil(t, el)
}

func TestRingBuffer_Concurrency(t *testing.T) {
	size := 10000
	buffer := NewRingBuffer[int](uint(10000))
	var wg sync.WaitGroup

	overwrite := 123123123123
	buffer.Write(&overwrite)
	for i := 0; i < size; i++ {
		buffer.Write(&i)
	}

	// Concurrent reads
	reader1 := buffer.Element()
	reader2 := buffer.Element()

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := size - 1; i >= 0; i-- {
			assert.NotNil(t, reader1)
			assert.Equal(t, *reader1.Value(), i, "Expected to read %d", i)
			reader1 = reader1.Next(0)
		}
	}()
	go func() {
		defer wg.Done()
		for i := size - 1; i >= 0; i-- {
			assert.NotNil(t, reader2)
			assert.Equal(t, *reader2.Value(), i, "Expected to read %d", i)
			reader2 = reader2.Next(0)
		}
	}()

	wg.Wait()
}

func TestRingBuffer_Concurrency_Int(t *testing.T) {
	size := 1000
	buffer := NewRingBuffer[int](uint(size))
	var wg sync.WaitGroup

	for i := 0; i < size; i++ {
		buffer.Write(&i)
	}

	reader := buffer.Element()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := size - 1; i >= 0; i-- {
			assert.NotNil(t, reader)
			assert.Equal(t, *reader.Value(), i)
			reader = reader.Next(0)
		}
	}()
	wg.Wait()
}

func TestRingBuffer_NewReaderAndListener(t *testing.T) {
	buffer := NewRingBuffer[int](3)

	// Write logs to the buffer
	items := []int{1, 2, 3}
	for _, item := range items {
		buffer.Write(&item)
	}

	// Create an element and listener channel
	el, listener := buffer.ElementAndListener(t.Context())
	el2, listener2 := buffer.ElementAndListener(t.Context())

	// Verify element reads logs in reverse order
	slices.Reverse(items)
	for _, expected := range items {
		assert.NotNil(t, el)
		assert.NotNil(t, el2)
		assert.Equal(t, expected, *el.Value())
		assert.Equal(t, expected, *el2.Value())
		el = el.Next(0)
		el2 = el2.Next(0)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	var l *int = nil
	var l2 *int = nil

	go func() {
		defer wg.Done()
		l = <-listener
	}()

	go func() {
		defer wg.Done()
		l2 = <-listener2
	}()

	for i := range ListenerBufferSize {
		buffer.Write(&i)
	}

	wg.Wait()

	tmp := 5
	buffer.Write(&tmp)
	assert.NotNil(t, l)
	assert.NotNil(t, l2)
	assert.Equal(t, 0, *l)
	assert.Equal(t, 0, *l2)

	// Now we fill the buffer and have no readers
	// it should close the buffer because the writes would be stale

	for i := range 10 {
		buffer.Write(&i)
	}

	for range ListenerBufferSize {
		<-listener
		<-listener2
	}

	_, ok1 := <-listener
	_, ok2 := <-listener2
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestRingBuffer_Next_WithDifferentOffsets(t *testing.T) {
	buffer := NewRingBuffer[int](5)

	// Write integers 10, 20, 30, 40, 50
	values := []int{10, 20, 30, 40, 50}
	for _, v := range values {
		buffer.Write(&v)
	}

	el := buffer.Element()
	assert.NotNil(t, el)
	assert.Equal(t, 50, *el.Value())

	// Next(0) should return 40
	el1 := el.Next(0)
	assert.NotNil(t, el1)
	assert.Equal(t, 40, *el1.Value())

	tmp := el1.Next(1)
	assert.NotNil(t, tmp)
	assert.Equal(t, 20, *tmp.Value())

	// Next(1) should return 30
	el2 := el.Next(1)
	assert.NotNil(t, el2)
	assert.Equal(t, 30, *el2.Value())

	// Next(2) should return 20
	el3 := el.Next(2)
	assert.NotNil(t, el3)
	assert.Equal(t, 20, *el3.Value())

	// Next(3) should return 10
	el4 := el.Next(3)
	assert.NotNil(t, el4)
	assert.Equal(t, 10, *el4.Value())

	// Next(4) should return nil (out of bounds)
	el5 := el.Next(4)
	assert.Nil(t, el5)
}

func TestLoopAdd64(t *testing.T) {
	maxUint64 := uint64(math.MaxUint64)
	size := uint64(20)

	res := loopAdd64(uint64(19), uint64(2), size)
	assert.Equal(t, uint64(1), res)

	res = loopAdd64(maxUint64, uint64(1), size)
	assert.Equal(t, uint64(16), res)
}
