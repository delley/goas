package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewAppHelp(t *testing.T) {
	var out bytes.Buffer
	app := newApp()
	app.Writer = &out
	app.ErrWriter = &out

	if err := app.Run([]string{"goas", "--help"}); err != nil {
		t.Fatalf("run help: %v", err)
	}

	got := out.String()
	for _, want := range []string{"goas [options]", "--module-path", "--output", "--help, -h"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q in %q", want, got)
		}
	}
}
