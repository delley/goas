package ast

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"strings"
)

// PackageCache caches parsed AST packages by directory path.
type PackageCache struct {
	cache map[string]map[string]*ast.Package
}

func NewPackageCache() *PackageCache {
	return &PackageCache{cache: map[string]map[string]*ast.Package{}}
}

func (c *PackageCache) GetPackageAST(pkgPath string) (map[string]*ast.Package, error) {
	if cache, ok := c.cache[pkgPath]; ok {
		return cache, nil
	}

	ignoreFileFilter := func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}

	astPackages, err := goparser.ParseDir(token.NewFileSet(), pkgPath, ignoreFileFilter, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse package %s: %w", pkgPath, err)
	}
	c.cache[pkgPath] = astPackages
	return astPackages, nil
}
