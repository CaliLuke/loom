package codegen

import "github.com/CaliLuke/loom/expr"

// formUnionElement retains the form field's physical body declaration while
// sharing the parameter matcher and its resolved codec grammar.
func formUnionElement(body *TypeData, field *FormFieldData) *Element {
	key, physical := body.attribute.FindAttribute(field.AttributeName)
	if physical == nil {
		panic("missing form union carrier " + field.AttributeName)
	}
	context := body.context
	return &Element{
		attribute: physical,
		context:   context,
		location:  "form",
		HTTPName:  field.HTTPName,
		Codec:     field.Codec,
		Namespace: field.Namespace,
		Styled:    true,
		AttributeData: &AttributeData{
			Name:         field.AttributeName,
			Type:         physical.Type,
			FieldType:    physical.Type,
			FieldName:    field.FieldName,
			FieldPointer: expr.ServiceFieldLayout(body.attribute, key, physical, context.Pointer, context.UseDefault).Pointer,
			Required:     field.Required,
			Pointer:      true,
			TypeName:     context.Scope.Name(physical, context.Pkg(physical), context.Pointer, context.UseDefault),
			TypeRef:      context.Scope.Ref(physical, context.Pkg(physical)),
		},
	}
}
