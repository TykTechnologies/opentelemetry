package trace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	LoggedMessage string
}

func (m *mockLogger) Error(args ...interface{}) {
	m.LoggedMessage = fmt.Sprintf("%v", args[0])
}

func (m *mockLogger) Info(args ...interface{}) {}

func TestErrHandler_Handle(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "non-nil error",
			err:      fmt.Errorf("test error"),
			expected: "error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			eh := &errHandler{logger: logger}
			eh.Handle(tt.err)
			assert.Equal(t, tt.expected, logger.LoggedMessage)
		})
	}
}
