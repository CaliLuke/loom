package expr

// CanonicalizeExample normalizes example values so Loom unions use their
// canonical discriminator/value JSON shape.
func CanonicalizeExample(att *AttributeExpr, example any) any {
	if att == nil || att.Type == nil || att.Type == Empty {
		return example
	}

	switch dt := att.Type.(type) {
	case UserType:
		return CanonicalizeExample(dt.Attribute(), example)
	case *Object:
		m, ok := example.(map[string]any)
		if !ok {
			return example
		}
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = CanonicalizeExample(att.Find(k), v)
		}
		return out
	case *Array:
		s, ok := example.([]any)
		if !ok {
			return example
		}
		out := make([]any, len(s))
		for i, v := range s {
			out[i] = CanonicalizeExample(dt.ElemType, v)
		}
		return out
	case *Map:
		m, ok := example.(map[string]any)
		if !ok {
			return example
		}
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = CanonicalizeExample(dt.ElemType, v)
		}
		return out
	case *Union:
		if example == nil || len(dt.Values) == 0 {
			return example
		}
		chosen := pickUnionVariantForExample(dt, example)
		if chosen == nil {
			return example
		}
		return map[string]any{
			dt.GetTypeKey():  UnionVariantTag(chosen),
			dt.GetValueKey(): CanonicalizeExample(chosen.Attribute, example),
		}
	default:
		return example
	}
}

func pickUnionVariantForExample(u *Union, example any) *NamedAttributeExpr {
	if m, ok := example.(map[string]any); ok {
		matches := make([]*NamedAttributeExpr, 0, len(u.Values))
		for _, nat := range u.Values {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			if unionVariantMatchesObjectKeys(nat.Attribute, m) {
				matches = append(matches, nat)
			}
		}
		if len(matches) == 1 {
			return matches[0]
		}
		return nil
	}

	for _, nat := range u.Values {
		if nat == nil || nat.Attribute == nil || nat.Attribute.Type == nil {
			continue
		}
		attr := unwrapUserTypeAttr(nat.Attribute)
		if attr == nil || attr.Type == nil {
			continue
		}
		if attr.Type.IsCompatible(example) {
			return nat
		}
	}

	return nil
}

func unionVariantMatchesObjectKeys(att *AttributeExpr, example map[string]any) bool {
	attr := unwrapUserTypeAttr(att)
	if attr == nil {
		return false
	}
	obj, ok := attr.Type.(*Object)
	if !ok {
		return false
	}
	fields := make(map[string]struct{}, len(*obj))
	for _, nat := range *obj {
		if nat == nil {
			continue
		}
		fields[nat.Name] = struct{}{}
	}
	if len(fields) == 0 {
		return false
	}
	for key := range example {
		if _, ok := fields[key]; !ok {
			return false
		}
	}
	if attr.Validation != nil {
		for _, name := range attr.Validation.Required {
			if _, ok := example[name]; !ok {
				return false
			}
		}
	}
	return true
}

func unwrapUserTypeAttr(att *AttributeExpr) *AttributeExpr {
	if att == nil || att.Type == nil {
		return att
	}
	if ut, ok := att.Type.(UserType); ok {
		return unwrapUserTypeAttr(ut.Attribute())
	}
	return att
}
