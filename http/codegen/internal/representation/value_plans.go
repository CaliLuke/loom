package representation

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

type (
	httpValuePlanCapture struct {
		target   *expr.AttributeExpr
		context  *codegen.AttributeContext
		request  expr.ValuePlanRequest
		boundary string
	}

	httpValuePlanBuilder struct {
		request         expr.ValuePlanRequest
		seen            map[*expr.AttributeExpr]bool
		boundary        string
		projectedRoot   *expr.AttributeExpr
		projectedFields map[string]contributorField
	}
)

// BuildValuePlan runs at the actual body declaration owner, after layout
// and view selection. Its policies consume the same tags used by the emitter.
func BuildValuePlan(value *transportir.ValueTarget, target *expr.AttributeExpr, context *codegen.AttributeContext, use expr.ValuePlanUse) *transportir.ValueTarget {
	return buildValuePlan(value, target, context, use, nil)
}

// buildValuePlan builds the plan of target. A projected contributor supplies
// its parameter contract and the selected body's common-field wire policies;
// ordinary targets capture their contract and derive policies normally.
func buildValuePlan(value *transportir.ValueTarget, target *expr.AttributeExpr, context *codegen.AttributeContext, use expr.ValuePlanUse, contract *contributorContract) *transportir.ValueTarget {
	if value == nil {
		return nil
	}
	result := *value
	if result.Error != nil || result.Source == nil {
		return &result
	}
	capture, err := captureValuePlan(value, target, context, use, contract)
	if err != nil {
		clearTargetValues(&result)
		result.ProjectionContext = capture.context
		result.Error = err
		return &result
	}
	return buildCapturedValuePlan(value, capture, use)
}

// captureValuePlan owns one finalized body/codec snapshot. Media slots of that
// exact body may reuse it in the same semantic context. Plans and their effect
// caches remain independent; no lookup survives the enclosing body preparation.
func captureValuePlan(value *transportir.ValueTarget, target *expr.AttributeExpr, context *codegen.AttributeContext, use expr.ValuePlanUse, contract *contributorContract) (*httpValuePlanCapture, error) {
	target = TargetGraph(target)
	var parameter *parameterContract
	if contract != nil {
		parameter = &contract.parameter
	}
	boundaries := normalizeParameterTarget(target, value.Codec, parameter)
	if value.Codec == expr.ValueCodecText {
		local := *context
		local.JSONPresence = false
		local.CollectionElementPresence = false
		context = &local
	}
	capture := &httpValuePlanCapture{target: target, context: captureProjectionContext(context, target)}
	builder := httpValuePlanBuilder{
		request: expr.ValuePlanRequest{Codec: value.Codec, Use: use},
		seen:    make(map[*expr.AttributeExpr]bool),
	}
	if contract != nil {
		builder.projectedRoot = contributorObject(target)
		builder.projectedFields = contract.fields
	}
	if value.SSEDataField != "" {
		builder.request.Codecs = mappedSSECodecs(target, context, value.SSEDataField)
	}
	if value.Documentary && use != expr.ValuePlanSchema {
		builder.request.Use = expr.ValuePlanDocumentation
	}
	if err := builder.walk(target, context, false, true); err != nil {
		return capture, err
	}
	builder.request.Codecs = appendJSONContentCodecs(builder.request.Codecs, boundaries)
	capture.boundary = builder.boundary
	captured, err := value.Source.Context.NewOccurrence(target)
	if err != nil {
		return capture, err
	}
	builder.request.TargetOccurrence = captured
	capture.request = builder.request
	return capture, nil
}

func buildCapturedValuePlan(value *transportir.ValueTarget, capture *httpValuePlanCapture, use expr.ValuePlanUse) *transportir.ValueTarget {
	result := *value
	// Each slot owns its attachments and fresh projection/effect caches, even
	// when immutable captured attributes and policy locations are shared.
	clearTargetValues(&result)
	result.ProjectionContext = capture.context
	request := capture.request
	request.Selection = append([]string(nil), value.Selection...)
	if capture.boundary != "" {
		result.Boundary = capture.boundary
	}
	result.Plan, result.Error = result.Source.Context.NewValuePlan(result.Source.Occurrence, request)
	if result.Error != nil || use != expr.ValuePlanSchema {
		return &result
	}
	result.Error = buildDocumentationPlans(&result, capture.target, request)
	return &result
}

func buildDocumentationPlans(result *transportir.ValueTarget, target *expr.AttributeExpr, request expr.ValuePlanRequest) (err error) {
	// Documentation examples are JSON values attached to OpenAPI schemas even
	// when the runtime target is text, multipart or raw. Node-local custom/SSE
	// codec policies remain explicit boundaries in the copied request.
	documentation := DocumentationPlanRequest(target, request)
	if result.Anchor != nil {
		if result.Anchor.Context != result.Source.Context {
			captured, err := result.Anchor.Context.NewOccurrence(target)
			if err != nil {
				return err
			}
			documentation.TargetOccurrence = captured
		}
		result.AnchorPlan, err = result.Anchor.Context.NewValuePlan(result.Anchor.Occurrence, documentation)
		if err != nil {
			return err
		}
	}
	if result.ExampleOccurrence.ID() != (expr.ValueIdentity{}) {
		documentation.Selection = nil
		documentation.TargetOccurrence = request.TargetOccurrence
		result.ExamplePlan, err = result.Source.Context.NewValuePlan(result.ExampleOccurrence, documentation)
		if err != nil {
			return err
		}
	}
	return nil
}

