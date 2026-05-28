package otlp

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc/stats"
)

type wireByteContextKey struct{}

// wireStatsHandler implements grpc/stats.Handler and tracks compressed wire
// bytes sent per RPC call. The caller sets up a per-call counter in the
// context before each UploadTraces call; the handler increments it for every
// OutPayload event on that context.
type wireStatsHandler struct{}

func (wireStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (wireStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	out, ok := s.(*stats.OutPayload)
	if !ok {
		return
	}
	if counter, ok := ctx.Value(wireByteContextKey{}).(*atomic.Int64); ok {
		counter.Add(int64(out.WireLength))
	}
}

func (wireStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (wireStatsHandler) HandleConn(context.Context, stats.ConnStats) {}

// withWireByteCounter attaches a fresh counter to ctx and returns both the
// new context and the counter. Pass the context to UploadTraces, then read
// the counter after the call.
func withWireByteCounter(ctx context.Context) (context.Context, *atomic.Int64) {
	var counter atomic.Int64
	return context.WithValue(ctx, wireByteContextKey{}, &counter), &counter
}
