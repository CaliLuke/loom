package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestPreparedEnumProvenanceConstructors(t *testing.T) {
	for _, variant := range []string{"plain", "alias", "local required"} {
		for _, mode := range []string{"source", "copy", "mapped", "remapped", "request", "response", "explicit request", "explicit response", "streaming"} {
			name := mode
			name += "/" + variant
			t.Run(name, func(t *testing.T) {
				source := preparedEnumSource(t, variant == "alias")
				if variant == "local required" {
					source.Validation = &ValidationExpr{Required: []string{"n"}}
				}
				service := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "numbers"}}
				endpoint := &HTTPEndpointExpr{
					MethodExpr: &MethodExpr{Name: "read", Payload: source, StreamingPayload: source, Stream: ClientStreamKind},
					Service:    service, Headers: NewEmptyMappedAttributeExpr(), Cookies: NewEmptyMappedAttributeExpr(), Params: NewEmptyMappedAttributeExpr(),
					wireBindings: &httpWireBindingPlan{bodyExclusions: make(map[string]bool)},
				}
				var target *AttributeExpr
				switch mode {
				case "source":
					target = source
				case "copy":
					target = DupAtt(source)
				case "mapped":
					target = NewMappedAttributeExpr(source).AttributeExpr
				case "remapped":
					mapped := NewMappedAttributeExpr(source)
					mapped.Remap()
					target = DupMappedAtt(mapped).Attribute()
				case "request":
					target = httpRequestBody(endpoint)
				case "response":
					target = buildHTTPResponseBody("read", source, &HTTPResponseExpr{Headers: NewEmptyMappedAttributeExpr()}, service)
				case "explicit request":
					target = explicitHTTPRequestBody(source, "ReadRequestBody", "RequestBody", "numbers#read")
				case "explicit response":
					target = explicitHTTPResponseBody(source, "ReadResponseBody", "ResponseBody", service)
				case "streaming":
					target = httpStreamingBody(endpoint)
				}
				assertPreparedEnumContract(t, target)
			})
		}
	}
}

func preparedEnumSource(t *testing.T, alias bool) *AttributeExpr {
	t.Helper()
	eval.Reset()
	base := &UserTypeExpr{TypeName: "EnumBase", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "n", Attribute: &AttributeExpr{Type: Int}}},
		Validation: &ValidationExpr{Values: []any{map[string]any{"n": 1}, map[string]any{"n": 2}}},
	}}
	refined := &UserTypeExpr{TypeName: "EnumRefined", AttributeExpr: &AttributeExpr{
		Type:  &Object{{Name: "n", Attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{Minimum: boundForTest(2)}}}},
		Bases: []DataType{base},
	}}
	root := &RootExpr{Types: []UserType{base, refined}}
	root.composeDesign()
	require.Empty(t, eval.Context.Errors)
	var typ DataType = refined
	if alias {
		typ = &UserTypeExpr{TypeName: "EnumAlias", AttributeExpr: &AttributeExpr{Type: refined}}
	}
	return &AttributeExpr{Type: typ}
}

func assertPreparedEnumContract(t *testing.T, target *AttributeExpr) {
	t.Helper()
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(target)
	require.NoError(t, err)
	require.Equal(t, "EnumBase", occurrence.EffectiveConstraints().EnumSource().Declaration)
	for _, n := range []int{1, 2, 3} {
		result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"n": n}}), ValueRoleDefault)
		require.Equal(t, n == 2, result.Outcome() == ValueResolved, "n=%d outcome=%v diagnostics=%v", n, result.Outcome(), result.Diagnostics())
	}
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
	require.NoError(t, err)
	require.Len(t, annotations.Enums, 1)
	require.Len(t, annotations.Enums[0], 2)
	for index, member := range annotations.Enums[0] {
		require.Equal(t, "EnumBase", member.Declaration().Declaration)
		wire, emitted := member.Projection().JSON()
		require.True(t, emitted)
		require.JSONEq(t, fmt.Sprintf(`{"n":%d}`, index+1), string(wire))
	}
}

// This test-only construction compares representations; it is not an alternate
// production constructor. The retained source edge must survive flattening.
func TestPreparedEnumProvenanceRepresentationCandidates(t *testing.T) {
	for _, mode := range []string{"lowered only", "retained declaration", "copied wrapper", "alias wrapper"} {
		t.Run(mode, func(t *testing.T) {
			source := preparedEnumSource(t, true)
			constraints, err := EffectiveConstraintsFor(source)
			require.NoError(t, err)
			mapped := NewMappedAttributeExpr(source).AttributeExpr
			mapped.Validation = constraints.Validation().Lowered()
			mapped.constraintParents, mapped.constraintLocal = nil, nil
			if mode != "lowered only" {
				mapped.constraintLocal = &AttributeExpr{Type: mapped.Type}
				mapped.constraintParents = []*AttributeExpr{DupAtt(source)}
			}
			target := mapped
			if mode == "copied wrapper" || mode == "alias wrapper" {
				target = &AttributeExpr{Type: &UserTypeExpr{TypeName: "PreparedBody", AttributeExpr: mapped}}
				if mode == "copied wrapper" {
					target.Validation = mapped.Validation.Dup()
				}
			}
			if mode == "lowered only" {
				_, err := NewValueContext().NewOccurrence(target)
				require.ErrorContains(t, err, "violates its declared type")
				return
			}
			assertPreparedEnumContract(t, target)
		})
	}
}

func TestPreparedEnumProvenanceMappedProjection(t *testing.T) {
	source := preparedEnumSource(t, false)
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	mapped := NewMappedAttributeExpr(source)
	mapped.Map("wire_n", "n")
	target := mapped.Attribute()
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
	require.NoError(t, err)
	for index, value := range annotations.Enums[0] {
		actual, ok := value.Projection().JSON()
		require.True(t, ok)
		require.JSONEq(t, fmt.Sprintf(`{"wire_n":%d}`, index+1), string(actual))
	}
}