// appendJSONContentCodecs binds each JSON content boundary of a form or
// multipart target to the JSON codec. One policy owns an exact target: a
// target that already has a custom or SSE policy keeps it, and a custom codec
// below a boundary still overrides the JSON codec its descendants inherit.
func appendJSONContentCodecs(codecs []expr.ValueCodecPolicy, boundaries []*expr.AttributeExpr) []expr.ValueCodecPolicy {
	owned := make(map[*expr.AttributeExpr]bool, len(codecs)+len(boundaries))
	for _, policy := range codecs {
		owned[policy.Target] = true
	}
	for _, target := range boundaries {
		if !owned[target] {
			owned[target] = true
			codecs = append(codecs, expr.ValueCodecPolicy{Target: target, Codec: expr.ValueCodecJSON})
		}
	}
	return codecs
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
	if boundary := CustomCodec(attribute); boundary != "" {
		if !visited {
			b.request.Codecs = append(b.request.Codecs, expr.ValueCodecPolicy{Target: attribute, Codec: expr.ValueCodecCustom})
		}
		if reachable && b.boundary == "" {
			b.boundary = boundary
		}
	}
	local := *context
	ApplyLayout(&local, attribute)
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

func emittedHTTPFieldPolicy(parent *expr.AttributeExpr, mapped *expr.ObjectFields, field *expr.NamedAttributeExpr, context *codegen.AttributeContext) (expr.ValueFieldPolicy, error) {
	name := expr.AttributeName(field.Name)
	resolved := ResolveField(mapped, name, field.Attribute, context.Pointer, context.UseDefault, context.JSONPresence)
	tags, err := emittedFieldTags(resolved.Encoding, field.Name)
	if err != nil {
		return expr.ValueFieldPolicy{}, err
	}
	parts := strings.Split(tags.Get("json"), ",")
	wireName := emittedJSONFieldName(field.Attribute, field.Name, tags)
	presence := expr.ValueFieldRetain
	switch resolved.Encoding.JSON.Omission {
	case encodingmeta.JSONOmitEmpty:
		presence = expr.ValueFieldOmitEmpty
	case encodingmeta.JSONOmitAbsent:
		presence = expr.ValueFieldOmitAbsent
	case encodingmeta.JSONOmitEmptyOrNativeZero:
		presence = expr.ValueFieldOmitEmptyOrZero
	case encodingmeta.JSONOmitNativeZero:
		presence = expr.ValueFieldOmitZero
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
		Visible: parts[0] != "-", Required: mapped.FieldFacts(name).Required, Presence: presence, NumericKind: numeric,
	}, nil
}

// CustomCodec follows the same replacement sites as goTypeDef.
// Finalization removes recognized matching framework Nullable metadata; ordinary
// Optional/Nullable/JSONValue layout chosen by the emitter needs no replacement.
// Any residual authored replacement keeps its external codec contract.
func CustomCodec(attribute *expr.AttributeExpr) string {
	typeName := encodingmeta.Replacement(attribute.Meta)
	_, primitive := attribute.Type.(expr.Primitive)
	if typeName != "" && (primitive || codegen.IsExplicitPresenceType(attribute)) {
		return fmt.Sprintf("emitted type %s owns an external codec", typeName)
	}
	return ""
}

func (b *httpValuePlanBuilder) object(attribute *expr.AttributeExpr, actual *expr.Object, context *codegen.AttributeContext, untaggedBranch, reachable, visited bool) error {
	if !visited {
		b.request.Containers = append(b.request.Containers, expr.ValueContainerPolicy{
			Target: attribute, RejectUnknownMembers: untaggedBranch && attribute.ObjectOpenness() == expr.ObjectClosed,
			PreserveAdditional: false,
		})
	}
	mapped := expr.NewObjectFields(attribute)
	for _, field := range *actual {
		policy, err := emittedHTTPFieldPolicy(attribute, mapped, field, context)
		if err != nil {
			return err
		}
		if attribute == b.projectedRoot {
			if physical, found := b.projectedFields[field.Name]; found {
				policy.WireName, policy.Visible, policy.Presence = physical.WireName, physical.Visible, physical.Presence
			}
		}
		if !visited {
			b.request.Fields = append(b.request.Fields, policy)
		}
		if err := b.walk(field.Attribute, context, false, reachable && policy.Visible); err != nil {
			return err
		}
	}
	return b.walk(attribute.AdditionalAttributes, context, false, reachable)
}

// clearTargetValues invalidates attachments keyed by replaced plan nodes.
func clearTargetValues(target *transportir.ValueTarget) {
	target.Examples = nil
	target.Representative = nil
	target.ExamplesPrepared = false
	target.ExampleSets = nil
	target.Annotations = nil
	target.AnnotationsPrepared = false
}
