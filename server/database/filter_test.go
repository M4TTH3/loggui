package database

import (
	"github.com/m4tth3/loggui/core"
	"testing"
	"time"
)

func TestNewFieldFilters(t *testing.T) {
	level := core.INFO
	lf := NewLevelFilter(&level)
	if lf.Eq == nil || *lf.Eq != core.INFO {
		t.Errorf("NewLevelFilter did not set Eq correctly")
	}

	str := "source"
	sf := NewStringFilter(&str)
	if sf.Eq == nil || *sf.Eq != "source" {
		t.Errorf("NewStringFilter did not set Eq correctly")
	}

	now := time.Now()
	tf := NewTimeFilter(&now, &now, &now)
	if tf.Eq == nil || tf.Le == nil || tf.Ge == nil {
		t.Errorf("NewTimeFilter did not set fields correctly")
	}
}

func TestFilter_Filter(t *testing.T) {
	level := core.INFO
	source := "app"
	group := "test"
	msg := "hello world"
	badMsg := "bad message"
	now := time.Now()
	before := now.Add(1 * time.Hour)
	after := now.Add(-1 * time.Hour)
	log := &core.Log{
		Level:      level,
		Source:     &source,
		Group:      &group,
		Message:    msg,
		RecordedAt: now,
		ReceivedAt: &now,
	}

	tests := []struct {
		name   string
		filter *Filter
		want   bool
	}{
		{
			name:   "match level",
			filter: &Filter{Level: NewLevelFilter(&level)},
			want:   true,
		},
		{
			name:   "mismatch level",
			filter: &Filter{Level: NewLevelFilter(new(core.Level))},
			want:   false,
		},
		{
			name:   "match source",
			filter: &Filter{Source: NewStringFilter(&source)},
			want:   true,
		},
		{
			name:   "mismatch source",
			filter: &Filter{Source: NewStringFilter(&badMsg)},
			want:   false,
		},
		{
			name:   "match group",
			filter: &Filter{Group: NewStringFilter(&group)},
			want:   true,
		},
		{
			name:   "mismatch group",
			filter: &Filter{Group: NewStringFilter(&badMsg)},
			want:   false,
		},
		{
			name:   "match message regex",
			filter: &Filter{Message: NewStringFilter(&msg)},
			want:   true,
		},
		{
			name:   "mismatch message regex",
			filter: &Filter{Message: NewStringFilter(&badMsg)},
			want:   false,
		},
		{
			name:   "match exact time",
			filter: &Filter{ReceivedAt: NewTimeFilter(&now, nil, nil)},
			want:   true,
		},
		{
			name:   "mismatch exact time",
			filter: &Filter{ReceivedAt: NewTimeFilter(&before, nil, nil)},
			want:   false,
		},
		{
			name:   "match before",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, &before, nil)},
			want:   true,
		},
		{
			name:   "mismatch before",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, &after, nil)},
			want:   false,
		},
		{
			name:   "match after",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, nil, &after)},
			want:   true,
		},
		{
			name:   "mismatch after",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, nil, &before)},
			want:   false,
		},
		{
			name:   "match between",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, &before, &after)},
			want:   true,
		},
		{
			name:   "mismatch between",
			filter: &Filter{ReceivedAt: NewTimeFilter(nil, &after, &before)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Filter(log)
			if got != tt.want {
				t.Errorf("Filter.Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Equal(t *testing.T) {
	level := core.INFO
	otherLevel := core.ERROR
	source := "app"
	otherSource := "other"
	group := "test"
	msg := "hello world"
	now := time.Now()
	before := now.Add(-time.Hour)

	cases := []struct {
		name string
		f1   *Filter
		f2   *Filter
		want bool
	}{
		{
			name: "identical filters (all fields)",
			f1: &Filter{
				Level:      NewLevelFilter(&level),
				Source:     NewStringFilter(&source),
				Group:      NewStringFilter(&group),
				Message:    NewStringFilter(&msg),
				ReceivedAt: NewTimeFilter(&now, &now, &before),
			},
			f2: &Filter{
				Level:      NewLevelFilter(&level),
				Source:     NewStringFilter(&source),
				Group:      NewStringFilter(&group),
				Message:    NewStringFilter(&msg),
				ReceivedAt: NewTimeFilter(&now, &now, &before),
			},
			want: true,
		},
		{
			name: "different level",
			f1:   &Filter{Level: NewLevelFilter(&level)},
			f2:   &Filter{Level: NewLevelFilter(&otherLevel)},
			want: false,
		},
		{
			name: "different source",
			f1:   &Filter{Source: NewStringFilter(&source)},
			f2:   &Filter{Source: NewStringFilter(&otherSource)},
			want: false,
		},
		{
			name: "one nil field",
			f1:   &Filter{Level: NewLevelFilter(&level)},
			f2:   &Filter{},
			want: false,
		},
		{
			name: "both nil fields",
			f1:   &Filter{},
			f2:   &Filter{},
			want: true,
		},
		{
			name: "identical time filters",
			f1:   &Filter{ReceivedAt: NewTimeFilter(&now, &now, &before)},
			f2:   &Filter{ReceivedAt: NewTimeFilter(&now, &now, &before)},
			want: true,
		},
		{
			name: "different time filters",
			f1:   &Filter{ReceivedAt: NewTimeFilter(&now, &now, &before)},
			f2:   &Filter{ReceivedAt: NewTimeFilter(&before, &now, &now)},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f1.Equal(tc.f2)
			if got != tc.want {
				t.Errorf("Filter.Equal() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilter_Equal_AllParams(t *testing.T) {
	level := core.INFO
	level2 := core.ERROR
	source := "app"
	source2 := "other"
	group := "group1"
	group2 := "group2"
	msg := "hello world"
	msg2 := "bye world"
	now := time.Now()
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)

	tests := []struct {
		name string
		f1   *Filter
		f2   *Filter
		want bool
	}{
		{
			name: "all params identical",
			f1: &Filter{
				Level:      NewLevelFilter(&level),
				Source:     NewStringFilter(&source),
				Group:      NewStringFilter(&group),
				Message:    NewStringFilter(&msg),
				ReceivedAt: NewTimeFilter(&now, &after, &before),
			},
			f2: &Filter{
				Level:      NewLevelFilter(&level),
				Source:     NewStringFilter(&source),
				Group:      NewStringFilter(&group),
				Message:    NewStringFilter(&msg),
				ReceivedAt: NewTimeFilter(&now, &after, &before),
			},
			want: true,
		},
		{
			name: "different level",
			f1:   &Filter{Level: NewLevelFilter(&level)},
			f2:   &Filter{Level: NewLevelFilter(&level2)},
			want: false,
		},
		{
			name: "different source",
			f1:   &Filter{Source: NewStringFilter(&source)},
			f2:   &Filter{Source: NewStringFilter(&source2)},
			want: false,
		},
		{
			name: "different group",
			f1:   &Filter{Group: NewStringFilter(&group)},
			f2:   &Filter{Group: NewStringFilter(&group2)},
			want: false,
		},
		{
			name: "different message",
			f1:   &Filter{Message: NewStringFilter(&msg)},
			f2:   &Filter{Message: NewStringFilter(&msg2)},
			want: false,
		},
		{
			name: "different ReceivedAt Eq",
			f1:   &Filter{ReceivedAt: NewTimeFilter(&now, nil, nil)},
			f2:   &Filter{ReceivedAt: NewTimeFilter(&after, nil, nil)},
			want: false,
		},
		{
			name: "different ReceivedAt Le",
			f1:   &Filter{ReceivedAt: NewTimeFilter(nil, &after, nil)},
			f2:   &Filter{ReceivedAt: NewTimeFilter(nil, &before, nil)},
			want: false,
		},
		{
			name: "different ReceivedAt Ge",
			f1:   &Filter{ReceivedAt: NewTimeFilter(nil, nil, &before)},
			f2:   &Filter{ReceivedAt: NewTimeFilter(nil, nil, &after)},
			want: false,
		},
		{
			name: "all nil fields",
			f1:   &Filter{},
			f2:   &Filter{},
			want: true,
		},
		{
			name: "one nil, one set",
			f1:   &Filter{Level: NewLevelFilter(&level)},
			f2:   &Filter{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.f1.Equal(tt.f2)
			if got != tt.want {
				t.Errorf("Filter.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
