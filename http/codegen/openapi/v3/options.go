package openapiv3

import (
	"github.com/CaliLuke/loom/codegen/service"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
)

// Option configures OpenAPI generation.
type Option func(*generationOptions)

type generationOptions struct {
	semantic *service.SemanticData
}

// WithServiceSemantics uses the compilation's captured service occurrences.
// The owner must have been captured from the same root after design preparation.
// Documentation example selection remains independent of Go generation.
func WithServiceSemantics(semantic *service.SemanticData) Option {
	return func(options *generationOptions) {
		options.semantic = semantic
	}
}

func analyzerOptions(options []Option) []openapiir.AnalyzerOption {
	var settings generationOptions
	for _, option := range options {
		option(&settings)
	}
	return []openapiir.AnalyzerOption{openapiir.WithServiceSemantics(settings.semantic)}
}
