package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/delley/goas/goas"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestOptionsFromContext(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		want   goas.Options
	}{
		{
			name: "defaults",
			want: goas.Options{
				ModulePath:  ".",
				FileRefPath: ".",
				OutputPath:  "oas.json",
			},
		},
		{
			name: "explicit values",
			values: map[string]string{
				"module-path":    "./example",
				"main-file-path": "./main.go",
				"handler-path":   "./handlers",
				"file-ref-path":  "./files",
				"output":         "-",
				"debug":          "true",
				"omit-packages":  "true",
				"show-hidden":    "true",
			},
			want: goas.Options{
				ModulePath:   "./example",
				MainFilePath: "./main.go",
				HandlerPath:  "./handlers",
				FileRefPath:  "./files",
				OutputPath:   "-",
				Debug:        true,
				OmitPackages: true,
				ShowHidden:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			for _, cliFlag := range flags {
				cliFlag.Apply(set)
			}
			for name, value := range tt.values {
				if err := set.Set(name, value); err != nil {
					t.Fatalf("set %s: %v", name, err)
				}
			}

			ctx := cli.NewContext(newApp(), set, nil)
			if got := optionsFromContext(ctx); got != tt.want {
				t.Fatalf("optionsFromContext() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
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

func TestOutputWriterReportsCreateError(t *testing.T) {
	_, _, _, err := outputWriter(goas.Options{OutputPath: filepath.Join(t.TempDir(), "missing", "oas.json")})
	if err == nil {
		t.Fatal("outputWriter() error = nil, want error")
	}
}

func TestRunWritesOutputAtomicallyOnSuccess(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "oas.json")

	var stdout, stderr bytes.Buffer
	args := []string{
		"goas",
		"--module-path", "../../example",
		"--main-file-path", "main.go",
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

func TestReadmeContractExamples(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	require.NoError(t, err)

	text := string(content)
	require.NotContains(t, text, "@Tags xxx")
	require.Contains(t, text, "@Tags \"{tag}\"")
	require.Contains(t, text, "@Router")
	require.NotContains(t, text, "(cd example && go run ../cmd/goas --module-path . --main-file-path ./main.go --output ./example.json)")
}
