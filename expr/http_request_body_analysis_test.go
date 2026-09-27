package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHTTPRequestBodyAnalysisPreservesSource checks that validation may derive
// a body repeatedly without consuming the authored type graph or renaming it.
func TestHTTPRequestBodyAnalysisPreservesSource(t *testing.T) {
	for _, name := range []string{"Item", "ItemRequestBody"} {
		for _, shape := range []string{"array", "map", "object"} {
			t.Run(name+"/"+shape, func(t *testing.T) {
				item := &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: &Object{}}}
				var bodyType DataType
				switch shape {
				case "array":
					bodyType = &Array{ElemType: &AttributeExpr{Type: item}}
				case "map":
					bodyType = &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: item}}
				case "object":
					bodyType = &Object{&NamedAttributeExpr{Name: "item", Attribute: &AttributeExpr{Type: item}}}
				}
				authored := &AttributeExpr{Type: bodyType, Meta: MetaExpr{"http:body": {"items"}}}
				endpoint := &HTTPEndpointExpr{
					MethodExpr: &MethodExpr{Name: "Send"},
					Service:    &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "Items"}},
					Body:       authored,
				}
				for range 3 {
					derived := httpRequestBody(endpoint)
					require.Same(t, authored, endpoint.Body, "analysis must not replace the authored body")
					require.Equal(t, name, item.Name(), "analysis must not rename the service type")
					names := []string{}
					walk(derived.Type, func(ut UserType) {
						names = append(names, ut.Name())
					})
					require.Contains(t, names, name+"RequestBody")
					require.NotContains(t, names, name+"RequestBodyRequestBody")
				}
			})
		}
	}
}

// TestHTTPRequestBodyPreparationPreservesInheritedExamples checks that the
// prepared body retains examples before type finalization removes its bases.
func TestHTTPRequestBodyPreparationPreservesInheritedExamples(t *testing.T) {
	base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{
		Type:         &Object{},
		UserExamples: []*ExampleExpr{{Summary: "authored", Value: map[string]any{}}},
	}}
	wrapper := &UserTypeExpr{TypeName: "Wrapper", AttributeExpr: &AttributeExpr{
		Type: &Object{}, Bases: []DataType{base},
	}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{Name: "Send"},
		Service:    &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "Items"}},
		Body:       &AttributeExpr{Type: wrapper},
	}
	endpoint.initTransportAttributes()
	wrapper.Finalize()
	require.Empty(t, wrapper.Bases)
	derived := httpRequestBody(endpoint)
	require.Len(t, derived.UserExamples, 1)
	require.Equal(t, "authored", derived.UserExamples[0].Summary)
}
