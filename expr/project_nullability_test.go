package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectRetainsDeclarationNullability(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, selected := range []string{DefaultView, "tiny"} {
			t.Run(fmt.Sprintf("nullable=%t/view=%s", nullable, selected), func(t *testing.T) {
				source := nullableProjectionResult(nullable)
				plan := newValueViewPlan()
				projected, err := plan.project(source, selected)
				require.NoError(t, err)
				again, err := plan.project(source, selected)
				require.NoError(t, err)
				require.Same(t, projected, again)
				require.NotSame(t, source, projected)
				require.Equal(t, nullable, IsNullable(&AttributeExpr{Type: projected}))
				require.Equal(t, nullable, source.Nullable)
				require.Len(t, *AsObject(source.Type), 2)
				projected.Nullable = !nullable
				require.Equal(t, nullable, source.Nullable)
			})
		}
	}
}

func TestProjectNullableOccurrencesReuseDeclarations(t *testing.T) {
	for _, intrinsic := range []bool{false, true} {
		for _, reversed := range []bool{false, true} {
			t.Run(fmt.Sprintf("intrinsic=%t/reversed=%t", intrinsic, reversed), func(t *testing.T) {
				child := nullableProjectionResult(intrinsic)
				names := []string{"plain", "nullable", "small"}
				if reversed {
					names[0], names[1] = names[1], names[0]
				}
				parent := resultType(names[0], child, names[1], child, "small", child,
					view(DefaultView, names[0], child, names[1], child, "small:tiny", child))
				AsObject(parent.Type).Attribute("nullable").Nullable = true
				projected, err := Project(parent, DefaultView)
				require.NoError(t, err)
				fields := AsObject(projected.Type)
				plain, local, small := fields.Attribute("plain"), fields.Attribute("nullable"), fields.Attribute("small")
				require.NotSame(t, plain, local)
				require.Same(t, plain.Type, local.Type)
				require.NotSame(t, plain.Type, small.Type)
				require.False(t, plain.Nullable)
				require.True(t, local.Nullable)
				require.Equal(t, intrinsic, plain.Type.(*ResultTypeExpr).Nullable)
				require.Equal(t, intrinsic, small.Type.(*ResultTypeExpr).Nullable)
				require.Len(t, *AsObject(plain.Type), 2)
				require.Len(t, *AsObject(small.Type), 1)
				require.Equal(t, intrinsic, child.Nullable)
				require.Len(t, *AsObject(child.Type), 2)
			})
		}
	}
}

func TestProjectNullableCollectionContract(t *testing.T) {
	for _, root := range []bool{false, true} {
		for _, intrinsic := range []bool{false, true} {
			for _, occurrence := range []bool{false, true} {
				t.Run(fmt.Sprintf("root=%t/intrinsic=%t/occurrence=%t", root, intrinsic, occurrence), func(t *testing.T) {
					child := nullableProjectionResult(intrinsic)
					source := collection(child)
					source.Nullable = root
					AsArray(source.Type).ElemType.Nullable = occurrence
					projected, err := Project(source, "tiny")
					require.NoError(t, err)
					element := AsArray(projected.Type).ElemType
					require.Equal(t, root, projected.Nullable)
					require.Equal(t, occurrence, element.Nullable)
					require.Equal(t, intrinsic, element.Type.(*ResultTypeExpr).Nullable)
					require.Equal(t, intrinsic || occurrence, IsNullable(element))
					require.Len(t, *AsObject(element.Type), 1)
					require.Equal(t, intrinsic, child.Nullable)
					require.Equal(t, occurrence, AsArray(source.Type).ElemType.Nullable)
				})
			}
		}
	}
}

func TestProjectNullableSemanticValues(t *testing.T) {
	for _, position := range []string{"root", "nested", "collection", "collection-root"} {
		for _, nullable := range []bool{false, true} {
			for _, null := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/nullable=%t/null=%t", position, nullable, null), func(t *testing.T) {
					child := nullableProjectionResult(nullable)
					var raw any = map[string]any{"text": "kept", "extra": "hidden"}
					wire := `{"text":"kept"}`
					if null {
						raw, wire = nil, `null`
					}
					source := child
					switch position {
					case "nested":
						source = resultType("child", child, view("tiny", "child:tiny", child))
						raw = map[string]any{"child": raw}
						wire = `{"child":` + wire + `}`
					case "collection", "collection-root":
						source = collection(child)
						raw = []any{raw}
						wire = `[` + wire + `]`
						if position == "collection-root" {
							source.Nullable = nullable
							if null {
								raw, wire = nil, `null`
							}
						}
					}
					target, err := Project(source, "tiny")
					require.NoError(t, err)
					context := NewValueContext()
					occurrence, err := context.NewOccurrence(source.AttributeExpr)
					require.NoError(t, err)
					resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
					if null && !nullable {
						require.NotEqual(t, ValueResolved, resolved.Outcome())
						return
					}
					require.Equal(t, ValueResolved, resolved.Outcome(), "%v", resolved.Diagnostics())
					plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target.AttributeExpr, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
					require.NoError(t, err)
					projected := context.ProjectJSON(resolved, plan)
					require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
					actual, present := projected.JSON()
					require.True(t, present)
					require.JSONEq(t, wire, string(actual))
				})
			}
		}
	}
}

func nullableProjectionResult(nullable bool) *ResultTypeExpr {
	result := resultType("text", String, "extra", String,
		view(DefaultView, "text", String, "extra", String), view("tiny", "text", String))
	result.Nullable = nullable
	return result
}
