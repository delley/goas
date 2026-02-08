package load

import (
	"bufio"
	"os"
	"strings"
)

func IsMainFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var isMainPackage, hasMainFunc bool

	bs := bufio.NewScanner(f)
	for bs.Scan() {
		l := bs.Text()
		if !isMainPackage && strings.HasPrefix(l, "package main") {
			isMainPackage = true
		}
		if !hasMainFunc && strings.HasPrefix(l, "func main()") {
			hasMainFunc = true
		}
		if isMainPackage && hasMainFunc {
			break
		}
	}
	if err := bs.Err(); err != nil {
		return false, err
	}

	return isMainPackage && hasMainFunc, nil
}
