package openapi

import "strings"

const schemaRefPrefix = "#/components/schemas/"

// SchemaRef builds a canonical OpenAPI schema reference from a component name.
// It preserves existing names when already prefixed and normalizes Windows-style
// separators to the slash form used by OpenAPI pointers.
func SchemaRef(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, schemaRefPrefix) {
		return NormalizeComponentSchemaKey(name)
	}
	return NormalizeComponentSchemaKey(schemaRefPrefix + name)
}

// NormalizeComponentSchemaKey converts a schema component identifier into the
// canonical map key used in the OpenAPI components registry.
func NormalizeComponentSchemaKey(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}
