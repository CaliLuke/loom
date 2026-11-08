package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// buildOutgoingHeaderPlan checks the selected native fields and their mapped
// contracts before preparing an isolated header map. The caller checks its
// shared transformErr before publishing either status or headers.
func buildOutgoingHeaderPlan(response *ResponseData, carrier *expr.AttributeExpr, context *codegen.AttributeContext, value, statusVar, headerVar, view string) codegen.TransformPlan {
	preparedValue := response.EncodePlan != nil && response.EncodePlan.sourceProjection != nil && response.EncodePlan.sourceProjection.statusBound
	if preparedValue {
		value = response.EncodePlan.FieldSource
	}
	encoded, err := renderResponseHeaders(response, value, headerVar)
	if err != nil {
		panic(err)
	}
	var checks []string
	if !preparedValue && strings.HasPrefix(context.Scope.Ref(carrier, context.Pkg(carrier)), "*") {
		checks = append(checks, fmt.Sprintf("if %s == nil {\nreturn fmt.Errorf(\"response metadata is absent\")\n}", value))
	}
	checks = append(checks, responseRequiredValidation(response, carrier, context, value, view))
	for _, header := range response.Headers {
		key, native := carrier.FindAttribute(header.Name)
		if native == nil {
			key, native = "", carrier
		}
		checks = append(checks, outgoingMetadataFieldValidation(header.Element, carrier, native, key, context, value))
	}
	checks = append(checks, "if err != nil {\nreturn err\n}", encoded)
	for _, header := range response.Headers {
		if header.Required {
			checks = append(checks, fmt.Sprintf("if _, present := %s[%q]; !present {\nreturn loom.MissingFieldError(%q, \"response headers\")\n}", headerVar, header.CanonicalName, header.Name))
		}
	}
	checks = append(checks, "return nil")
	selector := fmt.Sprintf("%#v", responseContractSelectorValue(response.Selector))
	return codegen.TransformPlan{
		Code:       fmt.Sprintf("%s := make(http.Header)\nif *transformErr == nil {\nif err := func() (err error) {\n%s\n}(); err != nil {\n*transformErr = &loomhttp.StatusViolationError{Code: %s, Selector: %s}\n}\n}\n", headerVar, strings.Join(checks, "\n"), statusVar, selector),
		ErrorAware: true,
	}
}

func outgoingMetadataFieldValidation(field *Element, carrier, native *expr.AttributeExpr, key string, context *codegen.AttributeContext, value string) string {
	target := value
	if field.FieldName != "" {
		target += "." + field.FieldName
	}
	fieldContext := context.Dup()
	fieldContext.Pointer = field.FieldPointer
	var checks []string
	for _, contract := range []*expr.AttributeExpr{native, field.attribute} {
		attribute := *contract
		attribute.Type, attribute.Meta = native.Type, native.Meta
		validationTarget := target
		required := field.Required
		validationContext := fieldContext
		if isStringMetaType(&attribute) {
			attribute = *rawTextValidationAttribute(&attribute)
			validationTarget = transportScalarValue(field.Type, native.Type, transportFieldReference(field.AttributeData, value), true)
			validationContext = context.Dup()
			validationContext.Pointer = false
			required = true
		}
		validation := codegen.AttributeValidationCode(&attribute, nil, validationContext, required, expr.IsAlias(attribute.Type), validationTarget, field.Name)
		if validation == "" || len(checks) > 0 && checks[0] == validation {
			continue
		}
		checks = append(checks, validation)
	}
	validation := strings.Join(checks, "\n")
	if validation == "" {
		return ""
	}
	layout := expr.ServiceFieldLayout(carrier, key, native, context.Pointer, context.UseDefault)
	if layout.Pointer && (!expr.IsPrimitive(native.Type) || field.IsTextUnmarshaler) {
		validation = fmt.Sprintf("if %s != nil {\n%s\n}", target, validation)
	}
	return validation
}
