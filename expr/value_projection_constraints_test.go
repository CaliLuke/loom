package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueProjectionOccurrenceConstraints(t *testing.T) {
	minimum := 2
	shapes := []struct {
		name       string
		typ        DataType
		raw, other any
	}{
		{"string", String, "x", "other"},
		{"bytes", Bytes, []byte("x"), []byte("other")},
		{"array", &Array{ElemType: &AttributeExpr{Type: String}}, []string{"x"}, []string{"other"}},
		{"map", &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}, map[string]string{"x": "v"}, map[string]string{"other": "v"}},
	}
	for _, shape := range shapes {
		for _, alias := range []bool{false, true} {
			for _, enumeration := range []bool{false, true} {
				name := shape.name
				if alias {
					name += "/alias"
				} else {
					name += "/native"
				}
				if enumeration {
					name += "/enum"
				} else {
					name += "/length"
				}
				t.Run(name, func(t *testing.T) {
					typ := shape.typ
					if alias {
						typ = &UserTypeExpr{TypeName: "Constrained", AttributeExpr: &AttributeExpr{Type: typ}}
					}
					source := &AttributeExpr{Type: typ}
					target := DupAtt(source)
					target.Validation = &ValidationExpr{MinLength: &minimum}
					if enumeration {
						target.Validation = &ValidationExpr{Values: []any{shape.other}}
					}
					for _, use := range []ValuePlanUse{ValuePlanDocumentation, ValuePlanRuntime} {
						checkProjectionOccurrenceRule(t, source, target, shape.raw, use, ProjectionUnrepresentable)
					}
				})
			}
		}
	}
	for _, alias := range []bool{false, true} {
		for _, enumeration := range []bool{false, true} {
			name := "key"
			if alias {
				name += "/alias"
			} else {
				name += "/native"
			}
			if enumeration {
				name += "/enum"
			} else {
				name += "/length"
			}
			t.Run(name, func(t *testing.T) {
				var typ DataType = String
				if alias {
					typ = &UserTypeExpr{TypeName: "Key", AttributeExpr: &AttributeExpr{Type: String}}
				}
				source := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: typ}, ElemType: &AttributeExpr{Type: String}}}
				target := DupAtt(source)
				key := target.Type.(*Map).KeyType
				key.Validation = &ValidationExpr{MinLength: &minimum}
				if enumeration {
					key.Validation = &ValidationExpr{Values: []any{"allowed"}}
				}
				raw := map[string]string{"x": "value"}
				// Emitted map schemas do not constrain keys. Runtime constraints must
				// still reject; schema admission cannot stand in for decoder admission.
				checkProjectionOccurrenceRule(t, source, target, raw, ValuePlanDocumentation, ProjectionEmitted)
				checkProjectionOccurrenceRule(t, source, target, raw, ValuePlanRuntime, ProjectionUnrepresentable)
			})
		}
	}
}

func checkProjectionOccurrenceRule(t *testing.T, source, target *AttributeExpr, raw any, use ValuePlanUse, want ProjectionOutcome) {
	t.Helper()
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: use})
	require.NoError(t, err)
	require.Equal(t, want, context.ProjectJSON(resolved, plan).Outcome(), "use %v", use)
}

func TestValueProjectionKeyEnumDomain(t *testing.T) {
	cases := []struct {
		name         string
		typ          DataType
		allowed, raw any
		accepted     bool
	}{
		{"String allowed", String, "allowed", "allowed", true},
		{"String rejected", String, "allowed", "other", false},
		{"Any string allowed", Any, "allowed", "allowed", true},
		{"Any string rejected", Any, "allowed", "other", false},
		{"Any integer spelling", Any, int64(1), "1", true},
		{"Any boolean spelling", Any, true, "true", true},
		{"Any heterogeneous wrong spelling", Any, true, "1", false},
		{"Any float width same", Any, float32(.1), float32(.1), true},
		{"Any float width distinct name", Any, float32(.1), float64(float32(.1)), false},
		{"Int64 normalized integer", Int64, int32(1), int64(1), true},
		{"Int64 rejected", Int64, int32(1), int64(2), false},
		{"Boolean allowed", Boolean, true, true, true},
		{"Boolean rejected", Boolean, true, false, false},
		{"Float32 allowed", Float32, float32(.1), float64(float32(.1)), true},
		{"Float32 rejected", Float32, float32(.1), float32(.2), false},
		{"Float64 allowed", Float64, float64(.1), float64(.1), true},
	}
	for _, tc := range cases {
		for _, alias := range []bool{false, true} {
			name := tc.name + "/native"
			if alias {
				name = tc.name + "/alias"
			}
			t.Run(name, func(t *testing.T) {
				typ := tc.typ
				if alias {
					typ = &UserTypeExpr{TypeName: "EnumKey", AttributeExpr: &AttributeExpr{Type: typ}}
				}
				source := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: typ}, ElemType: &AttributeExpr{Type: String}}}
				target := DupAtt(source)
				target.Type.(*Map).KeyType.Validation = &ValidationExpr{Values: []any{tc.allowed}}
				want := ProjectionUnrepresentable
				if tc.accepted {
					want = ProjectionEmitted
				}
				checkProjectionOccurrenceRule(t, source, target, map[any]string{tc.raw: "value"}, ValuePlanRuntime, want)
			})
		}
	}
}
