package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// PrepareStreamSchema pairs a finalized inline or view-projected stream target
// with the prepared stream's source and codec. It uses the service's existing
// declaration layout; neither media labels nor schema shape select the codec.
func PrepareStreamSchema(endpoint *transportir.Endpoint, target *expr.AttributeExpr, inbound bool) *transportir.ValueTarget {
	if target == nil || target.Type == expr.Empty {
		return nil
	}
	if endpoint == nil || endpoint.Stream == nil || endpoint.Service == nil {
		return &transportir.ValueTarget{Error: fmt.Errorf("stream schema requires a prepared endpoint")}
	}
	value := endpoint.Stream.ResponseValue
	if inbound {
		value = endpoint.Stream.RequestValue
	}
	if value == nil {
		return &transportir.ValueTarget{Error: fmt.Errorf("stream schema has no prepared source occurrence")}
	}
	layout, _ := ServiceLayouts(endpoint.Service.Endpoints)
	context := schemaContext(inbound)
	if inbound || !endpoint.Stream.HasMixedResults {
		layout.Bind(context)
	}
	return BuildValuePlan(value, target, context, expr.ValuePlanSchema)
}

func mappedSSECodecs(target *expr.AttributeExpr, context *codegen.AttributeContext, name string) []expr.ValueCodecPolicy {
	local := *context
	context = &local
	parent := target
	for {
		typ, named := parent.Type.(expr.UserType)
		if !named {
			break
		}
		ApplyLayout(context, parent)
		parent = typ.Attribute()
	}
	object := expr.AsObject(parent.Type)
	if object == nil {
		return nil
	}
	field := object.Attribute(name)
	if field == nil || CustomCodec(field) != "" {
		return nil
	}
	mapped := expr.NewMappedAttributeExpr(parent)
	if context.FieldPresence(mapped, name, field) != codegen.NativePresence {
		return nil
	}
	encoding := MappedSSEEncoding(context.Scope.Ref(field, context.DefaultPkg), context.IsPrimitivePointer(name, parent))
	if encoding.Codec == expr.ValueCodecJSON {
		return nil
	}
	return []expr.ValueCodecPolicy{{Target: field, Codec: encoding.Codec}}
}
