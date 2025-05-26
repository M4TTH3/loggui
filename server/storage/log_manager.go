package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/m4tth3/loggui/core"
	"github.com/m4tth3/loggui/server/database"
	"sync"
	"sync/atomic"
	"time"
)

type Log = core.Log
type Chunk = uint64
type Filter = database.Filter

const (
	CacheSize = 50
)

type filterCache struct {
	filter *Filter
	cache  *RingBuffer[Log]
}

type LogReader struct {
	count   uint64
	req     chan Chunk
	filter  *Filter
	manager *LogManager

	once atomic.Int32
}

func (s *LogReader) Count() uint64 {
	return s.count
}

func (s *LogReader) OpenStream(ctx context.Context) (<-chan *Log, error) {
	if !s.once.CompareAndSwap(0, 1) {
		return nil, errors.New("stream already started")
	}

	out := make(chan *Log)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case chunk := <-s.req:
				if !s.readChunk(chunk, s.filter, out) {
					return
				}
			}
		}
	}()

	return out, nil
}

func (s *LogReader) RequestChunk(chunk Chunk) {
	if s.req == nil {
		panic("request channel is nil")
	}

	s.req <- chunk
}

func (s *LogReader) readChunk(chunk Chunk, filter *Filter, out chan<- *Log) bool {
	// First attempt to find the cache. Note cache should be small
	for el := s.manager.caches.Element(); el != nil; el = el.Next(0) {
		if filter.Equal(el.Value().filter) {

		}
	}

	return true
}

// LogManager is the main storage manager for logs
//
// Implements Manager[*Log]
// We can always assume that the buffer Cache is always up to date,
// and we can reference the database for historical logs
type LogManager struct {
	size         uint64 // Number of logs in total
	writeChannel chan *Log

	caches    *RingBuffer[filterCache]
	buffer    *RingBuffer[Log]
	writeLock sync.Mutex
}

func NewLogManager(size uint) *LogManager {

	l := &LogManager{
		size:         uint64(size),
		writeChannel: make(chan *Log, size),
		buffer:       NewRingBuffer[Log](size),
	}

	go l.processWriteChannel()

	return l
}

func (l *LogManager) GetReader() *LogReader {
	return &LogReader{}
}

// Write writes the log to the storage. We will store based on date received
// and then use a ring buffer to Cache the logs
func (l *LogManager) Write(log *Log) error {
	if log == nil {
		return errors.New("log is nil")
	}

	l.writeLock.Lock()
	defer l.writeLock.Unlock()

	log.RecordedAt = time.Now()
	l.writeChannel <- log

	return nil
}

func (l *LogManager) processWriteChannel() {
	var log *Log
	for {
		log = <-l.writeChannel
		fmt.Print(log)
	}
}
