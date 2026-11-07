package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPBodyTraversalUsesSourceIdentity(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			first := &UserTypeExpr{TypeName: "Leaf", UID: "leaf", AttributeExpr: &AttributeExpr{
				Type: &Object{}, Meta: MetaExpr{"struct:pkg:path": {"types/leaf"}},
			}}
			second := Dup(first).(*UserTypeExpr)
			AsObject(first).Set("next", &AttributeExpr{Type: first})
			AsObject(second).Set("next", &AttributeExpr{Type: second})
			fields := &Object{
				{Name: "value", Attribute: &AttributeExpr{Type: first}},
				{Name: "items", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: second}}}},
				{Name: "again", Attribute: &AttributeExpr{Type: first}},
			}
			if reverse {
				(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
			}
			original := &AttributeExpr{Type: fields, AdditionalAttributes: &AttributeExpr{Type: second}}
			body := DupAtt(original)
			RemovePkgPath(body)
			appendSuffix(body, "RequestBody")
			visits := make(map[UserType]int)
			walkUserTypes(body, func(ut UserType) {
				visits[ut]++
				require.NotContains(t, ut.Attribute().Meta, "struct:pkg:path")
				require.Equal(t, "LeafRequestBody", ut.Name())
			})
			require.Len(t, visits, 2, "distinct same-ID nodes each need a transport projection")
			for _, count := range visits {
				require.Equal(t, 1, count, "shared and recursive edges must not repeat mutation")
			}
			for _, ut := range []UserType{first, second} {
				require.Equal(t, "Leaf", ut.Name())
				require.Equal(t, []string{"types/leaf"}, ut.Attribute().Meta["struct:pkg:path"])
			}
		})
	}
}
