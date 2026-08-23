package ast

import (
	"path/filepath"
	"testing"
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
