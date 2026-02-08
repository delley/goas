package goas

// Options holds the configuration options for the OpenAPI Specification generator.
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
