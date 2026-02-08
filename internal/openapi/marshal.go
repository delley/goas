package openapi

import "encoding/json"

type MarshalOptions struct {
	Prefix string
	Indent string
}

// Marshal serialize the spec to JSON with indentation.
func Marshal(spec *OpenAPIObject, opt MarshalOptions) ([]byte, error) {
	prefix := opt.Prefix
	indent := opt.Indent
	if indent == "" {
		indent = "  "
	}
	return json.MarshalIndent(spec, prefix, indent)
}
