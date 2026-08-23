package types

import (
	"fmt"
	"go/ast"
	"strings"
)

// BasicGoTypes maps Go built-in type names to a marker value.
var BasicGoTypes = map[string]bool{
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

// IsBasicGoType reports whether typeName is a built-in Go type.
func IsBasicGoType(typeName string) bool {
	_, ok := BasicGoTypes[typeName]
	return ok
}

// GoTypesOASTypes maps Go types to OpenAPI Schema types.
// Only types that map to standard OAS types are included.
var GoTypesOASTypes = map[string]string{
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

// IsGoTypeOASType reports whether typeName is a Go type that maps to an OAS type.
func IsGoTypeOASType(typeName string) bool {
	_, ok := GoTypesOASTypes[typeName]
	return ok
}

// GoTypesOASFormats maps Go types to OpenAPI Schema format strings.
var GoTypesOASFormats = map[string]string{
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

// GetOASType returns the OpenAPI type for a Go type, or empty string if not found.
func GetOASType(goType string) string {
	return GoTypesOASTypes[goType]
}

// GetOASFormat returns the OpenAPI format for a Go type, or empty string if not found.
func GetOASFormat(goType string) string {
	return GoTypesOASFormats[goType]
}

// TypeAsString converts an AST type expression to its string representation.
// Handles arrays, maps, interfaces, pointers, and selector expressions.
func TypeAsString(fieldType interface{}) string {
	astArrayType, ok := fieldType.(*ast.ArrayType)
	if ok {
		return fmt.Sprintf("[]%v", TypeAsString(astArrayType.Elt))
	}

	astMapType, ok := fieldType.(*ast.MapType)
	if ok {
		return fmt.Sprintf("map[]%v", TypeAsString(astMapType.Value))
	}

	_, ok = fieldType.(*ast.InterfaceType)
	if ok {
		return "interface{}"
	}

	astStarExpr, ok := fieldType.(*ast.StarExpr)
	if ok {
		// Dereference pointers for OpenAPI purposes
		return fmt.Sprintf("%v", TypeAsString(astStarExpr.X))
	}

	astSelectorExpr, ok := fieldType.(*ast.SelectorExpr)
	if ok {
		packageNameIdent, _ := astSelectorExpr.X.(*ast.Ident)
		return packageNameIdent.Name + "." + astSelectorExpr.Sel.Name
	}

	return fmt.Sprint(fieldType)
}

// IsSliceOrMapType reports whether typeName is a slice, map, or interface.
func IsSliceOrMapType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") ||
		strings.HasPrefix(typeName, "map[]") ||
		typeName == "interface{}"
}

// IsSpecialType reports whether typeName is a special type like time.Time or uuid.UUID.
func IsSpecialType(typeName string) bool {
	return typeName == "time.Time" || typeName == "uuid.UUID"
}

// NormalizeTypeName removes package prefix if present.
func NormalizeTypeName(typeName string) string {
	if index := strings.LastIndex(typeName, "."); index >= 0 {
		return typeName[index+1:]
	}
	return typeName
}
