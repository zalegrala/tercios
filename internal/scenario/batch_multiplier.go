package scenario

import (
	"context"

	"github.com/javiermolinar/tercios/internal/model"
)

type batchMultiplier struct {
	inner BatchGenerator
	n     int
}

// NewBatchMultiplier wraps inner so each GenerateBatch call invokes inner n
// times and concatenates the results into one slice. n < 1 is treated as 1.
func NewBatchMultiplier(inner BatchGenerator, n int) BatchGenerator {
	if n < 1 {
		n = 1
	}
	return &batchMultiplier{inner: inner, n: n}
}

func (m *batchMultiplier) GenerateBatch(ctx context.Context) ([]model.Span, error) {
	if m.n == 1 {
		return m.inner.GenerateBatch(ctx)
	}
	var out []model.Span
	for i := 0; i < m.n; i++ {
		batch, err := m.inner.GenerateBatch(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
}
