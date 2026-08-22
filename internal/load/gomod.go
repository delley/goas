package load

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

func GoModCache() (string, error) {
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return "", err
	}
	cache := strings.TrimSpace(string(output))
	if cache == "" {
		return "", fmt.Errorf("go env GOMODCACHE returned an empty path")
	}
	return cache, nil
}

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
