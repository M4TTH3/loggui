package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type logReader struct {
	count   uint64
	req     chan Chunk
	filter  *Filter
	reader  *RingBuffer[Log]
	manager *LogManager

	once atomic.Int32
}

func (s *logReader) Count() uint64 {
	return s.count
}

func (s *logReader) OpenStream(ctx context.Context) (<-chan *Log, error) {
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

func (s *logReader) RequestChunk(chunk Chunk) error {
	if s.req == nil {
		return errors.New("request channel is nil")
	}

	s.req <- chunk
	return nil
}

func (s *logReader) readChunk(chunk Chunk, filter *Filter, out chan<- *Log) bool {

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

	buffer    *RingBuffer[Log]
	writeLock sync.Mutex
}

func NewLogManager(size uint) Manager[Log] {

	l := &LogManager{
		size:         uint64(size),
		writeChannel: make(chan *Log, size),
		buffer:       NewRingBuffer[Log](size),
	}

	go l.processWriteChannel()

	return l
}

func (l *LogManager) GetReader() Reader[Log] {
	return &logReader{}
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
