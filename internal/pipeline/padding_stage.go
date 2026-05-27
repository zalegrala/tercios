package pipeline

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/javiermolinar/tercios/internal/model"
	"go.opentelemetry.io/otel/attribute"
)

type paddingStage struct {
	buffer []byte
}

// NewPaddingStage returns a BatchStage that appends a pseudo-random gen.padding
// string attribute of exactly size bytes to every span. seed=0 uses a random
// per-process seed, consistent with --scenario-run-seed=0 behaviour.
func NewPaddingStage(size int, seed int64) BatchStage {
	if size <= 0 {
		return &paddingStage{}
	}
	s := uint64(seed)
	if s == 0 {
		s = uint64(time.Now().UnixNano())
	}
	r := rand.New(rand.NewPCG(s, 0))
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(r.IntN(95)) + 0x20 // printable ASCII 0x20–0x7E
	}
	return &paddingStage{buffer: buf}
}

func (s *paddingStage) name() string { return "padding" }

func (s *paddingStage) process(_ context.Context, spans []model.Span) ([]model.Span, error) {
	if len(s.buffer) == 0 || len(spans) == 0 {
		return spans, nil
	}
	for i := range spans {
		spans[i].Attributes["gen.padding"] = attribute.StringValue(rotatePadding(s.buffer, i))
	}
	return spans, nil
}

// rotatePadding returns buf cyclically shifted left by n positions. Each span
// index produces a distinct string, preventing gzip from collapsing repeated
// identical values within the same OTLP message.
func rotatePadding(buf []byte, n int) string {
	size := len(buf)
	offset := n % size
	if offset == 0 {
		return string(buf)
	}
	out := make([]byte, size)
	copy(out, buf[offset:])
	copy(out[size-offset:], buf[:offset])
	return string(out)
}
