package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/examplevalue"
)

// selectedExampleUnion changes only example generation in a private design copy.
// The ordinary expression API continues to return unwrapped branch examples.
type selectedExampleUnion struct {
	*expr.Union
}

func synthesizedOpenAPIExample(attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) any {
	if len(attribute.ExtractUserExamples()) > 0 {
		return attribute.Example(generator)
	}
	if generator == nil || generator.Randomizer == nil {
		return nil
	}
	copy := expr.DupAtt(attribute)
	if !preserveExampleSelections(copy, make(map[expr.DataType]expr.DataType)) {
		return attribute.Example(generator)
	}
	// A raw example memo must not supply values to the selection-preserving
	// design, nor may its intermediate values escape into a caller's memo.
	return copy.Example(&expr.ExampleGenerator{Randomizer: generator.Randomizer})
}

func preserveExampleSelections(attribute *expr.AttributeExpr, seen map[expr.DataType]expr.DataType) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if previous, ok := seen[attribute.Type]; ok {
		attribute.Type = previous
		_, selected := previous.(*selectedExampleUnion)
		return selected
	}
	original := attribute.Type
	seen[original] = original
	found := false
	switch actual := original.(type) {
	case expr.UserType:
		found = preserveExampleSelections(actual.Attribute(), seen)
	case *expr.Object:
		for _, field := range *actual {
			found = preserveExampleSelections(field.Attribute, seen) || found
		}
	case *expr.Array:
		found = preserveExampleSelections(actual.ElemType, seen)
	case *expr.Map:
		found = preserveExampleSelections(actual.KeyType, seen)
		found = preserveExampleSelections(actual.ElemType, seen) || found
	case *expr.Union:
		found = true
		wrapped := &selectedExampleUnion{Union: actual}
		seen[original] = wrapped
		attribute.Type = wrapped
		for _, branch := range actual.Values {
			preserveExampleSelections(branch.Attribute, seen)
		}
	}
	return found
}

func (union *selectedExampleUnion) Example(generator *expr.ExampleGenerator) any {
	if len(union.Values) == 0 {
		return nil
	}
	index := generator.Int() % len(union.Values)
	value := union.Values[index].Attribute.Example(generator)
	if value == nil {
		return nil
	}
	return examplevalue.Union{Branch: index, Value: value}
}
