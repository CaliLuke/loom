package ir

import (
	"maps"
	"reflect"
	"slices"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	// sseEventOutcome is the one status-independent analysis of a server SSE
	// event, owned by the stream's ResponseValue. Every implicit success payload
	// and, when eligible, the derived async message reuse this completed outcome.
	sseEventOutcome struct {
		attribute *expr.AttributeExpr
		target    *transportir.ValueTarget
		schema    *Schema
		// eligible reports whether async documentation derives from this outcome
		// instead of its independent acquisition.
		eligible bool
	}

	// eventGraphComparison pairs the completed event target plan with a
	// finalized response target plan through captured immutable node queries.
	// It never invokes projectors or codecs; the pair guard stops recursion.
	eventGraphComparison struct {
		seen map[[2]expr.ValuePlanNode]bool
	}
)

// authoredBodyMarker is the provenance key the Body DSL adds to its occurrence.
// It is bookkeeping, not part of the authored contract.
const authoredBodyMarker = "http:body"

// derivedAsyncCandidate applies the endpoint-level conditions of the derived
// async branch: a pure server SSE stream with the exact !HasMixedResults bridge,
// no event projections, all five SSE mappings null and a nonempty event.
func derivedAsyncCandidate(endpoint *transportir.Endpoint) bool {
	stream := endpoint.Stream
	if endpoint.IsJSONRPC || stream == nil || !stream.IsSSE || stream.SSE == nil || stream.Direction != "server" {
		return false
	}
	if endpoint.Response == nil || endpoint.Response.HasMixedResults || len(endpoint.Response.Responses) == 0 {
		return false
	}
	sse := stream.SSE
	if len(sse.Projections) > 0 || sse.RequestIDField != "" || sse.DataField != "" || sse.IDField != "" || sse.EventField != "" || sse.RetryField != "" {
		return false
	}
	return stream.ResponseMessage != nil && stream.ResponseMessage.Type != expr.Empty
}

// derivedAsyncEligible decides, before any example synthesis, whether the async
// message derives from the completed event outcome. Every success response must
// correspond to the event target; no representative or first response stands
// in for the others.
func derivedAsyncEligible(endpoint *transportir.Endpoint, event *transportir.ValueTarget) bool {
	if !derivedAsyncCandidate(endpoint) || !preparedPlan(event) {
		return false
	}
	for _, response := range endpoint.Response.Responses {
		if !eventCorrespondence(endpoint.Stream.ResponseMessage, event, response) {
			return false
		}
	}
	return true
}

// eventCorrespondence compares the three role-specific operands of one success
// response. A typed handshake owns no body: its payload is the event itself.
// For a result-derived response, the source is the implicit ResultValue bridged
// by !HasMixedResults, or the pristine authored Body whose local facts must
// equal the event's; the finalized response target plan must then equal the
// completed event target plan node by node.
func eventCorrespondence(authoredEvent *expr.AttributeExpr, event *transportir.ValueTarget, response *transportir.ResponseStatus) bool {
	if response.IndependentDocumentBody || len(response.MediaVariants) > 0 || len(response.DocumentValues) > 0 {
		return false
	}
	if response.Handshake {
		return true
	}
	if response.BodyOrigin != "" || response.StatusField != nil || len(response.Headers) > 0 || len(response.Cookies) > 0 {
		return false
	}
	if response.Body == nil || response.Body.Type == expr.Empty || !preparedPlan(response.BodyValue) {
		return false
	}
	finalized := response.BodyValue
	if len(finalized.Selection) > 0 || finalized.Source == nil || finalized.Codec != event.Codec {
		return false
	}
	if projection := response.Body.HTTPBodyProjection(); projection != nil && len(projection.Excluded) > 0 {
		return false
	}
	if authored := response.AuthoredBody; authored != nil {
		meta := maps.Clone(authored.Meta)
		delete(meta, authoredBodyMarker)
		if !sameAuthoredRoot(authored, meta, authoredEvent) {
			return false
		}
	}
	comparison := &eventGraphComparison{seen: make(map[[2]expr.ValuePlanNode]bool)}
	return comparison.nodes(event.Plan.Root(), finalized.Plan.Root())
}

// eventOwnedResponse reports whether the response's payload documentation is
// the completed event outcome. An independent OpenAPIBody always keeps its own.
func eventOwnedResponse(event *sseEventOutcome, response *transportir.ResponseStatus) bool {
	if event == nil || response.IsError || response.IndependentDocumentBody {
		return false
	}
	return response.Handshake || event.eligible
}

