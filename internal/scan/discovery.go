package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/delley/goas/internal/load"
)

// Package describes a Go package discovered by the module scan.
type Package struct {
	Name string
	Path string
}

// PackageSet is the deterministic set of discovered packages.
type PackageSet struct {
	Packages []Package
}

func DiscoverPackages(ctx *load.ModuleContext) (*PackageSet, error) {
	if ctx == nil {
		return nil, nil
	}

	result := &PackageSet{Packages: []Package{}}
	seen := map[string]struct{}{}

	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || !info.IsDir() {
			return nil
		}
		// Skip excluded directories: .git, vendor, node_modules, and similar metadata
		relPath := strings.TrimPrefix(path, ctx.ModulePath)
		relPath = strings.TrimPrefix(relPath, "/")
		dir := filepath.Base(path)
		if strings.HasPrefix(dir, ".") || dir == "vendor" || dir == "node_modules" {
			return filepath.SkipDir
		}
		fns, err := filepath.Glob(filepath.Join(path, "*.go"))
		if len(fns) == 0 || err != nil {
			return nil
		}

		name := filepath.Join(ctx.ModuleName, strings.TrimPrefix(path, ctx.ModulePath))
		name = filepath.ToSlash(name)
		if name == "." || name == "/" {
			name = ctx.ModuleName
		}
		if _, ok := seen[path]; ok {
			return nil
		}
		seen[path] = struct{}{}
		result.Packages = append(result.Packages, Package{Name: name, Path: path})
		return nil
	}
	if err := filepath.Walk(ctx.ModulePath, walk); err != nil {
		return nil, err
	}

	sort.Slice(result.Packages, func(i, j int) bool {
		if result.Packages[i].Path == result.Packages[j].Path {
			return result.Packages[i].Name < result.Packages[j].Name
		}
		return result.Packages[i].Path < result.Packages[j].Path
	})
	return result, nil
}
