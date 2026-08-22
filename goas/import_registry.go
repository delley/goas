package goas

import "strings"

type importRegistry struct {
	pkgNameImportedPkgAlias map[string]map[string][]string
}

func newImportRegistry() *importRegistry {
	return &importRegistry{pkgNameImportedPkgAlias: map[string]map[string][]string{}}
}

func (r *importRegistry) recordImport(pkgName, importedPkgName, importedPkgAlias string) {
	if r.pkgNameImportedPkgAlias[pkgName] == nil {
		r.pkgNameImportedPkgAlias[pkgName] = map[string][]string{}
	}
	if _, ok := r.pkgNameImportedPkgAlias[pkgName][importedPkgAlias]; !ok {
		r.pkgNameImportedPkgAlias[pkgName][importedPkgAlias] = []string{}
	}
	for _, v := range r.pkgNameImportedPkgAlias[pkgName][importedPkgAlias] {
		if v == importedPkgName {
			return
		}
	}
	r.pkgNameImportedPkgAlias[pkgName][importedPkgAlias] = append(r.pkgNameImportedPkgAlias[pkgName][importedPkgAlias], importedPkgName)
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
	if astImportName != nil && astImportName.Name != "." && astImportName.Name != "_" {
		return astImportName.String()
	}
	s := strings.Split(astImportPath, "/")
	return s[len(s)-1]
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
