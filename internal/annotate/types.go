package annotate

import "net/http"

type RouteSpec struct {
	Path   string
	Method string
}

func (r RouteSpec) MethodUpper() string {
	return http.CanonicalHeaderKey(r.Method)
}

type OperationMeta struct {
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Security    []map[string][]string // compatível com OpenAPI (security requirement)
}

type ParamSpec struct {
	Name        string
	In          string // "query", "path", "header", "cookie"
	Required    bool
	Description string
	GoType      string
	ExampleRaw  string
}

type ResponseSpec struct {
	Status      string // mantemos string porque o engine usa map[string]...
	JSONType    string // pode ser "" se não informado
	GoType      string // normalizado ([] em vez de [x]) ou "" se não informado
	Description string // sem aspas (trim de ")
}
