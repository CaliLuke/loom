package ir

import (
	"encoding/json/jsontext"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func preparedOpenAPIExampleValue(example transportir.ValueExample) (any, bool) {
	if example.Source == nil {
		return nil, false
	}
	projected := example.Source.Context.ProjectJSON(example.Source.Example, example.Plan)
	raw, ok := projected.JSON()
	if !ok {
		return nil, false
	}
	return materializeOpenAPIRawJSON(jsontext.Value(raw)), true
}

func preparedDeclaredExampleValue(example transportir.ValueExample) (any, bool) {
	if example.Source == nil {
		return nil, false
	}
	return example.Source.Context.DeclaredJSONValue(example.Source.Example, example.Plan)
}

func initPreparedExamples(target interface {
	setExample(any)
	setExamples(map[string]*ExampleRef)
}, prepared *transportir.ValueTarget) bool {
	if prepared == nil || !prepared.ExamplesPrepared {
		return false
	}
	if len(prepared.Examples) == 0 {
		if prepared.Representative != nil {
			if value, ok := preparedOpenAPIExampleValue(*prepared.Representative); ok {
				target.setExample(value)
			}
		}
		return true
	}
	examples := scopedPreparedExamples(prepared.Examples, "media")
	if len(examples) == 1 &&
		!hasStructuredExampleMetadata(examples[0].Meta) &&
		metaValue(examples[0].Meta, "openapi:component:example") == "" {
		if value, ok := preparedOpenAPIExampleValue(examples[0]); ok {
			target.setExample(value)
		}
		return true
	}
	references := make(map[string]*ExampleRef, len(examples))
	for _, example := range examples {
		value, ok := preparedOpenAPIExampleValue(example)
		if !ok {
			continue
		}
		name := example.Summary
		if name == "" {
			name = "default"
		}
		references[name] = &ExampleRef{Value: buildPreparedExample(example, value)}
	}
	if len(references) > 0 {
		target.setExamples(references)
	}
	return true
}

func scopedPreparedExamples(examples []transportir.ValueExample, scope string) []transportir.ValueExample {
	selected := make([]transportir.ValueExample, 0, len(examples))
	for _, example := range examples {
		if exampleInScope(example.Meta, scope) {
			selected = append(selected, example)
		}
	}
	return selected
}

func exampleInScope(meta expr.MetaExpr, scope string) bool {
	owner, present := meta.Last("openapi:example:scope")
	return !present || owner == scope
}

func buildPreparedExample(example transportir.ValueExample, value any) *Example {
	summary := example.Summary
	if authored, ok := example.Meta.Last("openapi:example:summary"); ok {
		summary = authored
	}
	result := &Example{
		Summary:       summary,
		Description:   example.Description,
		ComponentName: metaValue(example.Meta, "openapi:component:example"),
		Value:         value,
	}
	if _, ok := example.Meta["openapi:example:dataValue"]; ok {
		result.DataValue = value
		result.Value = nil
	}
	if serialized, ok := example.Meta.Last("openapi:example:serializedValue"); ok {
		result.SerializedValue = serialized
	}
	return result
}
