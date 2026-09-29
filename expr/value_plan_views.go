package expr

import (
	"fmt"
	"mime"

	"github.com/CaliLuke/loom/eval"
)

type (
	// valueViewPlan reuses projected declarations, never field attributes. Its
	// source keys retain declaration identity independently of generated names.
	valueViewPlan struct {
		types map[valueViewKey]DataType
	}

	valueViewKey struct {
		source    DataType
		selection *AttributeExpr
		view      string
	}
)

func newValueViewPlan() *valueViewPlan {
	return &valueViewPlan{
		types: make(map[valueViewKey]DataType),
	}
}

func (p *valueViewPlan) project(rt *ResultTypeExpr, name string) (*ResultTypeExpr, error) {
	_, params, _ := mime.ParseMediaType(rt.Identifier)
	if params["view"] == name {
		return rt, nil
	}
	view := rt.View(name)
	if view == nil {
		return nil, fmt.Errorf("unknown view %#v", name)
	}
	key := valueViewKey{source: rt, selection: view.AttributeExpr, view: name}
	if prior, ok := p.types[key]; ok {
		return prior.(*ResultTypeExpr), nil
	}
	if _, ok := rt.Type.(*Array); ok {
		return p.collection(rt, view, key)
	}
	object := &Object{}
	projected := &ResultTypeExpr{
		Identifier: rt.projectIdentifier(name),
		UserTypeExpr: &UserTypeExpr{
			TypeName: projectedResultTypeName(rt, name),
			UID:      rt.projectIdentifier(name),
			AttributeExpr: &AttributeExpr{
				Type:        object,
				Description: projectedResultDescription(rt, name),
				Validation:  projectedResultValidation(rt, view),
				valueOrigin: valueAttributeOrigin(rt.AttributeExpr),
			},
		},
	}
	p.types[key] = projected
	source := AsObject(rt.Type)
	for _, field := range *AsObject(view.Type) {
		attribute := source.Attribute(field.Name)
		if attribute == nil {
			attribute = field.Attribute
		}
		copy, err := p.attribute(attribute, field.Attribute, name)
		if err != nil {
			return nil, fmt.Errorf("view %#v on field %#v cannot be computed: %w", name, field.Name, err)
		}
		*object = append(*object, &NamedAttributeExpr{Name: field.Name, Attribute: copy})
	}
	viewAttribute := copyValueAttribute(view.AttributeExpr)
	viewAttribute.Type = object
	projected.UserExamples = viewAttribute.UserExamples
	projected.Views = []*ViewExpr{{Name: DefaultView, AttributeExpr: viewAttribute, Parent: projected}}
	return projected, nil
}

func (p *valueViewPlan) collection(rt *ResultTypeExpr, view *ViewExpr, key valueViewKey) (*ResultTypeExpr, error) {
	array := rt.Type.(*Array)
	element := array.ElemType.Type.(*ResultTypeExpr)
	projected := &ResultTypeExpr{Identifier: rt.projectIdentifier(view.Name), UserTypeExpr: &UserTypeExpr{}}
	p.types[key] = projected
	child, err := p.project(element, view.Name)
	if err != nil {
		return nil, fmt.Errorf("collection element: %w", err)
	}
	attribute := copyValueAttribute(rt.AttributeExpr)
	// This projection creates a new declaration name. The transport body
	// rename provenance belongs to the unprojected declaration only.
	delete(attribute.Meta, "name:original")
	attribute.Description = rt.TypeName + " is the result type for an array of " + element.TypeName + " (" + view.Name + " view)"
	elementAttribute := copyValueAttribute(array.ElemType)
	elementAttribute.Type = child
	attribute.Type = &Array{ElemType: elementAttribute, NonNullableElems: array.NonNullableElems}
	projected.AttributeExpr = attribute
	projected.TypeName = child.TypeName + "Collection"
	projected.UID = projected.Identifier
	viewAttribute := copyValueAttribute(child.View(DefaultView).AttributeExpr)
	viewAttribute.Type = child.View(DefaultView).Type
	projected.Views = []*ViewExpr{{Name: DefaultView, AttributeExpr: viewAttribute, Parent: child}}
	if !eval.Execute(projected.DSL(), projected) {
		return nil, eval.Context.Errors
	}
	return projected, nil
}

