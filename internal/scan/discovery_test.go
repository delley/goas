package scan

import (
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
