package goas

import internalImports "github.com/delley/goas/internal/imports"

type importRegistry struct {
	registry                *internalImports.Registry
	pkgNameImportedPkgAlias map[string]map[string][]string
}

func newImportRegistry() *importRegistry {
	registry := internalImports.NewRegistry()
	return &importRegistry{registry: registry, pkgNameImportedPkgAlias: registry.PkgNameImportedPkgAlias}
}

func (r *importRegistry) recordImport(pkgName, importedPkgName, importedPkgAlias string) {
	r.registry.Record(pkgName, importedPkgName, importedPkgAlias)
}

func (r *importRegistry) importAliasFor(pkgName, importedPkgName string) string {
	for alias, imports := range r.pkgNameImportedPkgAlias[pkgName] {
		for _, imported := range imports {
			if imported == importedPkgName {
				return alias
			}
		}
	}
	return ""
}

func (r *importRegistry) resolveImportedPkgAlias(pkgName string, astImportPath string, astImportName *importName) string {
	if astImportName == nil {
		return internalImports.ResolveAlias(astImportPath, "")
	}
	return internalImports.ResolveAlias(astImportPath, astImportName.Name)
}

type importName struct {
	Name string
}

func (n *importName) String() string {
	if n == nil {
		return ""
	}
	return n.Name
}
