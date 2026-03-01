package scenario

import "fmt"

func NewBatchGeneratorFromFiles(paths []string, strategy SelectionStrategy) (BatchGenerator, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one scenario file is required")
	}

	definitions := make([]Definition, 0, len(paths))
	selectionSeed := uint64(0)
	for i, path := range paths {
		cfg, err := LoadFromJSON(path)
		if err != nil {
			return nil, fmt.Errorf("invalid scenario file %q: %w", path, err)
		}
		definition, err := cfg.Build()
		if err != nil {
			return nil, fmt.Errorf("invalid scenario definition in %q: %w", path, err)
		}
		definitions = append(definitions, definition)
		selectionSeed ^= uint64(definition.Seed) ^ (uint64(i+1) * 0x9e3779b97f4a7c15)
	}

	if len(definitions) == 1 {
		return NewGenerator(definitions[0]), nil
	}

	return NewMultiGenerator(definitions, strategy, int64(selectionSeed))
}
