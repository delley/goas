package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaRef(t *testing.T) {
	require.Equal(t, "", SchemaRef(""))
	require.Equal(t, "#/components/schemas/User", SchemaRef("User"))
	require.Equal(t, "#/components/schemas/User", SchemaRef("#/components/schemas/User"))
	require.Equal(t, "#/components/schemas/User/Type", SchemaRef("User\\Type"))
	require.Equal(t, "#/components/schemas/User/Type", SchemaRef("#/components/schemas/User\\Type"))
}

func TestNormalizeComponentSchemaKey(t *testing.T) {
	require.Equal(t, "User/Type", NormalizeComponentSchemaKey("User\\Type"))
	require.Equal(t, "User/Type", NormalizeComponentSchemaKey("User/Type"))
	require.Equal(t, "User", NormalizeComponentSchemaKey("User"))
}
