package valuecontract

import (
	"encoding/json/jsontext"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/enumvalue"
)

type productionProjectionCase struct {
	name        string
	attribute   *expr.AttributeExpr
	raw         any
	fields      map[string]expr.ValueFieldPresence
	hidden      map[string]bool
	untagged    bool
	selected    string
	targetEnums []any
}

// Target contracts are translated from the authored fixture graph and explicit
// emitter policies, never from Resolve, ProjectJSON, or their output wire.
func productionProjectionTargets(t *testing.T, graph *productionGraph, tc productionProjectionCase) []any {
	t.Helper()
	targets := make([]any, 0, len(graph.declarations))
	for _, raw := range graph.declarations {
		declaration := raw.(map[string]any)
		var schemaEnums any
		enumeration := declaration["enumeration"]
		root := declaration["identity"] == graph.declarations[0].(map[string]any)["identity"]
		if enumeration != nil || root && tc.targetEnums != nil {
			require.Equal(t, graph.declarations[0], declaration, "this corpus authors enums at the semantic root")
			values := tc.targetEnums
			if values == nil {
				values = tc.attribute.Validation.Values
			}
			semantic := make([]any, 0, len(values))
			for _, value := range values {
				semantic = append(semantic, graph.literal(tc.attribute, value))
			}
			enumeration = semantic
			wires := make([]any, 0, len(values))
			for _, value := range values {
				normalized := enumvalue.Normalize(tc.attribute, value)
				wires = append(wires, productionJSONWire(t, jsontext.Value(productionJSON(t, normalized))))
			}
			schemaEnums = wires
		}
		target := declaration["contract"]
		if constructor, ok := target.(map[string]any); ok {
			for kind, rawFields := range constructor {
				fields := maps.Clone(rawFields.(map[string]any))
				switch kind {
				case "scalar":
					fields["encoding"] = "json"
				case "object":
					members := []any{}
					for _, rawMember := range fields["members"].([]any) {
						member := rawMember.(map[string]any)
						name := member["sourceName"].(string)
						if tc.hidden[name] {
							continue
						}
						presence := "explicit"
						if tc.fields[name] == expr.ValueFieldOmitEmpty {
							presence = "omitEmpty"
						}
						members = append(members, map[string]any{"identity": member["identity"], "wireName": member["wireAlias"], "required": member["required"], "presence": presence, "child": member["child"]})
					}
					fields = map[string]any{"members": members, "preserveAdditional": false}
				case "union":
					alternatives := make([]any, 0, len(fields["alternatives"].([]any)))
					for _, rawBranch := range fields["alternatives"].([]any) {
						branch := rawBranch.(map[string]any)
						alternatives = append(alternatives, map[string]any{"identity": branch["identity"], "wireName": branch["name"], "child": branch["child"]})
					}
					fields["alternatives"] = alternatives
					fields["style"] = any(referenceConstructor("tagged", map[string]any{"tagKey": "type", "valueKey": "value"}))
					if tc.untagged {
						fields["style"] = "untagged"
					}
				}
				target = referenceConstructor(kind, fields)
			}
		}
		targets = append(targets, map[string]any{"identity": declaration["identity"], "expansionRank": declaration["expansionRank"], "target": target, "enumeration": enumeration, "schemaEnumeration": schemaEnums, "schemaAllowsUnknown": true, "decoderRejectsUnknown": false})
	}
	return targets
}

func productionProjectionPlan(tc productionProjectionCase, use expr.ValuePlanUse) expr.ValuePlanRequest {
	request := expr.ValuePlanRequest{Target: tc.attribute, Codec: expr.ValueCodecJSON, Use: use}
	seen := make(map[*expr.AttributeExpr]bool)
	var visit func(*expr.AttributeExpr)
	visit = func(attribute *expr.AttributeExpr) {
		if seen[attribute] {
			return
		}
		seen[attribute] = true
		switch typ := attribute.Type.(type) {
		case expr.UserType:
			visit(typ.Attribute())
		case *expr.Array:
			visit(typ.ElemType)
		case *expr.Map:
			visit(typ.ElemType)
		case *expr.Union:
			request.Containers = append(request.Containers, expr.ValueContainerPolicy{Target: attribute})
			for _, branch := range typ.Values {
				visit(branch.Attribute)
			}
		case *expr.Object:
			request.Containers = append(request.Containers, expr.ValueContainerPolicy{Target: attribute})
			for _, field := range *typ {
				presence := tc.fields[field.Name]
				if presence == 0 {
					presence = expr.ValueFieldRetain
				}
				request.Fields = append(request.Fields, expr.ValueFieldPolicy{Parent: attribute, Target: field.Attribute, Name: field.Name, WireName: expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute), Visible: !tc.hidden[field.Name], Required: attribute.IsRequired(field.Name), Presence: presence})
				visit(field.Attribute)
			}
		}
	}
	if tc.targetEnums != nil {
		request.Target = expr.DupAtt(tc.attribute)
		if request.Target.Validation == nil {
			request.Target.Validation = &expr.ValidationExpr{}
		}
		request.Target.Validation.Values = tc.targetEnums
	}
	if tc.selected != "" {
		request.Target = tc.attribute.Find(tc.selected)
		request.Selection = []string{tc.selected}
	}
	visit(request.Target)
	return request
}

