package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/javiermolinar/tercios/internal/model"
	"go.opentelemetry.io/otel/sdk/trace"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxFailureSamplesPerClass = 3

type Stats struct {
	durations        []time.Duration
	successes        int
	failures         int
	failureBreakdown map[string]int
	failureSamples   map[string][]string
}

func NewStats() *Stats {
	return &Stats{
		failureBreakdown: make(map[string]int),
		failureSamples:   make(map[string][]string),
	}
}

func (s *Stats) Record(duration time.Duration, err error) {
	s.durations = append(s.durations, duration)
	if err != nil {
		s.failures++
		class := classifyError(err)
		s.failureBreakdown[class]++
		s.recordFailureSample(class, err)
	} else {
		s.successes++
	}
}

func (s *Stats) recordFailureSample(class string, err error) {
	if err == nil {
		return
	}
	if s.failureSamples == nil {
		s.failureSamples = make(map[string][]string)
	}
	normalized := normalizeErrorMessage(err.Error())
	if normalized == "" {
		return
	}
	samples := s.failureSamples[class]
	for _, sample := range samples {
		if sample == normalized {
			return
		}
	}
	if len(samples) >= maxFailureSamplesPerClass {
		return
	}
	s.failureSamples[class] = append(samples, normalized)
}

type Summary struct {
	Total            int
	Successes        int
	Failures         int
	AvgLatency       time.Duration
	P95Latency       time.Duration
	FailureBreakdown map[string]int
	FailureSamples   map[string][]string
}

func (s *Stats) Summary() Summary {
	total := len(s.durations)
	if total == 0 {
		return Summary{
			Total:            0,
			Successes:        s.successes,
			Failures:         s.failures,
			FailureBreakdown: cloneBreakdown(s.failureBreakdown),
			FailureSamples:   cloneSamples(s.failureSamples),
		}
	}

	durations := make([]time.Duration, total)
	copy(durations, s.durations)
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := time.Duration(int64(sum) / int64(total))
	p95Index := int(float64(total-1) * 0.95)
	p95 := durations[p95Index]

	return Summary{
		Total:            total,
		Successes:        s.successes,
		Failures:         s.failures,
		AvgLatency:       avg,
		P95Latency:       p95,
		FailureBreakdown: cloneBreakdown(s.failureBreakdown),
		FailureSamples:   cloneSamples(s.failureSamples),
	}
}

