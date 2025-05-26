package database

import "github.com/m4tth3/loggui/core"

// QueryHandler is an interface to abstract operations across
// different databases.
type QueryHandler interface {
	Init() error
	GetLogs(filter *Filter) (chan *core.Log, error)
	WriteLog(log *core.Log) error
}
