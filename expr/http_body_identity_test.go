package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHTTPScalarBodyCopyIdentity keeps a named union wrapper distinct from
// its branch when transport suffixing gives them the same name.
func TestHTTPScalarBodyCopyIdentity(t *testing.T) {
	for _, c := range []struct{ name, suffix string }{
		{"Other", "RequestBody"},
		{"svc#Other", "RequestBody"},
		{"Other", "Response"},
	} {
		t.Run(c.name+"/"+c.suffix, func(t *testing.T) {
			branch := &UserTypeExpr{TypeName: c.name, AttributeExpr: &AttributeExpr{Type: &Object{}}}
			union := &UserTypeExpr{TypeName: "Choice", AttributeExpr: &AttributeExpr{Type: &Union{
				TypeName: "Choice",
				Values:   []*NamedAttributeExpr{{Name: "Other", Attribute: &AttributeExpr{Type: branch}}},
			}}}
			payload := &AttributeExpr{Type: union}
			svc := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "svc"}}
			endpoint := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "other", Payload: payload}, Service: svc}
			endpoint.initTransportAttributes()
			var body *AttributeExpr
			if c.suffix == "RequestBody" {
				body = scalarHTTPBody(endpoint, payload, "Other"+c.suffix, c.suffix)
			} else {
				response := &HTTPResponseExpr{Headers: NewEmptyMappedAttributeExpr()}
				body = buildHTTPResponseBody("other", payload, response, svc)
			}
			copied := DupAtt(body)
			copiedUnion := AsUnion(copied.Type)
			require.NotNil(t, copiedUnion)
			require.True(t, IsObject(copiedUnion.Values[0].Attribute.Type), "copy replaced the object branch with its union wrapper")
			require.Equal(t, c.name, branch.Name())
		})
	}
}

// TestHTTPBodyCopyNamespace checks that wrapper ownership survives copying and
// that keeping an authored type cannot also keep a same-ID transport wrapper.
func TestHTTPBodyCopyNamespace(t *testing.T) {
	for _, result := range []bool{false, true} {
		name := "user"
		if result {
			name = "result"
		}
		t.Run(name, func(t *testing.T) {
			authored := &UserTypeExpr{TypeName: "Same", UID: "same-id", AttributeExpr: &AttributeExpr{Type: &Object{}}}
			wrapper := authored.Dup(&AttributeExpr{Type: &Object{}}).(*UserTypeExpr)
			wrapper.httpBody = true
			var source, body UserType = authored, wrapper
			if result {
				source = &ResultTypeExpr{UserTypeExpr: authored, Identifier: "application/same"}
				body = &ResultTypeExpr{UserTypeExpr: wrapper, Identifier: "application/same"}
			}
			wrapper.Type = &Object{{Name: "value", Attribute: &AttributeExpr{Type: source}}}
			copied, kept := DupKeeping(body, []UserType{source})
			require.NotSame(t, body, copied)
			require.Same(t, source, AsObject(copied).Attribute("value").Type)
			require.Len(t, kept, 1)
			require.Equal(t, typeCopyKey(body), typeCopyKey(copied.(UserType)))
			require.NotEqual(t, typeCopyKey(source), typeCopyKey(copied.(UserType)))
		})
	}
}

// TestHTTPBodyExamplesSeparateAuthoredIdentity keeps example memoization from
// treating a wrapper and an authored branch with the same ID as recursion.
func TestHTTPBodyExamplesSeparateAuthoredIdentity(t *testing.T) {
	for _, authoredFirst := range []bool{false, true} {
		name := "body first"
		if authoredFirst {
			name = "authored first"
		}
		t.Run(name, func(t *testing.T) {
			branch := &UserTypeExpr{TypeName: "svc#OtherRequestBody", AttributeExpr: &AttributeExpr{
				Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}},
			}}
			body := &UserTypeExpr{TypeName: "OtherRequestBody", UID: branch.ID(), httpBody: true, AttributeExpr: &AttributeExpr{
				Type: &Object{{Name: "branch", Attribute: &AttributeExpr{Type: branch}}},
			}}
			random := &ExampleGenerator{Randomizer: NewDeterministicRandomizer()}
			wantBranch := map[string]any{"value": "abc123"}
			if authoredFirst {
				require.Equal(t, wantBranch, branch.Example(random))
			}
			require.Equal(t, map[string]any{"branch": wantBranch}, body.Example(random))
			require.Equal(t, wantBranch, branch.Example(random))
			seen, ok := random.PreviouslySeen(branch.ID())
			require.True(t, ok, "public cache uses authored identities")
			require.Equal(t, wantBranch, *seen)
			require.Equal(t, body.Example(random), Dup(body).Example(random))
		})
	}
}