func hasTypedHandshake(endpoint *transportir.Endpoint) bool {
	if endpoint.Response == nil || endpoint.Stream == nil || !endpoint.Stream.IsSSE {
		return false
	}
	for _, response := range endpoint.Response.Responses {
		if response.Handshake {
			return true
		}
	}
	return false
}

func preparedPlan(target *transportir.ValueTarget) bool {
	return target != nil && target.Error == nil && target.Plan.Root().Valid()
}

// sameAuthoredRoot compares an authored Body occurrence with the authored
// event occurrence: declaration identity of the root type and every
// occurrence-local fact, with only the Body DSL marker removed from meta.
func sameAuthoredRoot(body *expr.AttributeExpr, bodyMeta expr.MetaExpr, event *expr.AttributeExpr) bool {
	if !sameRootType(body.Type, event.Type) || !maps.EqualFunc(bodyMeta, event.Meta, slices.Equal[[]string]) {
		return false
	}
	return sameLocalFacts(body, event)
}

// sameRootType accepts the same named declaration or the same primitive. Inline
// structures are compared through the captured plans instead.
func sameRootType(left, right expr.DataType) bool {
	leftNamed, leftOK := left.(expr.UserType)
	rightNamed, rightOK := right.(expr.UserType)
	if leftOK || rightOK {
		return leftOK && rightOK && leftNamed.ID() == rightNamed.ID()
	}
	return left == right || expr.AsObject(left) != nil && expr.AsObject(right) != nil
}

// sameLocalFacts compares occurrence-local contract facts other than type and
// metadata: annotations, constraints, defaults, nullability and the ordered
// authored examples with their metadata. Values compare structurally.
func sameLocalFacts(left, right *expr.AttributeExpr) bool {
	if left.Description != right.Description || left.Title != right.Title || left.Nullable != right.Nullable {
		return false
	}
	if !reflect.DeepEqual(left.Docs, right.Docs) || !reflect.DeepEqual(left.Validation, right.Validation) ||
		!reflect.DeepEqual(left.DefaultValue, right.DefaultValue) || len(left.UserExamples) != len(right.UserExamples) {
		return false
	}
	for index, example := range left.UserExamples {
		peer := right.UserExamples[index]
		if example.Summary != peer.Summary || example.Description != peer.Description || example.ExplicitNull != peer.ExplicitNull ||
			!maps.EqualFunc(example.Meta, peer.Meta, slices.Equal[[]string]) || !reflect.DeepEqual(example.Value, peer.Value) {
			return false
		}
	}
	return true
}

// sameRenderedMeta compares the rendered annotation metadata of two target
// nodes. Synthesis bookkeeping and naming keys are outside schemaAnnotationMeta.
func sameRenderedMeta(left, right expr.MetaExpr) bool {
	keys := make(map[string]bool)
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}
	for key := range keys {
		if schemaAnnotationMeta(key) && !slices.Equal(left[key], right[key]) {
			return false
		}
	}
	return true
}

// sameEffectiveFacts compares the node-local facts of two target nodes. A
// named node contributes its occurrence facts over its declaration through the
// existing component composition, so a wrapper that carries copied declaration
// facts and an occurrence that inherits them compare equal.
func sameEffectiveFacts(event, response expr.ValuePlanNode) bool {
	left, right := effectiveFacts(event), effectiveFacts(response)
	leftValidation, leftOK := effectiveValidation(event.Attribute())
	rightValidation, rightOK := effectiveValidation(response.Attribute())
	if !leftOK || !rightOK || !reflect.DeepEqual(leftValidation, rightValidation) {
		return false
	}
	left.Validation, right.Validation = nil, nil
	return left.Nullable == right.Nullable && sameLocalFacts(left, right) && sameRenderedMeta(left.Meta, right.Meta)
}

// effectiveValidation is the lowered conjunction of every constraint the node
// inherits through its named chain, from expr's effective-constraint owner;
// transport flattening captures exactly this form.
func effectiveValidation(attribute *expr.AttributeExpr) (*expr.ValidationExpr, bool) {
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	if err != nil {
		return nil, false
	}
	validation := constraints.Validation().Lowered()
	if validation != nil && validation.HasRequiredOnly() && len(validation.Required) == 0 {
		validation = nil
	}
	return validation, true
}

