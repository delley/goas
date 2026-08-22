package goas

import (
	"go/ast"

	"github.com/delley/goas/internal/openapi"
	internalSchema "github.com/delley/goas/internal/schema"
)

type schemaRegistry struct {
	registry       *internalSchema.Registry
	omitPackages   bool
	knownIDSchema  map[string]*openapi.SchemaObject
	apiSchemaNames map[string]map[string]string
}

func newSchemaRegistry(omitPackages bool) *schemaRegistry {
	registry := internalSchema.NewRegistry(omitPackages)
	return &schemaRegistry{
		registry:       registry,
		omitPackages:   omitPackages,
		knownIDSchema:  registry.KnownIDSchema,
		apiSchemaNames: registry.ApiSchemaNames,
	}
}

func (sr *schemaRegistry) validateSchemaNames() error {
	return sr.registry.ValidateSchemaNames()
}

func (sr *schemaRegistry) parseTypeAnnotations(pkgName string, typeName string, commentGroup *ast.CommentGroup) error {
	return sr.registry.ParseTypeAnnotations(pkgName, typeName, commentGroup)
}

func (sr *schemaRegistry) registerSchemaObject(pkgName, typeName string, schemaObject *openapi.SchemaObject) error {
	return sr.registry.RegisterSchemaObject(pkgName, typeName, schemaObject)
}

func (sr *schemaRegistry) genSchemaObjectID(pkgName, typeName string) string {
	return sr.registry.SchemaObjectID(pkgName, typeName)
}
