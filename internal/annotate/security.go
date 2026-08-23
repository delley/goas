package annotate

import (
	"fmt"
	"strings"

	"github.com/delley/goas/internal/openapi"
)

// SecuritySchemeSpec captures a parsed @SecurityScheme comment.
type SecuritySchemeSpec struct {
	Name        string
	Type        string
	Flow        string
	In          string
	APIKeyName  string
	Scheme      string
	Description string
	OAuthFlows  *SecuritySchemeOAuthFlowsSpec
	LineNumber  int // Line number where this scheme was defined
}

// SecuritySchemeOAuthFlowsSpec captures OAuth2 flow metadata parsed from a security scheme.
type SecuritySchemeOAuthFlowsSpec struct {
	AuthorizationCode     *SecuritySchemeOAuthFlowSpec
	Implicit              *SecuritySchemeOAuthFlowSpec
	ResourceOwnerPassword *SecuritySchemeOAuthFlowSpec
	ClientCredentials     *SecuritySchemeOAuthFlowSpec
}

// SecuritySchemeOAuthFlowSpec captures OAuth flow configuration.
type SecuritySchemeOAuthFlowSpec struct {
	AuthorizationURL string
	TokenURL         string
	Scopes           map[string]string
}

// ServerSpec captures a parsed @Server comment.
type ServerSpec struct {
	URL         string
	Description string
}

// SecurityRequirementSpec captures a parsed @Security comment.
type SecurityRequirementSpec struct {
	Name   string
	Scopes []string
}

func ParseServerComment(comment string) (ServerSpec, error) {
	fields := strings.Fields(comment)
	if len(fields) == 0 {
		return ServerSpec{}, nil
	}

	url := fields[0]
	description := strings.TrimSpace(strings.TrimPrefix(comment, url))
	return ServerSpec{URL: url, Description: description}, nil
}

func ParseSecurityComment(comment string) (SecurityRequirementSpec, error) {
	fields := strings.Fields(comment)
	if len(fields) < 2 {
		return SecurityRequirementSpec{}, nil
	}

	return SecurityRequirementSpec{
		Name:   fields[0],
		Scopes: fields[1:],
	}, nil
}

