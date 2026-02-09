package annotate

import (
	"errors"
	"fmt"
	"strings"
)

// ParseRouteComment parses a comment string to extract route information.
//
// The expected format of the comment is:
//
//	"@router /path [METHOD]"
//
// or
//
//	"@route /path [METHOD]"
//
// For example:
//
//	"@router /users [GET]"
//	"@route /users/{id} [POST]"
//
// It returns a RouteSpec containing the path and HTTP method, or an error if the comment is invalid.
func ParseRouteComment(comment string) (RouteSpec, error) {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return RouteSpec{}, errors.New("empty comment")
	}

	parts := strings.Fields(comment)
	if len(parts) < 3 {
		return RouteSpec{}, fmt.Errorf("invalid route comment format: %q", comment)
	}

	tag := strings.ToLower(parts[0])
	if tag != "@router" && tag != "@route" {
		return RouteSpec{}, fmt.Errorf("not a route comment: %q", comment)
	}

	path := parts[1]

	methodToken := parts[2]
	if !strings.HasPrefix(methodToken, "[") || !strings.HasSuffix(methodToken, "]") {
		return RouteSpec{}, fmt.Errorf("invalid HTTP method format: %q", methodToken)
	}

	method := strings.Trim(methodToken, "[]")

	return RouteSpec{
		Path:   path,
		Method: method,
	}, nil
}
