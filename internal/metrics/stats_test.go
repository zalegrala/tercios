package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestStatsSummaryIncludesFailureBreakdown(t *testing.T) {
	stats := NewStats()
	stats.Record(5*time.Millisecond, nil)
	stats.Record(7*time.Millisecond, context.DeadlineExceeded)
	stats.Record(9*time.Millisecond, errors.New("dial tcp 127.0.0.1:4317: connect: connection refused"))

	summary := stats.Summary()
	if summary.Successes != 1 {
		t.Fatalf("expected 1 success, got %d", summary.Successes)
	}
	if summary.Failures != 2 {
		t.Fatalf("expected 2 failures, got %d", summary.Failures)
	}
	if got := summary.FailureBreakdown["timeout"]; got != 1 {
		t.Fatalf("expected timeout breakdown=1, got %d", got)
	}
	if got := summary.FailureBreakdown["connection_refused"]; got != 1 {
		t.Fatalf("expected connection_refused breakdown=1, got %d", got)
	}
	if len(summary.FailureSamples["timeout"]) == 0 {
		t.Fatalf("expected timeout sample")
	}
}

func TestFormatSummaryPrintsFailureBreakdown(t *testing.T) {
	summary := Summary{
		Total:      3,
		Successes:  1,
		Failures:   2,
		AvgLatency: 6 * time.Millisecond,
		P95Latency: 9 * time.Millisecond,
		FailureBreakdown: map[string]int{
			"timeout":            1,
			"connection_refused": 1,
		},
		FailureSamples: map[string][]string{
			"timeout": {"context deadline exceeded"},
		},
	}

	formatted := FormatSummary(summary)
	if !strings.Contains(formatted, "Failure breakdown:") {
		t.Fatalf("expected failure breakdown section, got %q", formatted)
	}
	if !strings.Contains(formatted, "- timeout: 1") {
		t.Fatalf("expected timeout breakdown line, got %q", formatted)
	}
	if !strings.Contains(formatted, "sample: context deadline exceeded") {
		t.Fatalf("expected timeout sample line, got %q", formatted)
	}
}
