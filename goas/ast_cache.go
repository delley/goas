package goas

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"strings"
)

type astPackageCache struct {
	cache map[string]map[string]*ast.Package
}

func newASTPackageCache() *astPackageCache {
	return &astPackageCache{cache: map[string]map[string]*ast.Package{}}
}

func (c *astPackageCache) getPkgAst(pkgPath string) (map[string]*ast.Package, error) {
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
