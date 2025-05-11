package core

import (
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{TRACE, "trace"},
		{DEBUG, "debug"},
		{INFO, "info"},
		{WARN, "warn"},
		{ERROR, "error"},
		{FATAL, "fatal"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if result := test.level.String(); result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}

	t.Run("unknown level", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for unknown log level, but no panic occurred")
			}
		}()
		_ = Level(999).String() // This should panic
	})
}
