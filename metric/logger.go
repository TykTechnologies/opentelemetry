package metric

import "fmt"

// Logger represents the internal library logger used for error and info messages.
type Logger interface {
	Info(args ...interface{})
	Error(args ...interface{})
}

type noopLogger struct{}

func (n *noopLogger) Error(args ...interface{}) {}

func (n *noopLogger) Info(args ...interface{}) {}

// errHandler implements otel.ErrorHandler for handling OpenTelemetry errors.
type errHandler struct {
	logger Logger
}

func (eh *errHandler) Handle(err error) {
	if eh.logger != nil && err != nil {
		eh.logger.Error(fmt.Sprintf("error: %v", err.Error()))
	}
}
