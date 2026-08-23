package buildselect

import (
	"go/build"
	"os"
	"strings"
)

// Selector applies the Go toolchain's file selection rules to a directory.
type Selector struct {
	context build.Context
}

func New() *Selector {
	return NewWithContext(build.Default)
}

func NewWithContext(context build.Context) *Selector {
	return &Selector{context: context}
}

func (s *Selector) MatchFile(dir string, info os.FileInfo) (bool, error) {
	if info == nil || info.IsDir() {
		return false, nil
	}
	name := info.Name()
	if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		return false, nil
	}
	return s.context.MatchFile(dir, name)
}
