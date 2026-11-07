package expr

import "slices"

// publishTemplate refreshes one authored template only after its semantic
// dependencies have been composed. The local layer never replaces the source.
func (c *designComposer) publishTemplate(attribute *AttributeExpr, path string) bool {
	binding := attribute.template
	if binding == nil {
		return true
	}
	if binding.session.active {
		compositionError(attribute, "%s: template authoring has not completed", path)
		return false
	}
	c.walk(binding.source, path+"[template]")
	if binding.typeSource != nil {
		c.walk(binding.typeSource, path+"[authored-type]")
	}
	source := binding.resolve()
	if source == nil {
		attribute.template = nil
		return true
	}
	constraints, err := EffectiveConstraintsFor(source)
	if err != nil {
		compositionError(attribute, "%s: invalid template source: %v", path, err)
		return false
	}
	if binding.selectedRoot {
		parent := *source
		parent.Validation = constraints.Validation().Lowered()
		attribute.template = nil
		attribute.References = append([]DataType{source.Type}, attribute.References...)
		attribute.Inherit(&parent)
		return true
	}

	currentType := attribute.Type
	declaration := attribute.DSLSource()
	copy := newTemplateDupper().DupAttribute(source)
	copy.template = nil
	// Reference copies effective occurrence properties. Its local choices do
	// not become Extend refinements of the source's old occurrence layers.
	copy.constraintParents, copy.constraintLocal = nil, nil
	copy.Validation = constraints.Validation().Lowered()
	if value, present := constraints.Default(); present {
		copy.DefaultValue = copyValueRaw(value)
	}
	if binding.typeSource != nil {
		copy.Type = newTemplateDupper().DupType(binding.typeSource.Type)
	}
	copy.SetDSLSource(declaration)
	applyTemplateProperties(copy, binding)
	c.publishTemplateChildren(copy.Type, currentType, path)
	if binding.untaggedSet {
		if union := AsUnion(copy.Type); union != nil {
			union.Untagged = true
		}
	}
	*attribute = *copy
	return true
}

func newTemplateDupper() *dupper {
	copy := newDupper()
	copy.templateSnapshot = true
	return copy
}

// publishTemplateChildren overlays only occurrences actually refined by the
// callback. Untouched members come from the completed source graph.
func (c *designComposer) publishTemplateChildren(target, authored DataType, path string) {
	if object := AsObject(target); object != nil {
		if local := AsObject(authored); local != nil {
			for _, field := range *local {
				if field.Attribute.template == nil {
					continue
				}
				c.walk(field.Attribute, path+"."+field.Name)
				object.Set(field.Name, field.Attribute)
			}
		}
		return
	}
	if array := AsArray(target); array != nil {
		if local := AsArray(authored); local != nil && local.ElemType.template != nil {
			c.walk(local.ElemType, path+"[]")
			array.ElemType = local.ElemType
		}
		return
	}
	if mapping := AsMap(target); mapping != nil {
		if local := AsMap(authored); local != nil {
			if local.KeyType.template != nil {
				c.walk(local.KeyType, path+"[key]")
				mapping.KeyType = local.KeyType
			}
			if local.ElemType.template != nil {
				c.walk(local.ElemType, path+"[value]")
				mapping.ElemType = local.ElemType
			}
		}
		return
	}
	if union := AsUnion(target); union != nil {
		if local := AsUnion(authored); local != nil {
			for _, branch := range local.Values {
				if branch.Attribute.template != nil {
					c.walk(branch.Attribute, path+"."+branch.Name)
					union.Values = append(union.Values, branch)
				}
			}
		}
	}
}

func applyTemplateProperties(target *AttributeExpr, binding *attributeTemplate) {
	local := binding.local
	target.Bases = append(target.Bases, local.Bases...)
	target.References = append(target.References, local.References...)
	if local.Description != "" {
		target.Description = local.Description
	}
	if binding.titleSet {
		target.Title = local.Title
	}
	if local.Docs != nil {
		docs := *local.Docs
		target.Docs = &docs
	}
	if local.DefaultValue != nil {
		target.DefaultValue = copyValueRaw(local.DefaultValue)
	}
	target.Nullable = target.Nullable || local.Nullable
	for key := range local.removedMeta {
		target.DeleteMeta(key)
	}
	for key, values := range local.Meta {
		if target.Meta == nil {
			target.Meta = make(MetaExpr)
		}
		target.Meta[key] = append(target.Meta[key], values...)
	}
	for _, example := range local.UserExamples {
		target.UserExamples = append(target.UserExamples, copyValueExample(example))
	}
	if local.AdditionalAttributes != nil {
		target.AdditionalAttributes = DupAtt(local.AdditionalAttributes)
	}
	applyTemplateValidation(target, local.Validation, binding.patternSet)
}

func applyTemplateValidation(target *AttributeExpr, local *ValidationExpr, patternSet bool) {
	if local == nil {
		return
	}
	validation := target.mutableValidation()
	if local.Values != nil {
		validation.Values = slices.Clone(local.Values)
		validation.EnumClauses = nil
	}
	if local.Format != "" {
		validation.Format = local.Format
		validation.FormatClauses = nil
	}
	if patternSet {
		validation.Pattern = local.Pattern
		validation.PatternClauses = nil
	}
	if local.Minimum != nil {
		validation.Minimum = copyValuePointer(local.Minimum)
	}
	if local.Maximum != nil {
		validation.Maximum = copyValuePointer(local.Maximum)
	}
	if local.ExclusiveMinimum != nil {
		validation.ExclusiveMinimum = copyValuePointer(local.ExclusiveMinimum)
	}
	if local.ExclusiveMaximum != nil {
		validation.ExclusiveMaximum = copyValuePointer(local.ExclusiveMaximum)
	}
	if local.MinLength != nil {
		validation.MinLength = copyValuePointer(local.MinLength)
	}
	if local.MaxLength != nil {
		validation.MaxLength = copyValuePointer(local.MaxLength)
	}
	validation.AddRequired(local.Required...)
}
