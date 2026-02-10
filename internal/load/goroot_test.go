package load

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCorePackages(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		_, err := CorePackages("")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("non-existent path returns error", func(t *testing.T) {
		_, err := CorePackages(filepath.Join(os.TempDir(), "this-path-should-not-exist-123456789"))
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("collects only dirs that contain .go files", func(t *testing.T) {
		root := t.TempDir()

		// /root/a has .go -> include "a"
		mkdir(t, filepath.Join(root, "a"))
		writeFile(t, filepath.Join(root, "a", "a.go"), "package a\n")

		// /root/a/sub has .go -> include "a/sub"
		mkdir(t, filepath.Join(root, "a", "sub"))
		writeFile(t, filepath.Join(root, "a", "sub", "sub.go"), "package sub\n")

		// /root/b has no .go -> do NOT include "b"
		mkdir(t, filepath.Join(root, "b"))
		writeFile(t, filepath.Join(root, "b", "readme.txt"), "hello")

		// /root/c has non-go files only -> do NOT include "c"
		mkdir(t, filepath.Join(root, "c"))
		writeFile(t, filepath.Join(root, "c", "c.md"), "# doc")

		got, err := CorePackages(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !got["a"] {
			t.Fatalf("expected package %q to be present", "a")
		}
		if !got["a/sub"] {
			t.Fatalf("expected package %q to be present", "a/sub")
		}
		if got["b"] {
			t.Fatalf("did not expect package %q to be present", "b")
		}
		if got["c"] {
			t.Fatalf("did not expect package %q to be present", "c")
		}
	})

	t.Run("go files at root are ignored by current behavior", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "main.go"), "package main\n")

		got, err := CorePackages(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Pela implementação atual, "." / "" é ignorado.
		// Se você quiser incluir root como um pacote, ajuste o CorePackages e este teste.
		if len(got) != 0 {
			t.Fatalf("expected empty map, got: %#v", got)
		}
	})
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %q: %v", path, err)
	}
}
