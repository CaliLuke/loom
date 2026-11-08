package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// buildOutgoingCookiePlan validates native and mapped cookie fields, then uses
// the runtime policy owner to prepare values in the already detached header map.
func buildOutgoingCookiePlan(response *ResponseData, carrier *expr.AttributeExpr, context *codegen.AttributeContext, value, statusVar, headerVar, requestContext string) codegen.TransformPlan {
	if response.EncodePlan != nil && response.EncodePlan.sourceProjection != nil && response.EncodePlan.sourceProjection.statusBound {
		value = response.EncodePlan.FieldSource
	}

	if len(response.Cookies) == 0 {
		return codegen.TransformPlan{}
	}
	var checks []string
	for _, cookie := range response.Cookies {
		key, native := carrier.FindAttribute(cookie.Name)
		if native == nil {
			key, native = "", carrier
		}
		checks = append(checks, outgoingMetadataFieldValidation(cookie.Element, carrier, native, key, context, value))
	}
	checks = append(checks, "return err")
	selector := fmt.Sprintf("%#v", responseContractSelectorValue(response.Selector))
	encoded, err := renderResponseCookies(response, value, headerVar, requestContext)
	if err != nil {
		panic(err)
	}
	code := fmt.Sprintf("if *transformErr == nil {\nif err := func() (err error) {\n%s\n}(); err != nil {\n*transformErr = &loomhttp.StatusViolationError{Code: %s, Selector: %s}\n}\n}\n", strings.Join(checks, "\n"), statusVar, selector)
	code += fmt.Sprintf("if *transformErr == nil {\nif err := func() error {\n%s\nreturn nil\n}(); err != nil {\n*transformErr = err\n}\n}\n", encoded)
	return codegen.TransformPlan{Code: code, ErrorAware: true}
}
