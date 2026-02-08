package load

type ModuleInfo struct {
	Name string // ex: github.com/delley/goas
	Dir  string // base dir (onde está o go.mod)
}

type EntryPoint struct {
	PkgPath string // ex: github.com/delley/goas/example
	PkgName string // ex: example
	File    string // path do arquivo main.go encontrado
	Dir     string // diretório do entrypoint
}
