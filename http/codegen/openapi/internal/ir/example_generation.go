package ir

import "github.com/CaliLuke/loom/expr"

type openAPIExampleSource struct {
	value    any
	declared bool
	present  bool
}

func synthesizedOpenAPIExample(attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) openAPIExampleSource {
	if len(attribute.ExtractUserExamples()) > 0 {
		return openAPIExampleSource{value: attribute.Example(generator), present: true}
	}
	if generator == nil || generator.Randomizer == nil {
		return openAPIExampleSource{}
	}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return openAPIExampleSource{}
	}
	suppress := false
	if disabled, ok := attribute.Meta.Last("openapi:example"); ok && disabled == "false" {
		suppress = true
	}
	selection := context.SelectExample(occurrence, expr.ExamplePolicy{
		Reachable:         true,
		SuppressGenerated: suppress,
	})
	result := context.Synthesize(selection, generator)
	value, ok := result.DeclaredJSONValue()
	if !ok {
		return openAPIExampleSource{}
	}
	return openAPIExampleSource{value: value, declared: true, present: true}
}
