package expr

import "slices"

type (
	attributeTemplate struct {
		source       *AttributeExpr
		steps        []templateStep
		local        *AttributeExpr
		session      *templateCapture
		typeSource   *AttributeExpr
		selectedRoot bool
		titleSet     bool
		patternSet   bool
		untaggedSet  bool
	}

	templateCapture struct {
		active bool
	}

	templateStep struct {
		name string
		role templateRole
	}

	templateRole uint8
)

const (
	templateMember templateRole = iota
	templateElement
	templateKey
)

// NewBodyTemplate creates an untyped selection template bound to the complete
// source occurrence. Its ordinary DSL callback declares the selected shape and
// must run before capture ends. A callback without a shape remains untyped.
func NewBodyTemplate(source *AttributeExpr) *AttributeExpr {
	return &AttributeExpr{
		template: &attributeTemplate{
			source: source, local: &AttributeExpr{},
			session: &templateCapture{active: true}, selectedRoot: true,
		},
	}
}

// EndTemplateCapture ends authoring for this body and its nested occurrences.
// Framework preparation may then write effective properties without recording
// those inherited properties as explicit author choices.
func (a *AttributeExpr) EndTemplateCapture() {
	if a.template != nil {
		a.template.session.active = false
	}
}

// TemplateAttribute returns a selected field during active template authoring.
// The boolean distinguishes a template occurrence from an ordinary attribute.
func (a *AttributeExpr) TemplateAttribute(name string) (*AttributeExpr, bool) {
	if a.templateLocal() == nil {
		return nil, false
	}
	var existing *AttributeExpr
	if object := AsObject(a.Type); object != nil {
		_, existing = objectAttribute(object, name)
	}
	if existing != nil && existing.template != nil {
		return DupAtt(existing), true
	}
	source, steps := a.templateTypeSource()
	steps = append(steps, templateStep{name: name, role: templateMember})
	binding := &attributeTemplate{source: source, steps: steps, local: &AttributeExpr{}, session: a.template.session}
	var selected *AttributeExpr
	if existing != nil {
		selected = DupAtt(existing)
	} else if inherited := binding.resolve(); inherited != nil {
		selected = DupAtt(inherited)
	} else {
		selected = &AttributeExpr{}
		for _, reference := range a.References {
			owner := &AttributeExpr{Type: reference}
			if named, ok := reference.(UserType); ok {
				owner = named.Attribute()
			}
			if _, field := owner.FindAttribute(name); field != nil {
				selected = DupAtt(field)
				binding.source = owner
				binding.steps = []templateStep{{name: name, role: templateMember}}
				break
			}
		}
		if selected.Type == nil {
			for _, base := range a.Bases {
				if _, field := (&AttributeExpr{Type: base}).FindAttribute(name); field != nil {
					selected.Type = field.Type
					break
				}
			}
		}
	}
	selected.template = binding
	return selected, true
}

// CaptureTemplateChild binds an existing array element or map key/value to its
// parent's active template before an Elem or Key callback refines it.
func (a *AttributeExpr) CaptureTemplateChild(child *AttributeExpr) {
	if a.templateLocal() == nil || child.template != nil {
		return
	}
	role := templateElement
	switch typ := a.Type.(type) {
	case *Array:
		if child != typ.ElemType {
			panic("template child is not the array element")
		}
	case *Map:
		if child == typ.KeyType {
			role = templateKey
		} else if child != typ.ElemType {
			panic("template child is not a map key or element")
		}
	default:
		panic("template child requires an array or map")
	}
	source, steps := a.templateTypeSource()
	child.template = &attributeTemplate{
		source: source, steps: append(steps, templateStep{role: role}),
		local: &AttributeExpr{}, session: a.template.session,
	}
}

// SetAuthoredType records an explicit type argument independently of the
// provisional type inferred from a template.
func (a *AttributeExpr) SetAuthoredType(typ DataType) {
	a.Type = typ
	if a.templateLocal() != nil {
		a.template.typeSource = &AttributeExpr{Type: typ}
	}
}

func (a *AttributeExpr) templateLocal() *AttributeExpr {
	if a.template == nil || !a.template.session.active {
		return nil
	}
	return a.template.local
}

func (a *AttributeExpr) templateTypeSource() (*AttributeExpr, []templateStep) {
	if a.template.typeSource != nil {
		return a.template.typeSource, nil
	}
	return a.template.source, slices.Clone(a.template.steps)
}

func (t *attributeTemplate) resolve() *AttributeExpr {
	attribute := t.source
	for _, step := range t.steps {
		if attribute == nil {
			return nil
		}
		switch step.role {
		case templateMember:
			_, attribute = attribute.FindAttribute(step.name)
		case templateElement:
			if array := AsArray(attribute.Type); array != nil {
				attribute = array.ElemType
			} else if mapping := AsMap(attribute.Type); mapping != nil {
				attribute = mapping.ElemType
			} else {
				return nil
			}
		case templateKey:
			mapping := AsMap(attribute.Type)
			if mapping == nil {
				return nil
			}
			attribute = mapping.KeyType
		}
	}
	return attribute
}

func (d *dupper) copyTemplate(source *attributeTemplate) *attributeTemplate {
	if source == nil {
		return nil
	}
	copy := *source
	copy.steps = slices.Clone(source.steps)
	copy.local = copyValueAttribute(source.local)
	if source.local.AdditionalAttributes != nil {
		copy.local.AdditionalAttributes = d.DupAttribute(source.local.AdditionalAttributes)
	}
	return &copy
}
