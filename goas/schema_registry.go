package goas

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/delley/goas/internal/annotate"
	"github.com/delley/goas/internal/openapi"
)

type schemaRegistry struct {
	omitPackages   bool
	knownIDSchema  map[string]*openapi.SchemaObject
	apiSchemaNames map[string]map[string]string
}

func newSchemaRegistry(omitPackages bool) *schemaRegistry {
	return &schemaRegistry{
		omitPackages:   omitPackages,
		knownIDSchema:  map[string]*openapi.SchemaObject{},
		apiSchemaNames: map[string]map[string]string{},
	}
}

func (sr *schemaRegistry) validateSchemaNames() error {
	potentialConflictsMap := map[string][]string{}
	for pkgName, schemaNames := range sr.apiSchemaNames {
		for typeName, schemaName := range schemaNames {
			potentialConflictsMap[schemaName] = append(potentialConflictsMap[schemaName], pkgName+"#"+typeName)
		}
	}

	conflicts := []string{}
	for schemaName := range potentialConflictsMap {
		if len(potentialConflictsMap[schemaName]) > 1 {
			conflicts = append(conflicts, schemaName+": "+strings.Join(potentialConflictsMap[schemaName], " | "))
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("conflicting schema names - %s", strings.Join(conflicts, ", "))
	}
	return nil
}

func (sr *schemaRegistry) parseTypeAnnotations(pkgName string, typeName string, commentGroup *ast.CommentGroup) error {
	alias, ok, err := annotate.ParseApiSchemaName(commentGroup)
	if err != nil || !ok {
		return err
	}
	if sr.apiSchemaNames[pkgName] == nil {
		sr.apiSchemaNames[pkgName] = map[string]string{}
	}
	sr.apiSchemaNames[pkgName][typeName] = alias
	return nil
}

func (sr *schemaRegistry) registerSchemaObject(pkgName, typeName string, schemaObject *openapi.SchemaObject) error {
	if schemaObject == nil || schemaObject.ID == "" {
		return nil
	}

	if _, ok := sr.knownIDSchema[schemaObject.ID]; !ok {
		sr.knownIDSchema[schemaObject.ID] = schemaObject
	}

	if _, ok := sr.apiSchemaNames[pkgName]; !ok {
		sr.apiSchemaNames[pkgName] = map[string]string{}
	}

	typeNameWithoutPackage := typeName
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeNameWithoutPackage = typeName[idx+1:]
	}
	if schemaName, ok := sr.apiSchemaNames[pkgName][typeNameWithoutPackage]; ok {
		if schemaName != schemaObject.ID {
			return fmt.Errorf("different schema object id for type %s#%s: %s vs %s", pkgName, typeNameWithoutPackage, schemaObject.ID, schemaName)
		}
		return nil
	}

	sr.apiSchemaNames[pkgName][typeNameWithoutPackage] = schemaObject.ID
	return nil
}

func (sr *schemaRegistry) genSchemaObjectID(pkgName, typeName string) string {
	apiSchemaName, ok := sr.apiSchemaNames[pkgName][typeName]
	if ok {
		return apiSchemaName
	}

	typeNameParts := strings.Split(typeName, ".")
	pkgName = replaceBackslash(pkgName)
	pkgNameParts := strings.Split(pkgName, "/")
	if sr.omitPackages || pkgNameParts[len(pkgNameParts)-1] == "" {
		return typeNameParts[len(typeNameParts)-1]
	}
	return strings.Join(append([]string{pkgNameParts[len(pkgNameParts)-1]}, typeNameParts[len(typeNameParts)-1]), ".")
}
