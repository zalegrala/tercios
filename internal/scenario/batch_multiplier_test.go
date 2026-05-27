package scenario

import (
	"context"
	"errors"
	"testing"

	"github.com/javiermolinar/tercios/internal/model"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type fixedGenerator struct {
	spans []model.Span
	calls int
}

func (f *fixedGenerator) GenerateBatch(_ context.Context) ([]model.Span, error) {
	f.calls++
	return f.spans, nil
}

type errorGenerator struct{}

func (e *errorGenerator) GenerateBatch(_ context.Context) ([]model.Span, error) {
	return nil, errors.New("generate failed")
}

func makeSpans(traceID string, count int) []model.Span {
	spans := make([]model.Span, count)
	for i := range spans {
		tid, _ := oteltrace.TraceIDFromHex(traceID)
		spans[i] = model.Span{TraceID: tid}
	}
	return spans
}

func TestBatchMultiplier_CombinesNBatches(t *testing.T) {
	inner := &fixedGenerator{spans: makeSpans("00000000000000000000000000000001", 3)}
	gen := NewBatchMultiplier(inner, 4)

	result, err := gen.GenerateBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 12 {
		t.Fatalf("expected 12 spans (4×3), got %d", len(result))
	}
	if inner.calls != 4 {
		t.Fatalf("expected inner called 4 times, got %d", inner.calls)
	}
}

func TestBatchMultiplier_NEqualsOne(t *testing.T) {
	inner := &fixedGenerator{spans: makeSpans("00000000000000000000000000000001", 5)}
	gen := NewBatchMultiplier(inner, 1)

	result, err := gen.GenerateBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 spans, got %d", len(result))
	}
}

func TestBatchMultiplier_PropagatesError(t *testing.T) {
	gen := NewBatchMultiplier(&errorGenerator{}, 3)

	_, err := gen.GenerateBatch(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
