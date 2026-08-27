package designfingerprint

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

type (
	externalMappingA struct{}
	externalMappingB struct{}
)

func TestDigestIsDeterministicForEquivalentDesigns(t *testing.T) {
	left := &expr.RootExpr{API: &expr.APIExpr{
		Name:             "widgets",
		Meta:             expr.MetaExpr{"zeta": {"last"}, "alpha": {"first"}},
		ExampleGenerator: expr.NewRandom("left-runtime-state"),
	}}
	right := &expr.RootExpr{API: &expr.APIExpr{
		Name:             "widgets",
		Meta:             expr.MetaExpr{"alpha": {"first"}, "zeta": {"last"}},
		ExampleGenerator: expr.NewRandom("right-runtime-state"),
	}}

	leftDigest, err := Digest(left, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	rightDigest, err := Digest(right, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.Equal(t, leftDigest, rightDigest)
}

func TestDigestChangesWithSemanticDesignOrGenerationInput(t *testing.T) {
	base := &expr.RootExpr{API: &expr.APIExpr{Name: "widgets", Title: "Widgets"}}
	changed := &expr.RootExpr{API: &expr.APIExpr{Name: "widgets", Title: "Inventory"}}

	baseDigest, err := Digest(base, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	changedDigest, err := Digest(changed, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	otherPackageDigest, err := Digest(base, "gen", "example.com/other/gen", 3)
	require.NoError(t, err)

	require.NotEqual(t, baseDigest, changedDigest)
	require.NotEqual(t, baseDigest, otherPackageDigest)
}

func TestDigestCoversGeneratedContractInputs(t *testing.T) {
	baseDigest, err := Digest(contractDesign(), "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	tests := map[string]func(*expr.RootExpr){
		"route": func(root *expr.RootExpr) {
			root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].Path = "/inventory"
		},
		"payload": func(root *expr.RootExpr) {
			root.Services[0].Methods[0].Payload.Description = "changed payload"
		},
		"result": func(root *expr.RootExpr) {
			root.Services[0].Methods[0].Result.Description = "changed result"
		},
		"error": func(root *expr.RootExpr) {
			root.Services[0].Methods[0].Errors[0].Name = "conflict"
		},
		"type mapping": func(root *expr.RootExpr) {
			root.Conversions[0].External = externalMappingB{}
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			root := contractDesign()
			change(root)
			digest, err := Digest(root, "gen", "example.com/service/gen", 3)
			require.NoError(t, err)
			require.NotEqual(t, baseDigest, digest)
		})
	}
}

func contractDesign() *expr.RootExpr {
	service := &expr.ServiceExpr{Name: "widgets"}
	method := &expr.MethodExpr{
		Name:    "show",
		Service: service,
		Payload: &expr.AttributeExpr{Type: expr.String, Description: "payload"},
		Result:  &expr.AttributeExpr{Type: expr.String, Description: "result"},
		Errors: []*expr.ErrorExpr{{
			Name:          "not_found",
			AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		}},
	}
	service.Methods = []*expr.MethodExpr{method}
	httpRoot := &expr.HTTPExpr{}
	httpService := &expr.HTTPServiceExpr{Root: httpRoot, ServiceExpr: service}
	endpoint := &expr.HTTPEndpointExpr{MethodExpr: method, Service: httpService}
	endpoint.Routes = []*expr.RouteExpr{{Method: "GET", Path: "/widgets", Endpoint: endpoint}}
	httpService.HTTPEndpoints = []*expr.HTTPEndpointExpr{endpoint}
	httpRoot.Services = []*expr.HTTPServiceExpr{httpService}
	return &expr.RootExpr{
		API:         &expr.APIExpr{Name: "widgets", HTTP: httpRoot},
		Services:    []*expr.ServiceExpr{service},
		Conversions: []*expr.TypeMap{{External: externalMappingA{}}},
	}
}
