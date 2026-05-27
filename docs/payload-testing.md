# Payload Stress Testing

Two flags control per-request payload size independently of concurrency:

- **`--span-attribute-padding N`** — inflates every span with N bytes of pseudo-random
  content, growing the payload by `spans × N` bytes.
- **`--traces-per-batch N`** — bundles N traces into a single export call, multiplying
  span count per request.

Use them together to hit a target request size while keeping trace shapes realistic.
This is distinct from **volume** stress testing (`--exporters`, `--max-requests`), which
increases request rate rather than per-request size.

## Quick start

```bash
# ~5 MB single request — default 20-span scenario × 250 KB per span
tercios --endpoint=localhost:4317 \
  --insecure \
  --span-attribute-padding=250000 \
  --exporters=1 \
  --max-requests=3

# ~5 MB single request — 10 traces × 20 spans × 25 KB per span
tercios --endpoint=localhost:4317 \
  --insecure \
  --traces-per-batch=10 \
  --span-attribute-padding=25000 \
  --exporters=1 \
  --max-requests=3
```

If the endpoint enforces a 4 MB limit, expect an error response (gRPC status 13
wrapping HTTP 413 from Envoy). Without the limit active, all requests succeed.

## CLI flags

| Flag | Default | Description |
|---|---|---|
| `--span-attribute-padding` | `0` | Bytes of pseudo-random padding added as a `gen.padding` string attribute on every span before export. `0` disables. |
| `--traces-per-batch` | `1` | Number of traces bundled into each OTLP export call. Must be >= 1. |

## Sizing

The padding value is printable-ASCII pseudo-random content, which is effectively
incompressible. On-wire compressed size ≈ attribute value size.

```
wire size per request ≈ (traces-per-batch) × (spans per trace) × (padding bytes) + baseline overhead
```

With the embedded default scenario (~20 spans per trace, `--traces-per-batch=1`):

| `--span-attribute-padding` | Approx. on-wire size |
|---|---|
| `50000` | ~1 MB |
| `100000` | ~2 MB |
| `200000` | ~4 MB |
| `250000` | ~5 MB |

Increasing `--traces-per-batch` multiplies the per-request size proportionally, allowing
a lower per-span padding value for the same total payload size.

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