func ParseSecuritySchemeComment(comment string) (SecuritySchemeSpec, error) {
	fields := strings.Fields(comment)
	if len(fields) < 2 {
		return SecuritySchemeSpec{}, fmt.Errorf("security scheme requires a name and type")
	}

	schemeName := fields[0]
	schemeTypeToken := fields[1]
	lowerType := strings.ToLower(schemeTypeToken)

	spec := SecuritySchemeSpec{
		Name: schemeName,
		Type: schemeTypeToken,
		Flow: schemeTypeToken,
	}

	if strings.Contains(lowerType, "oauth2") {
		spec.Type = "oauth2"
	}

	switch lowerType {
	case "http":
		return parseHTTPSSecurityScheme(spec, fields[2:])
	case "apikey":
		return parseAPIKeySecurityScheme(spec, fields[2:])
	case "openidconnect":
		return parseOpenIDConnectSecurityScheme(spec, fields[2:])
	case "oauth2authcode":
		spec.Flow = "oauth2AuthCode"
		spec.OAuthFlows = &SecuritySchemeOAuthFlowsSpec{
			AuthorizationCode: &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}},
		}
		if len(fields) < 4 {
			return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 authorization code scheme %q requires an authorization URL and a token URL", spec.Name)
		}
		spec.OAuthFlows.AuthorizationCode.AuthorizationURL = fields[2]
		spec.OAuthFlows.AuthorizationCode.TokenURL = fields[3]
		spec.Description = strings.Join(fields[4:], " ")
	case "oauth2implicit":
		spec.Flow = "oauth2Implicit"
		spec.OAuthFlows = &SecuritySchemeOAuthFlowsSpec{
			Implicit: &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}},
		}
		if len(fields) < 3 {
			return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 implicit scheme %q requires an authorization URL", spec.Name)
		}
		spec.OAuthFlows.Implicit.AuthorizationURL = fields[2]
		spec.Description = strings.Join(fields[3:], " ")
	case "oauth2resourceownercredentials":
		spec.Flow = "oauth2ResourceOwnerCredentials"
		spec.OAuthFlows = &SecuritySchemeOAuthFlowsSpec{
			ResourceOwnerPassword: &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}},
		}
		if len(fields) < 3 {
			return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 resource owner credentials scheme %q requires a token URL", spec.Name)
		}
		spec.OAuthFlows.ResourceOwnerPassword.TokenURL = fields[2]
		spec.Description = strings.Join(fields[3:], " ")
	case "oauth2clientcredentials":
		spec.Flow = "oauth2ClientCredentials"
		spec.OAuthFlows = &SecuritySchemeOAuthFlowsSpec{
			ClientCredentials: &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}},
		}
		if len(fields) < 3 {
			return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 client credentials scheme %q requires a token URL", spec.Name)
		}
		spec.OAuthFlows.ClientCredentials.TokenURL = fields[2]
		spec.Description = strings.Join(fields[3:], " ")
	case "oauth2":
		spec.Type = "oauth2"
		spec.OAuthFlows = &SecuritySchemeOAuthFlowsSpec{}
		if len(fields) < 3 {
			return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 security scheme %q requires a flow", spec.Name)
		}

		flowName := strings.ToLower(fields[2])
		switch flowName {
		case "oauth2authcode":
			spec.Flow = "oauth2AuthCode"
			spec.OAuthFlows.AuthorizationCode = &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}}
			if len(fields) < 5 {
				return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 authorization code scheme %q requires an authorization URL and a token URL", spec.Name)
			}
			spec.OAuthFlows.AuthorizationCode.AuthorizationURL = fields[3]
			spec.OAuthFlows.AuthorizationCode.TokenURL = fields[4]
			spec.Description = strings.Join(fields[5:], " ")
		case "oauth2implicit":
			spec.Flow = "oauth2Implicit"
			spec.OAuthFlows.Implicit = &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}}
			if len(fields) < 4 {
				return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 implicit scheme %q requires an authorization URL", spec.Name)
			}
			spec.OAuthFlows.Implicit.AuthorizationURL = fields[3]
			spec.Description = strings.Join(fields[4:], " ")
		case "oauth2resourceownercredentials":
			spec.Flow = "oauth2ResourceOwnerCredentials"
			spec.OAuthFlows.ResourceOwnerPassword = &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}}
			if len(fields) < 4 {
				return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 resource owner credentials scheme %q requires a token URL", spec.Name)
			}
			spec.OAuthFlows.ResourceOwnerPassword.TokenURL = fields[3]
			spec.Description = strings.Join(fields[4:], " ")
		case "oauth2clientcredentials":
			spec.Flow = "oauth2ClientCredentials"
			spec.OAuthFlows.ClientCredentials = &SecuritySchemeOAuthFlowSpec{Scopes: map[string]string{}}
			if len(fields) < 4 {
				return SecuritySchemeSpec{}, fmt.Errorf("OAuth2 client credentials scheme %q requires a token URL", spec.Name)
			}
			spec.OAuthFlows.ClientCredentials.TokenURL = fields[3]
			spec.Description = strings.Join(fields[4:], " ")
		default:
			return SecuritySchemeSpec{}, fmt.Errorf("unsupported OAuth2 flow %q", fields[2])
		}
	default:
		return SecuritySchemeSpec{}, fmt.Errorf("unsupported security scheme type %q", schemeTypeToken)
	}

	return spec, nil
}

func parseHTTPSSecurityScheme(spec SecuritySchemeSpec, fields []string) (SecuritySchemeSpec, error) {
	if len(fields) < 1 {
		return SecuritySchemeSpec{}, fmt.Errorf("HTTP security scheme %q requires a scheme", spec.Name)
	}
	spec.Type = "http"
	spec.Scheme = fields[0]
	spec.Description = strings.Join(fields[1:], " ")
	return spec, nil
}

