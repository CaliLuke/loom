package expr

// copyValueOccurrenceAttribute copies local contract state without traversing
// declarations. The graph builder supplies owned type and source edges.
func copyValueOccurrenceAttribute(source *AttributeExpr) *AttributeExpr {
	copy := *source
	copy.valueOrigin = valueAttributeOrigin(source)
	copy.valueSourceOrigin = valueCopiedSourceOrigin(source)
	copy.Type, copy.Bases, copy.References = nil, nil, nil
	copy.DefaultValue, copy.UserExamples = nil, nil
	copy.Meta = copyValueMeta(source.Meta)
	if source.Docs != nil {
		docs := *source.Docs
		copy.Docs = &docs
	}
	if source.Validation != nil {
		validation := *source.Validation
		validation.Values = nil
		validation.EnumClauses = nil
		if source.Validation.Values != nil {
			validation.Values = make([]any, 0, len(source.Validation.Values))
		}
		if source.Validation.EnumClauses != nil {
			validation.EnumClauses = make([][]any, len(source.Validation.EnumClauses))
			for index, clause := range source.Validation.EnumClauses {
				validation.EnumClauses[index] = make([]any, 0, len(clause))
			}
		}
		validation.PatternClauses = append([]string(nil), source.Validation.PatternClauses...)
		validation.FormatClauses = append([]ValidationFormat(nil), source.Validation.FormatClauses...)
		validation.Required = append([]string(nil), source.Validation.Required...)
		validation.Minimum = copyValuePointer(source.Validation.Minimum)
		validation.Maximum = copyValuePointer(source.Validation.Maximum)
		validation.ExclusiveMinimum = copyValuePointer(source.Validation.ExclusiveMinimum)
		validation.ExclusiveMaximum = copyValuePointer(source.Validation.ExclusiveMaximum)
		validation.MinLength = copyValuePointer(source.Validation.MinLength)
		validation.MaxLength = copyValuePointer(source.Validation.MaxLength)
		copy.Validation = &validation
	}
	return &copy
}

func copyValueMeta(source MetaExpr) MetaExpr {
	if source == nil {
		return nil
	}
	copy := make(MetaExpr, len(source))
	for key, values := range source {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func copyValuePointer[T any](source *T) *T {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
}

// valueAttributeOrigin is structural ancestry only. A supplied semantic root
// determines which occurrence that ancestry may identify; copies never carry a
// resolved value or a context ID.
func valueAttributeOrigin(attribute *AttributeExpr) *AttributeExpr {
	if attribute == nil {
		return nil
	}
	if attribute.valueOrigin != nil {
		return attribute.valueOrigin
	}
	return attribute
}

// bindValueSource changes only the semantic transport mapping. The target's
// authored declaration remains the owner of naming and schema annotations.
func bindValueSource(target, source *AttributeExpr) {
	origin := valueCopiedSourceOrigin(source)
	if origin == target {
		origin = nil
	}
	target.valueSourceOrigin = origin
}

// valueCopiedSourceOrigin preserves normalized binding even when a constructor
// retains only the canonical declaration origin. Invalid cycles retain an edge
// to the invalid source so that copying cannot turn them into valid ancestry.
func valueCopiedSourceOrigin(source *AttributeExpr) *AttributeExpr {
	origin := valueSemanticOrigin(source)
	if origin == nil && source != nil {
		return source
	}
	return origin
}

// valueSemanticOrigin follows explicit bindings and controlled copies. A
// constructed wrapper may retain only the canonical copy origin, which itself
// carries a binding; cycle protection makes malformed internal graphs fail the
// plan's ancestry guard instead of equating two missing origins.
func valueSemanticOrigin(attribute *AttributeExpr) *AttributeExpr {
	seen := make(map[*AttributeExpr]bool)
	for attribute != nil && !seen[attribute] {
		seen[attribute] = true
		if attribute.valueSourceOrigin != nil {
			attribute = attribute.valueSourceOrigin
			continue
		}
		if origin := valueAttributeOrigin(attribute); origin != attribute {
			attribute = origin
			continue
		}
		return attribute
	}
	return nil
}

func copyValueAttribute(source *AttributeExpr) *AttributeExpr {
	copy := copyValueOccurrenceAttribute(source)
	copy.Type = source.Type
	copy.Bases = append([]DataType(nil), source.Bases...)
	copy.References = append([]DataType(nil), source.References...)
	if source.DefaultValue != nil {
		copy.DefaultValue = copyValueRaw(source.DefaultValue)
	}
	for _, example := range source.UserExamples {
		copy.UserExamples = append(copy.UserExamples, copyValueExample(example))
	}
	if source.Validation != nil {
		for _, value := range source.Validation.Values {
			copy.Validation.Values = append(copy.Validation.Values, copyValueRaw(value))
		}
		for clauseIndex, clause := range source.Validation.EnumClauses {
			for _, value := range clause {
				copy.Validation.EnumClauses[clauseIndex] = append(
					copy.Validation.EnumClauses[clauseIndex], copyValueRaw(value),
				)
			}
		}
	}
	return copy
}

func copyValueRaw(raw any) any {
	snapshot := snapshotValueSource(raw)
	// Copies retain the owned builtin graph, including cycles for later
	// diagnostics. Opaque/custom host values remain borrowed without callbacks.
	return snapshot.raw
}

func copyValueExample(source *ExampleExpr) *ExampleExpr {
	if source == nil {
		return nil
	}
	copy := *source
	copy.Meta = copyValueMeta(source.Meta)
	copy.Value = copyValueRaw(source.Value)
	copy.valueOrigin = valueExampleOrigin(source)
	return &copy
}

func valueExampleOrigin(example *ExampleExpr) *ExampleExpr {
	if example.valueOrigin != nil {
		return example.valueOrigin
	}
	return example
}
