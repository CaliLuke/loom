package codegen

import (
	"strconv"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// statusErrorNameArg supplies the validated constant through the same field
// reconstruction plan as the other transport inputs.
func (b *errorBuilder) statusErrorNameArg(response *transportir.ResponseStatus, context *codegen.AttributeContext, vars *transportVarScope) *InitArgData {
	name := response.StatusErrorName
	source := response.Error.Attribute
	_, field := source.FindAttribute(name)
	fieldName, fieldType, pointer := transportFieldBinding(name, field, source, context)
	varName, _ := vars.allocate(codegen.Goify(expr.AttributeName(name), false), nil)
	return &InitArgData{
		Ref: strconv.Quote(response.Error.Name),
		AttributeData: &AttributeData{
			Name:         name,
			VarName:      varName,
			FieldName:    fieldName,
			FieldType:    fieldType,
			FieldPointer: pointer,
			Required:     true,
			Type:         expr.String,
			TypeRef:      b.sd.Scope.GoTypeRef(&expr.AttributeExpr{Type: expr.String}),
		},
	}
}
