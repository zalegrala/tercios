package scenario

import (
	"fmt"
	"strings"
)

type FileFlags struct {
	paths []string
}

func (f *FileFlags) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(f.paths, ",")
}

func (f *FileFlags) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("scenario file path cannot be empty")
	}
	f.paths = append(f.paths, trimmed)
	return nil
}

func (f *FileFlags) Values() []string {
	if f == nil || len(f.paths) == 0 {
		return nil
	}
	out := make([]string, len(f.paths))
	copy(out, f.paths)
	return out
}
