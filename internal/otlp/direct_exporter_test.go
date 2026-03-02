package otlp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/javiermolinar/tercios/internal/config"
	"github.com/javiermolinar/tercios/internal/model"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type fakeOTLPClient struct {
	uploadErr error
	stopErr   error
}

func (f *fakeOTLPClient) Start(_ context.Context) error {
	return nil
}

func (f *fakeOTLPClient) Stop(_ context.Context) error {
	return f.stopErr
}

func (f *fakeOTLPClient) UploadTraces(_ context.Context, _ []*tracepb.ResourceSpans) error {
	return f.uploadErr
}

func TestDirectBatchExporterWrapsUploadErrorWithDiagnostics(t *testing.T) {
	client := &fakeOTLPClient{uploadErr: context.DeadlineExceeded}
	exporter := &directBatchExporter{
		client:   client,
		protocol: config.ProtocolHTTP,
		endpoint: "http://localhost:4318/v1/traces",
	}

	now := time.Now()
	batch := model.Batch{{
		TraceID:    oteltrace.TraceID{0x01},
		SpanID:     oteltrace.SpanID{0x02},
		Name:       "span",
		Kind:       oteltrace.SpanKindInternal,
		StartTime:  now,
		EndTime:    now.Add(5 * time.Millisecond),
		StatusCode: codes.Ok,
	}}

	err := exporter.ExportBatch(context.Background(), batch)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	message := err.Error()
	if !strings.Contains(message, "protocol=http") {
		t.Fatalf("expected protocol in error message, got %q", message)
	}
	if !strings.Contains(message, "endpoint=http://localhost:4318/v1/traces") {
		t.Fatalf("expected endpoint in error message, got %q", message)
	}
}

func TestDirectBatchExporterShutdownWrapsErrorWithDiagnostics(t *testing.T) {
	exporter := &directBatchExporter{
		client:   &fakeOTLPClient{stopErr: errors.New("shutdown failed")},
		protocol: config.ProtocolGRPC,
		endpoint: "localhost:4317",
	}

	err := exporter.Shutdown(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	message := err.Error()
	if !strings.Contains(message, "protocol=grpc") {
		t.Fatalf("expected protocol in error message, got %q", message)
	}
	if !strings.Contains(message, "endpoint=localhost:4317") {
		t.Fatalf("expected endpoint in error message, got %q", message)
	}
}
