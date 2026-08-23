package openapi

import "encoding/json"

type MarshalOptions struct {
	Prefix string
	Indent string
}

// Marshal serializes an OpenAPI document to JSON. It is the only serialization
// boundary used by the generator; parsing and schema registries do not marshal.
func Marshal(spec *OpenAPIObject, opt MarshalOptions) ([]byte, error) {
	prefix := opt.Prefix
	indent := opt.Indent
	if indent == "" {
		indent = "  "
	}
	return json.MarshalIndent(spec, prefix, indent)
}
