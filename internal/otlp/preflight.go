package otlp

import (
	"context"
	"fmt"
	"time"

	"github.com/javiermolinar/tercios/internal/config"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type preflightClient interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error
}

func RunPreflight(ctx context.Context, factory ExporterFactory, exportTimeout time.Duration) error {
	newClient := func() (preflightClient, error) {
		return factory.newOTLPClient()
	}
	return runPreflight(ctx, factory.Protocol, factory.Endpoint, exportTimeout, newClient)
}

func runPreflight(
	ctx context.Context,
	protocol config.Protocol,
	endpoint string,
	exportTimeout time.Duration,
	newClient func() (preflightClient, error),
) (err error) {
	if newClient == nil {
		return fmt.Errorf("preflight create client protocol=%s endpoint=%s: client builder not configured", protocol, endpoint)
	}

	exportCtx := ctx
	cancel := func() {}
	if exportTimeout > 0 {
		exportCtx, cancel = context.WithTimeout(ctx, exportTimeout)
	}
	defer cancel()

	client, err := newClient()
	if err != nil {
		return fmt.Errorf("preflight create client protocol=%s endpoint=%s: %w", protocol, endpoint, err)
	}
	if err := client.Start(exportCtx); err != nil {
		return fmt.Errorf("preflight start client protocol=%s endpoint=%s: %w", protocol, endpoint, err)
	}
	defer func() {
		stopCtx := context.Background()
		stopCancel := func() {}
		if exportTimeout > 0 {
			stopCtx, stopCancel = context.WithTimeout(context.Background(), exportTimeout)
		}
		defer stopCancel()
		if stopErr := client.Stop(stopCtx); stopErr != nil && err == nil {
			err = fmt.Errorf("preflight stop client protocol=%s endpoint=%s: %w", protocol, endpoint, stopErr)
		}
	}()

	if err := client.UploadTraces(exportCtx, []*tracepb.ResourceSpans{}); err != nil {
		return fmt.Errorf("preflight export protocol=%s endpoint=%s: %w", protocol, endpoint, err)
	}
	return nil
}
