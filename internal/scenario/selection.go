package scenario

import (
	"fmt"
	"strings"
)

type SelectionStrategy string

const (
	SelectionStrategyRoundRobin SelectionStrategy = "round-robin"
	SelectionStrategyRandom     SelectionStrategy = "random"
)

func ParseSelectionStrategy(value string) (SelectionStrategy, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", string(SelectionStrategyRoundRobin), "round_robin":
		return SelectionStrategyRoundRobin, nil
	case string(SelectionStrategyRandom):
		return SelectionStrategyRandom, nil
	default:
		return "", fmt.Errorf("unsupported scenario strategy %q (supported: %s, %s)", value, SelectionStrategyRoundRobin, SelectionStrategyRandom)
	}
}
