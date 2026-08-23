package goas

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/delley/goas/internal/load"
	"github.com/delley/goas/internal/scan"
	module "golang.org/x/mod/modfile"
)

// parseModule discovers all packages in the module path and registers them.
func (p *parser) parseModule() error {
	moduleCtx := &load.ModuleContext{ModulePath: p.ModulePath, ModuleName: p.ModuleName}
	pkgSet, err := scan.DiscoverPackages(moduleCtx)
	if err != nil {
		return err
	}
	for _, discovered := range pkgSet.Packages {
		if err := p.contextErr(); err != nil {
			return err
		}
		p.KnownPkgs = append(p.KnownPkgs, pkg{
			Name: discovered.Name,
			Path: discovered.Path,
		})
		p.KnownNamePkg[discovered.Name] = &p.KnownPkgs[len(p.KnownPkgs)-1]
		p.KnownPathPkg[discovered.Path] = &p.KnownPkgs[len(p.KnownPkgs)-1]
	}
	return nil
}

// parseGoMod parses the go.mod file and registers all dependencies.
func (p *parser) parseGoMod() error {
	b, err := os.ReadFile(p.GoModFilePath)
	if err != nil {
		return err
	}
	goMod, err := module.ParseLax(p.GoModFilePath, b, fixer)
	if err != nil {
		return err
	}
	for i := range goMod.Require {
		if err := p.contextErr(); err != nil {
			return err
		}
		pathRunes := []rune{}
		for _, v := range goMod.Require[i].Mod.Path {
			if !unicode.IsUpper(v) {
				pathRunes = append(pathRunes, v)
				continue
			}
			pathRunes = append(pathRunes, '!')
			pathRunes = append(pathRunes, unicode.ToLower(v))
		}
		pkgName := goMod.Require[i].Mod.Path
		pkgPath := filepath.Join(p.GoModCachePath, string(pathRunes)+"@"+goMod.Require[i].Mod.Version)
		pkgName = filepath.ToSlash(pkgName)
		p.KnownPkgs = append(p.KnownPkgs, pkg{
			Name: pkgName,
			Path: pkgPath,
		})
		p.KnownNamePkg[pkgName] = &p.KnownPkgs[len(p.KnownPkgs)-1]
		p.KnownPathPkg[pkgPath] = &p.KnownPkgs[len(p.KnownPkgs)-1]

		walker := func(path string, info os.FileInfo, err error) error {
			if contextErr := p.contextErr(); contextErr != nil {
				return contextErr
			}
			if err != nil {
				return err
			}
			if info != nil && info.IsDir() {
				if strings.HasPrefix(strings.Trim(strings.TrimPrefix(path, p.ModulePath), "/"), ".git") {
					return nil
				}
				fns, err := filepath.Glob(filepath.Join(path, "*.go"))
				if len(fns) == 0 || err != nil {
					return nil
				}
				// p.debug(path)
				name := filepath.Join(pkgName, strings.TrimPrefix(path, pkgPath))
				name = filepath.ToSlash(name)
				p.KnownPkgs = append(p.KnownPkgs, pkg{
					Name: name,
					Path: path,
				})
				p.KnownNamePkg[name] = &p.KnownPkgs[len(p.KnownPkgs)-1]
				p.KnownPathPkg[path] = &p.KnownPkgs[len(p.KnownPkgs)-1]
			}
			return nil
		}
		filepath.Walk(pkgPath, walker)
	}
	if p.Debug {
		for i := range p.KnownPkgs {
			p.debug(p.KnownPkgs[i].Name, "->", p.KnownPkgs[i].Path)
		}
	}
	return nil
}

// parseGoRoot loads core packages from Go root source.
func (p *parser) parseGoRoot() error {
	core, err := load.CorePackages(p.GoRootSrcPath)
	if err != nil {
		return err
	}
	p.CorePkgs = core
	return nil
}

// fixer is a helper for module.ParseLax to handle version fixing.
func fixer(path, version string) (string, error) {
	return version, nil
}
