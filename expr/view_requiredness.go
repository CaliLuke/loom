package expr

import "slices"

// FinalizeRequiredness computes selected fields' effective requirements from
// the parent result type and the view's authored overrides. It preserves field
// definitions and changes only the view validation.
func (v *ViewExpr) FinalizeRequiredness() {
	if v.Parent == nil {
		return
	}
	canonical := v.Parent.AttributeExpr
	if array := AsArray(canonical.Type); array != nil {
		if element, ok := array.ElemType.Type.(UserType); ok {
			canonical = element.Attribute()
		}
	}
	object := AsObject(canonical.Type)
	selected := AsObject(v.Type)
	if object == nil || selected == nil {
		return
	}
	required := make([]string, 0, len(*selected))
	for _, field := range *object {
		if selected.Attribute(field.Name) == nil {
			continue
		}
		if slices.Contains(v.RequiredOverrides, field.Name) || (canonical.IsRequired(field.Name) && !slices.Contains(v.OptionalOverrides, field.Name)) {
			required = append(required, field.Name)
		}
	}
	if canonical.Validation == nil && len(required) == 0 {
		v.Validation = nil
		return
	}
	validation := &ValidationExpr{}
	if canonical.Validation != nil {
		validation = canonical.Validation.Dup()
	}
	validation.Required = required
	v.Validation = validation
}

func (rt *ResultTypeExpr) finalizeViews() {
	for _, view := range rt.Views {
		view.FinalizeRequiredness()
	}
}
