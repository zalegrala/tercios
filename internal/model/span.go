package model

import (
	"sort"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Link is the relationship between two Spans.
type Link struct {
	SpanContext oteltrace.SpanContext
	Attributes  []attribute.KeyValue
}

// Event is a thing that happened during a Span's lifetime.
type Event struct {
	Name       string
	Time       time.Time
	Attributes []attribute.KeyValue
}

type Span struct {
	TraceID      oteltrace.TraceID
	SpanID       oteltrace.SpanID
	ParentSpanID oteltrace.SpanID

	Name      string
	Kind      oteltrace.SpanKind
	StartTime time.Time
	EndTime   time.Time

	Attributes         map[string]attribute.Value
	ResourceAttributes map[string]attribute.Value

	Links  []Link
	Events []Event

	StatusCode        codes.Code
	StatusDescription string
}

type Batch []Span

// ByteSize returns an approximate byte count for the batch, summing attribute
// key and value string lengths. String-valued attributes (e.g. gen.padding)
// contribute their exact byte count; other types contribute their string
// representation length. This is a model-level estimate, not the wire size.
func (b Batch) ByteSize() int {
	n := 0
	for _, span := range b {
		n += len(span.Name)
		for k, v := range span.Attributes {
			n += len(k) + len(v.AsString())
		}
		for k, v := range span.ResourceAttributes {
			n += len(k) + len(v.AsString())
		}
	}
	return n
}

func AttributesToMap(attributes []attribute.KeyValue) map[string]attribute.Value {
	if len(attributes) == 0 {
		return map[string]attribute.Value{}
	}
	out := make(map[string]attribute.Value, len(attributes))
	for _, kv := range attributes {
		out[string(kv.Key)] = kv.Value
	}
	return out
}

func AttributesFromMap(attributes map[string]attribute.Value) []attribute.KeyValue {
	if len(attributes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]attribute.KeyValue, 0, len(keys))
	for _, key := range keys {
		out = append(out, attribute.KeyValue{Key: attribute.Key(key), Value: attributes[key]})
	}
	return out
}
