package goas

import (
	"context"
	"encoding/json"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	astpkg "github.com/delley/goas/internal/ast"
	internalImports "github.com/delley/goas/internal/imports"
	"github.com/delley/goas/internal/openapi"
	internalSchema "github.com/delley/goas/internal/schema"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestGenerateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New().Generate(ctx, Options{ModulePath: "../example"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestNewParserLegacyCompatibility(t *testing.T) {
	p, err := NewParser("../example", "main.go", "", "", false, true, false)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotEmpty(t, p.ModulePath)
	require.Equal(t, filepath.Join(p.ModulePath, "main.go"), p.MainFilePath)
}

func TestNewParserIsDeprecated(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "parser.go", nil, goparser.ParseComments)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "NewParser" {
			continue
		}
		require.NotNil(t, fn.Doc)
		require.Contains(t, fn.Doc.Text(), "Deprecated:")
		return
	}

	t.Fatal("NewParser declaration not found")
}

func TestNewParserUsesToolchainModuleCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "not-created")
	t.Setenv("GOPATH", filepath.Join(t.TempDir(), "not-gopath"))
	t.Setenv("GOMODCACHE", cachePath)

	p, err := newParser("../example", "main.go", "", "", false, true, false)
	require.NoError(t, err)
	require.Equal(t, cachePath, p.GoModCachePath)
	require.Equal(t, filepath.Join(p.ModulePath, "main.go"), p.MainFilePath)
}

func setupParser() (*parser, error) {
	return newParser("../example/", "../example/main.go", "", "", false, true, false)
}
func Test_parseSchemaObject_missingTypeReturnsError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		p, err := setupParser()
		if err != nil {
			t.Fatalf("setupParser() error = %v", err)
		}
		_, err = p.parseSchemaObject("../example", "main", "DefinitelyMissingType", true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=Test_parseSchemaObject_missingTypeReturnsError")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "missing type should return an error instead of terminating the process: %s", string(out))
	require.NotContains(t, string(out), "Can not find definition")
}

func TestExample(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	expected, _ := os.ReadFile("../example/example.json")
	require.JSONEq(t, string(expected), string(bts))
}

func TestShowHiddenExample(t *testing.T) {
	p, err := newParser("../example/", "../example/main.go", "", "", false, true, true)
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	expected, _ := os.ReadFile("../example/example-show-hidden.json")
	require.JSONEq(t, string(expected), string(bts))
}

func TestDeterministic(t *testing.T) {
	var allOutputs []string
	for i := 0; i < 10; i++ {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.parse()
		require.NoError(t, err)

		bts, err := json.Marshal(p.OpenAPI)
		require.NoError(t, err)
		allOutputs = append(allOutputs, string(bts))
	}

	for i := 0; i < len(allOutputs)-1; i++ {
		require.Equal(t, allOutputs[i], allOutputs[i+1])
	}
}

func Test_parseStructTags_handlesSingleQuotedTagLiteral(t *testing.T) {
	field := &ast.Field{
		Names: []*ast.Ident{{Name: "Example"}},
		Tag:   &ast.BasicLit{Value: "'json:\"doubleAlias\"'"},
	}
	structSchema := &openapi.SchemaObject{DisabledFieldNames: map[string]struct{}{}}
	fieldSchema := &openapi.SchemaObject{}

	newName, skip := parseStructTags(field, structSchema, fieldSchema, "Example")
	require.False(t, skip)
	require.Equal(t, "doubleAlias", newName)
}

func Test_parseRouteComment(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	operation := &openapi.OperationObject{
		Responses: map[string]*openapi.ResponseObject{},
	}
	p.OpenAPI.Paths["v2/foo/bar"] = &openapi.PathItemObject{}
	p.OpenAPI.Paths["v2/foo/bar"].Get = operation

	duplicateError := p.parseRouteComment(operation, "@Router v2/foo/bar [get]")
	require.Error(t, duplicateError)
}

func Test_parseRouteCommentRejectsUnknownHTTPMethod(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	operation := &openapi.OperationObject{Responses: openapi.ResponsesObject{}}
	err = p.parseRouteComment(operation, "@Router /unknown [CONNECT]")

	require.EqualError(t, err, `unsupported HTTP method "CONNECT"`)
	require.NotContains(t, p.OpenAPI.Paths, "/unknown")
}

