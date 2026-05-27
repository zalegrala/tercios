package pipeline

import (
	"context"
	"testing"

	"github.com/javiermolinar/tercios/internal/model"
	"go.opentelemetry.io/otel/attribute"
)

func TestPaddingStage_AddsAttributeToEverySpan(t *testing.T) {
	const size = 100
	stage := NewPaddingStage(size, 42)
	input := []model.Span{
		{Attributes: map[string]attribute.Value{"existing": attribute.StringValue("keep")}},
		{Attributes: map[string]attribute.Value{}},
	}

	result, err := stage.process(context.Background(), input)
	if err != nil {
		t.Fatalf("process() error = %v", err)
	}
	if len(result) != len(input) {
		t.Fatalf("expected %d spans, got %d", len(input), len(result))
	}
	for i, s := range result {
		v, ok := s.Attributes["gen.padding"]
		if !ok {
			t.Fatalf("span %d: missing gen.padding attribute", i)
		}
		if len(v.AsString()) != size {
			t.Fatalf("span %d: expected %d bytes, got %d", i, size, len(v.AsString()))
		}
	}
	if result[0].Attributes["existing"].AsString() != "keep" {
		t.Fatal("existing attribute was removed or overwritten")
	}
}

func TestPaddingStage_UniquePerSpan(t *testing.T) {
	stage := NewPaddingStage(64, 42)
	input := []model.Span{
		{Attributes: map[string]attribute.Value{}},
		{Attributes: map[string]attribute.Value{}},
		{Attributes: map[string]attribute.Value{}},
	}

	result, _ := stage.process(context.Background(), input)

	v0 := result[0].Attributes["gen.padding"].AsString()
	v1 := result[1].Attributes["gen.padding"].AsString()
	v2 := result[2].Attributes["gen.padding"].AsString()
	if v0 == v1 || v1 == v2 || v0 == v2 {
		t.Fatal("adjacent spans have identical padding values — rotation not working")
	}
}

func TestPaddingStage_EmptyBatch(t *testing.T) {
	stage := NewPaddingStage(100, 42)
	result, err := stage.process(context.Background(), nil)
	if err != nil {
		t.Fatalf("process() error = %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d spans", len(result))
	}
}

func TestPaddingStage_Name(t *testing.T) {
	stage := NewPaddingStage(1, 42)
	if stage.name() != "padding" {
		t.Fatalf("expected name %q, got %q", "padding", stage.name())
	}
}
