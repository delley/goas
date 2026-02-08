package load

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ModuleNameFromGoMod(t *testing.T) {
	t.Run("valid go.mod", func(t *testing.T) {
		dir := t.TempDir()
		goModPath := filepath.Join(dir, "go.mod")

		err := os.WriteFile(goModPath, []byte("module github.com/delley/goas\n\ngo 1.23\n"), 0o644)
		require.NoError(t, err)

		name, err := ModuleNameFromGoMod(goModPath)

		require.NoError(t, err)
		require.Equal(t, "github.com/delley/goas", name)
	})

	t.Run("file not found", func(t *testing.T) {
		name, err := ModuleNameFromGoMod("does-not-exist/go.mod")

		require.Error(t, err)
		require.Equal(t, "", name)
	})

	t.Run("missing module directive", func(t *testing.T) {
		dir := t.TempDir()
		goModPath := filepath.Join(dir, "go.mod")

		err := os.WriteFile(goModPath, []byte("go 1.23\n"), 0o644)
		require.NoError(t, err)

		name, err := ModuleNameFromGoMod(goModPath)

		require.Error(t, err)
		require.Equal(t, "", name)
	})
}
