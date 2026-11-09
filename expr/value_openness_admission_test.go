package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamedObjectValueAdmission(t *testing.T) {
	for _, policy := range []ObjectOpenness{ObjectOpennessUnset, ObjectOpen, ObjectClosed} {
		for _, depth := range []int{0, 1, 3} {
			for _, role := range []ValueRole{ValueRoleExample, ValueRoleEnum, ValueRoleDefault} {
				t.Run(fmt.Sprintf("policy=%d/aliases=%d/role=%d", policy, depth, role), func(t *testing.T) {
					attribute := &AttributeExpr{Type: &Object{{Name: "street", Attribute: &AttributeExpr{Type: String}}}}
					if policy == ObjectClosed {
						attribute.CloseObject()
					} else if policy == ObjectOpen {
						attribute.Meta = MetaExpr{objectOpennessMeta: {"true"}}
					}
					for index := range depth {
						attribute = &AttributeExpr{Type: &UserTypeExpr{TypeName: fmt.Sprintf("Alias%d", index), AttributeExpr: attribute}}
					}
					context := NewValueContext()
					occurrence, err := context.NewOccurrence(attribute)
					require.NoError(t, err)
					for _, extra := range []bool{false, true} {
						raw := map[string]any{"street": "x"}
						if extra {
							raw["extra"] = 1
						}
						result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), role)
						want := ValueResolved
						if extra && policy == ObjectClosed {
							want = ValueInvalid
						}
						require.Equal(t, want, result.Outcome(), "%v", result.Diagnostics())
					}
				})
			}
		}
	}
}

func TestExampleOpennessDefersOpaqueCodecs(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{{Name: "id", Attribute: &AttributeExpr{Type: String}}}}
	attribute.CloseObject()
	attribute.UserExamples = []*ExampleExpr{{Value: collisionCodecObject{Value: "codec-owned"}}}
	collisionCompatibilityCodecCalls = 0
	require.Empty(t, attribute.validateExamples("", attribute).Errors)
	require.Zero(t, collisionCompatibilityCodecCalls)
}
