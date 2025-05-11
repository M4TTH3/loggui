package storage

import (
	"github.com/m4tth3/loggui/core"
)

// A filter is used to filter logs based on a given criteria
type filter interface {
	Filter(*core.Log) bool
	SqlFilter() string
}
