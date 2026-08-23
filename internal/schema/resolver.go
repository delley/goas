package schema

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"

	"github.com/delley/goas/internal/imports"
	"github.com/delley/goas/internal/openapi"
	"github.com/delley/goas/internal/types"
)

// Resolver handles semantic resolution of Go types to OpenAPI schemas.
// It depends on type indices, import tracking, and schema registry,
// but does not depend on module loading or JSON serialization.
type Resolver struct {
	Types    *TypeIndex
	Imports  *imports.Registry
	Registry *Registry
}

// TypeIndex maps package names and type names to AST type specs.
type TypeIndex struct {
	// TypeSpecs maps package name -> type name -> ast.TypeSpec
	TypeSpecs map[string]map[string]*ast.TypeSpec
}

// NewResolver creates a new resolver with the given dependencies.
func NewResolver(omitPackages bool) *Resolver {
	return &Resolver{
		Types: &TypeIndex{
			TypeSpecs: make(map[string]map[string]*ast.TypeSpec),
		},
		Imports:  imports.NewRegistry(),
		Registry: NewRegistry(omitPackages),
	}
}

// RegisterTypeSpec associates a type spec with a package and type name.
func (r *Resolver) RegisterTypeSpec(pkgName, typeName string, typeSpec *ast.TypeSpec) {
	if r.Types.TypeSpecs[pkgName] == nil {
		r.Types.TypeSpecs[pkgName] = make(map[string]*ast.TypeSpec)
	}
	r.Types.TypeSpecs[pkgName][typeName] = typeSpec
}

// getTypeSpec looks up a type spec by package and type name.
func (r *Resolver) getTypeSpec(pkgName, typeName string) (*ast.TypeSpec, bool) {
	pkgTypeSpecs, exist := r.Types.TypeSpecs[pkgName]
	if !exist {
		return nil, false
	}
	astTypeSpec, exist := pkgTypeSpecs[typeName]
	if !exist {
		return nil, false
	}
	return astTypeSpec, true
}

// ResolveBasicType returns a schema for a basic Go type like int, string, etc.
func (r *Resolver) ResolveBasicType(typeName string) *openapi.SchemaObject {
	if !types.IsGoTypeOASType(typeName) {
		return nil
	}
	schema := &openapi.SchemaObject{}
	oasType := types.GetOASType(typeName)
	schema.Type = &oasType
	return schema
}

// HandleSliceType resolves a slice type like []SomeType.
// Returns nil if the type is not a slice.
func (r *Resolver) HandleSliceType(pkgName, typeName string) (*openapi.SchemaObject, error) {
	if !strings.HasPrefix(typeName, "[]") {
		return nil, nil
	}

	schema := &openapi.SchemaObject{}
	arrayType := "array"
	schema.Type = &arrayType

	itemTypeName := typeName[2:]
	itemSchema, err := r.ResolveType(pkgName, itemTypeName, true)
	if err != nil {
		return nil, err
	}

	if itemSchema.ID != "" {
		schema.Items = &openapi.SchemaObject{Ref: openapi.SchemaRef(itemSchema.ID)}
	}

	schema.Items = itemSchema
	return schema, nil
}

// HandleMapType resolves a map type like map[]ValueType.
// Returns nil if the type is not a map.
func (r *Resolver) HandleMapType(pkgName, typeName string) (*openapi.SchemaObject, error) {
	if !strings.HasPrefix(typeName, "map[]") {
		return nil, nil
	}

	schema := &openapi.SchemaObject{}
	objectType := "object"
	schema.Type = &objectType

	valueTypeName := typeName[5:]
	valueSchema, err := r.ResolveType(pkgName, valueTypeName, true)
	if err != nil {
		return nil, err
	}

	if valueSchema.ID != "" {
		schema.AdditionalProperties = &openapi.SchemaObject{Ref: openapi.SchemaRef(valueSchema.ID)}
	}

	schema.AdditionalProperties = valueSchema
	return schema, nil
}

// HandleCompoundType resolves compound types like oneOf(), anyOf(), allOf(), not().
// Returns nil if the type is not a compound type.
func (r *Resolver) HandleCompoundType(pkgName, typeName string) (*openapi.SchemaObject, error) {
	re := regexp.MustCompile(`(?i)(oneOf|anyOf|allOf|not)\(([^\)]*)\)`)
	matches := re.FindStringSubmatch(typeName)
	if len(matches) < 3 {
		return nil, nil
	}

	op := strings.ToLower(matches[1])
	if matches[2] == "" {
		return nil, fmt.Errorf("expected 1 or more arguments, received '%s'", typeName)
	}

	args := trimSplit(matches[2])

	// not only supports one arg
	if op == "not" && len(args) != 1 {
		return nil, fmt.Errorf("invalid number of arguments for not compound type, expected 1 received %d", len(args))
	}

	var schemas []*openapi.SchemaObject
	for _, arg := range args {
		result, err := r.ResolveType(pkgName, arg, true)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, result)
	}

	schema := &openapi.SchemaObject{}
	switch op {
	case "not":
		schema.Not = schemas[0]
	case "oneof":
		schema.OneOf = schemas
	case "anyof":
		schema.AnyOf = schemas
	case "allof":
		schema.AllOf = schemas
	default:
		return nil, fmt.Errorf("invalid compound type '%s'", op)
	}

	return schema, nil
}

