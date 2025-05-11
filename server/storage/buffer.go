package storage

import (
	"errors"
	"sync"
)

// BufferReader allows the buffer to read from an index and shift next.
// It is thread safe and can be used by multiple threads to read from the buffer.
type BufferReader[T any] interface {
	Position() int

	// Get returns the next item in the buffer and shifts the index
	//
	// Returns the next item and whether it's valid
	Get() (*T, bool)
}

// BufferListener allows the buffer to listen for new writes.
// It is thread safe and can be used by multiple threads to listen for new writes.
//
// Must call Close() to close the listener explicitly
type BufferListener[T any] interface {
	Listen() <-chan *T
	Close()
}

type Buffer[T any] interface {
	Index() int   // return the index of the next position to insert into
	MaxSize() int // return the max size of the buffer
	NewReader() BufferReader[T]

	// NewReaderAndListener returns a new reader and a channel to listen to new writes
	// Listener must be Closed explictly
	NewReaderAndListener() (BufferReader[T], BufferListener[T])

	Write(*T) error // write to the buffer
}

type ringBufferReader[T any] struct {
	pos     int
	counter int64 // counter relative to the buffer
	valid   bool
	buffer  *RingBuffer[T]
	m       sync.Mutex
}

func (r *ringBufferReader[T]) Position() int {
	r.m.Lock()
	defer r.m.Unlock()
	return r.pos
}

// Get returns the next item in the buffer and shifts the index
// if the item is out of bounds, it returns an error
// if the next item is overwritten, it returns nil
func (r *ringBufferReader[T]) Get() (*T, bool) {
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

func newRingBufferReader[T any](buffer *RingBuffer[T]) *ringBufferReader[T] {
	return &ringBufferReader[T]{
		pos:     posMod(buffer.index-1, buffer.maxSize),
		counter: buffer.counter,
		valid:   true,
		buffer:  buffer,
		m:       sync.Mutex{},
	}
}

type ringBufferListener[T any] struct {
	c      chan *T
	buffer *RingBuffer[T]
}

func (l *ringBufferListener[T]) Listen() <-chan *T {
	return l.c
}

func (l *ringBufferListener[T]) Close() {
	close(l.c)
	l.buffer.listeners.Delete(l)
}

// newLogBufferListener creates a new ringBufferListener.
func newLogBufferListener[T any](buffer *RingBuffer[T]) *ringBufferListener[T] {
	c := make(chan *T, buffer.maxSize)
	listener := &ringBufferListener[T]{
		c:      c,
		buffer: buffer,
	}

	buffer.listeners.Store(listener, (chan<- *T)(c))
	return listener
}

type RingBuffer[T any] struct {
	maxSize int
	data    []*T

	// Protected by RWMutex
	index   int
	counter int64 // counter of number of writes

	// listeners is a map of BufferListener to their channels
	listeners sync.Map

	sync.RWMutex
}

func NewRingBuffer[T any](maxSize int) *RingBuffer[T] {
	return &RingBuffer[T]{
		maxSize:   maxSize,
		data:      make([]*T, maxSize),
		index:     0,
		RWMutex:   sync.RWMutex{},
		counter:   0,
		listeners: sync.Map{},
	}
}

func (l *RingBuffer[T]) Index() int {
	return l.index
}

func (l *RingBuffer[T]) MaxSize() int {
	return l.maxSize
}

func (l *RingBuffer[T]) NewReader() BufferReader[T] {
	l.RLock()
	defer l.RUnlock()

	return newRingBufferReader[T](l)
}

func (l *RingBuffer[T]) NewReaderAndListener() (BufferReader[T], BufferListener[T]) {
	l.RLock()
	defer l.RUnlock()

	return newRingBufferReader(l), newLogBufferListener(l)
}

func (l *RingBuffer[T]) Write(item *T) error {
	if item == nil {
		return errors.New("item cannot be nil")
	}

	l.Lock()
	defer l.Unlock()

	l.data[l.index] = item
	l.index = (l.index + 1) % l.maxSize
	l.counter++

	l.listeners.Range(func(key, value any) bool {
		if c, ok := value.(chan<- *T); ok {
			c <- item
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
