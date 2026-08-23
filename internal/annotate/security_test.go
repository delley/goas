package annotate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecuritySchemeComment(t *testing.T) {
	t.Run("oauth2 authorization code", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("oauth2 oauth2AuthCode https://example.com/auth https://example.com/token OAuth login")
		require.NoError(t, err)
		require.Equal(t, "oauth2", spec.Name)
		require.Equal(t, "oauth2", spec.Type)
		require.NotNil(t, spec.OAuthFlows)
		require.NotNil(t, spec.OAuthFlows.AuthorizationCode)
		require.Equal(t, "https://example.com/auth", spec.OAuthFlows.AuthorizationCode.AuthorizationURL)
		require.Equal(t, "https://example.com/token", spec.OAuthFlows.AuthorizationCode.TokenURL)
		require.Equal(t, "OAuth login", spec.Description)
	})

	t.Run("http scheme accepts uppercase type names", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("MyApiAuth HTTP bearer Login com seu token")
		require.NoError(t, err)
		require.Equal(t, "MyApiAuth", spec.Name)
		require.Equal(t, "http", spec.Type)
		require.Equal(t, "bearer", spec.Scheme)
		require.Equal(t, "Login com seu token", spec.Description)
	})

	t.Run("rejects incomplete fixed-parameter schemes", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth apiKey header")
		require.EqualError(t, err, "API key security scheme \"MyApiAuth\" requires an in location and a name")

		_, err = ParseSecuritySchemeComment("MyApiAuth http")
		require.EqualError(t, err, "HTTP security scheme \"MyApiAuth\" requires a scheme")
	})

	t.Run("api key description keeps spaces", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("MyApiAuth apiKey header X-MyCustomHeader Login com seu token")
		require.NoError(t, err)
		require.Equal(t, "MyApiAuth", spec.Name)
		require.Equal(t, "apiKey", spec.Type)
		require.Equal(t, "header", spec.In)
		require.Equal(t, "X-MyCustomHeader", spec.APIKeyName)
		require.Equal(t, "Login com seu token", spec.Description)
	})

	t.Run("oauth2 generic flow keeps other flow variants valid", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("MyApiAuth oauth2 oauth2Implicit https://example.com/auth")
		require.NoError(t, err)
		require.Equal(t, "oauth2", spec.Type)
		require.NotNil(t, spec.OAuthFlows)
		require.NotNil(t, spec.OAuthFlows.Implicit)
		require.Equal(t, "https://example.com/auth", spec.OAuthFlows.Implicit.AuthorizationURL)
	})

	t.Run("oauth2 generic flow rejects partial input", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth oauth2 oauth2ClientCredentials")
		require.EqualError(t, err, "OAuth2 client credentials scheme \"MyApiAuth\" requires a token URL")
	})

	t.Run("oauth2 implicit flow parses correctly", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("oauth2implicit oauth2Implicit https://example.com/authorize OAuth implicit flow")
		require.NoError(t, err)
		require.Equal(t, "oauth2implicit", spec.Name)
		require.Equal(t, "oauth2", spec.Type)
		require.NotNil(t, spec.OAuthFlows)
		require.NotNil(t, spec.OAuthFlows.Implicit)
		require.Equal(t, "https://example.com/authorize", spec.OAuthFlows.Implicit.AuthorizationURL)
		require.Equal(t, "OAuth implicit flow", spec.Description)
	})

	t.Run("oauth2 implicit flow requires authorization url", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth oauth2Implicit")
		require.EqualError(t, err, "OAuth2 implicit scheme \"MyApiAuth\" requires an authorization URL")
	})

	t.Run("oauth2 resource owner password flow parses correctly", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("oauth2resourceowner oauth2ResourceOwnerCredentials https://example.com/token OAuth resource owner flow")
		require.NoError(t, err)
		require.Equal(t, "oauth2resourceowner", spec.Name)
		require.Equal(t, "oauth2", spec.Type)
		require.NotNil(t, spec.OAuthFlows)
		require.NotNil(t, spec.OAuthFlows.ResourceOwnerPassword)
		require.Equal(t, "https://example.com/token", spec.OAuthFlows.ResourceOwnerPassword.TokenURL)
		require.Equal(t, "OAuth resource owner flow", spec.Description)
	})

	t.Run("oauth2 resource owner password flow requires token url", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth oauth2ResourceOwnerCredentials")
		require.EqualError(t, err, "OAuth2 resource owner credentials scheme \"MyApiAuth\" requires a token URL")
	})

	t.Run("oauth2 client credentials flow parses correctly", func(t *testing.T) {
		spec, err := ParseSecuritySchemeComment("oauth2client oauth2ClientCredentials https://example.com/token OAuth client credentials flow")
		require.NoError(t, err)
		require.Equal(t, "oauth2client", spec.Name)
		require.Equal(t, "oauth2", spec.Type)
		require.NotNil(t, spec.OAuthFlows)
		require.NotNil(t, spec.OAuthFlows.ClientCredentials)
		require.Equal(t, "https://example.com/token", spec.OAuthFlows.ClientCredentials.TokenURL)
		require.Equal(t, "OAuth client credentials flow", spec.Description)
	})

	t.Run("oauth2 client credentials flow requires token url", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth oauth2ClientCredentials")
		require.EqualError(t, err, "OAuth2 client credentials scheme \"MyApiAuth\" requires a token URL")
	})

	t.Run("oauth2 authorization code flow requires both urls", func(t *testing.T) {
		_, err := ParseSecuritySchemeComment("MyApiAuth oauth2AuthCode https://example.com/auth")
		require.EqualError(t, err, "OAuth2 authorization code scheme \"MyApiAuth\" requires an authorization URL and a token URL")
	})
}

func TestParseSecurityScopeComment(t *testing.T) {
	spec, err := ParseSecurityScopeComment("oauth2 read Read access")
	require.NoError(t, err)
	require.Equal(t, "oauth2", spec.SchemeName)
	require.Equal(t, "read", spec.ScopeName)
	require.Equal(t, "Read access", spec.Description)

	_, err = ParseSecurityScopeComment("oauth2 read")
	require.EqualError(t, err, "security scope requires a scheme name, scope name, and description")
}

func TestParseServerComment(t *testing.T) {
	server, err := ParseServerComment("https://api.example.com Production API")
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com", server.URL)
	require.Equal(t, "Production API", server.Description)
}

func TestParseSecurityComment(t *testing.T) {
	requirement, err := ParseSecurityComment("ApiKey read write")
	require.NoError(t, err)
	require.Equal(t, "ApiKey", requirement.Name)
	require.Equal(t, []string{"read", "write"}, requirement.Scopes)
}
