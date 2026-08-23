package goas

import (
	"fmt"
	"go/ast"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/delley/goas/internal/annotate"
	"github.com/delley/goas/internal/load"
	"github.com/delley/goas/internal/openapi"
)

// parsePaths iterates through all packages looking for operations defined in function/variable comments.
func (p *parser) parsePaths() error {
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name

		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parsePaths: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}
		for _, astPackageKey := range load.SortedKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range load.SortedKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astDeclaration := range astFile.Decls {
					if astFuncDeclaration, ok := astDeclaration.(*ast.FuncDecl); ok {
						if astFuncDeclaration.Doc != nil && astFuncDeclaration.Doc.List != nil {
							err = p.parseOperation(pkgPath, pkgName, astFuncDeclaration.Doc.List)
							if err != nil {
								return err
							}
						}
					} else if astVarDeclaration, ok := astDeclaration.(*ast.GenDecl); ok {
						if astVarDeclaration.Doc != nil && astVarDeclaration.Doc.List != nil {
							err = p.parseOperation(pkgPath, pkgName, astVarDeclaration.Doc.List)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// parseOperation processes comments on a function or variable to extract route and operation metadata.
func (p *parser) parseOperation(pkgPath, pkgName string, astComments []*ast.Comment) error {
	operation := &openapi.OperationObject{
		Responses: map[string]*openapi.ResponseObject{},
	}
	if !isWithinPath(p.ModulePath, pkgPath) {
		// ignore this pkgName
		// p.debugf("parseOperation ignores %s", pkgPath)
		return nil
	} else if p.HandlerPath != "" && !isWithinPath(p.HandlerPath, pkgPath) {
		return nil
	}
	if annotate.IsHidden(astComments, p.ShowHidden) {
		return nil
	}
	var err error
	var tagList []string
	for _, tag := range p.OpenAPI.Tags {
		tagList = append(tagList, tag.Name)
	}

	for _, astComment := range astComments {
		comment := strings.TrimSpace(strings.TrimLeft(astComment.Text, "/"))
		if len(comment) == 0 {
			// ignore empty lines
			continue
		}
		attribute := strings.Fields(comment)[0]
		value := strings.TrimSpace(comment[len(attribute):])
		switch strings.ToLower(attribute) {
		case "@title":
			operation.Summary = value
		case "@description":
			err = p.parseDescription(operation, value)
		case "@operationid":
			operation.OperationID = value
		case "@param":
			err = p.parseParamComment(pkgPath, pkgName, operation, value)
		case "@success", "@failure":
			err = p.parseResponseComment(pkgPath, pkgName, operation, value)
		case "@resource", "@tag":
			resource := value
			if resource == "" {
				resource = "others"
			}

			if !slices.Contains(tagList, resource) && !p.ShowHidden {
				err = fmt.Errorf("could not find tag \"%s\" in the main list of tags", resource)
			} else if !slices.Contains(operation.Tags, resource) {
				operation.Tags = append(operation.Tags, resource)
			}
		case "@route", "@router":
			err = p.parseRouteComment(operation, comment)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// parseRouteComment processes @route/@router annotations to register operation in paths.
func (p *parser) parseRouteComment(operation *openapi.OperationObject, comment string) error {
	spec, err := annotate.ParseRouteComment(comment)
	if err != nil {
		return err
	}

	route := spec.Path
	method := spec.Method
	if !supportedHTTPMethod(method) {
		return fmt.Errorf("unsupported HTTP method %q", method)
	}

	pi, ok := p.OpenAPI.Paths[route]
	if !ok || pi == nil {
		p.OpenAPI.Paths[route] = &openapi.PathItemObject{}
	} else if p.routeAndMethodExist(route, method) {
		return fmt.Errorf("already exists, %q [%q]", route, method)
	}

	switch strings.ToUpper(method) {
	case http.MethodGet:
		p.OpenAPI.Paths[route].Get = operation
	case http.MethodPost:
		p.OpenAPI.Paths[route].Post = operation
	case http.MethodPatch:
		p.OpenAPI.Paths[route].Patch = operation
	case http.MethodPut:
		p.OpenAPI.Paths[route].Put = operation
	case http.MethodDelete:
		p.OpenAPI.Paths[route].Delete = operation
	case http.MethodOptions:
		p.OpenAPI.Paths[route].Options = operation
	case http.MethodHead:
		p.OpenAPI.Paths[route].Head = operation
	case http.MethodTrace:
		p.OpenAPI.Paths[route].Trace = operation
	}

	return nil
}

func supportedHTTPMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodHead, http.MethodTrace:
		return true
	default:
		return false
	}
}

func isWithinPath(parent, child string) bool {
	parent, err := filepath.Abs(filepath.Clean(parent))
	if err != nil {
		return false
	}
	child, err = filepath.Abs(filepath.Clean(child))
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// routeAndMethodExist checks if a route and method combination already exists.
func (p *parser) routeAndMethodExist(route string, method string) bool {
	pi, ok := p.OpenAPI.Paths[route]
	if !ok || pi == nil {
		return false
	}

	switch strings.ToUpper(method) {
	case http.MethodGet:
		return p.OpenAPI.Paths[route].Get != nil
	case http.MethodPost:
		return p.OpenAPI.Paths[route].Post != nil
	case http.MethodPatch:
		return p.OpenAPI.Paths[route].Patch != nil
	case http.MethodPut:
		return p.OpenAPI.Paths[route].Put != nil
	case http.MethodDelete:
		return p.OpenAPI.Paths[route].Delete != nil
	case http.MethodOptions:
		return p.OpenAPI.Paths[route].Options != nil
	case http.MethodHead:
		return p.OpenAPI.Paths[route].Head != nil
	case http.MethodTrace:
		return p.OpenAPI.Paths[route].Trace != nil
	}

	return false
}
