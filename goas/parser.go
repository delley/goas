package goas

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/delley/goas/internal/annotate"
	astpkg "github.com/delley/goas/internal/ast"
	"github.com/delley/goas/internal/desc"
	internalImports "github.com/delley/goas/internal/imports"
	"github.com/delley/goas/internal/load"
	"github.com/delley/goas/internal/openapi"
	"github.com/delley/goas/internal/scan"
	internalSchema "github.com/delley/goas/internal/schema"
	internalTypes "github.com/delley/goas/internal/types"
	"github.com/iancoleman/orderedmap"
)

// parser is the internal implementation of module discovery, annotation
// parsing, and OpenAPI model construction. Callers should use Generator.
type parser struct {
	ctx context.Context

	ModulePath string
	ModuleName string

	MainFilePath string

	HandlerPath string

	GoModFilePath string

	GoModCachePath string
	GoRootSrcPath  string

	OpenAPI openapi.OpenAPIObject

	schemaRegistry *internalSchema.Registry
	resources      *parseResources

	CorePkgs      map[string]bool
	KnownPkgs     []pkg
	KnownNamePkg  map[string]*pkg
	KnownPathPkg  map[string]*pkg
	KnownIDSchema map[string]*openapi.SchemaObject

	TypeSpecs map[string]map[string]*ast.TypeSpec

	// map of package name to type name to schema name
	ApiSchemaNames map[string]map[string]string

	Debug        bool
	OmitPackages bool
	ShowHidden   bool
	FileRefPath  string
}

type pkg = scan.Package

type parseResources struct {
	astCache       *astpkg.PackageCache
	importRegistry *internalImports.Registry
}

func newParseResources() *parseResources {
	return &parseResources{
		astCache:       astpkg.NewPackageCache(),
		importRegistry: internalImports.NewRegistry(),
	}
}

var (
	objectType = "object"
	stringType = "string"
	arrayType  = "array"
)

// Deprecated: NewParser is retained only for legacy source compatibility.
// It is not part of the supported public contract; use Generator and Options
// for all supported integrations. The returned value is intentionally opaque
// because the underlying parser type is an internal implementation detail.
func NewParser(modulePath, mainFilePath, handlerPath, descriptionRefPath string, debug, omitPackages, showHidden bool) (*parser, error) {
	return newParserContext(context.Background(), modulePath, mainFilePath, handlerPath, descriptionRefPath, debug, omitPackages, showHidden)
}

func newParser(modulePath, mainFilePath, handlerPath, descriptionRefPath string, debug, omitPackages, showHidden bool) (*parser, error) {
	return newParserContext(context.Background(), modulePath, mainFilePath, handlerPath, descriptionRefPath, debug, omitPackages, showHidden)
}

func newParserContext(ctx context.Context, modulePath, mainFilePath, handlerPath, descriptionRefPath string, debug, omitPackages, showHidden bool) (*parser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := &parser{
		ctx:           ctx,
		CorePkgs:      map[string]bool{},
		KnownPkgs:     []pkg{},
		KnownNamePkg:  map[string]*pkg{},
		KnownPathPkg:  map[string]*pkg{},
		KnownIDSchema: map[string]*openapi.SchemaObject{},
		TypeSpecs:     map[string]map[string]*ast.TypeSpec{},
		Debug:         debug,
		OmitPackages:  omitPackages,
		ShowHidden:    showHidden,
		FileRefPath:   descriptionRefPath,
	}
	p.resources = newParseResources()
	p.schemaRegistry = internalSchema.NewRegistry(omitPackages)
	p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
	p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
	p.OpenAPI.OpenAPI = openapi.OpenAPIVersion
	p.OpenAPI.Paths = make(openapi.PathsObject)
	p.OpenAPI.Security = []map[string][]string{}
	p.OpenAPI.Components.Schemas = make(map[string]*openapi.SchemaObject)
	p.OpenAPI.Components.SecuritySchemes = map[string]*openapi.SecuritySchemeObject{}

	moduleCtx, err := load.ResolveModuleContext(modulePath, mainFilePath, handlerPath, descriptionRefPath)
	if err != nil {
		return nil, err
	}
	p.ModulePath = moduleCtx.ModulePath
	p.debugf("module path: %s", p.ModulePath)
	p.GoModFilePath = moduleCtx.GoModFilePath
	p.debugf("go.mod file path: %s", p.GoModFilePath)
	p.MainFilePath = moduleCtx.MainFilePath
	p.debugf("main file path: %s", p.MainFilePath)
	p.ModuleName = moduleCtx.ModuleName
	p.debugf("module name: %s", p.ModuleName)
	p.GoModCachePath = moduleCtx.GoModCachePath
	p.debugf("go module cache path: %s", p.GoModCachePath)
	p.GoRootSrcPath = moduleCtx.GoRootSrcPath
	p.debugf("go root src path: %s", p.GoRootSrcPath)
	p.HandlerPath = moduleCtx.HandlerPath
	p.debugf("handler path: %s", p.HandlerPath)
	p.FileRefPath = moduleCtx.FileRefPath

	if p.ApiSchemaNames == nil {
		p.ApiSchemaNames = map[string]map[string]string{}
	}

	return p, nil
}

