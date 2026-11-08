package uniongen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/formgen"
)

// ConfigureFormField records captured native form fields with their actual Go
// selectors and presence layout. FieldType is already allocated by NameScope.
// Unsupported form contracts remain usable in JSON-only declarations.
func ConfigureFormField(field *Field, attribute *expr.AttributeExpr, context *codegen.AttributeContext) {
	field.FormFields = nil
	field.FormValue = formgen.Field{}
	field.FormError = ""
	field.FormBranchAllowsEmpty = false
	field.FlatFormObject = expr.IsObject(attribute.Type)
	field.EmptyValueExpr = field.FieldType + "{}"
	if strings.HasPrefix(field.FieldType, "*") {
		field.EmptyValueExpr = "&" + strings.TrimPrefix(field.FieldType, "*") + "{}"
	}
	captured, err := expr.HTTPFormTaggedBranchEncoding(attribute, "value")
	if err != nil {
		field.FormError = err.Error()
		return
	}
	field.FormBranchAllowsEmpty = captured.AllowsEmpty
	if captured.Value != nil {
		field.FormValue = formgen.Field{Codec: captured.Value.Codec, AllowsNull: expr.AllowsNull(attribute)}
		return
	}
	for _, member := range *expr.AsObject(attribute.Type) {
		codec := captured.Object.Fields[member.Name]
		if codec.Omitted {
			continue
		}
		key := expr.AttributeName(member.Name)
		selector := codegen.GoifyAtt(member.Attribute, key, true)
		field.FormFields = append(field.FormFields, &FormField{
			Field: formgen.Field{Codec: codec.Codec, AllowsNull: expr.AllowsNull(member.Attribute)},
			Name:  codec.Name, Selector: selector, Required: attribute.IsRequired(key),
			Presence: formgen.Presence(context, attribute, key, member.Attribute, "value."+selector),
		})
	}
}
