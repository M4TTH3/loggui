package storage

import (
	"context"
	"math/big"
	"sync"
)

// Ring buffer is a thread-safe circular buffer that allows for efficient
// reading and writing of data. It is a fixed-size buffer that offers
// fast Seek operations.
//
// For faster RW without O(1) seeking, use a fixed buffer.

const (
	ListenerBufferSize = 100
)

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

	e.buffer.rw.RLock()
	defer e.buffer.rw.RUnlock()

	nextCounter := e.counter - uint64(offset)
	itemPos := loopSubtract(e.pos, offset, e.buffer.Size())
	item := e.buffer.data[itemPos]

	if e.buffer.counter-nextCounter >= uint64(e.buffer.Size()) || item == nil {
		return nil
	}

	return &Element[T]{
		value:   item,
		pos:     itemPos,
		counter: nextCounter,
		buffer:  e.buffer,
	}
}

func newElement[T any](b *RingBuffer[T]) *Element[T] {
	if b.counter == 0 {
		// no writes ever written
		return nil
	}

	pos := loopSubtract(b.index, 1, b.Size())

	return &Element[T]{
		value:   b.data[pos],
		pos:     pos,
		counter: b.counter,
		buffer:  b,
	}
}

type RingBuffer[T any] struct {
	data []*T
	size uint

	// Protected by RWMutex

	index   uint
	counter uint64 // counter of number of writes

	// prependBefore to prepend an item (end -> beginning when space is nil)
	prependBefore uint

	// listeners is a map of BufferListener to their channels
	listeners sync.Map

	rw sync.RWMutex
}

func NewRingBuffer[T any](size uint) *RingBuffer[T] {
	return &RingBuffer[T]{
		data:          make([]*T, size),
		size:          size,
		index:         0,
		rw:            sync.RWMutex{},
		counter:       0,
		listeners:     sync.Map{},
		prependBefore: size,
	}
}

func (l *RingBuffer[T]) Index() uint {
	return l.index
}

func (l *RingBuffer[T]) Size() uint {
	return l.size
}

func (l *RingBuffer[T]) Element() *Element[T] {
	l.rw.RLock()
	defer l.rw.RUnlock()

	return newElement(l)
}

// ElementAndListener returns the current element and a buffered channel with the same size
//
// If the buffer is full and there is no active readers, it will be closed
func (l *RingBuffer[T]) ElementAndListener(ctx context.Context) (*Element[T], <-chan *T) {
	l.rw.RLock()
	defer l.rw.RUnlock()

	c := make(chan *T, ListenerBufferSize)
	l.listeners.Store(c, (chan<- *T)(c))

	go func() {
		<-ctx.Done()
		l.listeners.Delete(c)
	}()

	return newElement(l), c
}

func (l *RingBuffer[T]) Write(item *T) {
	if item == nil {
		panic("item cannot be nil")
	}

	l.rw.Lock()
	defer l.rw.Unlock()

	l.data[l.index] = item
	l.index = loopAdd(l.index, 1, l.Size())
	l.counter++

	l.listeners.Range(func(key, value any) bool {
		if c, ok := value.(chan<- *T); ok {
			select {
			case c <- item:
			default:
				{
					// Stopped listening and buffer is full
					l.listeners.Delete(key)
					close(c)
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

	l.rw.Lock()
	defer l.rw.Unlock()

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
	bigA := big.NewInt(int64(a))
	bigB := big.NewInt(int64(b))
	bigSize := big.NewInt(int64(size))

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
