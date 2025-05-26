package database

import (
	"github.com/m4tth3/loggui/core"
	"regexp"
	"strings"
	"time"
)

type FieldFilter[T comparable] struct {
	Le *T
	Ge *T
	Eq *T

	// Currently not supported
	Ne *T
}

func (f *FieldFilter[T]) Equal(other *FieldFilter[T]) bool {
	if f == other {
		return true
	}

	if f == nil || other == nil {
		return false
	}

	if !isValid(
		compare(f.Eq, other.Eq),
		compare(f.Ne, other.Ne),
		compare(f.Le, other.Le),
		compare(f.Ge, other.Ge),
	) {
		return false
	}

	return true
}

func NewLevelFilter(eq *core.Level) *FieldFilter[core.Level] {
	return &FieldFilter[core.Level]{Eq: eq}
}

func NewStringFilter(eq *string) *FieldFilter[string] {
	return &FieldFilter[string]{Eq: eq}
}

func NewTimeFilter(eq, le, ge *time.Time) *FieldFilter[time.Time] {
	return &FieldFilter[time.Time]{Eq: eq, Le: le, Ge: ge}
}

type Filter struct {
	Level      *FieldFilter[core.Level]
	Source     *FieldFilter[string]
	Group      *FieldFilter[string]
	Message    *FieldFilter[string]
	ReceivedAt *FieldFilter[time.Time]
}

func (f *Filter) IsEmpty() bool {
	return f.Level == nil && f.Source == nil && f.Group == nil && f.Message == nil && f.ReceivedAt == nil
}

func (f *Filter) Equal(other *Filter) bool {
	if f == other {
		return true
	}

	if !isValid(
		f.Level.Equal(other.Level),
		f.Source.Equal(other.Source),
		f.Group.Equal(other.Group),
		f.Message.Equal(other.Message),
		f.ReceivedAt.Equal(other.ReceivedAt),
	) {
		return false
	}

	return true
}

func (f *Filter) Filter(log *core.Log) bool {
	if !isValid(
		ifField(f.Level, func() bool {
			return *f.Level.Eq == log.Level
		}),
		ifField(f.Source, log.Source, func() bool {
			return strings.Contains(*log.Source, *f.Source.Eq)
		}),
		ifField(f.Group, log.Group, func() bool {
			return strings.Contains(*log.Group, *f.Group.Eq)
		}),
		ifField(f.Message, func() bool {
			ok, err := regexp.MatchString(*f.Message.Eq, log.Message)
			if err != nil {
				panic(err)
			}

			return ok
		}),
		ifField(f.ReceivedAt, log.ReceivedAt, func() bool {
			switch {
			case f.ReceivedAt.Eq != nil:
				return log.RecordedAt.Equal(*f.ReceivedAt.Eq)
			case f.ReceivedAt.Le != nil && f.ReceivedAt.Ge != nil:
				return (log.RecordedAt.Before(*f.ReceivedAt.Le) || log.RecordedAt.Equal(*f.ReceivedAt.Le)) &&
					(log.RecordedAt.After(*f.ReceivedAt.Ge) || log.RecordedAt.Equal(*f.ReceivedAt.Ge))
			case f.ReceivedAt.Le != nil:
				return log.RecordedAt.Before(*f.ReceivedAt.Le) || log.RecordedAt.Equal(*f.ReceivedAt.Le)
			case f.ReceivedAt.Ge != nil:
				return log.RecordedAt.After(*f.ReceivedAt.Ge) || log.RecordedAt.Equal(*f.ReceivedAt.Ge)
			default:
				panic("ReceivedAt filter is not set")
			}
		}),
	) {
		return false
	}

	return true
}

func compare[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ifField checks if the field is not nil. If it is, it returns true.
// If the field is not nil, it checks if all provided values are valid.
func ifField[T any](field *T, values ...any) bool {
	switch field {
	case nil:
		return true
	default:
		return isValid(values...)
	}
}

// isValid checks if all provided fields are not nil, if they are of type bool,
// that they are true. If it's type func() bool, it calls the function and checks
// if it returns true.
func isValid(values ...any) bool {
	for _, v := range values {
		switch t := v.(type) {
		case bool:
			if !t {
				return false
			}
		case func() bool:
			if !t() {
				return false
			}
		case *string:
			if t == nil {
				return false
			}
		case *core.Level:
			if t == nil {
				return false
			}
		case *time.Time:
			if t == nil {
				return false
			}
		}
	}

	return true
}
