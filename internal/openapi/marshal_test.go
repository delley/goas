package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalIndent(t *testing.T) {
	spec := &OpenAPIObject{
		OpenAPI: OpenAPIVersion,
		Info:    InfoObject{Title: "t", Version: "v"},
		Paths:   PathsObject{},
	}
	b, err := Marshal(spec, MarshalOptions{Indent: "  "})
	require.NoError(t, err)
	require.Contains(t, string(b), "\n  ")
}
