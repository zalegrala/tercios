package scenario

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestNewBatchGeneratorFromFilesSingle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.json")
	if err := os.WriteFile(path, []byte(minimalScenarioJSON("single", 1, "root-single")), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	generator, err := NewBatchGeneratorFromFiles([]string{path}, SelectionStrategyRoundRobin)
	if err != nil {
		t.Fatalf("NewBatchGeneratorFromFiles() error = %v", err)
	}

	batch, err := generator.GenerateBatch(context.Background())
	if err != nil {
		t.Fatalf("GenerateBatch() error = %v", err)
	}
	if len(batch) == 0 {
		t.Fatalf("expected non-empty batch")
	}
	if batch[0].Name != "root-single" {
		t.Fatalf("expected root-single, got %q", batch[0].Name)
	}
}

func TestNewBatchGeneratorFromFilesMultiple(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "scenario-a.json")
	pathB := filepath.Join(dir, "scenario-b.json")
	if err := os.WriteFile(pathA, []byte(minimalScenarioJSON("a", 1, "root-a")), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(pathB, []byte(minimalScenarioJSON("b", 2, "root-b")), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	generator, err := NewBatchGeneratorFromFiles([]string{pathA, pathB}, SelectionStrategyRoundRobin)
	if err != nil {
		t.Fatalf("NewBatchGeneratorFromFiles() error = %v", err)
	}

	first, err := generator.GenerateBatch(context.Background())
	if err != nil {
		t.Fatalf("GenerateBatch() error = %v", err)
	}
	second, err := generator.GenerateBatch(context.Background())
	if err != nil {
		t.Fatalf("GenerateBatch() error = %v", err)
	}
	if first[0].Name != "root-a" || second[0].Name != "root-b" {
		t.Fatalf("expected round robin root-a/root-b, got %q/%q", first[0].Name, second[0].Name)
	}
}

func TestNewBatchGeneratorFromFilesRejectsEmpty(t *testing.T) {
	_, err := NewBatchGeneratorFromFiles(nil, SelectionStrategyRoundRobin)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func minimalScenarioJSON(name string, seed int64, rootSpanName string) string {
	return `{
  "name": "` + name + `",
  "seed": ` + strconv.FormatInt(seed, 10) + `,
  "services": {
    "svc": {
      "resource": {
        "service.name": { "type": "string", "value": "` + name + `" }
      }
    }
  },
  "nodes": {
    "root": { "service": "svc", "span_name": "` + rootSpanName + `" },
    "child": { "service": "svc", "span_name": "child" }
  },
  "root": "root",
  "edges": [
    { "from": "root", "to": "child", "kind": "internal", "repeat": 1, "duration_ms": 10 }
  ]
}`
}
