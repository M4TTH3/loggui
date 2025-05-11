package core

import "time"

type Level int

// Log.Level(s) are defined as follows:
const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level.
// It is used for logging and displaying the log level in the UI.
func (l Level) String() string {
	switch l {
	case TRACE:
		return "trace"
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	case FATAL:
		return "fatal"
	}

	panic("unknown log level")
}

// Log is the main data type sent/received by the server.
//
// Source is an identifier we can label the sending source with.
// Group is an identifier to group related logs together
type Log struct {
	Level Level `json:"level"`

	Source *string `json:"source"`
	Group  *string `json:"group"`

	Message string `json:"message"`

	// MessageJson is a map of kvp from Message if Message is a JSON object
	MessageJson *map[string]any `json:"message_json"`

	RecordedAt time.Time `json:"recorded_at"`

	// We will use this time as the main source of time
	ReceivedAt *time.Time `json:"created_at"`
}
