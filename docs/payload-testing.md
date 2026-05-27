# Payload Stress Testing

tercios supports two distinct large-payload attack shapes, both configured through
scenario files using the `string` typed value:

| Shape | Config | Wire behavior |
|---|---|---|
| **Incompressible** | `{"type": "string", "size": N, "random": true}` | Full N bytes visible on the wire |
| **Compressible** | `{"type": "string", "size": N}` | Tiles a seed string; compresses well with gzip |

Use **incompressible** payloads to validate byte-limit enforcement at the network layer
(Envoy buffer filters, gRPC max message size). Use **compressible** payloads to test
decompressed-size limits and memory pressure on the receiver.

## Example scenarios

Two ready-to-use scenarios are in `examples/`:

| File | Shape | Per-span size | Spans per trace |
|---|---|---|---|
| `examples/large_payload_incompressible.json` | Random (incompressible) | 200 KB | 5 |
| `examples/large_payload_compressible.json` | Tiled (compressible) | 200 KB | 5 |

## Quick start

```bash
# Single incompressible request (~1 MB per trace)
tercios --endpoint=localhost:4317 \
  --insecure \
  --scenario-file=examples/large_payload_incompressible.json \
  --max-requests=1

# Ramp up to find the tipping point
tercios --endpoint=localhost:4317 \
  --insecure \
  --scenario-file=examples/large_payload_incompressible.json \
  --exporters=4 \
  --ramp-up=10 \
  --for=60
```

## Sizing

```
wire size per request ≈ (spans per trace) × (size per attribute) × (attributes with size)
```

Adjust `size` in the scenario JSON to hit a target request size. Use `--dry-run -o json`
to measure the model-level byte count before sending:

```bash
tercios --dry-run -o json \
  --scenario-file=examples/large_payload_incompressible.json \
  | python3 -c "
import sys, json
d = json.load(sys.stdin)
total = sum(len(v) for s in d['spans'] for v in s.get('attributes', {}).values() if isinstance(v, str))
print(f'{len(d[\"spans\"])} spans, ~{total/1024/1024:.2f} MB attribute data')
"
```

## Bundling multiple traces per request

Use `--traces-per-batch` to send N traces in a single OTLP export call:

```bash
# 5 traces × 5 spans × 200 KB = ~5 MB per request
tercios --endpoint=localhost:4317 \
  --insecure \
  --scenario-file=examples/large_payload_incompressible.json \
  --traces-per-batch=5 \
  --max-requests=1
```

## Finding the tipping point

Ramp concurrency with a fixed scenario to find where the system under test starts
rejecting or dropping requests:

```bash
tercios --endpoint=<target> \
  --scenario-file=examples/large_payload_incompressible.json \
  --traces-per-batch=5 \
  --exporters=8 \
  --ramp-up=30 \
  --for=120 \
  --request-interval=0.1
```

Watch `Failures` and `Payload rate` in the summary — the failure rate climbing while
payload rate plateaus indicates the system's limit.

## gRPC connection stickiness

With `--exporters=1` (default), all requests share one persistent gRPC connection
routed to a single upstream pod. Increase `--exporters` to spread load across pods,
or keep it at 1 to replicate a sticky single-client incident pattern.

## Composing with chaos

Large-payload scenarios compose with chaos policies. Use `--chaos-policies-file` to
inject errors or latency on top of oversized batches:

```bash
tercios --endpoint=localhost:4317 \
  --insecure \
  --scenario-file=examples/large_payload_incompressible.json \
  --chaos-policies-file=my-chaos.json \
  --traces-per-batch=5 \
  --for=60
```
