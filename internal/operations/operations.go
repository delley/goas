package operations

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/delley/goas/internal/annotate"
	"github.com/delley/goas/internal/desc"
	"github.com/delley/goas/internal/openapi"
)

// Input describes a single operation comment block to be parsed into an OpenAPI operation.
type Input struct {
	Comments    []string
	Tags        []string
	ShowHidden  bool
	FileRefPath string
}

// Parser is a small, side-effect free parser for operation metadata and route application.
type Parser struct {
	FileRefPath string
	Tags        []string
	ShowHidden  bool
}

func (p *Parser) Parse(ctx context.Context, input Input) (*openapi.OperationObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if p == nil {
		p = &Parser{}
	}
	if p.FileRefPath == "" {
		p.FileRefPath = input.FileRefPath
	}
	if len(p.Tags) == 0 {
		p.Tags = input.Tags
	}
	if !p.ShowHidden {
		p.ShowHidden = input.ShowHidden
	}

	operation := &openapi.OperationObject{Responses: openapi.ResponsesObject{}}
	for _, raw := range input.Comments {
		comment := strings.TrimSpace(strings.TrimLeft(raw, "/"))
		if comment == "" {
			continue
		}
		if len(strings.Fields(comment)) == 0 {
			continue
		}
		attribute := strings.Fields(comment)[0]
		value := strings.TrimSpace(comment[len(attribute):])
		switch strings.ToLower(attribute) {
		case "@title":
			operation.Summary = value
		case "@description":
			if err := parseDescription(ctx, p.FileRefPath, operation, value); err != nil {
				return nil, err
			}
		case "@operationid":
			operation.OperationID = value
		case "@tag", "@resource":
			resource := value
			if resource == "" {
				resource = "others"
			}
			if !slices.Contains(p.Tags, resource) && !p.ShowHidden {
				return nil, fmt.Errorf("could not find tag %q in the main list of tags", resource)
			}
			if !slices.Contains(operation.Tags, resource) {
				operation.Tags = append(operation.Tags, resource)
			}
		}
	}
	return operation, nil
}

func parseDescription(ctx context.Context, fileRefPath string, operation *openapi.OperationObject, description string) error {
	descText, err := desc.FetchRefContext(ctx, fileRefPath, description)
	if err != nil {
		return err
	}
	if operation.Description == "" {
		operation.Description = descText
	} else {
		operation.Description = operation.Description + " " + descText
	}
	return nil
}

func ParseRouteComment(comment string) (annotate.RouteSpec, error) {
	return annotate.ParseRouteComment(comment)
}

func ApplyRoute(doc *openapi.OpenAPIObject, operation *openapi.OperationObject, route, method string) error {
	if doc == nil {
		return fmt.Errorf("openapi document is nil")
	}
	if doc.Paths == nil {
		doc.Paths = openapi.PathsObject{}
	}
	if operation == nil {
		return fmt.Errorf("operation is nil")
	}

	method = strings.TrimSpace(method)
	if method == "" {
		return fmt.Errorf("missing HTTP method for route %q", route)
	}
	method = strings.ToUpper(method)

	item, ok := doc.Paths[route]
	if !ok || item == nil {
		doc.Paths[route] = &openapi.PathItemObject{}
		item = doc.Paths[route]
	}

	if hasMethod(item, method) {
		return fmt.Errorf("already exists, %q [%q]", route, method)
	}

	switch method {
	case http.MethodGet:
		item.Get = operation
	case http.MethodPost:
		item.Post = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodDelete:
		item.Delete = operation
	case http.MethodOptions:
		item.Options = operation
	case http.MethodHead:
		item.Head = operation
	case http.MethodTrace:
		item.Trace = operation
	default:
		return fmt.Errorf("unsupported HTTP method %q", method)
	}
	return nil
}

func hasMethod(item *openapi.PathItemObject, method string) bool {
	if item == nil {
		return false
	}
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return item.Get != nil
	case http.MethodPost:
		return item.Post != nil
	case http.MethodPatch:
		return item.Patch != nil
	case http.MethodPut:
		return item.Put != nil
	case http.MethodDelete:
		return item.Delete != nil
	case http.MethodOptions:
		return item.Options != nil
	case http.MethodHead:
		return item.Head != nil
	case http.MethodTrace:
		return item.Trace != nil
	default:
		return false
	}
}
