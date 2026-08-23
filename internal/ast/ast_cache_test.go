package ast

import (
	"go/ast"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/delley/goas/internal/buildselect"
)

func TestPackageCache_GetPackageAST(t *testing.T) {
	cache := NewPackageCache()
	pkgPath := filepath.Join("..", "..", "example")

	first, err := cache.GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("GetPackageAST returned error: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("expected AST packages for example module")
	}

	second, err := cache.GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("second GetPackageAST returned error: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("cache should be stable: got %d != %d", len(first), len(second))
	}
}

func TestPackageCache_GetPackageAST_InvalidPath(t *testing.T) {
	cache := NewPackageCache()
	if _, err := cache.GetPackageAST(filepath.Join("..", "..", "this-path-does-not-exist")); err == nil {
		t.Fatal("expected error for invalid package path")
	}
}

func TestPackageCache_IgnoresTestFiles(t *testing.T) {
	cache := NewPackageCache()
	pkgPath := filepath.Join("..", "..", "example")
	pkgs, err := cache.GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("GetPackageAST returned error: %v", err)
	}
	for _, pkg := range pkgs {
		for fileName := range pkg.Files {
			if fileName == "" {
				continue
			}
			if len(fileName) >= 5 && fileName[len(fileName)-5:] == "_test.go" {
				t.Fatalf("_test.go file should be ignored: %s", fileName)
			}
		}
	}
}

func TestPackageCache_SelectsFilesForBuildContext(t *testing.T) {
	pkgPath := t.TempDir()
	files := map[string]string{
		"platform_linux.go":   "package platform\n\nconst Name = \"linux\"\n",
		"platform_windows.go": "package platform\n\nconst Name = \"windows\"\n",
		"platform_test.go":    "package platform\n\nfunc broken( {\n",
		"excluded.go":         "//go:build wave11_never_enabled\n\npackage platform\n\nfunc broken( {\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(pkgPath, name), []byte(contents), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	linuxCache := NewPackageCacheWithSelector(buildselect.NewWithContext(build.Context{GOOS: "linux", GOARCH: "amd64"}))
	linuxPackages, err := linuxCache.GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("linux AST parsing returned error: %v", err)
	}
	assertSelectedFiles(t, linuxPackages, "platform_linux.go")

	windowsCache := NewPackageCacheWithSelector(buildselect.NewWithContext(build.Context{GOOS: "windows", GOARCH: "amd64"}))
	windowsPackages, err := windowsCache.GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("windows AST parsing returned error: %v", err)
	}
	assertSelectedFiles(t, windowsPackages, "platform_windows.go")
}

func TestPackageCache_ReturnsEmptyForTestOnlyPackage(t *testing.T) {
	pkgPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkgPath, "only_test.go"), []byte("package testonly\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	packages, err := NewPackageCache().GetPackageAST(pkgPath)
	if err != nil {
		t.Fatalf("GetPackageAST returned error: %v", err)
	}
	if len(packages) != 0 {
		t.Fatalf("expected no AST packages, got %d", len(packages))
	}
}

func assertSelectedFiles(t *testing.T, packages map[string]*ast.Package, expected string) {
	t.Helper()
	found := false
	for _, pkg := range packages {
		for fileName := range pkg.Files {
			if strings.HasSuffix(fileName, expected) {
				found = true
			}
			if strings.HasSuffix(fileName, "_test.go") || strings.HasSuffix(fileName, "windows.go") && expected != "platform_windows.go" || strings.HasSuffix(fileName, "linux.go") && expected != "platform_linux.go" {
				t.Fatalf("unexpected AST file %s", fileName)
			}
		}
	}
	if !found {
		t.Fatalf("expected AST file %s", expected)
	}
}
