package load

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveModuleContext_ResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package main\n"), 0o644))

	ctx, err := ResolveModuleContext(dir, "main.go", "handler.go", "docs/overview.md")
	require.NoError(t, err)
	require.Equal(t, dir, ctx.ModulePath)
	require.Equal(t, filepath.Join(dir, "go.mod"), ctx.GoModFilePath)
	require.Equal(t, filepath.Join(dir, "main.go"), ctx.MainFilePath)
	require.Equal(t, filepath.Join(dir, "handler.go"), ctx.HandlerPath)
	require.Equal(t, filepath.Join(dir, "docs", "overview.md"), ctx.FileRefPath)
}

func TestResolveModuleContext_AutodetectsMainFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server.go"), []byte("package main\nfunc main() {}\n"), 0o644))

	ctx, err := ResolveModuleContext(dir, "", "", "")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "server.go"), ctx.MainFilePath)
}

func TestResolveModuleContext_ErrorsWhenMainFileIsMissing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "library.go"), []byte("package library\n"), 0o644))

	_, err := ResolveModuleContext(dir, "", "", "")
	require.EqualError(t, err, "main file not found in "+dir)
}

func TestResolveModuleContext_ErrorsWhenAutodetectedGoFileIsInvalid(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package main\nfunc main( {\n"), 0o644))

	_, err := ResolveModuleContext(dir, "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot parse Go file")
}

func TestResolveModuleContext_ErrorsOnInvalidMainFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644))

	_, err := ResolveModuleContext(dir, filepath.Join(dir, "nested"), "", "")
	require.Error(t, err)
}
