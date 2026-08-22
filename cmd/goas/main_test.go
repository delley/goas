package main

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if got := run([]string{"goas", "--help"}, &stdout, &stderr); got != 0 {
		t.Fatalf("run help exit code = %d, want 0", got)
	}
	if stdout.Len() == 0 {
		t.Fatal("run help wrote no output to stdout")
	}
	if stderr.Len() != 0 {
		t.Fatalf("run help wrote to stderr: %q", stderr.String())
	}
}

func TestRunWithoutArgumentsShowsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalArgs := os.Args
	os.Args = []string{"goas"}
	defer func() { os.Args = originalArgs }()

	if got := run([]string{"goas"}, &stdout, &stderr); got != 0 {
		t.Fatalf("run without arguments exit code = %d, want 0", got)
	}
	if stdout.Len() == 0 {
		t.Fatal("run without arguments wrote no help to stdout")
	}
	if stderr.Len() != 0 {
		t.Fatalf("run without arguments wrote to stderr: %q", stderr.String())
	}
}

func TestRunReportsUsageError(t *testing.T) {
	for _, args := range [][]string{
		{"goas", "--bogus"},
		{"goas", "--module-path"},
	} {
		t.Run(strings.Join(args[1:], "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if got := run(args, &stdout, &stderr); got == 0 {
				t.Fatal("usage error exit code = 0, want non-zero")
			}
			if stdout.Len() == 0 {
				t.Fatal("usage error wrote no help to stdout")
			}
			if !strings.Contains(stderr.String(), "Error:") {
				t.Fatalf("usage error missing stderr message: %q", stderr.String())
			}
		})
	}
}

func TestRunReportsGenerationError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	args := []string{
		"goas",
		"--module-path", "/path/that/does/not/exist",
		"--output", "-",
	}
	if got := run(args, &stdout, &stderr); got == 0 {
		t.Fatal("generation error exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Fatalf("generation error missing stderr message: %q", stderr.String())
	}
}

func TestRunPreservesExistingFileOnGenerationFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "oas.json")
	const original = "existing api\n"
	if err := os.WriteFile(dest, []byte(original), 0o600); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{
		"goas",
		"--module-path", "/path/that/does/not/exist",
		"--output", dest,
	}
	if got := run(args, &stdout, &stderr); got == 0 {
		t.Fatal("generation failure exit code = 0, want non-zero")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination after failure: %v", err)
	}
	if string(data) != original {
		t.Fatalf("destination changed after generation failure: got %q, want %q", string(data), original)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read destination dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "oas.json" {
		t.Fatalf("leftover temp files after failure: %#v", entries)
	}
}

func TestRunWritesOutputAtomicallyOnSuccess(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "oas.json")

	var stdout, stderr bytes.Buffer
	args := []string{
		"goas",
		"--module-path", "../../example",
		"--main-file-path", "../../example/main.go",
		"--output", dest,
	}
	if got := run(args, &stdout, &stderr); got != 0 {
		t.Fatalf("generation success exit code = %d, want 0: stderr=%q", got, stderr.String())
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination after success: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file was empty after successful generation")
	}
	if stdout.Len() != 0 {
		t.Fatalf("successful file output wrote to stdout unexpectedly: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful file output wrote errors to stderr: %q", stderr.String())
	}
}
