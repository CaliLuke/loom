package service

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// interceptorMethodBindings keeps physical field storage out of the shared
	// accessor interface. Its index identifies one declared read/write role.
	interceptorMethodBindings [interceptorAccessCount][]*interceptorAttributeBinding

	interceptorAttributeBinding struct {
		*AttributeData
		logicalName string
		selector    string
		indirect    bool
		address     bool
		valueType   string
	}

	interceptorSignatureKey struct {
		interceptor string
		role        int
		field       string
	}

	interceptorSignatureOrigin struct {
		method *expr.MethodExpr
		ref    string
	}
)

const (
	interceptorReadPayload = iota
	interceptorWritePayload
	interceptorReadResult
	interceptorWriteResult
	interceptorReadStreamingPayload
	interceptorWriteStreamingPayload
	interceptorReadStreamingResult
	interceptorWriteStreamingResult
	interceptorAccessCount
)

func collectInterceptorMethodBindings(i *expr.InterceptorExpr, m *expr.MethodExpr, scope *codegen.NameScope) interceptorMethodBindings {
	return interceptorMethodBindings{
		collectInterceptorBindings(i.ReadPayload, m.Payload, scope),
		collectInterceptorBindings(i.WritePayload, m.Payload, scope),
		collectInterceptorBindings(i.ReadResult, m.Result, scope),
		collectInterceptorBindings(i.WriteResult, m.Result, scope),
		collectInterceptorBindings(i.ReadStreamingPayload, m.StreamingPayload, scope),
		collectInterceptorBindings(i.WriteStreamingPayload, m.StreamingPayload, scope),
		collectInterceptorBindings(i.ReadStreamingResult, m.EffectiveStreamingResult(), scope),
		collectInterceptorBindings(i.WriteStreamingResult, m.EffectiveStreamingResult(), scope),
	}
}

func collectInterceptorBindings(names, parent *expr.AttributeExpr, scope *codegen.NameScope) []*interceptorAttributeBinding {
	if names == nil || expr.AsObject(names.Type) == nil {
		return nil
	}
	fields := *expr.AsObject(names.Type)
	bindings := make([]*interceptorAttributeBinding, len(fields))
	for index, field := range fields {
		attribute := parent.Find(field.Name)
		if attribute == nil {
			continue
		}
		layout := expr.ServiceFieldLayout(parent, field.Name, attribute, false, true)
		logical := attribute
		if layout.WrapNullable {
			copy := *attribute
			copy.Nullable = true
			logical = &copy
		}
		var pkg string
		if loc := codegen.UserTypeLocation(attribute.Type); loc != nil {
			pkg = scope.PackageName(loc)
		}
		ref := scope.GoFullTypeRef(logical, pkg)
		valueType := scope.GoFullTypeName(logical, pkg)
		logicalPointer := ref != valueType
		bindings[index] = &interceptorAttributeBinding{
			AttributeData: &AttributeData{
				Name:    codegen.Goify(field.Name, true),
				TypeRef: ref,
			},
			logicalName: field.Name,
			selector:    codegen.GoifyAtt(attribute, field.Name, true),
			indirect:    layout.Pointer && !logicalPointer,
			address:     !layout.Pointer && logicalPointer,
			valueType:   valueType,
		}
	}
	return bindings
}

func interceptorSignatures(bindings []*interceptorAttributeBinding) []*AttributeData {
	if len(bindings) == 0 {
		return nil
	}
	fields := make([]*AttributeData, len(bindings))
	for index, binding := range bindings {
		if binding != nil {
			fields[index] = binding.AttributeData
		}
	}
	return fields
}

// admitInterceptorContracts compares logical signatures over both invocation
// sides before declaration coverage or output files can be published.
func admitInterceptorContracts(ctx *codegen.Context, server, client []*InterceptorData) {
	seen := make(map[interceptorSignatureKey]interceptorSignatureOrigin)
	roles := [...]string{
		"ReadPayload", "WritePayload", "ReadResult", "WriteResult",
		"ReadStreamingPayload", "WriteStreamingPayload", "ReadStreamingResult", "WriteStreamingResult",
	}
	for _, side := range [][]*InterceptorData{server, client} {
		for _, interceptor := range side {
			for _, method := range interceptor.Methods {
				for role, bindings := range method.attributes {
					for _, binding := range bindings {
						key := interceptorSignatureKey{interceptor.DesignName, role, binding.logicalName}
						if prior, ok := seen[key]; ok && prior.ref != binding.TypeRef {
							err := fmt.Errorf("interceptor %s %s field %q has incompatible signatures: method %q uses %s, method %q uses %s", interceptor.DesignName, roles[role], binding.logicalName, prior.method.Name, prior.ref, method.design.Name, binding.TypeRef)
							panic(codegen.NewError(ctx.WithMethod(method.design), method.design, err))
						}
						seen[key] = interceptorSignatureOrigin{method: method.design, ref: binding.TypeRef}
					}
				}
			}
		}
	}
}
