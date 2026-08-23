package ast

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/delley/goas/internal/buildselect"
)

// PackageCache caches parsed AST packages by directory path.
type PackageCache struct {
	cache    map[string]map[string]*ast.Package
	selector *buildselect.Selector
}

func NewPackageCache() *PackageCache {
	return NewPackageCacheWithSelector(buildselect.New())
}

func NewPackageCacheWithSelector(selector *buildselect.Selector) *PackageCache {
	return &PackageCache{cache: map[string]map[string]*ast.Package{}, selector: selector}
}

func (c *PackageCache) GetPackageAST(pkgPath string) (map[string]*ast.Package, error) {
	if cache, ok := c.cache[pkgPath]; ok {
		return cache, nil
	}
	if c.selector == nil {
		return nil, fmt.Errorf("build selector is nil")
	}

	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("parse package %s: %w", pkgPath, err)
	}
	selected := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("parse package %s: %w", pkgPath, err)
		}
		matches, err := c.selector.MatchFile(pkgPath, info)
		if err != nil {
			return nil, fmt.Errorf("parse package %s file %s: %w", pkgPath, filepath.Base(info.Name()), err)
		}
		if matches {
			selected[info.Name()] = struct{}{}
		}
	}

	ignoreFileFilter := func(info os.FileInfo) bool {
		_, ok := selected[info.Name()]
		return ok
	}
	astPackages, err := goparser.ParseDir(token.NewFileSet(), pkgPath, ignoreFileFilter, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse package %s: %w", pkgPath, err)
	}
	c.cache[pkgPath] = astPackages
	return astPackages, nil
}
