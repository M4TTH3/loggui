package storage

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestLogRingBuffer_Write(t *testing.T) {
	buffer := NewRingBuffer[Log](3)

	// Test Write with valid logs
	logs := []*Log{
		{Message: "Log 1"},
		{Message: "Log 2"},
		{Message: "Log 3"},
		{Message: "Log 4"}, // Overwrites the first log
	}

	for _, log := range logs {
		err := buffer.Write(log)
		assert.NoError(t, err, "Expected no error when writing a valid log")
	}

	expectedMessages := []string{"Log 4", "Log 2", "Log 3"}
	for i, expected := range expectedMessages {
		assert.Equal(t, expected, buffer.data[i].Message, "Unexpected log message at index %d", i)
	}

	// Test Write with nil log
	err := buffer.Write(nil)
	assert.Error(t, err, "Expected an error when writing a nil log")
}

func TestLogRingBuffer_Get(t *testing.T) {
	buffer := NewRingBuffer[Log](3)

	// Write logs to the buffer
	logs := []*Log{
		{Message: "Log 1"},
		{Message: "Log 2"},
		{Message: "Log 3"},
	}
	for _, log := range logs {
		err := buffer.Write(log)
		assert.NoError(t, err, "Expected no error when writing a valid log")
	}

	// Create a reader and read logs
	reader := buffer.NewReader()

	expectedMessages := []string{"Log 3", "Log 2", "Log 1"}
	for _, expected := range expectedMessages {
		item, valid := reader.Get()
		assert.True(t, valid, "Expected valid log when reading")
		assert.NotNil(t, item, "Expected non-nil log when reading")
		assert.Equal(t, expected, item.Message, "Unexpected log message")
	}

	// Attempt to read out of bounds
	item, valid := reader.Get()
	assert.False(t, valid, "Expected invalid log when reading out of bounds")
	assert.Nil(t, item, "Expected log to be nil when reading out of bounds")
}

func TestLogRingBuffer_Get_Overwritten(t *testing.T) {
	buffer := NewRingBuffer[Log](3)
	reader := buffer.NewReader()

	// Attempt to read from an empty buffer
	item, valid := reader.Get()
	assert.False(t, valid, "Expected invalid log when reading an empty buffer")
	assert.Nil(t, item, "Expected log to be nil when reading an empty buffer")

	logs := []*Log{
		{Message: "Log 1"},
		{Message: "Log 2"},
	}

	for _, log := range logs {
		err := buffer.Write(log)
		assert.NoError(t, err, "Expected no error when writing a valid log")
	}

	reader = buffer.NewReader()
	item, valid = reader.Get()
	assert.True(t, valid, "Expected valid log when reading")
	assert.NotNil(t, item, "Expected non-nil log when reading")

	logs = []*Log{
		{Message: "Log 3"},
		{Message: "Log 4"},
	}

	for _, log := range logs {
		err := buffer.Write(log)
		assert.NoError(t, err, "Expected no error when writing a valid log")
	}

	item, valid = reader.Get()

	assert.False(t, valid, "Expected invalid log when reading an overwritten log")
	assert.Nil(t, item, "Expected log to be nil when reading an overwritten log")
}

func TestLogRingBuffer_Concurrency(t *testing.T) {
	buffer := NewRingBuffer[Log](5)
	var wg sync.WaitGroup

	// Concurrent writes
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			err := buffer.Write(&Log{Message: "Writer 1 - Log " + string(rune(i))})
			assert.NoError(t, err, "Expected no error during concurrent writes")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			err := buffer.Write(&Log{Message: "Writer 2 - Log " + string(rune(i))})
			assert.NoError(t, err, "Expected no error during concurrent writes")
		}
	}()
	wg.Wait()

	// Concurrent reads
	reader1 := buffer.NewReader()
	reader2 := buffer.NewReader()

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			item, valid := reader1.Get()
			if valid {
				assert.NotNil(t, item, "Expected non-nil log during concurrent reads")
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			item, valid := reader2.Get()
			if valid {
				assert.NotNil(t, item, "Expected non-nil log during concurrent reads")
			}
		}
	}()
	wg.Wait()
}

func TestLogRingBuffer_NewReaderAndListener(t *testing.T) {
	buffer := NewRingBuffer[Log](3)

	// Write logs to the buffer
	logs := []*Log{
		{Message: "Log 1"},
		{Message: "Log 2"},
		{Message: "Log 3"},
	}
	for _, log := range logs {
		err := buffer.Write(log)
		assert.NoError(t, err, "Expected no error when writing a valid log")
	}

	// Create a reader and listener
	reader, listener := buffer.NewReaderAndListener()

	// Verify reader reads logs in reverse order
	expectedMessages := []string{"Log 3", "Log 2", "Log 1"}
	for _, expected := range expectedMessages {
		item, valid := reader.Get()
		assert.True(t, valid, "Expected valid log when reading")
		assert.NotNil(t, item, "Expected non-nil log when reading")
		assert.Equal(t, expected, item.Message, "Unexpected log message")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	var l *Log = nil

	go func() {
		defer wg.Done()
		l = <-listener.Listen()
	}()

	err := buffer.Write(&Log{Message: "Log 1"})
	assert.NoError(t, err, "Expected no error when writing a valid log")
	wg.Wait()
	assert.NotNil(t, l, "Expected non-nil log from listener")
	assert.Equal(t, "Log 1", l.Message, "Unexpected log message from listener")
}
