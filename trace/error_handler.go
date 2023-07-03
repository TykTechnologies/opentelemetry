package trace

import "fmt"

type errHandler struct {
	logger Logger
}

func (eh *errHandler) Handle(err error) {
	if eh.logger != nil && err != nil {
		eh.logger.Error(fmt.Sprintf("error: %v", err.Error()))
	}
}

type noopLogger struct{}

func (n *noopLogger) Error(args ...interface{}) {}

func (n *noopLogger) Info(args ...interface{}) {}
