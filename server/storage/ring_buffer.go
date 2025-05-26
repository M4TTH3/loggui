package storage

import (
	"context"
	"math/big"
	"sync"
)

// Ring buffer is a thread-safe circular buffer that allows for efficient
// reading and writing of data. It is a fixed-capacity buffer that offers
// fast Seek operations.
//
// For faster RW without O(1) seeking, use a fixed buffer.

const (
	ListenerBufferSize = 100
)

// SafeElement is an interface that defines a cleanup method for elements
// in the ring buffer. This is useful for elements that need to perform
// cleanup operations when they are pushed out of the buffer.
type SafeElement interface {
	Cleanup()
}

type Element[T any] struct {
	value *T

	pos uint

	// A counter relative to the RingBuffer.counter to determine which write it's on
	counter uint64
	buffer  *RingBuffer[T]
}

func (e *Element[T]) Value() *T {
	return e.value
}

// Next gets the next item with an offset from the current position.
//
// Example: [1, 2, 3] if i = 0, then with offset 0 we get item at index 1
func (e *Element[T]) Next(offset uint) *Element[T] {
	offset++
	if uint64(offset) > e.counter {
		// no possible item to read
		return nil
	}

	e.buffer.mutex.RLock()
	defer e.buffer.mutex.RUnlock()

	nextCounter := e.counter - uint64(offset)
	itemPos := loopSubtract(e.pos, offset, e.buffer.Capacity())
	item := e.buffer.data[itemPos]

	newEl := &Element[T]{
		value:   item,
		pos:     itemPos,
		counter: nextCounter,
		buffer:  e.buffer,
	}

	if !newEl.Valid() {
		return nil
	}

	return newEl
}

// Valid checks if the element is valid (inside the buffer)
func (e *Element[T]) Valid() bool {
	return e != nil && e.value != nil && e.buffer.counter-e.counter < uint64(e.buffer.Capacity())
}

func newElement[T any](b *RingBuffer[T]) *Element[T] {
	if b.counter == 0 {
		// no writes ever written
		return nil
	}

	pos := loopSubtract(b.index, 1, b.Capacity())

	return &Element[T]{
		value:   b.data[pos],
		pos:     pos,
		counter: b.counter,
		buffer:  b,
	}
}

type listener[T any] struct {
	c      chan<- *T
	cancel context.CancelFunc
}

type RingBuffer[T any] struct {
	data     []*T
	capacity uint

	// Protected by RWMutex

	index   uint
	counter uint64 // counter of number of writes

	// prependBefore to prepend an item (end -> beginning when space is nil)
	prependBefore uint

	// listeners is a map of BufferListener to their channels
	listeners sync.Map

	mutex sync.RWMutex
}

func NewRingBuffer[T any](size uint) *RingBuffer[T] {
	if size == 0 {
		panic("ring buffer size must be > 0")
	}

	return &RingBuffer[T]{
		data:          make([]*T, size),
		capacity:      size,
		index:         0,
		mutex:         sync.RWMutex{},
		counter:       0,
		listeners:     sync.Map{},
		prependBefore: size,
	}
}

func (l *RingBuffer[T]) Index() uint {
	return l.index
}

func (l *RingBuffer[T]) Capacity() uint {
	return l.capacity
}

func (l *RingBuffer[T]) Element() *Element[T] {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return newElement(l)
}

// ElementAndListener returns the current element and a buffered channel with the same capacity
//
// If the buffer is full and there is no active readers, it will be closed
func (l *RingBuffer[T]) ElementAndListener(ctx context.Context) (*Element[T], <-chan *T) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	newCtx, cancel := context.WithCancel(ctx)
	c := make(chan *T, ListenerBufferSize)
	l.listeners.Store(c, listener[T]{
		c:      c,
		cancel: cancel,
	})

	go func() {
		<-newCtx.Done()
		l.listeners.Delete(c)
		close(c)
	}()

	return newElement(l), c
}

func (l *RingBuffer[T]) Write(item *T) {
	if item == nil {
		panic("item cannot be nil")
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	prev := l.data[l.index]
	if safeEl, ok := any(prev).(SafeElement); prev != nil && ok {
		defer safeEl.Cleanup()
	}

	l.data[l.index] = item
	l.index = loopAdd(l.index, 1, l.Capacity())
	l.counter++

	l.listeners.Range(func(key, value any) bool {
		if v, ok := value.(listener[T]); ok {
			select {
			case v.c <- item:
			default:
				{
					// Stopped listening and buffer is full
					l.listeners.Delete(key)
					v.cancel()
				}
			}
		} else {
			panic("listener channel is not a channel")
		}

		return true
	})
}

// WriteLastEmpty is will insert if space (not empty) a previous index to
// allow older writes to be "prepended". It will NOT overwrite newer items.
func (l *RingBuffer[T]) WriteLastEmpty(item *T) bool {
	if item == nil || l.prependBefore == 0 {
		return false
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.prependBefore--

	if l.data[l.prependBefore] != nil {
		// Make it a bad state
		l.prependBefore = 0
		return false
	}

	l.data[l.index] = item
	return true
}

func loopAdd64(a, b, size uint64) uint64 {
	bigA := big.NewInt(0).SetUint64(a)
	bigB := big.NewInt(0).SetUint64(b)
	bigSize := big.NewInt(0).SetUint64(size)

	bigA = bigA.Add(bigA, bigB)

	return bigA.Mod(bigA, bigSize).Uint64()
}

func loopSubtract64(a, b, size uint64) uint64 {
	if a < b {
		return size - (b - a)
	}

	return a - b
}

func loopSubtract(a, b, size uint) uint {
	return uint(loopSubtract64(uint64(a), uint64(b), uint64(size)))
}

func loopAdd(a, b, size uint) uint {
	return uint(loopAdd64(uint64(a), uint64(b), uint64(size)))
}
