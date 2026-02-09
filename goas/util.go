package goas

import (
	"log"
	"strings"

	"github.com/delley/goas/internal/openapi"
)

func isInStringList(list []string, s string) bool {
	for i := range list {
		if list[i] == s {
			return true
		}
	}
	return false
}

var basicGoTypes = map[string]bool{
	"bool":       true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"float32":    true,
	"float64":    true,
	"string":     true,
	"complex64":  true,
	"complex128": true,
	"byte":       true,
	"rune":       true,
	"uintptr":    true,
	"error":      true,
}

func isBasicGoType(typeName string) bool {
	_, ok := basicGoTypes[typeName]
	return ok
}

var goTypesOASTypes = map[string]string{
	"bool":    "boolean",
	"uint":    "integer",
	"uint8":   "integer",
	"uint16":  "integer",
	"uint32":  "integer",
	"uint64":  "integer",
	"int":     "integer",
	"int8":    "integer",
	"int16":   "integer",
	"int32":   "integer",
	"int64":   "integer",
	"float32": "number",
	"float64": "number",
	"string":  "string",
}

func isGoTypeOASType(typeName string) bool {
	_, ok := goTypesOASTypes[typeName]
	return ok
}

var goTypesOASFormats = map[string]string{
	"bool":    "boolean",
	"uint":    "int64",
	"uint8":   "int64",
	"uint16":  "int64",
	"uint32":  "int64",
	"uint64":  "int64",
	"int":     "int64",
	"int8":    "int64",
	"int16":   "int64",
	"int32":   "int64",
	"int64":   "int64",
	"float32": "float",
	"float64": "double",
	"string":  "string",
}

func addSchemaRefLinkPrefix(name string) string {
	if name == "" {
		log.Fatalln("schema does not reference valid name")
	}
	if strings.HasPrefix(name, "#/components/schemas/") {
		return replaceBackslash(name)
	}
	return replaceBackslash("#/components/schemas/" + name)
}

func replaceBackslash(origin string) string {
	return strings.ReplaceAll(origin, "\\", "/")
}

// checkFormatInt64 will see if the type is int64 and add to Format property if true
func checkFormatInt64(typeName string, schemaObject *openapi.SchemaObject) {
	if typeName == "int64" {
		schemaObject.Format = "int64"
	}
}
