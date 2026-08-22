package imports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryTracksExternalAliasesWithoutDuplicates(t *testing.T) {
	registry := NewRegistry()
	registry.Record("example.com/app", "github.com/acme/types", "types")
	registry.Record("example.com/app", "github.com/acme/types", "types")

	require.Equal(t, "types", registry.AliasFor("example.com/app", "github.com/acme/types"))
	require.Len(t, registry.PkgNameImportedPkgAlias["example.com/app"]["types"], 1)
}

func TestResolveAlias(t *testing.T) {
	require.Equal(t, "custom", ResolveAlias("github.com/acme/types", "custom"))
	require.Equal(t, "types", ResolveAlias("github.com/acme/types", ""))
	require.Equal(t, "types", ResolveAlias("github.com/acme/types", "."))
	require.Equal(t, "types", ResolveAlias("github.com/acme/types", "_"))
}
