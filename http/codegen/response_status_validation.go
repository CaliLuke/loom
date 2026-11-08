package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// responseConstructedStatusValidation follows the constructor's actual returned
// representation before validating its native status field. Named nullable
// errors return an owner wrapper; its checked Payload extraction owns identity.
func responseConstructedStatusValidation(response *ResponseData, value string) string {
	if !response.HasStatusBinding() {
		return ""
	}
	init := response.ResultInit
	if init == nil || init.returnContext == nil {
		panic("native status reconstruction has no captured constructor context")
	}
	var preparation string
	if init.errorConstructor != "" {
		preparation = fmt.Sprintf("statusPayload, err := %s.Payload()\nif err != nil {\nreturn err\n}\n", value)
		value = "statusPayload"
	}
	return preparation + responseNativeStatusValidation(response, init.returnContext, value)
}

// responseNativeStatusValidation checks the reconstructed native field through
// its authored contract. It does not apply an emission domain to observations.
func responseNativeStatusValidation(response *ResponseData, context *codegen.AttributeContext, value string) string {
	if response.statusBinding == nil {
		return ""
	}
	key, attribute := response.Carrier.FindAttribute(response.statusBinding.Name)
	if attribute == nil {
		panic("HTTP status binding is absent from its native carrier")
	}
	fieldContext := context.Dup()
	fieldContext.Pointer = context.IsPrimitivePointer(key, response.Carrier)
	target := value + "." + context.Scope.Field(attribute, key, true)
	nullable := expr.IsNullable(response.Carrier)
	if nullable {
		target = "statusValue." + context.Scope.Field(attribute, key, true)
	}
	validation := codegen.AttributeValidationCode(attribute, nil, fieldContext, true, expr.IsAlias(attribute.Type), target, response.statusBinding.Name)
	if nullable {
		variable := "statusValue"
		if validation == "" {
			variable = "_"
		}
		return fmt.Sprintf("if %s, statusPresent := %s.Value(); !statusPresent {\nerr = loom.MissingFieldError(%q, \"response\")\n} else {\n%s\n}\n", variable, value, response.statusBinding.Name, validation)
	}
	return validation
}
