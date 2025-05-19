package database

import (
	"github.com/m4tth3/loggui/core"
)

// Filter is used to filter logs based on a given criteria
type Filter[T any] interface {
	Filter(*T) bool
	Sql() string
}

type LogVarFilter interface {
	Filter(*core.Log) bool

	// SqlCondition returns the SQL condition for the filter
	// i.e. "field > value"
	SqlCondition() string
}

type LogFilter struct {
	varFilters []LogVarFilter
}

func (f *LogFilter) Filter(log *core.Log) bool {
	for _, filter := range f.varFilters {
		if !filter.Filter(log) {
			return false
		}
	}
	return true
}

func (f *LogFilter) Sql() string {
	sql := "WHERE "

	for _, filter := range f.varFilters {
		if sql != "" {
			sql += " AND "
		}
		sql += filter.SqlCondition()
	}
	return sql
}
