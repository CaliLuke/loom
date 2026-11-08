package representation

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	// WireContext is the resolved wire contract of one HTTP body occurrence:
	// its codec, selected layout and resolved field omission, and for form
	// and multipart bodies, the field contracts that
	// the media contract owners captured from the actual body, keyed by full
	// authored field identity.
	WireContext struct {
		codec    expr.ValueCodec
		contract parameterContract
		context  *codegen.AttributeContext
		fields   map[string]contributorField
		identity string
	}

	// Contributor is a declaration projected into the wire context of an HTTP
	// body occurrence: its own contract without the members the body
	// excludes, with the body's wire facts for every member the body shares.
	Contributor struct {
		// Attribute is the projected declaration occurrence.
		Attribute *expr.AttributeExpr
		// Target is the schema target of Attribute, normalized with the
		// projected contract.
		Target *transportir.ValueTarget
		// Key identifies the declaration, its selected members and their
		// resolved wire facts. Equal keys denote equal projections.
		Key string
	}

	// contributorKey is the content identity of a projected contributor.
	contributorKey struct {
		Context     string                                     `json:"context"`
		Declaration string                                     `json:"declaration"`
		Codec       expr.ValueCodec                            `json:"codec"`
		Excluded    []string                                   `json:"excluded"`
		Form        map[string]httpencoding.FieldCodec         `json:"form,omitempty"`
		Multipart   map[string]expr.HTTPMultipartFieldEncoding `json:"multipart,omitempty"`
		Custom      bool                                       `json:"custom,omitempty"`
	}
)

// CaptureWireContext captures the wire context of the HTTP body object body
// from its prepared target and exact plan position. It retains selected
// layout and common-field wire policies without copying semantic constraints.
func CaptureWireContext(body *expr.AttributeExpr, target *transportir.ValueTarget, node expr.ValuePlanNode) (*WireContext, error) {
	if target == nil || target.ProjectionContext == nil {
		return nil, fmt.Errorf("HTTP body contributor requires a prepared projection context")
	}
	codec := target.Codec
	if node.Valid() {
		codec = node.Codec()
	}
	contract, err := captureParameterContract(body, codec)
	if err != nil {
		return nil, err
	}
	context := captureProjectionContext(target.ProjectionContext, nil)
	fields := make(map[string]contributorField)
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	for _, member := range node.Members() {
		fields[member.Name] = contributorField{member.WireName, member.Visible, member.Presence}
	}
	wire := &WireContext{codec: codec, contract: contract, context: context, fields: fields}
	wire.identity, err = wireContextIdentity(wire)
	return wire, err
}

// ProjectContributor projects declaration into wire: declaration's own
// contract through projection, normalized with the declaration's own captured
// contract rebound to the body's entry for every member the body shares. A
// member the body does not share, such as a sibling's own field, keeps the
// entry its declaration resolves.
func ProjectContributor(wire *WireContext, projection *expr.HTTPBodyProjection, declaration expr.UserType) (*Contributor, error) {
	attribute := expr.ProjectHTTPBodyContributor(declaration, projection)
	contract, err := projectContract(wire, attribute)
	if err != nil {
		return nil, fmt.Errorf("project %s into its HTTP body context: %w", declaration.Name(), err)
	}
	key, err := contributorIdentity(wire, declaration, attribute, contract)
	if err != nil {
		return nil, err
	}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return nil, err
	}
	data := &service.ValueData{Context: context, Occurrence: occurrence}
	request := captureProjectionContext(wire.context, nil)
	value := &transportir.ValueTarget{Source: data, Anchor: data, Codec: wire.codec, ExampleOccurrence: occurrence}
	target := buildValuePlan(value, attribute, request, expr.ValuePlanSchema, &contributorContract{parameter: contract, fields: wire.fields})
	if target.Error != nil {
		return nil, fmt.Errorf("project %s into its HTTP body context: %w", declaration.Name(), target.Error)
	}
	return &Contributor{Attribute: attribute, Target: target, Key: key}, nil
}

// projectContract captures the own contract of the projected contributor and
// rebinds every field the body shares to the body's resolved entry. Every
// field of the contributor must have an entry.
func projectContract(wire *WireContext, attribute *expr.AttributeExpr) (parameterContract, error) {
	own, err := captureParameterContract(attribute, wire.codec)
	if err != nil {
		return parameterContract{}, err
	}
	object := expr.AsObject(attribute.Type)
	switch wire.codec {
	case expr.ValueCodecForm:
		projected := *own.form
		projected.Fields = maps.Clone(own.form.Fields)
		for name := range projected.Fields {
			if shared, ok := wire.contract.form.Fields[name]; ok {
				projected.Fields[name] = shared
			}
		}
		for _, field := range *object {
			if _, ok := projected.Fields[field.Name]; !ok {
				return parameterContract{}, fmt.Errorf("form field %q has no contract", field.Name)
			}
		}
		return parameterContract{form: &projected}, nil
	case expr.ValueCodecMultipart:
		if wire.contract.multipart.Custom {
			return parameterContract{multipart: &expr.HTTPMultipartEncoding{Custom: true}}, nil
		}
		projected := expr.HTTPMultipartEncoding{Fields: make(map[string]expr.HTTPMultipartFieldEncoding)}
		if !own.multipart.Custom {
			projected = *own.multipart
			projected.Fields = maps.Clone(own.multipart.Fields)
		}
		for _, field := range *object {
			if shared, ok := wire.contract.multipart.Fields[field.Name]; ok {
				projected.Fields[field.Name] = shared
			}
			if _, ok := projected.Fields[field.Name]; !ok {
				return parameterContract{}, fmt.Errorf("multipart field %q has no contract", field.Name)
			}
		}
		return parameterContract{multipart: &projected}, nil
	default:
		return parameterContract{}, nil
	}
}

// contributorIdentity serializes the content identity of a projection: the
// declaration, its excluded members and the resolved entry of every field.
func contributorIdentity(wire *WireContext, declaration expr.UserType, attribute *expr.AttributeExpr, contract parameterContract) (string, error) {
	key := contributorKey{Context: wire.identity, Declaration: declaration.ID(), Codec: wire.codec}
	if projection := attribute.HTTPBodyProjection(); projection != nil {
		key.Excluded = slices.Sorted(slices.Values(projection.Excluded))
	}
	if contract.form != nil {
		key.Form = contract.form.Fields
	}
	if contract.multipart != nil {
		key.Multipart, key.Custom = contract.multipart.Fields, contract.multipart.Custom
	}
	encoded, err := json.Marshal(key, json.Deterministic(true))
	if err != nil {
		return "", fmt.Errorf("identify projected %s: %w", declaration.Name(), err)
	}
	return string(encoded), nil
}
