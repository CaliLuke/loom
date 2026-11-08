package codegen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// buildOutgoingStatusPlan lowers the captured domain against the actual server
// carrier. The caller declares statusVar, owns transformErr and checks it before publication.
func buildOutgoingStatusPlan(response *ResponseData, carrier *expr.AttributeExpr, context *codegen.AttributeContext, value, statusVar string) codegen.TransformPlan {
	if response.statusBinding == nil {
		return outgoingLiteralStatusPlan(response, statusVar)
	}
	if carrier == nil || context == nil {
		panic("HTTP status binding has no native carrier context")
	}
	violation := fmt.Sprintf("&loomhttp.StatusViolationError{Selector: %#v}", responseContractSelectorValue(response.Selector))
	var code strings.Builder
	preparedValue := response.EncodePlan != nil && response.EncodePlan.sourceProjection != nil && response.EncodePlan.sourceProjection.statusBound
	if preparedValue {
		code.WriteString(response.EncodePlan.sourceProjection.sourcePreparation(value, func(string) string {
			return "if *transformErr == nil {\n*transformErr = " + violation + "\n}"
		}, ""))
		value = response.EncodePlan.FieldSource
		carrier = representation.ConcreteAttribute(carrier)
	}
	key, field := carrier.FindAttribute(response.statusBinding.Name)
	if field == nil {
		panic("HTTP status binding is absent from its native carrier")
	}
	member := value + "." + context.Scope.Field(field, key, true)
	fieldRef := context.Scope.Ref(field, context.Pkg(field))
	code.WriteString("if *transformErr == nil {\n")
	guards := 0
	if !preparedValue && strings.HasPrefix(context.Scope.Ref(carrier, context.Pkg(carrier)), "*") {
		fmt.Fprintf(&code, "if %s == nil {\n*transformErr = %s\n} else {\n", value, violation)
		guards++
	}
	if context.IsPrimitivePointer(key, carrier) {
		fmt.Fprintf(&code, "if %s == nil {\n*transformErr = %s\n} else {\n", member, violation)
		member = "*" + member
		guards++
	}
	statuses := make([]string, len(response.Domain))
	for i, status := range response.Domain {
		statuses[i] = strconv.Itoa(status)
	}
	// Both comparisons use the actual native value. A conversion to a narrow
	// native type cannot make a different candidate match; a wide value is never
	// converted to int before membership is established. The selected int comes
	// from the captured domain itself.
	fmt.Fprintf(&code, "statusFound := false\nfor _, ownedStatus := range []int{%s} {\nif uint64(%s) == uint64(ownedStatus) && %s == %s(ownedStatus) {\n%s = ownedStatus\nstatusFound = true\nbreak\n}\n}\n", strings.Join(statuses, ", "), member, member, fieldRef, statusVar)
	fmt.Fprintf(&code, "if !statusFound {\n*transformErr = fmt.Errorf(\"native response status %%v violates the selected domain: %%w\", %s, %s)\n}\n", member, violation)
	validationResponse := *response
	validationResponse.Carrier = carrier
	if validation := responseNativeStatusValidation(&validationResponse, context, value); validation != "" {
		fmt.Fprintf(&code, "if *transformErr == nil {\nvar err error\n%s\nif err != nil {\n*transformErr = &loomhttp.StatusViolationError{Code: %s, Selector: %#v}\n}\n}\n", validation, statusVar, responseContractSelectorValue(response.Selector))
	}
	for range guards {
		code.WriteString("}\n")
	}
	code.WriteString("}\n")
	return codegen.TransformPlan{Code: code.String(), ErrorAware: true}
}

// outgoingLiteralStatusPlan uses the captured emission domain even when no
// native status field is present. Observation-only declarations have no member.
func outgoingLiteralStatusPlan(response *ResponseData, statusVar string) codegen.TransformPlan {
	status, ok := response.Selector.Exact()
	if !ok {
		panic("HTTP dynamic response selector has no native status binding")
	}
	if !slices.Contains(response.Domain, status) {
		return codegen.TransformPlan{
			Code:       fmt.Sprintf("if *transformErr == nil {\n*transformErr = &loomhttp.StatusViolationError{Code: %d, Selector: %#v}\n}\n", status, responseContractSelectorValue(response.Selector)),
			ErrorAware: true,
		}
	}
	return codegen.TransformPlan{Code: fmt.Sprintf("%s = %d\n", statusVar, status)}
}
