# Tyk OpenTelemetry Go Library

This repository contains Tyk's OpenTelemetry library. It provides an abstraction layer for working with OpenTelemetry, making it easier to instrument, generate, collect, and export telemetry data. The library is designed to be imported and reused across main Tyk's components.

## Getting Started

To start using the Tyk OpenTelemetry Go library, you can import this library as a dependency into your Go application.

```
go get github.com/TykTechnologies/opentelemetry
```

Ensure that you have Go installed on your machine.

```
import (
    "context"
	"os"
	"os/signal"
	"syscall"
    "github.com/TykTechnologies/opentelemetry/config"
	"github.com/TykTechnologies/opentelemetry/trace"
)

func main(){
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg := config.OpenTelemetry{
		Enabled:           true,
		Exporter:          "grpc",
		Endpoint:          "otel-collector:4317",
		ConnectionTimeout: 10,
		ResourceName:      "e2e-basic",
	}

	provider, err := trace.NewProvider(trace.WithContext(ctx), trace.WithConfig(&cfg))
	if err != nil {
		log.Printf("error on otel provider init %s", err.Error())
		return
	}
	defer provider.Shutdown(ctx)

	tracer := provider.Tracer()

	_, span := tracer.Start(ctx, "span-name")
	defer span.End()
}
```

## Compiling the Project

This repository includes a Taskfile that allows you to build the project to check everything is working as expected.
It will also build the "basic" repository, which is a simple application that uses the Tyk OpenTelemetry library to test it.

To build the project, use the build task:

```
task build
```

## Running Tests

The Taskfile also includes tasks for running unit tests.

```
task test
```

This command will run all unit tests in the repository.

### End-to-End (E2E) Testing

The repository provides several tasks to set up and run e2e tests.

1. **e2e-setup:** This task starts the basic application, the tracetest, and the otel-collector for e2e testing. It can be run with the following command:

```
task e2e-setup
```

2. **e2e-test:** This task runs the e2e test scenarios with tracetest:

```
task e2e-test
```

3. **e2e-stop:** After you've run the e2e tests, you can stop the e2e environment with this task:

```
task e2e-stop
```

4. **e2e**: This task combines all the previous steps (setup, run, and clean) to install, run, and clean the e2e tests:

```
task e2e
```
