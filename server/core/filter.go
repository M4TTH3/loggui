package core

// A filter is used to filter logs based on a given criteria
type filter interface {
	Filter(*Log) bool
	SqlFilter() string
}
