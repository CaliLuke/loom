package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// serverResponseValidation retains a declaration's physical context until every
// prepared body has contributed its server validation requirements. Type
// allocation and validation eligibility have different lifetimes: an ordinary
// response can allocate a type before another response uses it in a union.
type serverResponseValidation struct {
	data      *TypeData
	attribute *expr.AttributeExpr
	context   *codegen.AttributeContext
	root      *responseBodyTypeData
}

func (sd *ServiceData) finalizeServerResponseValidation() {
	for _, pending := range sd.responseValidations {
		typ := pending.attribute.Type.(expr.UserType)
		if pending.root != nil {
			applyUserResponseBodyValidation(pending.root, pending.attribute, typ, pending.context, sd, true)
			pending.data.ValidateDef = pending.root.validateDef
			pending.data.ValidateRef = pending.root.validateRef
			if pending.data.Init != nil {
				pending.data.Init.ServerArgs[0].Validate = pending.root.validateDef
			}
			continue
		}
		pending.data.ValidateDef = attributeValidationDefinition(typ, false, true, sd, pending.context)
		if pending.data.ValidateDef != "" {
			pending.data.ValidateRef = fmt.Sprintf("err = Validate%s(v)", pending.data.VarName)
		}
	}
	sd.responseValidations = nil
}
