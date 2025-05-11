package storage

import (
	"errors"
	"fmt"
	"github.com/m4tth3/loggui/core"
	"sync"
	"time"
)

type Log = core.Log

const (
	DEFAULT_RING_MAX_SIZE int = 10_000
)

type StorageContext[T any] interface {
	DataReader() <-chan T // Returns the data reader for the logs
	Count() int           // Returns the number of logs in the context
	sync.Locker
}

// DatabaseCache caches the logs in memory from the database for a short time
// if we exceed the cache size
type DatabaseCache struct {
}

type LogStorageContext struct {
	data   chan *Log
	count  int // how many logs we've read
	cache  *DatabaseCache
	reader BufferReader[Log]
	*sync.Mutex
}

func (s *LogStorageContext) DataReader() <-chan *Log {
	return s.data
}

func (s *LogStorageContext) Count() int {
	s.Lock()
	defer s.Unlock()
	return s.count
}

// StorageManager should allow settings and retrievals of data for the frontend
// client to consume.
//
// Should be a singleton instance that is shared across all requests.
type StorageManager[T any] interface {
	// Read reads the next "num" logs from the storage using the current context
	// and sends it back into the context
	// User should call Read from another goroutine for async processing
	//
	// Mutates the LogStorageContext[T] and adjusts the start and ends
	Read(num int, ctx StorageContext[T], filter filter) error

	// Write writes the log to the storage
	Write(T) error
}

// LogStorageManager is the main storage manager for logs
//
// Implements StorageManager[*Log]
// We can always assume that the buffer cache is always up to date,
// and we can reference the database for historical logs
type LogStorageManager struct {
	size         uint64 // Number of logs in total
	writeChannel chan *Log

	buffer    Buffer[Log]
	writeLock sync.Mutex
}

func NewLogStorageManager(size int) *LogStorageManager {
	if size <= 0 {
		size = DEFAULT_RING_MAX_SIZE
	}

	l := &LogStorageManager{
		size:         uint64(size),
		writeChannel: make(chan *Log, size),
		buffer:       NewRingBuffer[Log](size),
	}

	go l.processWriteChannel()

	return l
}

func (l *LogStorageManager) Read(num int, ctx *LogStorageContext, filter filter) error {
	if num <= 0 {
		return errors.New("num must be greater than 0")
	}

	if ctx == nil {
		return errors.New("log is nil")
	}

	return nil
}

// Write writes the log to the storage. We will store based on date received
// and then use a ring buffer to cache the logs
func (l *LogStorageManager) Write(log *Log) error {
	if log == nil {
		return errors.New("log is nil")
	}

	l.writeLock.Lock()
	defer l.writeLock.Unlock()

	log.RecordedAt = time.Now()
	l.writeChannel <- log

	return nil
}

func (l *LogStorageManager) processWriteChannel() {
	var log *Log
	for {
		log = <-l.writeChannel
		fmt.Print(log)
	}
}
