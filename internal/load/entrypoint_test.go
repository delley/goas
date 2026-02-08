package load

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsMainFile(t *testing.T) {
	t.Run("package main with func main()", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "main.go")

		err := os.WriteFile(p, []byte(`package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`), 0o644)
		require.NoError(t, err)

		ok, err := IsMainFile(p)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("package main without func main()", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "notmain.go")

		err := os.WriteFile(p, []byte(`package main

func x() {}
`), 0o644)
		require.NoError(t, err)

		ok, err := IsMainFile(p)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("func main() but not package main", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "foo.go")

		err := os.WriteFile(p, []byte(`package foo

func main() {}
`), 0o644)
		require.NoError(t, err)

		ok, err := IsMainFile(p)
		require.NoError(t, err)
		require.False(t, ok)
	})
}
