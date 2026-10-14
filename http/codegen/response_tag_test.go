package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestResponseTagBindings checks semantic identity and physical lowering together.
func TestResponseTagBindings(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResponseTagCarriersDSL)
	services := NewServicesData(service.NewServicesData(root), root.API.HTTP)
	data := services.Get("tagged")
	for i, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			binding := endpoint.Responses[1].TagBinding
			require.NotNil(t, binding)
			key, declared := endpoint.MethodExpr.Result.FindAttribute("kind")
			require.Equal(t, key, binding.Key)
			require.Same(t, declared, binding.Attribute)
			require.Equal(t, "accepted", binding.Value)
			physical := data.Endpoints[i].Result.Responses[0].Tag
			require.NotNil(t, physical)
			require.Nil(t, data.Endpoints[i].Result.Responses[1].Tag)
			switch endpoint.Name() {
			case "nullable", "named_nullable":
				require.Equal(t, `tag, ok := res.Kind.Value(); ok && tag == "accepted"`, physical.Match)
				require.Equal(t, `res.Kind.SetValue("accepted")`, physical.Assignment)
			case "renamed":
				require.Equal(t, "Disposition", physical.Field)
				require.Contains(t, physical.Assignment, "res.Disposition = &tmp")
			case "mapped":
				require.Equal(t, "kind:wire_kind", binding.Key)
				require.Equal(t, "Kind", physical.Field)
			case "alias":
				require.Contains(t, physical.Assignment, "var tmp tagged.Marker")
			case "viewed_alias":
				require.Contains(t, physical.Assignment, "var tmp taggedviews.MarkerView")
			}
			if endpoint.Name() == "viewed" || endpoint.Name() == "viewed_alias" {
				require.Contains(t, physical.Match, "res.Projected.Kind")
				require.Contains(t, physical.Assignment, "p.Kind")
			}
		})
	}
}
