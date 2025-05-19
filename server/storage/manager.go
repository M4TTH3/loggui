package storage

import (
	"context"
	"github.com/m4tth3/loggui/core"
	"github.com/m4tth3/loggui/server/database"
)

type Log = core.Log
type Chunk = uint64
type Filter = database.Filter[Log]

const (
	DefaultRingMaxSize uint = 10_000
)

// Reader is the interface for reading logs from the storage
//
// Receiving actions should only be called once per request
type Reader[T any] interface {
	Count() uint64 // Returns the number of items read in the context

	// OpenStream returns a channel that streams the logs from the storage.
	// Use getNext to get the next chunk of logs.
	//
	// OpenStream will end when the context is done
	OpenStream(context.Context) (out <-chan *T, err error)
	RequestChunk(chunk Chunk) error
}

// Manager should allow settings and retrievals of data for the frontend
// client to consume.
//
// Should be a singleton instance that is shared across all requests.
type Manager[T any] interface {
	GetReader() Reader[T]

	// Write writes the log to the storage
	Write(*T) error
}