func productionJSONWire(t *testing.T, raw jsontext.Value) any {
	t.Helper()
	switch raw.Kind() {
	case 'n':
		return "null"
	case 't', 'f':
		return referenceConstructor("boolean", map[string]any{"value": referenceDecode[bool](t, raw)})
	case '"':
		return referenceConstructor("text", map[string]any{"value": referenceDecode[string](t, raw)})
	case '0':
		return referenceConstructor("number", map[string]any{"text": string(raw)})
	case '[':
		items := referenceDecode[[]jsontext.Value](t, raw)
		values := make([]any, 0, len(items))
		for _, item := range items {
			values = append(values, productionJSONWire(t, item))
		}
		return referenceConstructor("array", map[string]any{"items": values})
	case '{':
		object := referenceDecode[map[string]jsontext.Value](t, raw)
		names := make([]string, 0, len(object))
		for name := range object {
			names = append(names, name)
		}
		slices.Sort(names)
		fields := make([]any, 0, len(names))
		for _, name := range names {
			fields = append(fields, []any{name, productionJSONWire(t, object[name])})
		}
		return referenceConstructor("object", map[string]any{"fields": fields})
	default:
		t.Fatalf("invalid wire %s", raw)
		return nil
	}
}

// Convert the reference Wire AST into JSON syntax. Pair-list object ordering is
// irrelevant; exact number lexemes and strings remain distinct.
func productionReferenceWire(t *testing.T, raw jsontext.Value) jsontext.Value {
	t.Helper()
	if string(raw) == `"null"` {
		return jsontext.Value("null")
	}
	constructor := referenceDecode[map[string]jsontext.Value](t, raw)
	require.Len(t, constructor, 1)
	for kind, payload := range constructor {
		fields := referenceDecode[map[string]jsontext.Value](t, payload)
		switch kind {
		case "boolean", "text":
			return fields["value"]
		case "number":
			return jsontext.Value(referenceDecode[string](t, fields["text"]))
		case "array":
			children := referenceDecode[[]jsontext.Value](t, fields["items"])
			values := make([]jsontext.Value, 0, len(children))
			for _, child := range children {
				values = append(values, productionReferenceWire(t, child))
			}
			return jsontext.Value(productionJSON(t, values))
		case "object":
			children := referenceDecode[[][2]jsontext.Value](t, fields["fields"])
			values := make(map[string]jsontext.Value, len(children))
			for _, child := range children {
				name := referenceDecode[string](t, child[0])
				_, duplicate := values[name]
				require.False(t, duplicate)
				values[name] = productionReferenceWire(t, child[1])
			}
			return jsontext.Value(productionJSON(t, values))
		default:
			t.Fatalf("unsupported reference wire %s", kind)
		}
	}
	return nil
}

func productionProjectionRoot(t *testing.T, graph *productionGraph, occurrence expr.ValueOccurrence, targets []any, selected string) ([]any, referenceIdentity) {
	t.Helper()
	root := graph.identity(occurrence.ID())
	if selected == "" {
		return targets, root
	}
	for _, member := range occurrence.Members() {
		if member.Name != selected {
			continue
		}
		child := graph.identity(member.Occurrence.ID())
		rank := uint64(0)
		for _, raw := range targets {
			target := raw.(map[string]any)
			if target["identity"] == child {
				rank = target["expansionRank"].(uint64)
			}
		}
		identity := referenceIdentity{Occurrence: graph.next + 1, Declaration: graph.next + 1}
		target := map[string]any{"identity": identity, "expansionRank": rank + 1, "target": referenceConstructor("select", map[string]any{"field": graph.identity(member.ID), "child": child}), "enumeration": nil, "schemaEnumeration": nil, "schemaAllowsUnknown": true, "decoderRejectsUnknown": false}
		return append(targets, target), identity
	}
	t.Fatalf("selected fixture field %q absent", selected)
	return nil, referenceIdentity{}
}
