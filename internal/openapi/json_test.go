package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReffableString(t *testing.T) {
	t.Run("marshals a string", func(t *testing.T) {
		result, err := json.Marshal(&ReffableString{Value: "Foobarbazington Esq."})

		require.NoError(t, err)
		require.Equal(t, "\"Foobarbazington Esq.\"", string(result))
	})

	t.Run("marshals an object", func(t *testing.T) {
		result, err := json.Marshal(&ReffableString{Value: "$ref:foo/bar/baz"})

		require.NoError(t, err)
		require.Equal(t, "{\"$ref\":\"foo/bar/baz\"}", string(result))
	})

	t.Run("missing url", func(t *testing.T) {
		_, err := json.Marshal(&ReffableString{Value: "$ref:"})

		require.Error(t, err)
	})
}

func TestOpenAPIObjectComponentsJSONCompatibility(t *testing.T) {
	spec := &OpenAPIObject{
		OpenAPI: OpenAPIVersion,
		Info:    InfoObject{Title: "t", Version: "v"},
		Paths:   PathsObject{},
		Components: ComponentsObject{
			Schemas: map[string]*SchemaObject{
				"User": {Type: strPtr("object")},
			},
		},
	}

	result, err := json.Marshal(spec)
	require.NoError(t, err)
	require.Contains(t, string(result), "\"components\":")
	require.Contains(t, string(result), "\"schemas\":")
}

func TestSecuritySchemeOAuthJSONCompatibility(t *testing.T) {
	scheme := &SecuritySchemeObject{
		Type: "oauth2",
		OAuthFlows: &SecuritySchemeOAuthObject{
			AuthorizationCode: &SecuritySchemeOAuthFlowObject{
				AuthorizationURL: "https://example.com/auth",
				TokenURL:         "https://example.com/token",
				Scopes:           map[string]string{"read": "Read access"},
			},
		},
	}

	result, err := json.Marshal(scheme)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"oauth2","flows":{"authorizationCode":{"authorizationUrl":"https://example.com/auth","tokenUrl":"https://example.com/token","scopes":{"read":"Read access"}}}}`, string(result))
}

func strPtr(v string) *string {
	return &v
}