// parse runs the internal parsing stages in dependency order. Keeping this
// pipeline behind buildSpec prevents parser details from becoming public API.
func (p *parser) parse() error {
	if err := p.contextErr(); err != nil {
		return err
	}
	// parse basic info
	err := p.parseEntryPoint()
	if err != nil {
		return err
	}
	if err := p.contextErr(); err != nil {
		return err
	}

	// parse sub-package
	err = p.parseModule()
	if err != nil {
		return err
	}
	if err := p.contextErr(); err != nil {
		return err
	}

	// parse go.mod info
	err = p.parseGoMod()
	if err != nil {
		return err
	}
	if err := p.contextErr(); err != nil {
		return err
	}

	// parse core packages
	err = p.parseGoRoot()
	if err != nil {
		return err
	}

	// parse APIs info
	err = p.parseAPIs()
	if err != nil {
		return err
	}

	return nil
}

func (p *parser) contextErr() error {
	if p.ctx == nil {
		return nil
	}
	return p.ctx.Err()
}

func (p *parser) validateSchemaNames() error {
	if p.schemaRegistry == nil {
		p.schemaRegistry = internalSchema.NewRegistry(p.OmitPackages)
		p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
		p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
	}
	return p.schemaRegistry.ValidateSchemaNames()
}





func (p *parser) getPkgAst(pkgPath string) (map[string]*ast.Package, error) {
	if p.resources == nil {
		p.resources = newParseResources()
	}
	return p.resources.astCache.GetPackageAST(pkgPath)
}

func (p *parser) parseAPIs() error {
	err := p.parseImportStatements()
	if err != nil {
		return err
	}

	err = p.parseTypeSpecs()
	if err != nil {
		return err
	}

	return p.parsePaths()
}

func (p *parser) parseImportStatements() error {
	if p.resources == nil {
		p.resources = newParseResources()
	}
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name

		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parseImportStatements: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}

		for _, astPackageKey := range load.SortedKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range load.SortedKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astImport := range astFile.Imports {
					importedPkgName := strings.Trim(astImport.Path.Value, "\"")
					importedPkgAlias := ""
					if astImport.Name != nil {
						importedPkgAlias = internalImports.ResolveAlias(importedPkgName, astImport.Name.Name)
					} else {
						importedPkgAlias = internalImports.ResolveAlias(importedPkgName, "")
					}
					p.resources.importRegistry.Record(pkgName, importedPkgName, importedPkgAlias)
				}
			}
		}
	}
	return nil
}