func Test_parseOperationDoesNotAcceptPathOutsideModule(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	err = p.parseOperation(p.ModulePath+"-extra", "main", []*ast.Comment{{Text: "// @Route /outside [GET]"}})

	require.NoError(t, err)
	require.NotContains(t, p.OpenAPI.Paths, "/outside")
}

func Test_parseOperationDoesNotAcceptPathOutsideHandler(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)
	p.HandlerPath = filepath.Join(p.ModulePath, "handlers")

	err = p.parseOperation(p.HandlerPath+"-extra", "main", []*ast.Comment{{Text: "// @Route /outside-handler [GET]"}})

	require.NoError(t, err)
	require.NotContains(t, p.OpenAPI.Paths, "/outside-handler")
}

func Test_parseResponseCommentAppliesJSONTypeToSchema(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		wantType string
	}{
		{name: "object", comment: `200 object FooResponse "ok"`, wantType: ""},
		{name: "array", comment: `200 array FooResponse "ok"`, wantType: "array"},
		{name: "array with slice type", comment: `200 array []FooResponse "ok"`, wantType: "array"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := setupParser()
			require.NoError(t, err)
			require.NoError(t, p.parse())
			operation := &openapi.OperationObject{Responses: openapi.ResponsesObject{}}

			err = p.parseResponseComment(p.ModulePath, "main", operation, test.comment)
			require.NoError(t, err)
			mediaType := operation.Responses["200"].Content[openapi.ContentTypeJson]
			require.NotNil(t, mediaType)
			if test.wantType == "" {
				require.Nil(t, mediaType.Schema.Type)
			} else {
				require.NotNil(t, mediaType.Schema.Type)
				require.Equal(t, test.wantType, *mediaType.Schema.Type)
			}
		})
	}
}

func Test_parseResponseCommentWithoutPayloadHasNoContent(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)
	operation := &openapi.OperationObject{Responses: openapi.ResponsesObject{}}

	err = p.parseResponseComment(p.ModulePath, "main", operation, `204 "No content"`)
	require.NoError(t, err)
	require.Empty(t, operation.Responses["204"].Content)
}

func Test_handleCompoundType(t *testing.T) {
	t.Run("oneOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "oneOf(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"oneOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("anyOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "anyOf(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"anyOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("allOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "allOf(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"allOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("case insensitive oneOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "oneof(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"oneOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("case insensitive anyOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "anyof(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"anyOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("case insensitive allOf", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "allof(string,[]string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"allOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("not", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "not(string)")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"not\":{\"type\":\"string\"}}", string(s))
	})

	t.Run("handles whitespace", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		result, err := p.handleCompoundType("./example", "example.com/example", "allOf(  string, []string )")
		require.NoError(t, err)
		s, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"allOf\":[{\"type\":\"string\"},{\"type\":\"array\",\"items\":{\"type\":\"string\"}}]}", string(s))
	})

	t.Run("not only accepts 1 arg", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		_, notErr := p.handleCompoundType("./example", "example.com/example", "not(string,int32)")
		require.Error(t, notErr)
	})

	t.Run("error when no args", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		_, notErr := p.handleCompoundType("./example", "example.com/example", "oneOf()")
		require.Error(t, notErr)
	})
}

func TestSchemaRef_emptyNameIsSafe(t *testing.T) {
	require.Equal(t, "", openapi.SchemaRef(""))
	require.Equal(t, "#/components/schemas/User", openapi.SchemaRef("User"))
	require.Equal(t, "#/components/schemas/User", openapi.SchemaRef("#/components/schemas/User"))
	require.Equal(t, "#/components/schemas/User/Type", openapi.SchemaRef("User\\Type"))
	require.Equal(t, "#/components/schemas/User/Type", openapi.SchemaRef("#/components/schemas/User\\Type"))
}

func Test_descriptions(t *testing.T) {
	t.Run("Description unchanged when not a ref", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		operation := &openapi.OperationObject{
			Responses: map[string]*openapi.ResponseObject{},
		}

		err = p.parseDescription(operation, "testing")
		require.NoError(t, err)

		require.Equal(t, "testing", operation.Description)
	})

	t.Run("Description inline when a ref", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "https://example.com",
			httpmock.NewStringResponder(200, "The quick brown fox jumped over the lazy dog"))

		p, err := setupParser()
		require.NoError(t, err)

		operation := &openapi.OperationObject{
			Responses: map[string]*openapi.ResponseObject{},
		}

		err = p.parseDescription(operation, "$ref:https://example.com")
		require.NoError(t, err)

		require.Equal(t, "The quick brown fox jumped over the lazy dog", operation.Description)
	})
}

