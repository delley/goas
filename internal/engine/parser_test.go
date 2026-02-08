package engine

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"os"
	"regexp"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func setupParser() (*parser, error) {
	return newParser("../../example/", "../../example/main.go", "", "", false, true, false)
}
func TestExample(t *testing.T) {
	p, err := setupParser()
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	expected, _ := os.ReadFile("../../example/example.json")
	require.JSONEq(t, string(expected), string(bts))
}

func TestShowHiddenExample(t *testing.T) {
	p, err := newParser("../../example/", "../../example/main.go", "", "", false, true, true)
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	expected, _ := os.ReadFile("../../example/example-show-hidden.json")
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
		p, err := newParser("../../example/", "../../example/main.go", "", "", false, true, false)
		require.NoError(t, err)

		result := p.genSchemaObjectID("test.sample", "sample")

		require.Equal(t, "sample", string(result))
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

		conflicts := p.validateSchemaNames()

		require.Empty(t, conflicts)
	})

	t.Run("Returns conflicts", func(t *testing.T) {
		p, err := setupParser()
		require.NoError(t, err)
		p.ApiSchemaNames["pkg/foo/bar"] = map[string]string{}
		p.ApiSchemaNames["pkg/foo/bar"]["BarRecord"] = "Record"
		p.ApiSchemaNames["pkg/baz/qux"] = map[string]string{}
		p.ApiSchemaNames["pkg/baz/qux"]["QuxRecord"] = "Record"

		conflicts := p.validateSchemaNames()

		require.Len(t, conflicts, 1)
		require.Contains(t, conflicts[0], "pkg/foo/bar#BarRecord")
		require.Contains(t, conflicts[0], "pkg/baz/qux#QuxRecord")
	})
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
