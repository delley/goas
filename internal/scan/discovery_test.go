package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/delley/goas/internal/load"
)

func TestDiscoverPackages(t *testing.T) {
	ctx, err := load.ResolveModuleContext("../../example", "", "", "")
	if err != nil {
		t.Fatalf("ResolveModuleContext returned error: %v", err)
	}

	set, err := DiscoverPackages(ctx)
	if err != nil {
		t.Fatalf("DiscoverPackages returned error: %v", err)
	}
	if len(set.Packages) == 0 {
		t.Fatal("expected at least one package from example module")
	}

	found := false
	for _, pkg := range set.Packages {
		if pkg.Path == ctx.ModulePath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected root package %q to be present", ctx.ModulePath)
	}
}

func TestDiscoverPackages_DeterministicOrder(t *testing.T) {
	ctx, err := load.ResolveModuleContext("../../example", "", "", "")
	if err != nil {
		t.Fatalf("ResolveModuleContext returned error: %v", err)
	}

	first, err := DiscoverPackages(ctx)
	if err != nil {
		t.Fatalf("first DiscoverPackages returned error: %v", err)
	}
	second, err := DiscoverPackages(ctx)
	if err != nil {
		t.Fatalf("second DiscoverPackages returned error: %v", err)
	}
	if len(first.Packages) != len(second.Packages) {
		t.Fatalf("package count mismatch: %d != %d", len(first.Packages), len(second.Packages))
	}
	for i := range first.Packages {
		if first.Packages[i] != second.Packages[i] {
			t.Fatalf("package order changed at index %d: %#v != %#v", i, first.Packages[i], second.Packages[i])
		}
	}
}

func TestDiscoverPackages_SkipsGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal go.mod in root
	modPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(modPath, []byte("module example.com/test\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create a valid main.go file in root (required by ResolveModuleContext)
	rootGoPath := filepath.Join(tmpDir, "main.go")
	mainContent := `package main

func main() {
}
`
	if err := os.WriteFile(rootGoPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	// Create .git directory with a Go file inside
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}
	gitGoPath := filepath.Join(gitDir, "hidden.go")
	if err := os.WriteFile(gitGoPath, []byte("package git\n"), 0644); err != nil {
		t.Fatalf("failed to create file in .git: %v", err)
	}

	// Discover packages
	ctx, err := load.ResolveModuleContext(tmpDir, "", "", "")
	if err != nil {
		t.Fatalf("ResolveModuleContext returned error: %v", err)
	}

	set, err := DiscoverPackages(ctx)
	if err != nil {
		t.Fatalf("DiscoverPackages returned error: %v", err)
	}

	// Verify .git directory was not traversed
	for _, pkg := range set.Packages {
		if strings.Contains(pkg.Path, ".git") {
			t.Fatalf("discovered package in .git directory: %s", pkg.Path)
		}
	}

	// Verify root package is still discovered
	if len(set.Packages) == 0 {
		t.Fatal("expected at least one package (root) from temporary module")
	}
}

