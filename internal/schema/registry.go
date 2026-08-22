package schema

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/delley/goas/internal/annotate"
	"github.com/delley/goas/internal/openapi"
)

// Registry owns schema names, schema identity deduplication, and conflict validation.
type Registry struct {
	OmitPackages   bool
	KnownIDSchema  map[string]*openapi.SchemaObject
	ApiSchemaNames map[string]map[string]string
}

func NewRegistry(omitPackages bool) *Registry {
	return &Registry{
		OmitPackages:   omitPackages,
		KnownIDSchema:  map[string]*openapi.SchemaObject{},
		ApiSchemaNames: map[string]map[string]string{},
	}
}

func (r *Registry) ValidateSchemaNames() error {
	potentialConflicts := map[string][]string{}
	for pkgName, schemaNames := range r.ApiSchemaNames {
		for typeName, schemaName := range schemaNames {
			potentialConflicts[schemaName] = append(potentialConflicts[schemaName], pkgName+"#"+typeName)
		}
	}

	conflicts := []string{}
	for schemaName, owners := range potentialConflicts {
		if len(owners) > 1 {
			conflicts = append(conflicts, schemaName+": "+strings.Join(owners, " | "))
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("conflicting schema names - %s", strings.Join(conflicts, ", "))
	}
	return nil
}

func (r *Registry) ParseTypeAnnotations(pkgName, typeName string, comments *ast.CommentGroup) error {
	alias, ok, err := annotate.ParseApiSchemaName(comments)
	if err != nil || !ok {
		return err
	}
	if r.ApiSchemaNames[pkgName] == nil {
		r.ApiSchemaNames[pkgName] = map[string]string{}
	}
	r.ApiSchemaNames[pkgName][typeName] = alias
	return nil
}

func (r *Registry) RegisterSchemaObject(pkgName, typeName string, schemaObject *openapi.SchemaObject) error {
	if schemaObject == nil || schemaObject.ID == "" {
		return nil
	}
	if _, exists := r.KnownIDSchema[schemaObject.ID]; !exists {
		r.KnownIDSchema[schemaObject.ID] = schemaObject
	}
	if r.ApiSchemaNames[pkgName] == nil {
		r.ApiSchemaNames[pkgName] = map[string]string{}
	}
	typeName = typeWithoutPackage(typeName)
	if schemaName, exists := r.ApiSchemaNames[pkgName][typeName]; exists {
		if schemaName != schemaObject.ID {
			return fmt.Errorf("different schema object id for type %s#%s: %s vs %s", pkgName, typeName, schemaObject.ID, schemaName)
		}
		return nil
	}
	r.ApiSchemaNames[pkgName][typeName] = schemaObject.ID
	return nil
}

func (r *Registry) SchemaObjectID(pkgName, typeName string) string {
	if schemaName, exists := r.ApiSchemaNames[pkgName][typeName]; exists {
		return schemaName
	}
	typeNameParts := strings.Split(typeName, ".")
	pkgNameParts := strings.Split(strings.ReplaceAll(pkgName, "\\", "/"), "/")
	if r.OmitPackages || pkgNameParts[len(pkgNameParts)-1] == "" {
		return typeNameParts[len(typeNameParts)-1]
	}
	return pkgNameParts[len(pkgNameParts)-1] + "." + typeNameParts[len(typeNameParts)-1]
}

func typeWithoutPackage(typeName string) string {
	if index := strings.LastIndex(typeName, "."); index >= 0 {
		return typeName[index+1:]
	}
	return typeName
}
