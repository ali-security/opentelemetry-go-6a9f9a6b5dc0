// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptrace_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type client struct {
	uploadErr   error
	logEndpoint string
}

var _ otlptrace.Client = &client{}

func (c *client) Start(ctx context.Context) error {
	return nil
}

func (c *client) Stop(ctx context.Context) error {
	return nil
}

func (c *client) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	return c.uploadErr
}

func (c *client) MarshalLog() interface{} {
	return struct{ Endpoint string }{Endpoint: c.logEndpoint}
}

func TestExporterMarshalLogDoesNotIncludeClientConfig(t *testing.T) {
	const sensitiveEndpoint = "user:pass@collector.internal:4318"

	exp := otlptrace.NewUnstarted(&client{logEndpoint: sensitiveEndpoint})

	marshaled := exp.MarshalLog()
	clientField := reflect.ValueOf(marshaled).FieldByName("Client")
	if assert.True(t, clientField.IsValid()) {
		// The client must be represented by its type only, never by its
		// value (which a logger would recursively marshal).
		assert.Equal(t, reflect.String, clientField.Kind())
		assert.Equal(t, "*otlptrace_test.client", fmt.Sprint(clientField.Interface()))
	}

	logged := fmt.Sprintf("%+v", marshaled)
	assert.Contains(t, logged, "otlptrace")
	assert.NotContains(t, logged, sensitiveEndpoint)
}

func TestExporterClientError(t *testing.T) {
	ctx := context.Background()
	exp, err := otlptrace.New(ctx, &client{
		uploadErr: context.Canceled,
	})
	assert.NoError(t, err)

	spans := tracetest.SpanStubs{{Name: "Span 0"}}.Snapshots()
	err = exp.ExportSpans(ctx, spans)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, strings.HasPrefix(err.Error(), "traces export: "), err)

	assert.NoError(t, exp.Shutdown(ctx))
}
