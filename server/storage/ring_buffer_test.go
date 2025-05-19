package storage

import (
	"github.com/stretchr/testify/assert"
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
