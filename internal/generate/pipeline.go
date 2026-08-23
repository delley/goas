package generate

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/delley/goas/internal/openapi"
)

// DebugLogger is a small dependency used by the pipeline to emit diagnostics
// without leaking domain-specific flags across the generation stack.
type DebugLogger interface {
	Printf(string, ...interface{})
}

type noopDebugLogger struct{}

func (noopDebugLogger) Printf(string, ...interface{}) {}

// NewDebugLogger returns a noop logger unless enabled, so generation code can
// stay independent from the CLI flag surface.
func NewDebugLogger(enabled bool) DebugLogger {
	if !enabled {
		return noopDebugLogger{}
	}
	return log.New(os.Stderr, "[goas] ", log.LstdFlags)
}

// Input represents the data needed to build an OpenAPI document.
type Input struct {
	ModulePath   string
	MainFilePath string
	HandlerPath  string
	FileRefPath  string

	OmitPackages bool
	ShowHidden   bool
	Debug        bool
}

// Document is the mutable unit passed between pipeline phases.
type Document struct {
	Spec *openapi.OpenAPIObject
}

// Phase is a named stage in the generation pipeline.
type Phase struct {
	Name string
	Run  func(context.Context, *Document) error
}

// Pipeline orchestrates generation in a sequence of explicit stages.
type Pipeline struct {
	ctx   context.Context
	debug DebugLogger
}

// NewPipeline creates a pipeline instance bound to ctx and a small debug sink.
func NewPipeline(ctx context.Context, debug DebugLogger) *Pipeline {
	if ctx == nil {
		ctx = context.Background()
	}
	if debug == nil {
		debug = noopDebugLogger{}
	}
	return &Pipeline{ctx: ctx, debug: debug}
}

// Run executes a build step and then each named phase in order. A cancelled or
// expired context is returned immediately before and after each stage.
func (p *Pipeline) Run(input Input, build func(context.Context, Input) (*Document, error), phases ...Phase) (*openapi.OpenAPIObject, error) {
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}

	doc, err := build(p.ctx, input)
	if err != nil {
		return nil, err
	}
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("generate: nil document")
	}
	if doc.Spec == nil {
		return nil, fmt.Errorf("generate: nil document spec")
	}

	for _, phase := range phases {
		if err := p.ctx.Err(); err != nil {
			return nil, err
		}
		if phase.Run != nil {
			p.debug.Printf("generate: running phase %s", phase.Name)
			if err := phase.Run(p.ctx, doc); err != nil {
				return nil, err
			}
		}
		if err := p.ctx.Err(); err != nil {
			return nil, err
		}
	}

	return doc.Spec, nil
}
