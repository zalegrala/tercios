# Payload Stress Testing

`--span-attribute-padding` inflates every span in each OTLP export call by adding a
pseudo-random `gen.padding` string attribute. Use this to validate per-request payload
size limits — for example, an Envoy buffer filter that rejects requests exceeding a
configured byte threshold with HTTP 413.

This is distinct from **volume** stress testing (`--exporters`, `--max-requests`).
Payload testing targets the per-request size rather than request throughput.

## Quick start

```bash
# ~5 MB single request — default 20-span scenario × 250 KB per span
tercios --endpoint=localhost:4317 \
  --insecure \
  --span-attribute-padding=250000 \
  --exporters=1 \
  --max-requests=3
```

If the endpoint enforces a 4 MB limit, expect an error response (gRPC status 13
wrapping HTTP 413 from Envoy). Without the limit active, all requests succeed.

## CLI flag

| Flag | Default | Description |
|---|---|---|
| `--span-attribute-padding` | `0` | Bytes of pseudo-random padding added as a `gen.padding` string attribute on every span before export. `0` disables. |

## Sizing

The padding value is printable-ASCII pseudo-random content, which is effectively
incompressible. On-wire compressed size ≈ attribute value size.

```
wire size per request ≈ (spans per trace) × (padding bytes) + baseline overhead
```

With the embedded default scenario (~20 spans per trace):

| `--span-attribute-padding` | Approx. on-wire size |
|---|---|
| `50000` | ~1 MB |
| `100000` | ~2 MB |
| `200000` | ~4 MB |
| `250000` | ~5 MB |

To count spans per trace for a custom scenario:

```bash
tercios --dry-run -o json --scenario-file=my-scenario.json 2>/dev/null \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d['spans']), 'spans')"
```

## gRPC connection stickiness

With `--exporters 1` (default), all requests share a single persistent gRPC
connection routed to one upstream pod. This replicates the incident pattern
where a sticky connection with oversized LLM-attribute batches saturated a
single pod's upload bandwidth. Use `--max-requests` to control how many
oversized requests are sent per connection.

## Composing with chaos

Padding is applied **after** chaos mutations — spans that chaos errors, adds
latency to, or modifies attributes on still receive `gen.padding` before
export. Combine both to stress the error-reporting path at large payload sizes:

```bash
tercios --endpoint=localhost:4317 \
  --insecure \
  --span-attribute-padding=250000 \
  --chaos-policies-file=my-chaos.json \
  --exporters=1 \
  --max-requests=10
```

## Seeding

Padding content is seeded from `--scenario-run-seed`. Use a non-zero value for
reproducible test runs; leave it at `0` (default) for a fresh random seed per process.
