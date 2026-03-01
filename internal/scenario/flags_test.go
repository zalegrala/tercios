package scenario

import "testing"

func TestFileFlagsSetAndValues(t *testing.T) {
	var flags FileFlags
	if err := flags.Set(" a.json "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := flags.Set("b.json"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	values := flags.Values()
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != "a.json" || values[1] != "b.json" {
		t.Fatalf("unexpected values: %#v", values)
	}

	values[0] = "mutated"
	if flags.Values()[0] != "a.json" {
		t.Fatalf("Values() must return a copy")
	}
}

func TestFileFlagsRejectsEmpty(t *testing.T) {
	var flags FileFlags
	if err := flags.Set("   "); err == nil {
		t.Fatalf("expected error for empty path")
	}
}
