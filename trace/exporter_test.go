package trace

import (
	"context"
	"fmt"
	"opentelemetry/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewGRPCClient(t *testing.T) {
	ctx := context.Background()
	endpoint := "localhost:4317"

	cfg := config.OpenTelemetry{
		Endpoint: endpoint,
	}

	client := newGRPCClient(ctx, cfg)
	assert.NotNil(t, client)
}

func Test_NewHTTPClient(t *testing.T) {
	ctx := context.Background()
	endpoint := "localhost:4317"

	cfg := config.OpenTelemetry{
		Endpoint: endpoint,
	}

	client := newHTTPClient(ctx, cfg)
	assert.NotNil(t, client)
}

func Test_ExporterFactory(t *testing.T) {
	tcs := []struct {
		name        string
		givenConfig config.OpenTelemetry
		expectedErr error
	}{
		{
			name: "invalid exporter type",
			givenConfig: config.OpenTelemetry{
				Exporter: "invalid",
			},
			expectedErr: fmt.Errorf("invalid exporter type: %s", "invalid"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			exporter, err := exporterFactory(ctx, tc.givenConfig)
			if tc.expectedErr != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, exporter)
			}
		})
	}
}
