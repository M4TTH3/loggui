package core

import (
	"errors"
	"sync"
)

// BufferReader allows the buffer to read from an index and shift next.
// It is thread safe and can be used by multiple threads to read from the buffer.
type BufferReader[T any] interface {
	Position() int

	// Get returns the next item in the buffer and shifts the index
	// Returns the next item and whether it's valid
	Get() (*T, bool)
}

type BufferListener[T any] interface {
	Listen() <-chan *T
	Close()
}

type Buffer[T any] interface {
	Index() int   // return the index of the next position to insert into
	MaxSize() int // return the max size of the buffer
	NewReader() BufferReader[T]

	// NewReaderAndListener returns a new reader and a channel to listen to new writes
	NewReaderAndListener() (BufferReader[T], BufferListener[T])

	Write(*T) error // write to the buffer
}

type logBufferReader struct {
	pos     int
	counter int64 // counter relative to the buffer
	valid   bool
	buffer  *LogRingBuffer
	m       *sync.Mutex
}

func (r *logBufferReader) Position() int {
	r.m.Lock()
	defer r.m.Unlock()
	return r.pos
}

// Get returns the next item in the buffer and shifts the index
// if the item is out of bounds, it returns an error
// if the next item is overwritten, it returns nil Log
func (r *logBufferReader) Get() (*Log, bool) {
	if r.buffer == nil {
		return nil, false
	}

	r.m.Lock()
	r.buffer.RLock()

	defer r.m.Unlock()
	defer r.buffer.RUnlock()

	item := r.buffer.data[r.pos]

	if r.buffer.counter-r.counter >= int64(r.buffer.maxSize) || item == nil {
		// Not an error, just out of range
		r.valid = false
		return nil, false
	}

	r.pos = posMod(r.pos-1, r.buffer.maxSize)
	r.counter--

	return item, true
}

func newLogBufferReader(buffer *LogRingBuffer) *logBufferReader {
	return &logBufferReader{
		pos:     posMod(buffer.index-1, buffer.maxSize),
		counter: buffer.counter,
		valid:   true,
		buffer:  buffer,
		m:       &sync.Mutex{},
	}
}

type logBufferListener struct {
	c      chan *Log
	buffer *LogRingBuffer
}

func (l *logBufferListener) Listen() <-chan *Log {
	return l.c
}

func (l *logBufferListener) Close() {
	close(l.c)
	l.buffer.listeners.Delete(l)
}

// newLogBufferListener creates a new logBufferListener.
func newLogBufferListener(buffer *LogRingBuffer) *logBufferListener {
	c := make(chan *Log, buffer.maxSize)
	listener := &logBufferListener{
		c:      c,
		buffer: buffer,
	}

	buffer.listeners.Store(listener, (chan<- *Log)(c))
	return listener
}

type LogRingBuffer struct {
	maxSize int
	data    []*Log

	// Protected by RWMutex
	index   int
	counter int64 // counter of number of writes

	// listeners is a map of BufferListener to their channels
	listeners *sync.Map

	*sync.RWMutex
}

func NewLogRingBuffer(maxSize int) *LogRingBuffer {
	return &LogRingBuffer{
		maxSize:   maxSize,
		data:      make([]*Log, maxSize),
		index:     0,
		RWMutex:   &sync.RWMutex{},
		counter:   0,
		listeners: &sync.Map{},
	}
}

func (l *LogRingBuffer) Index() int {
	return l.index
}

func (l *LogRingBuffer) MaxSize() int {
	return l.maxSize
}

func (l *LogRingBuffer) NewReader() BufferReader[Log] {
	l.RLock()
	defer l.RUnlock()

	return newLogBufferReader(l)
}

func (l *LogRingBuffer) NewReaderAndListener() (BufferReader[Log], BufferListener[Log]) {
	l.RLock()
	defer l.RUnlock()

	return newLogBufferReader(l), newLogBufferListener(l)
}

func (l *LogRingBuffer) Write(log *Log) error {
	l.Lock()
	defer l.Unlock()

	if log == nil {
		return errors.New("log cannot be nil")
	}

	l.data[l.index] = log
	l.index = (l.index + 1) % l.maxSize
	l.counter++

	l.listeners.Range(func(key, value any) bool {
		if c, ok := value.(chan<- *Log); ok {
			c <- log
		} else {
			return false
		}
		return true
	})

	return nil
}

func posMod(a, b int) int {
	return ((a % b) + b) % b
}
