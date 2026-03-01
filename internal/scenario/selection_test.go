package scenario

import "testing"

func TestParseSelectionStrategy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SelectionStrategy
		wantErr bool
	}{
		{name: "default-empty", input: "", want: SelectionStrategyRoundRobin},
		{name: "round-robin", input: "round-robin", want: SelectionStrategyRoundRobin},
		{name: "round_robin", input: "round_robin", want: SelectionStrategyRoundRobin},
		{name: "random", input: "random", want: SelectionStrategyRandom},
		{name: "invalid", input: "weighted", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSelectionStrategy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSelectionStrategy() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
