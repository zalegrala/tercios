package otlp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/javiermolinar/tercios/internal/config"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type fakePreflightClient struct {
	startErr    error
	uploadErr   error
	stopErr     error
	blockUpload bool

	started     bool
	uploadCalls int
	uploaded    []*tracepb.ResourceSpans
	stopped     bool
}

func (f *fakePreflightClient) Start(_ context.Context) error {
	if f.startErr != nil {
		return f.startErr
	}
	f.started = true
	return nil
}

func (f *fakePreflightClient) Stop(_ context.Context) error {
	f.stopped = true
	return f.stopErr
}

func (f *fakePreflightClient) UploadTraces(ctx context.Context, spans []*tracepb.ResourceSpans) error {
	f.uploadCalls++
	f.uploaded = spans
	if f.blockUpload {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.uploadErr
}

func TestRunPreflightSendsEmptyExportRequest(t *testing.T) {
	client := &fakePreflightClient{}
	err := runPreflight(context.Background(), config.ProtocolGRPC, "localhost:4317", 0, func() (preflightClient, error) {
		return client, nil
	})
	if err != nil {
		t.Fatalf("runPreflight() error = %v", err)
	}
	if !client.started || !client.stopped {
		t.Fatalf("expected client start/stop to be called")
	}
	if client.uploadCalls != 1 {
		t.Fatalf("expected one upload call, got %d", client.uploadCalls)
	}
	if client.uploaded == nil {
		t.Fatalf("expected uploaded slice, got nil")
	}
	if len(client.uploaded) != 0 {
		t.Fatalf("expected empty export payload, got %d resource spans", len(client.uploaded))
	}
}

func TestRunPreflightWrapsStartError(t *testing.T) {
	client := &fakePreflightClient{startErr: errors.New("dial failed")}
	err := runPreflight(context.Background(), config.ProtocolHTTP, "http://localhost:4318/v1/traces", 0, func() (preflightClient, error) {
		return client, nil
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "preflight start client") {
		t.Fatalf("expected wrapped start error, got %q", err.Error())
	}
}

func TestRunPreflightTimeout(t *testing.T) {
	client := &fakePreflightClient{blockUpload: true}
	err := runPreflight(context.Background(), config.ProtocolGRPC, "localhost:4317", 10*time.Millisecond, func() (preflightClient, error) {
		return client, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if !strings.Contains(err.Error(), "preflight export") {
		t.Fatalf("expected wrapped export error, got %q", err.Error())
	}
}
