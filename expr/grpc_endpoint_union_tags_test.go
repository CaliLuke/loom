package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRPCTagsRejectsConstructorUnionBranchTagCollidingWithSibling(t *testing.T) {
	fields := &Object{
		{
			Name: "id",
			Attribute: &AttributeExpr{
				Type: String,
				Meta: MetaExpr{"rpc:tag": []string{"1"}},
			},
		},
		{
			Name: "choice",
			Attribute: &AttributeExpr{
				Type: &Union{
					TypeName: "Choice",
					Values: []*NamedAttributeExpr{
						{
							Name: "Text",
							Attribute: &AttributeExpr{
								Type: String,
								Meta: MetaExpr{"rpc:tag": []string{"1"}},
							},
						},
						{
							Name: "JSON",
							Attribute: &AttributeExpr{
								Type: String,
								Meta: MetaExpr{"rpc:tag": []string{"2"}},
							},
						},
					},
				},
			},
		},
	}

	verr := validateRPCTags(fields, grpcEndpointForTagValidationTest())
	require.EqualError(t, verr, `service "Service" gRPC endpoint "Method": field number 1 in attribute "choice.Text" already exists for attribute "id"`)
}

func TestValidateRPCTagsRejectsDuplicateConstructorUnionBranchTags(t *testing.T) {
	fields := &Object{
		{
			Name: "choice",
			Attribute: &AttributeExpr{
				Type: &Union{
					TypeName: "Choice",
					Values: []*NamedAttributeExpr{
						{
							Name: "Text",
							Attribute: &AttributeExpr{
								Type: String,
								Meta: MetaExpr{"rpc:tag": []string{"1"}},
							},
						},
						{
							Name: "JSON",
							Attribute: &AttributeExpr{
								Type: String,
								Meta: MetaExpr{"rpc:tag": []string{"1"}},
							},
						},
					},
				},
			},
		},
	}

	verr := validateRPCTags(fields, grpcEndpointForTagValidationTest())
	require.EqualError(t, verr, `service "Service" gRPC endpoint "Method": field number 1 in attribute "choice.JSON" already exists for attribute "choice.Text"`)
}

func grpcEndpointForTagValidationTest() *GRPCEndpointExpr {
	service := &ServiceExpr{Name: "Service"}
	return &GRPCEndpointExpr{
		MethodExpr: &MethodExpr{Name: "Method"},
		Service:    &GRPCServiceExpr{ServiceExpr: service},
	}
}

// TestUnionFieldTags checks that union branches keep their own field numbers
// and that branches without one are numbered consecutively from the field
// number of the union attribute.
func TestUnionFieldTags(t *testing.T) {
	branch := func(name string, tag ...string) *NamedAttributeExpr {
		att := &AttributeExpr{Type: String}
		if len(tag) > 0 {
			att.Meta = MetaExpr{"rpc:tag": tag}
		}
		return &NamedAttributeExpr{Name: name, Attribute: att}
	}
	union := func(tag string, values ...*NamedAttributeExpr) *AttributeExpr {
		att := &AttributeExpr{Type: &Union{TypeName: "U", Values: values}}
		if tag != "" {
			att.Meta = MetaExpr{"rpc:tag": []string{tag}}
		}
		return att
	}
	cases := []struct {
		name string
		att  *AttributeExpr
		want []string
	}{
		{"nil", nil, nil},
		{"not a union", &AttributeExpr{Type: String, Meta: MetaExpr{"rpc:tag": []string{"1"}}}, nil},
		{"block branches", union("", branch("a", "4"), branch("b", "7")), []string{"4", "7"}},
		{"constructor branches", union("2", branch("a"), branch("b"), branch("c")), []string{"2", "3", "4"}},
		{"branch number wins", union("2", branch("a", "9"), branch("b")), []string{"9", "3"}},
		{"no numbers", union("", branch("a"), branch("b", "5")), []string{"", "5"}},
		{"invalid field number", union("x", branch("a")), []string{""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, c.att.UnionFieldTags())
		})
	}
}