func Summarize(stats []*Stats) Summary {
	var total int
	var successes int
	var failures int
	failureBreakdown := make(map[string]int)
	failureSamples := make(map[string][]string)

	for _, stat := range stats {
		if stat == nil {
			continue
		}
		total += len(stat.durations)
		successes += stat.successes
		failures += stat.failures
		mergeBreakdown(failureBreakdown, stat.failureBreakdown)
		mergeSamples(failureSamples, stat.failureSamples)
	}

	if total == 0 {
		return Summary{
			Total:            0,
			Successes:        successes,
			Failures:         failures,
			FailureBreakdown: failureBreakdown,
			FailureSamples:   failureSamples,
		}
	}

	durations := make([]time.Duration, 0, total)
	for _, stat := range stats {
		if stat == nil {
			continue
		}
		durations = append(durations, stat.durations...)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := time.Duration(int64(sum) / int64(total))
	p95Index := int(float64(total-1) * 0.95)
	p95 := durations[p95Index]

	return Summary{
		Total:            total,
		Successes:        successes,
		Failures:         failures,
		AvgLatency:       avg,
		P95Latency:       p95,
		FailureBreakdown: failureBreakdown,
		FailureSamples:   failureSamples,
	}
}

type InstrumentedExporter struct {
	inner trace.SpanExporter
	stats *Stats
}

func NewInstrumentedExporter(inner trace.SpanExporter, stats *Stats) *InstrumentedExporter {
	return &InstrumentedExporter{inner: inner, stats: stats}
}

func (e *InstrumentedExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	start := time.Now()
	err := e.inner.ExportSpans(ctx, spans)
	if e.stats != nil {
		e.stats.Record(time.Since(start), err)
	}
	return err
}

func (e *InstrumentedExporter) Shutdown(ctx context.Context) error {
	return e.inner.Shutdown(ctx)
}

type InstrumentedBatchExporter struct {
	inner model.BatchExporter
	stats *Stats
}

func NewInstrumentedBatchExporter(inner model.BatchExporter, stats *Stats) *InstrumentedBatchExporter {
	return &InstrumentedBatchExporter{inner: inner, stats: stats}
}

func (e *InstrumentedBatchExporter) ExportBatch(ctx context.Context, batch model.Batch) error {
	start := time.Now()
	err := e.inner.ExportBatch(ctx, batch)
	if e.stats != nil {
		e.stats.Record(time.Since(start), err)
	}
	return err
}

func (e *InstrumentedBatchExporter) Shutdown(ctx context.Context) error {
	return e.inner.Shutdown(ctx)
}

func FormatSummary(summary Summary) string {
	lines := []string{
		fmt.Sprintf("Sent %s requests", formatCount(summary.Total)),
		fmt.Sprintf("Success: %s", formatCount(summary.Successes)),
		fmt.Sprintf("Failures: %s", formatCount(summary.Failures)),
		fmt.Sprintf("Avg latency: %s", formatLatency(summary.AvgLatency)),
		fmt.Sprintf("P95 latency: %s", formatLatency(summary.P95Latency)),
	}

	if summary.Failures > 0 && len(summary.FailureBreakdown) > 0 {
		keys := make([]string, 0, len(summary.FailureBreakdown))
		for key := range summary.FailureBreakdown {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		lines = append(lines, "Failure breakdown:")
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("  - %s: %s", key, formatCount(summary.FailureBreakdown[key])))
			samples := summary.FailureSamples[key]
			for _, sample := range samples {
				lines = append(lines, fmt.Sprintf("    sample: %s", sample))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func FormatProgress(summary Summary, expected int) string {
	expectedText := formatCount(expected)
	if expected <= 0 {
		expectedText = "?"
	}
	return fmt.Sprintf(
		"Progress: %s/%s sent | Success: %s | Failures: %s | Avg: %s | P95: %s",
		formatCount(summary.Total),
		expectedText,
		formatCount(summary.Successes),
		formatCount(summary.Failures),
		formatLatency(summary.AvgLatency),
		formatLatency(summary.P95Latency),
	)
}

func formatCount(count int) string {
	if count >= 1000 {
		value := float64(count) / 1000.0
		if count%1000 == 0 {
			return fmt.Sprintf("%dk", count/1000)
		}
		return fmt.Sprintf("%.1fk", value)
	}
	return fmt.Sprintf("%d", count)
}

func formatLatency(duration time.Duration) string {
	if duration <= 0 {
		return "0ms"
	}
	return fmt.Sprintf("%dms", duration.Milliseconds())
}

func cloneBreakdown(in map[string]int) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneSamples(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(in))
	for key, samples := range in {
		out[key] = append([]string(nil), samples...)
	}
	return out
}

func mergeBreakdown(dst map[string]int, src map[string]int) {
	for key, value := range src {
		dst[key] += value
	}
}

func mergeSamples(dst map[string][]string, src map[string][]string) {
	for key, samples := range src {
		for _, sample := range samples {
			if len(dst[key]) >= maxFailureSamplesPerClass {
				break
			}
			exists := false
			for _, existing := range dst[key] {
				if existing == sample {
					exists = true
					break
				}
			}
			if !exists {
				dst[key] = append(dst[key], sample)
			}
		}
	}
}

func normalizeErrorMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " | ")
	return strings.Join(strings.Fields(trimmed), " ")
}

func classifyError(err error) string {
	if err == nil {
		return "other"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case grpccodes.DeadlineExceeded:
			return "timeout"
		case grpccodes.Canceled:
			return "canceled"
		case grpccodes.Unavailable:
			if strings.Contains(strings.ToLower(st.Message()), "connection refused") {
				return "connection_refused"
			}
			return "unavailable"
		case grpccodes.Unauthenticated:
			return "unauthenticated"
		case grpccodes.PermissionDenied:
			return "permission_denied"
		}
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "timed out"), strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "connection refused"):
		return "connection_refused"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "name resolution"):
		return "dns"
	case strings.Contains(msg, "x509"), strings.Contains(msg, "tls"), strings.Contains(msg, "certificate"):
		return "tls"
	case strings.Contains(msg, "unauthenticated"), strings.Contains(msg, "unauthorized"), strings.Contains(msg, "401"):
		return "unauthenticated"
	case strings.Contains(msg, "permission denied"), strings.Contains(msg, "403"):
		return "permission_denied"
	case strings.Contains(msg, "unavailable"):
		return "unavailable"
	default:
		return "other"
	}
}
