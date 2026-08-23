package annotate

type RouteSpec struct {
	Path   string
	Method string
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
