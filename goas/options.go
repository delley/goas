package goas

// Options is the public configuration contract for OpenAPI generation.
//
// The generator consumes these options as input; module discovery, parsing,
// schema registration, and serialization remain implementation details.
type Options struct {
	ModulePath   string
	MainFilePath string
	HandlerPath  string
	FileRefPath  string
	OutputPath   string
	Debug        bool
	OmitPackages bool
	ShowHidden   bool
}
