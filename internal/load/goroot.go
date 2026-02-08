package load

import (
	"os/exec"
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
