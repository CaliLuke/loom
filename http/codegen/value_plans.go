package codegen

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type httpValuePlanBuilder struct {
	request  expr.ValuePlanRequest
	service  *ServiceData
	server   bool
	seen     map[*expr.AttributeExpr]bool
	boundary string
}

// buildHTTPValuePlan runs at the actual body declaration owner, after layout
// and view selection. Its policies consume the same tags used by the emitter.
func buildHTTPValuePlan(value *transportir.ValueTarget, target *expr.AttributeExpr, context *codegen.AttributeContext, data *ServiceData, server bool) *transportir.ValueTarget {
	if value == nil {
		return nil
	}
	result := *value
	if result.Error != nil || result.Source == nil || result.Boundary != "" {
		return &result
	}
	target = httpValueTargetGraph(target)
	builder := httpValuePlanBuilder{
		request: expr.ValuePlanRequest{Target: target, Selection: append([]string(nil), value.Selection...), Codec: value.Codec, Use: expr.ValuePlanRuntime},
		service: data, server: server, seen: make(map[*expr.AttributeExpr]bool),
	}
	if err := builder.walk(target, context, false, true); err != nil {
		result.Error = err
		return &result
	}
	if builder.boundary != "" {
		result.Boundary = builder.boundary
		return &result
	}
	result.Plan, result.Error = result.Source.Context.NewValuePlan(result.Source.Occurrence, builder.request)
	return &result
}

func (b *httpValuePlanBuilder) walk(attribute *expr.AttributeExpr, context *codegen.AttributeContext, untaggedBranch, reachable bool) error {
	if attribute == nil {
		return nil
	}
	prior, visited := b.seen[attribute]
	if visited && (prior || !reachable) {
		return nil
	}
	b.seen[attribute] = prior || reachable
	if reachable && b.boundary == "" {
		b.boundary = httpCustomValueCodec(attribute)
	}
	local := *context
	if b.service != nil {
		applyUserTypeLayout(&local, b.service, attribute, b.server)
	}
	switch actual := attribute.Type.(type) {
	case expr.UserType:
		return b.walk(actual.Attribute(), &local, untaggedBranch, reachable)
	case *expr.Object:
		return b.object(attribute, actual, &local, untaggedBranch, reachable, visited)
	case *expr.Array:
		return b.walk(actual.ElemType, &local, false, reachable)
	case *expr.Map:
		if err := b.walk(actual.KeyType, &local, false, reachable); err != nil {
			return err
		}
		return b.walk(actual.ElemType, &local, false, reachable)
	case *expr.Union:
		// Generated tagged envelopes use ordinary json.Unmarshal into a
		// two-field struct, which ignores unknown envelope members.
		if !visited {
			b.request.Containers = append(b.request.Containers, expr.ValueContainerPolicy{Target: attribute})
		}
		for _, branch := range actual.Values {
			if err := b.walk(branch.Attribute, &local, actual.Untagged, reachable); err != nil {
				return err
			}
		}
	}
	return nil
}

func emittedHTTPFieldPolicy(parent *expr.AttributeExpr, mapped *expr.MappedAttributeExpr, field *expr.NamedAttributeExpr, context *codegen.AttributeContext) (expr.ValueFieldPolicy, error) {
	name := expr.AttributeName(field.Name)
	optional, omitZero := httpFieldOmission(mapped, name, field.Attribute, context.Pointer, context.UseDefault, context.JSONPresence)
	tags := attributeTags(field.Attribute, expr.ElementName(field.Name), optional, omitZero)
	unquoted, err := strconv.Unquote(strings.TrimSpace(tags))
	if err != nil {
		return expr.ValueFieldPolicy{}, fmt.Errorf("HTTP field %q tags: %w", field.Name, err)
	}
	parts := strings.Split(reflect.StructTag(unquoted).Get("json"), ",")
	wireName := parts[0]
	if wireName == "" {
		wireName = codegen.GoifyAtt(field.Attribute, name, true)
	}
	presence := expr.ValueFieldRetain
	for _, option := range parts[1:] {
		switch option {
		case "omitempty":
			presence = expr.ValueFieldOmitEmpty
		case "omitzero":
			presence = expr.ValueFieldOmitAbsent
		}
	}
	var numeric expr.Kind
	typ := field.Attribute.Type
	for {
		alias, ok := typ.(expr.UserType)
		if !ok {
			break
		}
		typ = alias.Attribute().Type
	}
	if primitive, ok := typ.(expr.Primitive); ok {
		kind := primitive.Kind()
		if kind >= expr.IntKind && kind <= expr.Float64Kind {
			numeric = kind
		}
	}
	return expr.ValueFieldPolicy{
		Parent: parent, Target: field.Attribute, Name: field.Name, WireName: wireName,
		Visible: parts[0] != "-", Required: mapped.IsRequired(name), Presence: presence, NumericKind: numeric,
	}, nil
}

// httpCustomValueCodec follows the same replacement sites as goTypeDef.
// Finalization removes recognized matching framework Nullable metadata; ordinary
// Optional/Nullable/JSONValue layout chosen by the emitter needs no replacement.
// Any residual authored replacement keeps its external codec contract.
func httpCustomValueCodec(attribute *expr.AttributeExpr) string {
	typeName, _ := codegen.GetMetaType(attribute)
	_, primitive := attribute.Type.(expr.Primitive)
	if typeName != "" && (primitive || codegen.IsExplicitPresenceType(attribute)) {
		return fmt.Sprintf("emitted type %s owns an external codec", typeName)
	}
	return ""
}

func (b *httpValuePlanBuilder) object(attribute *expr.AttributeExpr, actual *expr.Object, context *codegen.AttributeContext, untaggedBranch, reachable, visited bool) error {
	closed, _ := attribute.Meta.Last("openapi:additionalProperties")
	if !visited {
		b.request.Containers = append(b.request.Containers, expr.ValueContainerPolicy{
			Target: attribute, RejectUnknownMembers: untaggedBranch && closed == "false",
			PreserveAdditional: false,
		})
	}
	mapped := expr.NewMappedAttributeExpr(attribute)
	for _, field := range *actual {
		policy, err := emittedHTTPFieldPolicy(attribute, mapped, field, context)
		if err != nil {
			return err
		}
		if !visited {
			b.request.Fields = append(b.request.Fields, policy)
		}
		if err := b.walk(field.Attribute, context, false, reachable && policy.Visible); err != nil {
			return err
		}
	}
	return nil
}