func (p *parser) parseTypeSpecs() error {
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name

		_, ok := p.TypeSpecs[pkgName]
		if !ok {
			p.TypeSpecs[pkgName] = map[string]*ast.TypeSpec{}
		}
		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parseTypeSpecs: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}
		for _, astPackageKey := range load.SortedKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range load.SortedKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astDeclaration := range astFile.Decls {
					if astGenDeclaration, ok := astDeclaration.(*ast.GenDecl); ok && astGenDeclaration.Tok == token.TYPE {
						// find type declaration
						for _, astSpec := range astGenDeclaration.Specs {
							if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
								typeName := typeSpec.Name.String()
								p.TypeSpecs[pkgName][typeName] = typeSpec
								if astGenDeclaration.Doc != nil {
									err := p.parseTypeAnnotations(pkgName, typeName, astGenDeclaration.Doc)
									if err != nil {
										return err
									}
								}
							}
						}
					} else if astFuncDeclaration, ok := astDeclaration.(*ast.FuncDecl); ok {
						// find type declaration in func, method
						if astFuncDeclaration.Doc != nil && astFuncDeclaration.Doc.List != nil && astFuncDeclaration.Body != nil {
							funcName := astFuncDeclaration.Name.String()
							for _, astStmt := range astFuncDeclaration.Body.List {
								if astDeclStmt, ok := astStmt.(*ast.DeclStmt); ok {
									if astGenDeclaration, ok := astDeclStmt.Decl.(*ast.GenDecl); ok {
										for _, astSpec := range astGenDeclaration.Specs {
											if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
												// type in func
												if astFuncDeclaration.Recv == nil {
													p.TypeSpecs[pkgName][strings.Join([]string{funcName, typeSpec.Name.String()}, "@")] = typeSpec
													continue
												}
												// type in method
												var recvTypeName string
												if astStarExpr, ok := astFuncDeclaration.Recv.List[0].Type.(*ast.StarExpr); ok {
													recvTypeName = fmt.Sprintf("%s", astStarExpr.X)
												} else if astIdent, ok := astFuncDeclaration.Recv.List[0].Type.(*ast.Ident); ok {
													recvTypeName = astIdent.String()
												}
												p.TypeSpecs[pkgName][strings.Join([]string{recvTypeName, funcName, typeSpec.Name.String()}, "@")] = typeSpec
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *parser) parseTypeAnnotations(pkgName string, typeName string, commentGroup *ast.CommentGroup) error {
	if p.schemaRegistry == nil {
		p.schemaRegistry = internalSchema.NewRegistry(p.OmitPackages)
		p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
		p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
	}
	return p.schemaRegistry.ParseTypeAnnotations(pkgName, typeName, commentGroup)
}

func (p *parser) parseDescription(operation *openapi.OperationObject, description string) error {
	descText, err := desc.FetchRef(p.FileRefPath, description)
	if err != nil {
		return err
	}
	if operation.Description == "" {
		operation.Description = descText
	} else {
		operation.Description = operation.Description + " " + descText
	}
	return nil
}

func (p *parser) parseParamComment(pkgPath, pkgName string, operation *openapi.OperationObject, comment string) error {
	spec, err := annotate.ParseParamComment(comment)
	if err != nil {
		return err
	}

	// `file`, `form`
	if spec.In == "file" || spec.In == "files" || spec.In == "form" {
		if operation.RequestBody == nil {
			operation.RequestBody = &openapi.RequestBodyObject{
				Content: map[string]*openapi.MediaTypeObject{
					openapi.ContentTypeForm: {
						Schema: openapi.SchemaObject{
							Type:       &objectType,
							Properties: orderedmap.New(),
						},
					},
				},
				Required: spec.Required,
			}
		}
		if spec.In == "file" {
			operation.RequestBody.Content[openapi.ContentTypeForm].Schema.Properties.Set(spec.Name, &openapi.SchemaObject{
				Type:        &stringType,
				Format:      "binary",
				Description: spec.Description,
			})
		} else if spec.In == "files" {
			operation.RequestBody.Content[openapi.ContentTypeForm].Schema.Properties.Set(spec.Name, &openapi.SchemaObject{
				Type: &arrayType,
				Items: &openapi.SchemaObject{
					Type:   &stringType,
					Format: "binary",
				},
				Description: spec.Description,
			})
		} else if internalTypes.IsGoTypeOASType(spec.GoType) {
			localGoType := internalTypes.GetOASType(spec.GoType)
			operation.RequestBody.Content[openapi.ContentTypeForm].Schema.Properties.Set(spec.Name, &openapi.SchemaObject{
				Type:        &localGoType,
				Format:      internalTypes.GetOASFormat(spec.GoType),
				Description: spec.Description,
			})
		}
		return nil
	}

	// `path`, `query`, `header`, `cookie`
	if spec.In != "body" {
		parameterObject := openapi.ParameterObject{
			Name:        spec.Name,
			In:          spec.In,
			Description: spec.Description,
			Required:    spec.Required,
		}
		if spec.In == "path" {
			parameterObject.Required = true
		}
		if spec.GoType == "time.Time" {
			var err error
			parameterObject.Schema, err = p.parseSchemaObject(pkgPath, pkgName, spec.GoType, true)
			if err != nil {
				p.debug("parseParamComment cannot parse goType", spec.GoType)
			}
			operation.Parameters = append(operation.Parameters, parameterObject)
		} else if internalTypes.IsGoTypeOASType(spec.GoType) {
			localGoType := internalTypes.GetOASType(spec.GoType)
			parameterObject.Schema = &openapi.SchemaObject{
				Type:        &localGoType,
				Format:      internalTypes.GetOASFormat(spec.GoType),
				Description: spec.Description,
			}
			operation.Parameters = append(operation.Parameters, parameterObject)
		}
		return nil
	}

	if operation.RequestBody == nil {
		operation.RequestBody = &openapi.RequestBodyObject{
			Content:  map[string]*openapi.MediaTypeObject{},
			Required: spec.Required,
		}
	}

	s, err := p.parseBodyType(pkgPath, pkgName, spec.GoType)
	if err != nil {
		return err
	}
	operation.RequestBody.Content[openapi.ContentTypeJson] = &openapi.MediaTypeObject{
		Schema: *s,
	}

	// parse example
	if spec.ExampleRaw != "" {
		exampleRequestBody, err := annotate.ParseRequestBodyExample(spec.ExampleRaw)
		if err != nil {
			return err
		}
		operation.RequestBody.Content[openapi.ContentTypeJson].Example = exampleRequestBody
	}

	return nil
}

func (p *parser) parseBodyType(pkgPath, pkgName, typeName string) (*openapi.SchemaObject, error) {
	if strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "map[]") || typeName == "time.Time" {
		schema, err := p.parseSchemaObject(pkgPath, pkgName, typeName, true)
		if err != nil {
			p.debug("parseResponseComment cannot parse type", typeName)
		}
		return schema, nil
	}

	// handle oneOf/anyOf/allOf/not
	sob, err := p.handleCompoundType(pkgPath, pkgName, typeName)
	if sob != nil || err != nil {
		return sob, err
	}

	registeredTypeName, err := p.registerType(pkgPath, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if internalTypes.IsBasicGoType(registeredTypeName) {
		return &openapi.SchemaObject{
			Type: &stringType,
		}, nil
	} else {
		return &openapi.SchemaObject{
			Ref: openapi.SchemaRef(registeredTypeName),
		}, nil
	}
}

func (p *parser) parseResponseComment(pkgPath, pkgName string, operation *openapi.OperationObject, comment string) error {
	spec, err := annotate.ParseResponseComment(comment)
	if err != nil {
		return err
	}

	status := spec.Status

	responseObject := &openapi.ResponseObject{
		Content: map[string]*openapi.MediaTypeObject{},
	}
	responseObject.Description = spec.Description

	if spec.GoType != "" {
		goType := spec.GoType
		if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[]") {
			schema, err := p.parseSchemaObject(pkgPath, pkgName, goType, true)
			if err != nil {
				p.debug("parseResponseComment: cannot parse goType", goType)
			}
			responseObject.Content[openapi.ContentTypeJson] = &openapi.MediaTypeObject{Schema: *schema}
		} else {
			// aqui mantém seu comportamento original (mesmo detalhe do matches[3]):
			typeName, err := p.registerType(pkgPath, pkgName, goType)
			if err != nil {
				return err
			}
			if internalTypes.IsBasicGoType(typeName) {
				responseObject.Content[openapi.ContentTypeText] = &openapi.MediaTypeObject{
					Schema: openapi.SchemaObject{Type: &stringType},
				}
			} else {
				responseObject.Content[openapi.ContentTypeJson] = &openapi.MediaTypeObject{
					Schema: openapi.SchemaObject{Ref: openapi.SchemaRef(typeName)},
				}
			}
		}
	}
	operation.Responses[status] = responseObject
	return nil
}

func (p *parser) getSchemaObjectCached(pkgPath, pkgName, typeName string) (*openapi.SchemaObject, error) {
	var schemaObject *openapi.SchemaObject

	// see if we've already parsed this type
	if knownObj, ok := p.KnownIDSchema[p.genSchemaObjectID(pkgName, typeName)]; ok {
		schemaObject = knownObj
	} else if knownObj, ok := p.KnownIDSchema[typeName]; ok {
		schemaObject = knownObj
	} else {
		// if not, parse it now
		parsedObject, err := p.parseSchemaObject(pkgPath, pkgName, typeName, true)
		if err != nil {
			return schemaObject, err
		}
		schemaObject = parsedObject
	}

	return schemaObject, nil
}

func (p *parser) registerType(pkgPath, pkgName, typeName string) (string, error) {
	var registerTypeName string

	if internalTypes.IsBasicGoType(typeName) {
		registerTypeName = typeName
	} else {
		schemaObject, err := p.getSchemaObjectCached(pkgPath, pkgName, typeName)
		if err != nil {
			return "", err
		}
		registerTypeName = schemaObject.ID
	}

	if registerTypeName == "" {
		return "", fmt.Errorf("could not parse schema for %s %s %s", pkgName, pkgName, typeName)
	}

	return registerTypeName, nil
}

func trimSplit(csl string) []string {
	s := strings.Split(csl, ",")
	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}
	return s
}

func (p *parser) handleCompoundType(pkgPath, pkgName, typeName string) (*openapi.SchemaObject, error) {
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

	var sobs []*openapi.SchemaObject
	for i := range args {
		result, err := p.parseBodyType(pkgPath, pkgName, args[i])
		if err != nil {
			return nil, err
		}
		sobs = append(sobs, result)
	}

	sob := &openapi.SchemaObject{}
	switch op {
	case "not":
		sob.Not = sobs[0]
	case "oneof":
		sob.OneOf = sobs
	case "anyof":
		sob.AnyOf = sobs
	case "allof":
		sob.AllOf = sobs
	default:
		return nil, fmt.Errorf("invalid compound type '%s'", op)
	}

	return sob, nil
}

func (p *parser) parseSchemaObject(pkgPath, pkgName, typeName string, register bool) (*openapi.SchemaObject, error) {
	var typeSpec *ast.TypeSpec
	var exist bool
	var schemaObject openapi.SchemaObject

	// handler basic and some specific typeName
	if strings.HasPrefix(typeName, "[]") {
		schemaObject.Type = &arrayType
		itemTypeName := typeName[2:]
		schema, ok := p.KnownIDSchema[p.genSchemaObjectID(pkgName, itemTypeName)]
		if ok {
			schemaObject.Items = &openapi.SchemaObject{Ref: openapi.SchemaRef(schema.ID)}
			return &schemaObject, nil
		}

		newParsedSchema, err := p.parseSchemaObject(pkgPath, pkgName, itemTypeName, true)
		if err != nil {
			return nil, err
		}

		if newParsedSchema.ID != "" {
			schemaObject.Items = &openapi.SchemaObject{Ref: openapi.SchemaRef(newParsedSchema.ID)}
			return &schemaObject, nil
		}

		schemaObject.Items = newParsedSchema
		return &schemaObject, nil
	} else if strings.HasPrefix(typeName, "map[]") {
		schemaObject.Type = &objectType
		itemTypeName := typeName[5:]
		schema, ok := p.KnownIDSchema[p.genSchemaObjectID(pkgName, itemTypeName)]
		if ok {
			schemaObject.AdditionalProperties = &openapi.SchemaObject{Ref: openapi.SchemaRef(schema.ID)}
			return &schemaObject, nil
		}
		schemaProperty, err := p.parseSchemaObject(pkgPath, pkgName, itemTypeName, true)
		if err != nil {
			return nil, err
		}
		schemaObject.AdditionalProperties = schemaProperty
		return &schemaObject, nil
	} else if typeName == "time.Time" {
		schemaObject.Type = &stringType
		schemaObject.Format = "date-time"
		return &schemaObject, nil
	} else if typeName == "uuid.UUID" {
		schemaObject.Type = &stringType
		schemaObject.Format = "uuid"
		return &schemaObject, nil
	} else if strings.HasPrefix(typeName, "interface{}") {
		schemaObject.Type = nil
		return &schemaObject, nil
	} else if internalTypes.IsGoTypeOASType(typeName) {
		localGoType := internalTypes.GetOASType(typeName)
		schemaObject.Type = &localGoType
		return &schemaObject, nil
	}

	// handler other type
	typeNameParts := strings.Split(typeName, ".")
	if len(typeNameParts) == 1 {
		typeSpec, exist = p.getTypeSpec(pkgName, typeName)
		if !exist {
			for _, value := range p.KnownNamePkg {
				typeSpec, exist = p.getTypeSpec(value.Name, typeName)
				if exist {
					pkgPath = value.Path
					pkgName = value.Name
					break
				}
			}
			if !exist {
				return nil, fmt.Errorf("can not find definition of %s ast.TypeSpec in current package %s", typeName, pkgName)
			}
		}
		schemaObject.PkgName = pkgName
		schemaObject.ID = p.genSchemaObjectID(pkgName, typeName)
		p.KnownIDSchema[schemaObject.ID] = &schemaObject
	} else {
		guessPkgName := strings.Join(typeNameParts[:len(typeNameParts)-1], "/")
		guessPkgPath := ""
		for i := range p.KnownPkgs {
			if guessPkgName == p.KnownPkgs[i].Name {
				guessPkgPath = p.KnownPkgs[i].Path
				break
			}
		}
		guessTypeName := typeNameParts[len(typeNameParts)-1]
		typeSpec, exist = p.getTypeSpec(guessPkgName, guessTypeName)
		if !exist {
			found := false
			if p.resources != nil {
				for k := range p.resources.importRegistry.PkgNameImportedPkgAlias[pkgName] {
					if k == guessPkgName && len(p.resources.importRegistry.PkgNameImportedPkgAlias[pkgName][guessPkgName]) != 0 {
						found = true
						break
					}
				}
			}
			if !found {
				p.debugf("unknown guess %s ast.TypeSpec in package %s", guessTypeName, guessPkgName)
				return &schemaObject, nil
			}
			guessPkgName = p.resources.importRegistry.PkgNameImportedPkgAlias[pkgName][guessPkgName][0]
			guessPkgPath = ""
			for i := range p.KnownPkgs {
				if guessPkgName == p.KnownPkgs[i].Name {
					guessPkgPath = p.KnownPkgs[i].Path
					break
				}
			}
			// p.debugf("guess %s ast.TypeSpec in package %s", guessTypeName, guessPkgName)

			typeSpec, exist = p.getTypeSpec(guessPkgName, guessTypeName)
			if !exist {
				if p.CorePkgs[guessPkgName] {
					p.debugf("Ignoring missing type %s in core package %s", guessTypeName, guessPkgName)
					schemaObject.Type = &objectType
					return &schemaObject, nil
				}

				return nil, fmt.Errorf("cannot find definition of guessed %s ast.TypeSpec in package %s; if the definition is in a vendor dependency, try running `go mod tidy && go mod vendor`",
					guessTypeName, guessPkgName)
			}
			schemaObject.PkgName = guessPkgName
			schemaObject.ID = p.genSchemaObjectID(guessPkgName, guessTypeName)
			p.KnownIDSchema[schemaObject.ID] = &schemaObject
		}
		pkgPath, pkgName = guessPkgPath, guessPkgName
	}

	if internalTypes.IsGoTypeOASType(internalTypes.TypeAsString(typeSpec.Type)) && schemaObject.Ref == "" {
		typeAsString := internalTypes.TypeAsString(typeSpec.Type)
		localGoType := internalTypes.GetOASType(typeAsString)
		schemaObject.Type = &localGoType
		if internalTypes.GetOASFormat(typeAsString) == "int64" {
			schemaObject.Format = "int64"
		}

	} else if astIdent, ok := typeSpec.Type.(*ast.Ident); ok {
		// this is for type aliases to custom types
		newSchema, err := p.parseSchemaObject(pkgPath, pkgName, astIdent.Name, true)
		if err != nil {
			return nil, err
		}
		schemaObject.Ref = openapi.SchemaRef(newSchema.ID)
	} else if astStructType, ok := typeSpec.Type.(*ast.StructType); ok {
		schemaObject.Type = &objectType
		if astStructType.Fields != nil {
			p.parseSchemaPropertiesFromStructFields(pkgPath, pkgName, &schemaObject, astStructType.Fields.List)
		}
	} else if astArrayType, ok := typeSpec.Type.(*ast.ArrayType); ok {
		schemaObject.Type = &arrayType
		schemaObject.Items = &openapi.SchemaObject{}
		typeAsString := p.getTypeAsString(astArrayType.Elt)
		typeAsString = strings.TrimLeft(typeAsString, "*")

		if !internalTypes.IsBasicGoType(typeAsString) {
			itemsSchema, err := p.getSchemaObjectCached(pkgPath, pkgName, typeAsString)
			if err != nil {
				p.debug("parseSchemaObject parse array items err:", err)
			} else {
				if itemsSchema.ID != "" {
					schemaObject.Items.Ref = openapi.SchemaRef(itemsSchema.ID)
				} else {
					*schemaObject.Items = *itemsSchema
				}
			}
		} else if internalTypes.IsGoTypeOASType(typeAsString) {
			localGoType := internalTypes.GetOASType(typeAsString)
			schemaObject.Items.Type = &localGoType
		}
	} else if astMapType, ok := typeSpec.Type.(*ast.MapType); ok {
		schemaObject.Type = &objectType
		propertySchema := &openapi.SchemaObject{}
		schemaObject.AdditionalProperties = propertySchema
		typeAsString := p.getTypeAsString(astMapType.Value)
		typeAsString = strings.TrimLeft(typeAsString, "*")
		if !internalTypes.IsBasicGoType(typeAsString) {
			keySchema, err := p.getSchemaObjectCached(pkgPath, pkgName, typeAsString)
			if err != nil {
				p.debug("parseSchemaObject parse array items err:", err)
			} else {
				if keySchema.ID != "" {
					propertySchema.Ref = openapi.SchemaRef(keySchema.ID)
				} else {
					*propertySchema = *keySchema
				}
			}
		} else if internalTypes.IsGoTypeOASType(typeAsString) {
			localGoType := internalTypes.GetOASType(typeAsString)
			propertySchema.Type = &localGoType
		}
	} else if selectorType, ok := typeSpec.Type.(*ast.SelectorExpr); ok {
		// this case is for referencing third party packages.
		packageIdentifier, ok := selectorType.X.(*ast.Ident)
		usedTypeName := selectorType.Sel.Name
		if ok {
			if packageIdentifier.Name == "bson" {
				switch usedTypeName {
				case "ObjectId":
					schemaObject.Type = &stringType
				case "M":
					schemaObject.Type = &objectType
					schemaObject.AdditionalProperties = &openapi.SchemaObject{}
				}
			}

			packageName := packageIdentifier.Name
			for potentialPackage, typeSpecs := range p.TypeSpecs {
				if strings.HasSuffix(potentialPackage, packageName) {
					// iterate through types of that package
					for name := range typeSpecs {
						if name == usedTypeName {
							parsedPackageSchema, err := p.parseSchemaObject(potentialPackage, potentialPackage, usedTypeName, false)
							if err != nil {
								return nil, err
							}
							schemaObject.Type = parsedPackageSchema.Type
							schemaObject.Properties = parsedPackageSchema.Properties
							schemaObject.AdditionalProperties = parsedPackageSchema.AdditionalProperties
							break
						}
					}
				}
			}

		}
	} else if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
		// free form object since the interface can be "anything"
		schemaObject.Type = &objectType
		schemaObject.AdditionalProperties = &openapi.SchemaObject{}
	}

	// we don't want to register 3rd party library types
	if register {
		if p.schemaRegistry == nil {
			p.schemaRegistry = internalSchema.NewRegistry(p.OmitPackages)
			p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
			p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
		}
		if err := p.schemaRegistry.RegisterSchemaObject(pkgName, typeName, &schemaObject); err != nil {
			return nil, err
		}
		if _, ok := p.OpenAPI.Components.Schemas[openapi.NormalizeComponentSchemaKey(schemaObject.ID)]; !ok {
			p.OpenAPI.Components.Schemas[openapi.NormalizeComponentSchemaKey(schemaObject.ID)] = &schemaObject
		}
	}

	return &schemaObject, nil
}

func (p *parser) getTypeSpec(pkgName, typeName string) (*ast.TypeSpec, bool) {
	pkgTypeSpecs, exist := p.TypeSpecs[pkgName]
	if !exist {
		return nil, false
	}
	astTypeSpec, exist := pkgTypeSpecs[typeName]
	if !exist {
		return nil, false
	}
	return astTypeSpec, true
}

func (p *parser) parseAstFields(pkgPath, pkgName string, structSchema *openapi.SchemaObject, astFields []*ast.Field) {
	for _, astField := range astFields {
		p.parseAstField(pkgPath, pkgName, structSchema, astField)
	}
}

func (p *parser) parseAstField(pkgPath, pkgName string, structSchema *openapi.SchemaObject, astField *ast.Field) {
	fieldSchema := &openapi.SchemaObject{}
	typeAsString := internalTypes.TypeAsString(astField.Type)
	if renderedStruct := parseOverrideStructTag(astField); renderedStruct != "" {
		typeAsString = renderedStruct
	}

	isSliceOrMap := strings.HasPrefix(typeAsString, "[]") || strings.HasPrefix(typeAsString, "map[]")
	isInterface := strings.HasPrefix(typeAsString, "interface{}")
	if isSliceOrMap || isInterface || typeAsString == "time.Time" || typeAsString == "uuid.UUID" {
		splitType := strings.Split(typeAsString, "]")
		if len(splitType) > 1 && !internalTypes.IsBasicGoType(splitType[1]) {
			if _, ok := p.KnownIDSchema[splitType[1]]; ok {
				nestedType := internalTypes.TypeAsString(splitType[1])
				setNestedFieldSchemaProps(splitType[0], nestedType, fieldSchema, structSchema)
			} else {
				var err error
				p.registerType(pkgPath, pkgName, typeAsString)
				fieldSchema, err = p.parseSchemaObject(pkgPath, pkgName, typeAsString, true)
				if err != nil {
					p.debug(err)
					return
				}
			}
		} else {
			var err error
			fieldSchema, err = p.parseSchemaObject(pkgPath, pkgName, typeAsString, true)
			if err != nil {
				p.debug(err)
				return
			}
		}
	} else if !internalTypes.IsBasicGoType(typeAsString) {
		fieldSchemaObjectID, err := p.registerType(pkgPath, pkgName, typeAsString)
		if err != nil {
			p.debug("parseSchemaPropertiesFromStructFields err:", err)
		} else {
			fieldSchema.ID = fieldSchemaObjectID
			fieldSchema.Ref = openapi.SchemaRef(fieldSchemaObjectID)
		}
	} else if internalTypes.IsGoTypeOASType(typeAsString) {
		localGoType := internalTypes.GetOASType(typeAsString)
		fieldSchema.Type = &localGoType
		if internalTypes.GetOASFormat(typeAsString) == "int64" {
			fieldSchema.Format = "int64"
		}
	}
	// for embedded fields
	if len(astField.Names) == 0 {
		if fieldSchema.Properties != nil {
			for _, propertyName := range fieldSchema.Properties.Keys() {
				_, exist := structSchema.Properties.Get(propertyName)
				if exist {
					return
				}
				propertySchema, _ := fieldSchema.Properties.Get(propertyName)
				structSchema.Properties.Set(propertyName, propertySchema)
			}
			// for _, required := range fieldSchema.Required {
			// 	structSchema.Required = append(structSchema.Required, required)
			// }
			structSchema.Required = append(structSchema.Required, fieldSchema.Required...)
		} else if len(fieldSchema.Ref) != 0 && len(fieldSchema.ID) != 0 {
			refSchema, ok := p.KnownIDSchema[fieldSchema.ID]
			if ok {
				if refSchema.Properties == nil {
					p.debug("nil refSchema.Properties")
					return
				}
				for _, propertyName := range refSchema.Properties.Keys() {
					refPropertySchema, _ := refSchema.Properties.Get(propertyName)
					_, disabled := structSchema.DisabledFieldNames[refPropertySchema.(*openapi.SchemaObject).FieldName]
					if disabled {
						return
					}
					_, exist := structSchema.Properties.Get(propertyName)
					if exist {
						return
					}
					structSchema.Properties.Set(propertyName, refPropertySchema)
				}
				structSchema.Required = append(structSchema.Required, refSchema.Required...)
			}
		}
	} else {
		name := astField.Names[0].Name
		fieldSchema.FieldName = name
		_, disabled := structSchema.DisabledFieldNames[name]
		if disabled {
			return
		}

		newName, skip := parseStructTags(astField, structSchema, fieldSchema, name)
		if skip {
			return
		}

		name = newName

		structSchema.Properties.Set(name, fieldSchema)
	}
}

func (p *parser) parseSchemaPropertiesFromStructFields(pkgPath, pkgName string, structSchema *openapi.SchemaObject, astFields []*ast.Field) {
	if astFields == nil {
		return
	}
	structSchema.Properties = orderedmap.New()
	if structSchema.DisabledFieldNames == nil {
		structSchema.DisabledFieldNames = map[string]struct{}{}
	}

	p.parseAstFields(pkgPath, pkgName, structSchema, astFields)
}

// getTypeAsString delegates to internal/types.TypeAsString for backward compatibility.
// This method can be removed after all callers are updated to use internalTypes.TypeAsString directly.
func (p *parser) getTypeAsString(fieldType interface{}) string {
	return internalTypes.TypeAsString(fieldType)
}

func parseOverrideStructTag(astField *ast.Field) (renderedStructName string) {
	if astField.Tag != nil {
		astFieldTag := reflect.StructTag(strings.Trim(astField.Tag.Value, "`"))
		if renderedStructName := astFieldTag.Get("overrideApiSchemaType"); renderedStructName != "" {
			return renderedStructName
		}
	}
	return renderedStructName
}

func normalizeStructTagLiteral(tagLiteral string) string {
	tagLiteral = strings.TrimSpace(tagLiteral)
	if len(tagLiteral) >= 2 {
		if tagLiteral[0] == '`' && tagLiteral[len(tagLiteral)-1] == '`' {
			return tagLiteral[1 : len(tagLiteral)-1]
		}
		if tagLiteral[0] == '\'' && tagLiteral[len(tagLiteral)-1] == '\'' {
			return tagLiteral[1 : len(tagLiteral)-1]
		}
	}
	return tagLiteral
}

func parseStructTags(astField *ast.Field, structSchema *openapi.SchemaObject, fieldSchema *openapi.SchemaObject, name string) (newName string, skip bool) {
	if astField.Tag != nil {
		astFieldTag := reflect.StructTag(normalizeStructTagLiteral(astField.Tag.Value))
		tagText := ""

		if tag := astFieldTag.Get("goas"); tag != "" {
			tagText = tag
		}
		tagValues := strings.Split(tagText, ",")
		for _, v := range tagValues {
			if v == "-" {
				structSchema.DisabledFieldNames[name] = struct{}{}
				fieldSchema.Deprecated = true
				return "", true
			}
			parseTagValue := strings.Split(v, "=")
			if len(parseTagValue) > 0 {
				if parseTagValue[0] == "enum" {
					if fieldSchema.Type != nil && *fieldSchema.Type == "array" {
						fieldSchema.Items.Enum = strings.Split(parseTagValue[1], " ")
					} else {
						fieldSchema.Enum = strings.Split(parseTagValue[1], " ")
					}
				}
			}
		}

		if tag := astFieldTag.Get("json"); tag != "" {
			tagText = tag
		}
		tagValues = strings.Split(tagText, ",")
		isRequired := false
		for _, v := range tagValues {
			if v == "-" {
				structSchema.DisabledFieldNames[name] = struct{}{}
				fieldSchema.Deprecated = true
				return "", true
			} else if v != "" && v != "omitempty" {
				name = v
			}
		}

		if tag := astFieldTag.Get("example"); tag != "" {
			if fieldSchema.Type == nil {
				fieldSchema.Example = tag
			} else {
				switch *fieldSchema.Type {
				case "boolean":
					fieldSchema.Example, _ = strconv.ParseBool(tag)
				case "integer":
					fieldSchema.Example, _ = strconv.Atoi(tag)
				case "number":
					fieldSchema.Example, _ = strconv.ParseFloat(tag, 64)
				case "array":
					b, err := json.RawMessage(tag).MarshalJSON()
					if err != nil {
						fieldSchema.Example = "invalid example"
					} else {
						sliceOfInterface := []interface{}{}
						err := json.Unmarshal(b, &sliceOfInterface)
						if err != nil {
							fieldSchema.Example = "invalid example"
						} else {
							fieldSchema.Example = sliceOfInterface
						}
					}
				case "object":
					b, err := json.RawMessage(tag).MarshalJSON()
					if err != nil {
						fieldSchema.Example = "invalid example"
					} else {
						mapOfInterface := map[string]interface{}{}
						err := json.Unmarshal(b, &mapOfInterface)
						if err != nil {
							fieldSchema.Example = "invalid example"
						} else {
							fieldSchema.Example = mapOfInterface
						}
					}
				default:
					fieldSchema.Example = tag
				}
			}
		}
		if _, ok := astFieldTag.Lookup("required"); ok || isRequired {
			structSchema.Required = append(structSchema.Required, name)
		}

		if desc := astFieldTag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}
	}

	return name, false
}

func (p *parser) debug(v ...interface{}) {
	if p.Debug {
		log.Println(v...)
	}
}

func (p *parser) debugf(format string, args ...interface{}) {
	if p.Debug {
		log.Printf(format, args...)
	}
}

func (p *parser) genSchemaObjectID(pkgName, typeName string) string {
	if p.schemaRegistry == nil {
		p.schemaRegistry = internalSchema.NewRegistry(p.OmitPackages)
		p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
		p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
	}
	return p.schemaRegistry.SchemaObjectID(pkgName, typeName)
}

func setNestedFieldSchemaProps(valuePrefix, typeAsString string, fieldSchema, structSchema *openapi.SchemaObject) {
	if strings.HasPrefix(valuePrefix, "map") {
		fieldSchema.Type = &objectType
		fieldSchema.AdditionalProperties = &openapi.SchemaObject{Ref: openapi.SchemaRef(typeAsString)}
	} else {
		fieldSchema.Type = &arrayType
		fieldSchema.Items = &openapi.SchemaObject{Ref: openapi.SchemaRef(typeAsString)}
	}
}
