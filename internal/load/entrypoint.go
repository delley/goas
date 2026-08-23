package load

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func IsMainFile(path string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, err
	}

	if file.Name.Name != "main" {
		return false, nil
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == "main" && function.Type.Params.NumFields() == 0 && function.Type.Results == nil {
			return true, nil
		}
	}
	return false, nil
}