func Test_genSchemaObjectID(t *testing.T) {
	t.Run("empty package name", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		result := p.genSchemaObjectID("", "sample")

		require.Equal(t, "sample", string(result))
	})
	t.Run("simple package name", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		result := p.genSchemaObjectID("sample", "sample")

		require.Equal(t, "sample", string(result))
	})
	t.Run("multidepth package name", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		result := p.genSchemaObjectID("test.sample", "sample")

		require.Equal(t, "sample", string(result))
	})
	t.Run("omit package name", func(t *testing.T) {
		p, err := newParser("../example/", "../example/main.go", "", "", false, true, false)
		require.NoError(t, err)

		result := p.genSchemaObjectID("test.sample", "sample")

		require.Equal(t, "sample", string(result))
	})
}

func Test_parseEntryPointHelpers(t *testing.T) {
	t.Run("Parses security scheme, scopes and tags", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		oauthScopes := map[string]map[string]string{}
		require.NoError(t, p.parseSecuritySchemeComment("oauth2 oauth2AuthCode https://example.com/auth https://example.com/token", oauthScopes))
		require.NoError(t, p.parseSecurityScopeComment("oauth2 read Read access", oauthScopes))
		require.NoError(t, p.parseTagsComment("@Tags \"Foo\" \"Bar\""))

		require.Contains(t, p.OpenAPI.Components.SecuritySchemes, "oauth2")
		require.NotNil(t, p.OpenAPI.Components.SecuritySchemes["oauth2"].OAuthFlows)
		require.NotNil(t, p.OpenAPI.Components.SecuritySchemes["oauth2"].OAuthFlows.AuthorizationCode)
		require.Equal(t, "Read access", oauthScopes["oauth2"]["read"])
		require.Equal(t, "Foo", p.OpenAPI.Tags[0].Name)
	})

	t.Run("Keeps free-form descriptions with spaces for apiKey schemes", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		oauthScopes := map[string]map[string]string{}
		require.NoError(t, p.parseSecuritySchemeComment("MyApiAuth apiKey header X-MyCustomHeader Login com seu token", oauthScopes))

		scheme, ok := p.OpenAPI.Components.SecuritySchemes["MyApiAuth"]
		require.True(t, ok)
		require.Equal(t, "header", scheme.In)
		require.Equal(t, "X-MyCustomHeader", scheme.Name)
		require.Equal(t, "Login com seu token", scheme.Description)
	})
}

func Test_parseSchemaObject_externalAliases(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)
	require.NoError(t, p.parse())

	t.Run("BsonID is generated as string", func(t *testing.T) {
		schema, err := p.parseSchemaObject("../example", "main", "BsonID", true)
		require.NoError(t, err)
		require.Equal(t, "string", *schema.Type)
	})

	t.Run("Instruction is generated as free-form object", func(t *testing.T) {
		schema, err := p.parseSchemaObject("../example", "main", "Instruction", true)
		require.NoError(t, err)
		require.Equal(t, "object", *schema.Type)
		require.NotNil(t, schema.AdditionalProperties)
	})
}

func Test_parseOperationTags(t *testing.T) {
	t.Run("Parses operation tags", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		p.OpenAPI.Tags = append(p.OpenAPI.Tags, openapi.TagDefinition{Name: "foo", Description: &openapi.ReffableString{Value: "bar"}})

		var comment []*ast.Comment
		comment = append(comment, &ast.Comment{Slash: 0, Text: "// @Tag foo"})
		err = p.parseOperation(p.ModulePath, "", comment)
		require.NoError(t, err)
	})

	t.Run("Errors when tag in operation is not in list of tags", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		p.OpenAPI.Tags = append(p.OpenAPI.Tags, openapi.TagDefinition{Name: "foo", Description: &openapi.ReffableString{Value: "bar"}})

		var comment []*ast.Comment
		comment = append(comment, &ast.Comment{Slash: 0, Text: "// @Tag Foo"})
		err = p.parseOperation(p.ModulePath, "", comment)
		require.Error(t, err)
	})
}

