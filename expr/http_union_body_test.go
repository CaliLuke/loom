package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestUnionHTTPBodyBranchesAreSuffixed checks that the branch types of a
// union request or response body are copies renamed with the body suffix,
// like the attribute types of an object body, whether the union is anonymous,
// named or an explicit body attribute, that the service types keep their
// names and that every body keeps the discriminator values of the service
// union.
func TestUnionHTTPBodyBranchesAreSuffixed(t *testing.T) {
	cases := []struct {
		name     string
		named    bool
		explicit bool
	}{
		{"anonymous", false, false},
		{"named", true, false},
		{"explicit-body", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				leaf := Type("Leaf", func() {
					Attribute("name", String)
				})
				other := Type("Other", func() {
					Attribute("count", Int)
				})
				var union any = OneOf(leaf, other)
				if c.named {
					union = Type("U", OneOf(leaf, other))
				}
				Service("svc", func() {
					Method("pick", func() {
						if c.explicit {
							Payload(func() {
								Attribute("q", String)
								Attribute("u", union)
							})
						} else {
							Payload(union)
						}
						Result(union)
						HTTP(func() {
							POST("/")
							if c.explicit {
								Param("q")
								Body("u")
							}
						})
					})
				})
			})
			endpoint := root.API.HTTP.Service("svc").Endpoint("pick")
			requestBranches := []string{"LeafRequestBody", "OtherRequestBody"}
			if c.explicit {
				// The explicit body is renamed once when it is declared and
				// once when the endpoint is finalized.
				requestBranches = []string{"LeafRequestBodyRequestBody", "OtherRequestBodyRequestBody"}
			}
			assert.Equal(t, requestBranches, unionBranchNames(t, endpoint.Body))
			assert.Equal(t, []string{"LeafResponse", "OtherResponse"}, unionBranchNames(t, endpoint.Responses[0].Body))
			result := endpoint.MethodExpr.Result
			assert.Equal(t, []string{"Leaf", "Other"}, unionBranchNames(t, result))
			for _, body := range []*expr.AttributeExpr{endpoint.Body, endpoint.Responses[0].Body, result} {
				assert.Equal(t, []string{"Leaf", "Other"}, unionBranchTags(t, body), "the discriminator must not change")
			}
		})
	}
}

// unionBranchNames returns the type names of the branches of the union type
// of att, unwrapping a user type.
func unionBranchNames(t *testing.T, att *expr.AttributeExpr) []string {
	t.Helper()
	union := expr.AsUnion(att.Type)
	require.NotNil(t, union, "%T is not a union", att.Type)
	names := make([]string, 0, len(union.Values))
	for _, branch := range union.Values {
		names = append(names, branch.Attribute.Type.Name())
	}
	return names
}

// unionBranchTags returns the discriminator values of the branches of the
// union type of att, unwrapping a user type.
func unionBranchTags(t *testing.T, att *expr.AttributeExpr) []string {
	t.Helper()
	union := expr.AsUnion(att.Type)
	require.NotNil(t, union, "%T is not a union", att.Type)
	tags := make([]string, 0, len(union.Values))
	for _, branch := range union.Values {
		tags = append(tags, expr.UnionVariantTag(branch))
	}
	return tags
}
