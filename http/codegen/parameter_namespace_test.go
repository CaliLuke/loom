package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestParameterNamespaceUsesBindingIdentity(t *testing.T) {
	for _, name := range []string{"values", "credential sid"} {
		t.Run(name, func(t *testing.T) {
			parameter := &transportir.Parameter{
				Name:  name,
				In:    "query",
				Codec: httpencoding.Codec{Style: httpencoding.Form, Shape: httpencoding.ScalarMap, Explode: true},
				Keys:  httpencoding.KeySet{Residual: true},
			}
			peer := &transportir.Parameter{Name: name, Keys: httpencoding.KeySet{Literals: []string{"fixed"}}}
			builder := &payloadBuilder{endpointIR: &transportir.Endpoint{Security: &transportir.Security{
				CredentialBindings: []expr.HTTPCredentialBinding{{Name: "sid", Location: "query"}},
			}}}
			element := &Element{AttributeData: &AttributeData{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}}
			builder.configureParameterElement(element, parameter, []*transportir.Parameter{parameter, peer})
			require.Equal(t, []string{"fixed", "sid"}, element.Namespace.Literals)
		})
	}
}
