package goas

import (
	"fmt"
	goparser "go/parser"
	"go/token"
	"strings"

	"github.com/delley/goas/internal/annotate"
	"github.com/delley/goas/internal/openapi"
)

// parseEntryPoint extracts metadata from the main file (title, version, servers, etc).
func (p *parser) parseEntryPoint() error {
	fset := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fset, p.MainFilePath, nil, goparser.ParseComments)
	if err != nil {
		return fmt.Errorf("can not parse general API information: %w", err)
	}

	// Security Scopes are defined at a different level in the hierarchy as where they need to end up in the OpenAPI structure,
	// so a temporary list is needed.
	oauthScopes := make(map[string]map[string]string)

	if fileTree.Comments != nil {
		for i := range fileTree.Comments {
			lineNum := fset.Position(fileTree.Comments[i].Pos()).Line
			for _, comment := range strings.Split(fileTree.Comments[i].Text(), "\n") {
				comment = strings.TrimSpace(comment)
				if len(comment) == 0 {
					continue
				}
				fields := strings.Fields(comment)
				if len(fields) == 0 || fields[0][0] != '@' {
					continue
				}
				attribute := strings.ToLower(fields[0])
				value := strings.TrimSpace(comment[len(fields[0]):])
				if len(value) == 0 {
					continue
				}
				p.parseGlobalServiceComments(attribute, value, lineNum, oauthScopes)
			}
		}
	}

	// Apply security scopes to their security schemes
	for scheme := range p.OpenAPI.Components.SecuritySchemes {
		if p.OpenAPI.Components.SecuritySchemes[scheme].Type == "oauth2" {
			if scopes, ok := oauthScopes[scheme]; ok {
				p.OpenAPI.Components.SecuritySchemes[scheme].OAuthFlows.ApplyScopes(scopes)
			}
		}
	}

	if len(p.OpenAPI.Servers) < 1 {
		p.OpenAPI.Servers = append(p.OpenAPI.Servers, openapi.ServerObject{URL: "/", Description: "Default Server URL"})
	}

	if p.OpenAPI.Info.Title == "" {
		return fmt.Errorf("info.title cannot not be empty")
	}
	if p.OpenAPI.Info.Version == "" {
		return fmt.Errorf("info.version cannot not be empty")
	}
	for i := range p.OpenAPI.Servers {
		if p.OpenAPI.Servers[i].URL == "" {
			return fmt.Errorf("servers[%d].url cannot not be empty", i)
		}
	}

	// Return accumulated parsing errors (like duplicate security schemes)
	if len(p.ParseErrors) > 0 {
		// For now, return the first error; can be enhanced to collect all into a multi-error
		return p.ParseErrors[0]
	}

	return nil
}

// parseGlobalServiceComments processes global service-level annotations.
// Non-fatal errors (like duplicate security scheme names) are accumulated in p.ParseErrors.
func (p *parser) parseGlobalServiceComments(attribute, value string, lineNum int, oauthScopes map[string]map[string]string) error {
	switch attribute {
	case "@version":
		p.OpenAPI.Info.Version = value
	case "@title":
		p.OpenAPI.Info.Title = value
	case "@description":
		if p.OpenAPI.Info.Description == nil {
			p.OpenAPI.Info.Description = &openapi.ReffableString{}
		}
		p.OpenAPI.Info.Description.Value = value
	case "@termsofserviceurl":
		p.OpenAPI.Info.TermsOfService = value
	case "@contactname":
		if p.OpenAPI.Info.Contact == nil {
			p.OpenAPI.Info.Contact = &openapi.ContactObject{}
		}
		p.OpenAPI.Info.Contact.Name = value
	case "@contactemail":
		if p.OpenAPI.Info.Contact == nil {
			p.OpenAPI.Info.Contact = &openapi.ContactObject{}
		}
		p.OpenAPI.Info.Contact.Email = value
	case "@contacturl":
		if p.OpenAPI.Info.Contact == nil {
			p.OpenAPI.Info.Contact = &openapi.ContactObject{}
		}
		p.OpenAPI.Info.Contact.URL = value
	case "@licensename":
		if p.OpenAPI.Info.License == nil {
			p.OpenAPI.Info.License = &openapi.LicenseObject{}
		}
		p.OpenAPI.Info.License.Name = value
	case "@licenseurl":
		if p.OpenAPI.Info.License == nil {
			p.OpenAPI.Info.License = &openapi.LicenseObject{}
		}
		p.OpenAPI.Info.License.URL = value
	case "@server":
		spec, err := annotate.ParseServerComment(value)
		if err != nil {
			return err
		}
		if spec.URL == "" {
			return nil
		}
		s := openapi.ServerObject{URL: spec.URL, Description: spec.Description}
		p.OpenAPI.Servers = append(p.OpenAPI.Servers, s)
	case "@security":
		spec, err := annotate.ParseSecurityComment(value)
		if err != nil {
			return err
		}
		if spec.Name == "" {
			return nil
		}
		security := map[string][]string{spec.Name: spec.Scopes}
		p.OpenAPI.Security = append(p.OpenAPI.Security, security)
	case "@securityscheme":
		err := p.parseSecuritySchemeComment(value, lineNum, oauthScopes)
		if err != nil {
			// Accumulate non-fatal errors instead of returning immediately
			p.ParseErrors = append(p.ParseErrors, err)
		}
		return nil
	case "@securityscope":
		return p.parseSecurityScopeComment(value, oauthScopes)
	case "@tags":
		return p.parseTagsComment("@Tags " + value)
	}
	return nil
}

// parseSecuritySchemeComment processes @securityscheme annotations.
// Returns an error if a security scheme with the same name is already defined.
func (p *parser) parseSecuritySchemeComment(value string, lineNum int, oauthScopes map[string]map[string]string) error {
	spec, err := annotate.ParseSecuritySchemeComment(value)
	if err != nil {
		return err
	}
	if spec.Name == "" {
		return nil
	}
	spec.LineNumber = lineNum

	// Check for duplicate security scheme name
	if existing, exists := p.OpenAPI.Components.SecuritySchemes[spec.Name]; exists {
		return fmt.Errorf("security scheme '%s' already defined at line %d", spec.Name, existing.LineNumber)
	}

	securityScheme := spec.ToOpenAPISecurityScheme()
	// Store line number in the OpenAPI object for error reporting
	if securityScheme != nil {
		securityScheme.LineNumber = lineNum
	}
	p.OpenAPI.Components.SecuritySchemes[spec.Name] = securityScheme
	return nil
}

// parseSecurityScopeComment processes @securityscope annotations.
func (p *parser) parseSecurityScopeComment(value string, oauthScopes map[string]map[string]string) error {
	spec, err := annotate.ParseSecurityScopeComment(value)
	if err != nil {
		return err
	}
	if spec.SchemeName == "" {
		return nil
	}
	if _, ok := oauthScopes[spec.SchemeName]; !ok {
		oauthScopes[spec.SchemeName] = make(map[string]string)
	}
	if spec.ScopeName == "" {
		return nil
	}
	oauthScopes[spec.SchemeName][spec.ScopeName] = spec.Description
	return nil
}

// parseTagsComment processes @tags annotations.
func (p *parser) parseTagsComment(comment string) error {
	t, err := annotate.ParseTags(comment)
	if err != nil {
		return err
	}
	p.OpenAPI.Tags = append(p.OpenAPI.Tags, *t)
	return nil
}