func effectiveFacts(node expr.ValuePlanNode) *expr.AttributeExpr {
	effective := node.Attribute()
	if !node.Underlying().Valid() {
		return effective
	}
	// Component composition intentionally leaves nullable examples on the outer
	// occurrence. Comparison owns that full occurrence, not only its non-null
	// component, so retain the example group selected by the expression owner.
	examples := effective.ExtractUserExamples()
	nullable := effective.Nullable
	seen := make(map[expr.UserType]bool)
	for {
		named, ok := effective.Type.(expr.UserType)
		if !ok || seen[named] {
			break
		}
		seen[named] = true
		effective = componentAttribute(effective, named)
		nullable = nullable || effective.Nullable
	}
	effective.Nullable = nullable
	effective.UserExamples = examples
	return effective
}

// chainTerminal follows named edges to the structural node they denote.
func chainTerminal(node expr.ValuePlanNode) expr.ValuePlanNode {
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	return node
}

// eventFor returns the completed event outcome that documents response, or nil
// when the response keeps its own body documentation.
func (b *EndpointBodies) eventFor(response *transportir.ResponseStatus) *sseEventOutcome {
	if eventOwnedResponse(b.event, response) {
		return b.event
	}
	return nil
}

// nodes compares two target plan nodes and their captured children. Named
// edges compare declaration identity and their effective occurrence-over-
// declaration facts; a compiler body wrapper, recognized only through the
// plan's reuse record, stands for the occurrence it wraps. Structure compares
// members with wire names, visibility, requiredness and presence, union
// branches, elements, keys and additional members.
func (c *eventGraphComparison) nodes(event, response expr.ValuePlanNode) bool {
	if !event.Valid() || !response.Valid() {
		return event.Valid() == response.Valid()
	}
	pair := [2]expr.ValuePlanNode{event, response}
	if c.seen[pair] {
		return true
	}
	c.seen[pair] = true
	if event.Codec() != response.Codec() {
		return false
	}
	eventNamed, responseNamed := event.Underlying().Valid(), response.Underlying().Valid()
	if response.UnderlyingReusesSource() {
		// The wrapper carries the occurrence; its child is either a named
		// declaration copy (explicit Body) or the declaration's structure.
		inner := response.Underlying()
		switch {
		case !sameEffectiveFacts(event, response):
			return false
		case eventNamed && inner.Underlying().Valid():
			return c.nodes(event, inner)
		case eventNamed:
			return event.TargetDeclarationID() == response.TargetDeclarationID() && c.structure(event.Underlying(), inner)
		default:
			return c.structure(event, inner)
		}
	}
	if eventNamed && !responseNamed {
		// A transport body flattens a named alias chain into one node that keeps
		// the value-origin declaration identity and the chain's captured facts.
		identity := response.TargetDeclarationID()
		if identity == "" || identity != event.TargetDeclarationID() || !sameEffectiveFacts(event, response) {
			return false
		}
		return c.structure(chainTerminal(event), response)
	}
	if eventNamed != responseNamed || eventNamed && event.TargetDeclarationID() != response.TargetDeclarationID() {
		return false
	}
	if !sameEffectiveFacts(event, response) {
		return false
	}
	if eventNamed {
		return c.structure(event.Underlying(), response.Underlying())
	}
	return c.structure(event, response)
}

// structure compares the children of two nodes whose local facts were already
// compared. A named child restarts the declaration comparison.
func (c *eventGraphComparison) structure(event, response expr.ValuePlanNode) bool {
	if event.Underlying().Valid() || response.Underlying().Valid() {
		return c.nodes(event, response)
	}
	eventMembers, responseMembers := event.Members(), response.Members()
	if len(eventMembers) != len(responseMembers) {
		return false
	}
	for index, member := range eventMembers {
		peer := responseMembers[index]
		if member.Name != peer.Name || member.WireName != peer.WireName || member.Visible != peer.Visible ||
			member.Required != peer.Required || member.Presence != peer.Presence || !c.nodes(member.Node, peer.Node) {
			return false
		}
	}
	eventBranches, responseBranches := event.Branches(), response.Branches()
	if len(eventBranches) != len(responseBranches) {
		return false
	}
	for index, branch := range eventBranches {
		if branch.Tag != responseBranches[index].Tag || !c.nodes(branch.Node, responseBranches[index].Node) {
			return false
		}
	}
	return c.nodes(event.Element(), response.Element()) && c.nodes(event.Key(), response.Key()) &&
		c.nodes(event.Additional(), response.Additional())
}
