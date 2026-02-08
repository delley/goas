package load

import (
	"fmt"
	"os"

	"golang.org/x/mod/modfile"
)

func ModuleNameFromGoMod(goModPath string) (string, error) {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	f, err := modfile.Parse(goModPath, b, nil)
	if err != nil {
		return "", err
	}
	if f.Module == nil {
		return "", fmt.Errorf("no module directive in %s", goModPath)
	}
	return f.Module.Mod.Path, nil
}