// ResolveType resolves a Go type to an OpenAPI schema.
// If register is true, the schema is added to the registry.
func (r *Resolver) ResolveType(pkgName, typeName string, register bool) (*openapi.SchemaObject, error) {
	// Try compound type (oneOf, anyOf, allOf, not)
	if compound, err := r.HandleCompoundType(pkgName, typeName); compound != nil || err != nil {
		if err != nil {
			return nil, err
		}
		return compound, nil
	}

	// Try slice type
	if slice, err := r.HandleSliceType(pkgName, typeName); slice != nil || err != nil {
		if err != nil {
			return nil, err
		}
		return slice, nil
	}

	// Try map type
	if m, err := r.HandleMapType(pkgName, typeName); m != nil || err != nil {
		if err != nil {
			return nil, err
		}
		return m, nil
	}

	// Try basic Go type
	if types.IsBasicGoType(typeName) {
		if schema := r.ResolveBasicType(typeName); schema != nil {
			return schema, nil
		}
	}

	// Look up in type specs
	typeSpec, exist := r.getTypeSpec(pkgName, typeName)
	if !exist {
		return &openapi.SchemaObject{}, nil
	}

	// Resolve the actual type definition
	schema, err := r.resolveTypeSpec(pkgName, typeName, typeSpec, register)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

// resolveTypeSpec resolves a type spec to a schema object.
func (r *Resolver) resolveTypeSpec(pkgName, typeName string, typeSpec *ast.TypeSpec, register bool) (*openapi.SchemaObject, error) {
	schema := &openapi.SchemaObject{}

	// Handle type aliases to basic Go types
	if ident, ok := typeSpec.Type.(*ast.Ident); ok {
		if types.IsGoTypeOASType(ident.Name) {
			oasType := types.GetOASType(ident.Name)
			schema.Type = &oasType
			return schema, nil
		}

		// This is an alias to a custom type
		newSchema, err := r.ResolveType(pkgName, ident.Name, true)
		if err != nil {
			return nil, err
		}
		schema.Ref = openapi.SchemaRef(newSchema.ID)
		return schema, nil
	}

	// Handle struct types
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		objectType := "object"
		schema.Type = &objectType
		if structType.Fields != nil {
			r.parseStructFields(pkgName, schema, structType.Fields.List)
		}
		goto register_schema
	}

	// Handle array types
	if arrayType, ok := typeSpec.Type.(*ast.ArrayType); ok {
		arrayTypeStr := "array"
		schema.Type = &arrayTypeStr
		schema.Items = &openapi.SchemaObject{}
		typeAsString := types.TypeAsString(arrayType.Elt)
		typeAsString = strings.TrimLeft(typeAsString, "*")

		if !types.IsBasicGoType(typeAsString) {
			itemSchema, err := r.ResolveType(pkgName, typeAsString, true)
			if err != nil {
				return nil, err
			}
			if itemSchema.ID != "" {
				schema.Items.Ref = openapi.SchemaRef(itemSchema.ID)
			} else {
				*schema.Items = *itemSchema
			}
		} else if types.IsGoTypeOASType(typeAsString) {
			localGoType := types.GetOASType(typeAsString)
			schema.Items.Type = &localGoType
		}
		goto register_schema
	}

	// Handle map types
	if mapType, ok := typeSpec.Type.(*ast.MapType); ok {
		objectType := "object"
		schema.Type = &objectType
		propertySchema := &openapi.SchemaObject{}
		schema.AdditionalProperties = propertySchema
		typeAsString := types.TypeAsString(mapType.Value)
		typeAsString = strings.TrimLeft(typeAsString, "*")

		if !types.IsBasicGoType(typeAsString) {
			valueSchema, err := r.ResolveType(pkgName, typeAsString, true)
			if err != nil {
				return nil, err
			}
			if valueSchema.ID != "" {
				propertySchema.Ref = openapi.SchemaRef(valueSchema.ID)
			} else {
				*propertySchema = *valueSchema
			}
		} else if types.IsGoTypeOASType(typeAsString) {
			localGoType := types.GetOASType(typeAsString)
			propertySchema.Type = &localGoType
		}
		goto register_schema
	}

	// Handle interface types
	if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
		objectType := "object"
		schema.Type = &objectType
		schema.AdditionalProperties = &openapi.SchemaObject{}
		goto register_schema
	}

	// Handle selector types (e.g., time.Time, uuid.UUID)
	if selectorType, ok := typeSpec.Type.(*ast.SelectorExpr); ok {
		if ident, ok := selectorType.X.(*ast.Ident); ok {
			pkgAlias := ident.Name
			fieldName := selectorType.Sel.Name

			// Special handling for known types
			if pkgAlias == "bson" {
				switch fieldName {
				case "ObjectId":
					stringType := "string"
					schema.Type = &stringType
				case "M":
					objectType := "object"
					schema.Type = &objectType
					schema.AdditionalProperties = &openapi.SchemaObject{}
				}
			}
		}
		goto register_schema
	}

register_schema:
	if register {
		if err := r.Registry.RegisterSchemaObject(pkgName, typeName, schema); err != nil {
			return nil, err
		}
	}

	return schema, nil
}

// parseStructFields parses struct fields and adds them to the schema.
func (r *Resolver) parseStructFields(pkgName string, schema *openapi.SchemaObject, fields []*ast.Field) {
	// TODO: Implement struct field parsing with proper support for tags and embedded types
	// For now, this is a placeholder
}

func trimSplit(csl string) []string {
	s := strings.Split(csl, ",")
	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}
	return s
}
