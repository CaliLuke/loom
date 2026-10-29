package ir

import "github.com/CaliLuke/loom/expr"

func (a *Analyzer) applySchemaExample(s *Schema, attr *expr.AttributeExpr, context string) {
	if a.annotationOwner != nil {
		position := componentAnnotationPosition{context, fingerprintAttribute(attr, a.closeObjects)}
		if example, found := a.annotationOwner[position]; found {
			values := example.snapshot()
			s.Example = values.representative
			s.Examples = values.values
			return
		}
		defer func() {
			a.annotationOwner[position] = (componentExamples{representative: s.Example, values: s.Examples}).snapshot()
		}()
	}
	s.Example, s.Examples = nil, nil
	if a.suppressExamples != nil && a.suppressExamples(attr, a.closeObjects) {
		return
	}
	if a.customExampleValue {
		a.applyCustomSchemaExamples(s, attr, context)
		return
	}
	a.applyPreparedSchemaExamples(s)
}

func (a *Analyzer) applyCustomSchemaExamples(s *Schema, attr *expr.AttributeExpr, context string) {
	if authored := attr.ExtractUserExamples(); len(authored) > 0 {
		examples := make([]*expr.ExampleExpr, 0, len(authored))
		for _, example := range authored {
			if exampleInScope(example.Meta, "schema") {
				examples = append(examples, example)
			}
		}
		for index, entry := range examples {
			if value, ok := a.exampleValue(attr, entry.Value); ok {
				if len(examples) > 1 {
					s.Examples = append(s.Examples, value)
				}
				if index == len(examples)-1 {
					s.Example = value
				}
			}
		}
		return
	}
	generator := exampleGeneratorForAttribute(a.rand, attr, a.closeObjects, context)
	if example, ok := a.exampleValue(attr, attr.Example(generator)); ok {
		s.Example = example
	}
}

func (a *Analyzer) applyPreparedSchemaExamples(s *Schema) {
	if a.preparedExamples == nil {
		return
	}
	set, prepared := a.preparedExamples.ExampleSets[a.examplePlan]
	if !prepared || set == nil || !set.Prepared {
		return
	}
	project := preparedOpenAPIExampleValue
	if a.declaredExampleValue {
		project = preparedDeclaredExampleValue
	}
	if len(set.Examples) > 0 {
		examples := scopedPreparedExamples(set.Examples, "schema")
		for index, entry := range examples {
			if value, ok := project(entry); ok {
				if len(examples) > 1 {
					s.Examples = append(s.Examples, value)
				}
				if index == len(examples)-1 {
					s.Example = value
				}
			}
		}
		return
	}
	if set.Representative != nil {
		if value, ok := project(*set.Representative); ok {
			s.Example = value
		}
	}
}