func (p *valueViewPlan) attribute(source, selection *AttributeExpr, view string) (*AttributeExpr, error) {
	copy := copyValueAttribute(source)
	if _, ok := source.Type.(*ResultTypeExpr); ok {
		if selected, ok := selection.Meta.Last(ViewMetaKey); ok {
			view = selected
		} else if selected, ok := source.Meta.Last(ViewMetaKey); ok {
			view = selected
		} else {
			view = DefaultView
		}
	}
	typ, err := p.projectType(source.Type, selection, view)
	if err != nil {
		return nil, err
	}
	copy.Type = typ
	return copy, nil
}

func (p *valueViewPlan) projectType(source DataType, selection *AttributeExpr, view string) (DataType, error) {
	if result, ok := source.(*ResultTypeExpr); ok {
		return p.project(result, view)
	}
	if _, ok := source.(Primitive); ok {
		return source, nil
	}
	key := valueViewKey{source: source, selection: selection, view: view}
	if IsPrimitive(source) {
		key.selection = nil
	}
	if prior, ok := p.types[key]; ok {
		return prior, nil
	}
	switch actual := source.(type) {
	case UserType:
		attribute := copyValueAttribute(actual.Attribute())
		copy := actual.Dup(attribute)
		p.types[key] = copy
		typ, err := p.projectType(actual.Attribute().Type, selection, view)
		if err != nil {
			return nil, err
		}
		attribute.Type = typ
		return copy, nil
	case *Object:
		return p.object(actual, selection, view, key)
	case *Array:
		copy := &Array{NonNullableElems: actual.NonNullableElems}
		p.types[key] = copy
		child, err := p.attribute(actual.ElemType, selection, view)
		if err != nil {
			return nil, err
		}
		copy.ElemType = child
		return copy, nil
	case *Map, *Union:
		return p.plainType(source)
	default:
		return nil, fmt.Errorf("cannot project declaration %T", source)
	}
}

func (p *valueViewPlan) plainType(source DataType) (DataType, error) {
	key := valueViewKey{source: source}
	if prior, ok := p.types[key]; ok {
		return prior, nil
	}
	b := &valueOccurrenceBuilder{graph: &valueOccurrenceGraph{}, types: make(map[DataType]*valueDeclarationNode)}
	node, err := b.occurrence(&AttributeExpr{Type: source})
	if err != nil {
		return nil, err
	}
	for _, child := range b.graph.nodes {
		// The semantic capture follows authored source ancestry. Structural
		// views retain the copy's possibly transport-filtered legacy examples,
		// but own their mutable builtin values just like other view fields.
		local := copyValueAttribute(child.origin)
		child.attribute.DefaultValue = local.DefaultValue
		child.attribute.UserExamples = local.UserExamples
		if child.attribute.Validation != nil {
			child.attribute.Validation.Values = local.Validation.Values
		}
	}
	p.types[key] = node.attribute.Type
	return node.attribute.Type, nil
}

func (p *valueViewPlan) object(actual *Object, selection *AttributeExpr, view string, key valueViewKey) (DataType, error) {
	selected := AsObject(selection.Type)
	if selected == nil {
		return p.plainType(actual)
	}
	copy := &Object{}
	p.types[key] = copy
	for _, field := range *actual {
		choice := selected.Attribute(field.Name)
		if choice == nil {
			attribute := copyValueAttribute(field.Attribute)
			typ, err := p.plainType(field.Attribute.Type)
			if err != nil {
				return nil, err
			}
			attribute.Type = typ
			*copy = append(*copy, &NamedAttributeExpr{Name: field.Name, Attribute: attribute})
			continue
		}
		attribute, err := p.attribute(field.Attribute, choice, view)
		if err != nil {
			return nil, err
		}
		*copy = append(*copy, &NamedAttributeExpr{Name: field.Name, Attribute: attribute})
	}
	return copy, nil
}
