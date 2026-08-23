package load

import (
	"fmt"
	"os"
	"path/filepath"
)

// ModuleContext holds the resolved module metadata and path configuration used
// during parser initialization. It is an internal boundary: load resolves
// filesystem and toolchain state, while parsing consumes the resulting values.
type ModuleContext struct {
	ModulePath     string
	ModuleName     string
	MainFilePath   string
	HandlerPath    string
	GoModFilePath  string
	FileRefPath    string
	GoModCachePath string
	GoRootSrcPath  string
}

func ResolveModuleContext(modulePath, mainFilePath, handlerPath, fileRefPath string) (*ModuleContext, error) {
	modulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return nil, err
	}

	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(modulePath, path)
	}

	mainFilePath = resolvePath(mainFilePath)
	handlerPath = resolvePath(handlerPath)
	fileRefPath = resolvePath(fileRefPath)

	moduleInfo, err := os.Stat(modulePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", modulePath, err)
	}
	if !moduleInfo.IsDir() {
		return nil, fmt.Errorf("modulePath should be a directory")
	}

	goModFilePath := filepath.Join(modulePath, "go.mod")
	goModFileInfo, err := os.Stat(goModFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goModFilePath, err)
	}
	if goModFileInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a file", goModFilePath)
	}

	moduleName, err := ModuleNameFromGoMod(goModFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot get module name from %s: %w", goModFilePath, err)
	}

	mainFilePath, err = ResolveMainFile(modulePath, mainFilePath)
	if err != nil {
		return nil, err
	}

	if handlerPath != "" {
		_, err := os.Stat(handlerPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, err
			}
			return nil, fmt.Errorf("cannot get information of %s: %s", handlerPath, err)
		}
	}
	fileRefPath = resolvePath(fileRefPath)

	goModCachePath, err := GoModCache()
	if err != nil {
		return nil, fmt.Errorf("cannot get GOMODCACHE: %w", err)
	}

	goRoot, err := GoRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot get GOROOT: %w", err)
	}
	goRootSrcPath := filepath.Join(goRoot, "src")
	if _, err := os.Stat(goRootSrcPath); err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goRootSrcPath, err)
	}
	goRootSrcInfo, err := os.Stat(goRootSrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goRootSrcPath, err)
	}
	if !goRootSrcInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a directory", goRootSrcPath)
	}

	return &ModuleContext{
		ModulePath:     modulePath,
		ModuleName:     moduleName,
		MainFilePath:   mainFilePath,
		HandlerPath:    handlerPath,
		GoModFilePath:  goModFilePath,
		FileRefPath:    fileRefPath,
		GoModCachePath: goModCachePath,
		GoRootSrcPath:  goRootSrcPath,
	}, nil
}

func ResolveMainFile(modulePath, mainFilePath string) (string, error) {
	if mainFilePath == "" {
		fns, err := filepath.Glob(filepath.Join(modulePath, "*.go"))
		if err != nil {
			return "", err
		}
		for _, fn := range fns {
			isMain, err := IsMainFile(fn)
			if err != nil {
				return "", fmt.Errorf("cannot parse Go file %s: %w", fn, err)
			}
			if isMain {
				return fn, nil
			}
		}
		return "", fmt.Errorf("main file not found in %s", modulePath)
	}

	mainFilePath, err := filepath.Abs(mainFilePath)
	if err != nil {
		return "", err
	}
	mainFileInfo, err := os.Stat(mainFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("cannot get information of %s: %s", mainFilePath, err)
	}
	if mainFileInfo.IsDir() {
		return "", fmt.Errorf("mainFilePath should not be a directory")
	}
	return mainFilePath, nil
}
