package expr

// SetDescription sets the occurrence description and records a local template choice.
func (a *AttributeExpr) SetDescription(value string) {
	a.Description = value
	if local := a.templateLocal(); local != nil {
		local.Description = value
	}
}

// SetDocs sets the occurrence documentation and retains a detached local choice.
func (a *AttributeExpr) SetDocs(value *DocsExpr) {
	a.Docs = value
	if local := a.templateLocal(); local != nil && value != nil {
		cloned := *value
		local.Docs = &cloned
	}
}

// AddExample appends an authored occurrence example in declaration order.
func (a *AttributeExpr) AddExample(value *ExampleExpr) {
	a.UserExamples = append(a.UserExamples, value)
	if local := a.templateLocal(); local != nil {
		local.UserExamples = append(local.UserExamples, copyValueExample(value))
	}
}

// SetNullable adds explicit null admission to this occurrence.
func (a *AttributeExpr) SetNullable() {
	a.Nullable = true
	if local := a.templateLocal(); local != nil {
		local.Nullable = true
	}
}

// SetAdditionalAttributes records the authored contract for undeclared members.
// The caller validates that this object has no earlier additional declaration.
func (a *AttributeExpr) SetAdditionalAttributes(value *AttributeExpr) {
	a.AdditionalAttributes = value
	if local := a.templateLocal(); local != nil {
		local.AdditionalAttributes = value
	}
}

// SetUntagged selects direct JSON branch encoding for the current union.
// The caller must establish that the occurrence is a union.
func (a *AttributeExpr) SetUntagged() {
	AsUnion(a.Type).Untagged = true
	if a.templateLocal() != nil {
		a.template.untaggedSet = true
	}
}

// SetMinimum records a normalized Minimum constraint on this occurrence.
func (a *AttributeExpr) SetMinimum(value NumericBound) {
	a.mutableValidation().Minimum = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().Minimum = copyValuePointer(&value)
	}
}

// SetMaximum records a normalized Maximum constraint on this occurrence.
func (a *AttributeExpr) SetMaximum(value NumericBound) {
	a.mutableValidation().Maximum = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().Maximum = copyValuePointer(&value)
	}
}

// SetExclusiveMinimum records a normalized ExclusiveMinimum constraint on this occurrence.
func (a *AttributeExpr) SetExclusiveMinimum(value NumericBound) {
	a.mutableValidation().ExclusiveMinimum = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().ExclusiveMinimum = copyValuePointer(&value)
	}
}

// SetExclusiveMaximum records a normalized ExclusiveMaximum constraint on this occurrence.
func (a *AttributeExpr) SetExclusiveMaximum(value NumericBound) {
	a.mutableValidation().ExclusiveMaximum = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().ExclusiveMaximum = copyValuePointer(&value)
	}
}

// SetMinLength records a normalized MinLength constraint on this occurrence.
func (a *AttributeExpr) SetMinLength(value int) {
	a.mutableValidation().MinLength = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().MinLength = copyValuePointer(&value)
	}
}

// SetMaxLength records a normalized MaxLength constraint on this occurrence.
func (a *AttributeExpr) SetMaxLength(value int) {
	a.mutableValidation().MaxLength = &value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().MaxLength = copyValuePointer(&value)
	}
}

// SetFormat sets the validated format constraint for this occurrence.
func (a *AttributeExpr) SetFormat(value ValidationFormat) {
	a.mutableValidation().Format = value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().Format = value
	}
}

// SetPattern records an authored regular expression, including an empty one.
func (a *AttributeExpr) SetPattern(value string) {
	a.mutableValidation().Pattern = value
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().Pattern = value
		a.template.patternSet = true
	}
}

// SetEnum replaces the enum with normalized authored values.
func (a *AttributeExpr) SetEnum(values []any) {
	a.mutableValidation().Values = values
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().Values = make([]any, len(values))
		for i, value := range values {
			local.Validation.Values[i] = copyValueRaw(value)
		}
	}
}

// AddRequired adds required member names without removing inherited guarantees.
func (a *AttributeExpr) AddRequired(names ...string) {
	a.mutableValidation().AddRequired(names...)
	if local := a.templateLocal(); local != nil {
		local.mutableValidation().AddRequired(names...)
	}
}

// AddReference appends a selected-field template reference in authored order.
func (a *AttributeExpr) AddReference(typ DataType) {
	a.References = append(a.References, typ)
	if local := a.templateLocal(); local != nil {
		local.References = append(local.References, typ)
	}
}

// AddBase appends an authored Extend contributor without changing its contract.
func (a *AttributeExpr) AddBase(typ DataType) {
	a.Bases = append(a.Bases, typ)
	if local := a.templateLocal(); local != nil {
		local.Bases = append(local.Bases, typ)
	}
}

func (a *AttributeExpr) mutableValidation() *ValidationExpr {
	if a.Validation == nil {
		a.Validation = &ValidationExpr{}
	}
	return a.Validation
}
