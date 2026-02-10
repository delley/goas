package load

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

func GoRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func CorePackages(goRootSrcPath string) (map[string]bool, error) {
	if strings.TrimSpace(goRootSrcPath) == "" {
		return nil, errors.New("goRootSrcPath is empty")
	}

	out := make(map[string]bool)

	err := filepath.WalkDir(goRootSrcPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}

		fns, err := filepath.Glob(filepath.Join(path, "*.go"))
		if err != nil || len(fns) == 0 {
			return nil
		}

		rel, err := filepath.Rel(goRootSrcPath, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}

		name := filepath.ToSlash(rel)
		name = strings.TrimPrefix(name, "./")
		name = strings.TrimPrefix(name, "/")
		if name == "." || name == "" {
			// If there are .go files directly in the root, you can decide whether to register "" or "."
			// Here I ignore.
			return nil
		}

		out[name] = true
		return nil
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}