func parseAPIKeySecurityScheme(spec SecuritySchemeSpec, fields []string) (SecuritySchemeSpec, error) {
	if len(fields) < 2 {
		return SecuritySchemeSpec{}, fmt.Errorf("API key security scheme %q requires an in location and a name", spec.Name)
	}
	spec.Type = "apiKey"
	spec.In = fields[0]
	spec.APIKeyName = fields[1]
	spec.Description = strings.Join(fields[2:], " ")
	return spec, nil
}

func parseOpenIDConnectSecurityScheme(spec SecuritySchemeSpec, fields []string) (SecuritySchemeSpec, error) {
	if len(fields) < 1 {
		return SecuritySchemeSpec{}, fmt.Errorf("OpenID Connect security scheme %q requires a URL", spec.Name)
	}
	spec.Type = "openIdConnect"
	spec.Scheme = fields[0]
	spec.Description = strings.Join(fields[1:], " ")
	return spec, nil
}

func (s SecuritySchemeSpec) ToOpenAPISecurityScheme() *openapi.SecuritySchemeObject {
	scheme := &openapi.SecuritySchemeObject{Type: s.Type}
	switch s.Type {
	case "http":
		scheme.Scheme = s.Scheme
		scheme.Description = s.Description
	case "apiKey":
		scheme.In = s.In
		scheme.Name = s.APIKeyName
		scheme.Description = s.Description
	case "openIdConnect":
		scheme.OpenIDConnectURL = s.Scheme
		scheme.Description = s.Description
	case "oauth2":
		scheme.Type = "oauth2"
		scheme.Description = s.Description
		scheme.OAuthFlows = &openapi.SecuritySchemeOAuthObject{}
		if s.OAuthFlows != nil {
			if s.OAuthFlows.AuthorizationCode != nil {
				scheme.OAuthFlows.AuthorizationCode = &openapi.SecuritySchemeOAuthFlowObject{
					AuthorizationURL: s.OAuthFlows.AuthorizationCode.AuthorizationURL,
					TokenURL:         s.OAuthFlows.AuthorizationCode.TokenURL,
					Scopes:           make(map[string]string),
				}
			}
			if s.OAuthFlows.Implicit != nil {
				scheme.OAuthFlows.Implicit = &openapi.SecuritySchemeOAuthFlowObject{
					AuthorizationURL: s.OAuthFlows.Implicit.AuthorizationURL,
					Scopes:           make(map[string]string),
				}
			}
			if s.OAuthFlows.ResourceOwnerPassword != nil {
				scheme.OAuthFlows.ResourceOwnerPassword = &openapi.SecuritySchemeOAuthFlowObject{
					TokenURL: s.OAuthFlows.ResourceOwnerPassword.TokenURL,
					Scopes:   make(map[string]string),
				}
			}
			if s.OAuthFlows.ClientCredentials != nil {
				scheme.OAuthFlows.ClientCredentials = &openapi.SecuritySchemeOAuthFlowObject{
					TokenURL: s.OAuthFlows.ClientCredentials.TokenURL,
					Scopes:   make(map[string]string),
				}
			}
		}
	}
	return scheme
}

// SecurityScopeSpec captures a parsed @SecurityScope comment.
type SecurityScopeSpec struct {
	SchemeName  string
	ScopeName   string
	Description string
}

func ParseSecurityScopeComment(comment string) (SecurityScopeSpec, error) {
	fields := strings.Fields(comment)
	if len(fields) < 3 {
		return SecurityScopeSpec{}, fmt.Errorf("security scope requires a scheme name, scope name, and description")
	}
	return SecurityScopeSpec{
		SchemeName:  fields[0],
		ScopeName:   fields[1],
		Description: strings.Join(fields[2:], " "),
	}, nil
}
