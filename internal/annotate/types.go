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
	Security    []map[string][]string
}

type ParamSpec struct {
	Name        string
	In          string
	Required    bool
	Description string
	GoType      string
	ExampleRaw  string
}

type ResponseSpec struct {
	Status      string
	JSONType    string
	GoType      string
	Description string
}