func Test_validateSchemaNames(t *testing.T) {
	t.Run("Returns no conflicts", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.validateSchemaNames()

		require.NoError(t, err)
	})

	t.Run("Returns conflicts", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		p.ApiSchemaNames["pkg/foo/bar"] = map[string]string{}
		p.ApiSchemaNames["pkg/foo/bar"]["BarRecord"] = "Record"
		p.ApiSchemaNames["pkg/baz/qux"] = map[string]string{}
		p.ApiSchemaNames["pkg/baz/qux"]["QuxRecord"] = "Record"

		err = p.validateSchemaNames()

		require.Error(t, err)
	})

	t.Run("Schema registry validates aliases independently", func(t *testing.T) {
		reg := internalSchema.NewRegistry(false)
		reg.ApiSchemaNames["pkg/foo/bar"] = map[string]string{}
		reg.ApiSchemaNames["pkg/foo/bar"]["BarRecord"] = "Record"
		reg.ApiSchemaNames["pkg/baz/qux"] = map[string]string{}
		reg.ApiSchemaNames["pkg/baz/qux"]["QuxRecord"] = "Record"

		require.Error(t, reg.ValidateSchemaNames())
	})
}

func Test_astPackageCache(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	cache := astpkg.NewPackageCache()
	moduleAST, err := cache.GetPackageAST(p.ModulePath)
	require.NoError(t, err)
	require.NotEmpty(t, moduleAST)

	cachedAgain, err := cache.GetPackageAST(p.ModulePath)
	require.NoError(t, err)
	require.Equal(t, len(moduleAST), len(cachedAgain))
}

func Test_importRegistryTracksAliases(t *testing.T) {
	reg := internalImports.NewRegistry()
	reg.Record("example.com/foo", "github.com/example/pkg", "pkg")
	reg.Record("example.com/foo", "github.com/example/pkg", "pkg")
	reg.Record("example.com/foo", "github.com/example/other", "other")

	require.Len(t, reg.PkgNameImportedPkgAlias["example.com/foo"]["pkg"], 1)
	require.Len(t, reg.PkgNameImportedPkgAlias["example.com/foo"]["other"], 1)
}

func Test_parseOverrideStructTag(t *testing.T) {
	t.Run("found tag", func(t *testing.T) {
		ast := &ast.Field{
			Doc:   nil,
			Names: nil,
			Type:  nil,
			Tag: &ast.BasicLit{
				ValuePos: 0,
				Kind:     token.STRING,
				Value:    `overrideApiSchemaType:"Test"`},
		}
		result := parseOverrideStructTag(ast)

		require.Equal(t, "Test", result)
	})
}

func Test_parseResourcesUseInternalRegistries(t *testing.T) {
	resources := newParseResources()

	require.IsType(t, &internalImports.Registry{}, resources.importRegistry)
	require.IsType(t, &astpkg.PackageCache{}, resources.astCache)
	require.NotNil(t, resources.astCache)
}

func Test_parseGoMod(t *testing.T) {
	t.Run("Successfully parses go.mod file", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.parseGoMod()
		require.NoError(t, err)

		// Verify that packages were loaded from go.mod
		require.NotEmpty(t, p.KnownPkgs)
		require.NotEmpty(t, p.KnownNamePkg)
		require.NotEmpty(t, p.KnownPathPkg)
	})

	t.Run("Returns error when go.mod file cannot be read", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		// Set invalid go.mod path
		p.GoModFilePath = "/nonexistent/go.mod"

		err = p.parseGoMod()
		require.Error(t, err)
	})

	t.Run("Handles uppercase characters in module paths", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.parseGoMod()
		require.NoError(t, err)

		// Check that uppercase characters are converted to !lowercase format
		for _, pkg := range p.KnownPkgs {
			if pkg.Path != "" && pkg.Path != p.ModulePath {
				// Package paths from go.mod cache should not contain uppercase without ! prefix
				require.NotRegexp(t, regexp.MustCompile(`[^!][A-Z]`), pkg.Path)
			}
		}
	})

	t.Run("Maps package names to packages correctly", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.parseGoMod()
		require.NoError(t, err)

		// Verify bidirectional mapping
		for _, pkg := range p.KnownPkgs {
			if pkg.Name != "" {
				foundByName := p.KnownNamePkg[pkg.Name]
				require.NotNil(t, foundByName)
				require.Equal(t, pkg.Name, foundByName.Name)
			}
			if pkg.Path != "" {
				foundByPath := p.KnownPathPkg[pkg.Path]
				require.NotNil(t, foundByPath)
				require.Equal(t, pkg.Path, foundByPath.Path)
			}
		}
	})

	t.Run("Skips .git directories", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)

		err = p.parseGoMod()
		require.NoError(t, err)

		// Verify no .git paths are included
		for _, pkg := range p.KnownPkgs {
			require.NotContains(t, pkg.Path, ".git")
		}
	})
}
