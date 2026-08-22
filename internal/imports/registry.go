package imports

import "strings"

// Registry tracks package aliases per importing package.
type Registry struct {
	PkgNameImportedPkgAlias map[string]map[string][]string
}

func NewRegistry() *Registry {
	return &Registry{PkgNameImportedPkgAlias: map[string]map[string][]string{}}
}

func (r *Registry) Record(pkgName, importedPkgName, alias string) {
	if r.PkgNameImportedPkgAlias[pkgName] == nil {
		r.PkgNameImportedPkgAlias[pkgName] = map[string][]string{}
	}
	imports := r.PkgNameImportedPkgAlias[pkgName][alias]
	for _, imported := range imports {
		if imported == importedPkgName {
			return
		}
	}
	r.PkgNameImportedPkgAlias[pkgName][alias] = append(imports, importedPkgName)
}

func (r *Registry) AliasFor(pkgName, importedPkgName string) string {
	for alias, packageNames := range r.PkgNameImportedPkgAlias[pkgName] {
		for _, imported := range packageNames {
			if imported == importedPkgName {
				return alias
			}
		}
	}
	return ""
}

func ResolveAlias(astImportPath string, explicitAlias string) string {
	if explicitAlias != "" && explicitAlias != "." && explicitAlias != "_" {
		return explicitAlias
	}
	parts := strings.Split(astImportPath, "/")
	return parts[len(parts)-1]
}
