package goas

import (
	"github.com/delley/goas/internal/openapi"
)

// checkFormatInt64 will see if the type is int64 and add to Format property if true
func checkFormatInt64(typeName string, schemaObject *openapi.SchemaObject) {
	if typeName == "int64" {
		schemaObject.Format = "int64"
	}
}
