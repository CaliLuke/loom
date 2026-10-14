package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// JSONRPCIDData is the shared envelope-to-service field plan. It preserves
// physical field storage independently from protocol presence and correlation.
type JSONRPCIDData struct {
	// Argument is the validated service field supplied to payload constructors.
	Argument *InitArgData
	// DecodeCode admits, decodes and validates the envelope before conversion.
	DecodeCode string
	// FieldName is the actual service member, including authored renaming.
	FieldName string
	// ClientPresence is the physical presence expression for the service field.
	ClientPresence string
	// AllowsNull retains the field's explicit null admission for CLI decoding.
	AllowsNull bool
}

func jsonrpcIDDecodeCode(binding *expr.JSONRPCIDBinding, argument *InitArgData, carrier, converted, definition, validation, conversion string) string {
	nullCheck := ""
	if !expr.AllowsNull(binding.Attribute) {
		nullCheck = fmt.Sprintf("if req.ID == nil {\nreturn payload, loom.DecodePayloadError(%q)\n}\n", "JSON-RPC ID must not be null")
	}
	requiredCheck := ""
	if binding.Required {
		requiredCheck = fmt.Sprintf("if !req.HasID {\nreturn payload, loom.MissingFieldError(%q, \"envelope\")\n}\n", expr.AttributeName(binding.Name))
	}
	decode := fmt.Sprintf(`var %s %s
{
%s
%s := &%s{}
if req.HasID {
%s
idBytes := []byte("null")
if req.ID != nil {
 var encodeErr error
 idBytes, encodeErr = json.Marshal(jsonrpc.IDToString(req.ID))
 if encodeErr != nil {
  return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(encodeErr))
 }
}
if decodeErr := json.Unmarshal(idBytes, &%s.%s); decodeErr != nil {
 return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(decodeErr))
}
}
`, argument.VarName, argument.TypeRef, requiredCheck, carrier, definition, nullCheck, carrier, argument.FieldName)
	if validation != "" {
		decode += "var err error\n" + validation + "\nif err != nil {\nreturn payload, err\n}\n"
	}
	decode += conversion + fmt.Sprintf("\n%s = %s.%s\n}\n", argument.VarName, converted, argument.FieldName)
	return decode
}

func (b *payloadBuilder) buildJSONRPCID() *JSONRPCIDData {
	binding := b.endpointIR.Request.JSONRPCID
	if binding == nil {
		return nil
	}
	attribute := binding.Attribute
	name := binding.Name
	field := codegen.GoifyAtt(attribute, name, true)
	layout := expr.ServiceFieldLayout(b.payload, name, attribute, false, true)
	target := &expr.AttributeExpr{Type: &expr.Object{{Name: name, Attribute: attribute}}}
	if binding.Required {
		target.Validation = &expr.ValidationExpr{Required: []string{name}}
	}
	source := makeHTTPType(target)
	sourceContext := httpContext(b.sd.Scope, true, true)
	carrier := b.vars.names.Unique("rpcIDBody")
	converted := b.vars.names.Unique("rpcIDFields")
	value := b.vars.names.Unique("rpcID")
	typeRef := b.svc.Scope.GoFullTypeRef(attribute, b.pkg)
	if layout.Pointer {
		typeRef = "*" + typeRef
	}
	plan, err := codegen.GoTransform(source, target, carrier, converted, sourceContext, b.svcctx, "unmarshal", true)
	if err != nil {
		panic(codegen.NewError(b.sds.Ctx, attribute, fmt.Errorf("JSON-RPC ID conversion: %w", err)))
	}
	b.sd.ServerTransformHelpers = codegen.AppendHelpers(b.sd.ServerTransformHelpers, plan.Helpers)
	validation := codegen.ValidationCode(source, nil, sourceContext, true, false, false, carrier)
	defaultValue, _ := selectedTransportConstraints(attribute).Default()
	argument := &InitArgData{Ref: value, AttributeData: &AttributeData{
		Name: name, VarName: value, FieldName: field, Type: attribute.Type,
		FieldType: attribute.Type, TypeName: typeRef,
		TypeRef: typeRef, FieldPointer: layout.Pointer, Pointer: layout.Pointer,
		Required: binding.Required, DefaultValue: defaultValue,
		Validate: codegen.ValidationCode(attribute, nil, b.svcctx, binding.Required, expr.IsAlias(attribute.Type), false, value),
		Example:  attribute.Example(b.sds.examplesFor(b.sd)),
	}}
	presence := "true"
	if expr.AllowsNull(attribute) {
		presence = "p." + field + ".Present()"
	} else if layout.Pointer {
		presence = "p." + field + " != nil"
	}
	return &JSONRPCIDData{
		AllowsNull: expr.AllowsNull(attribute), ClientPresence: presence,
		Argument: argument, FieldName: field,
		DecodeCode: jsonrpcIDDecodeCode(binding, argument, carrier, converted,
			goTypeDef(b.sd.Scope, source, true, false, false), validation, inlineConversionCode(plan)),
	}
}
