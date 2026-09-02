package designfingerprint

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

type (
	externalMappingA struct{}
	externalMappingB struct{}
	cycleNode        struct {
		Name string
		Next *cycleNode
	}
	cyclePair struct {
		Left  *cycleNode
		Right *cycleNode
	}
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

func TestDigestIgnoresVetMetadata(t *testing.T) {
	base := &expr.RootExpr{API: &expr.APIExpr{Name: "widgets"}}
	configured := &expr.RootExpr{API: &expr.APIExpr{
		Name: "widgets",
		Meta: expr.MetaExpr{
			"loom:vet:http-entrypoint": {"./cmd/api"},
			"loom:vet:ignore":          {"generated-design-skew"},
		},
	}}

	baseDigest, err := Digest(base, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	configuredDigest, err := Digest(configured, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.Equal(t, baseDigest, configuredDigest)
	generated := &expr.RootExpr{API: &expr.APIExpr{
		Name: "widgets",
		Meta: expr.MetaExpr{"http:generate": {"server"}},
	}}
	generatedDigest, err := Digest(generated, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.NotEqual(t, baseDigest, generatedDigest)
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

func TestDigestHandlesDeepSharedDesignGraphs(t *testing.T) {
	root := contractDesign()
	root.Services[0].Methods[0].Result = sharedAttributeGraph(32)

	first, err := Digest(root, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	second, err := Digest(root, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestDigestHandlesRepeatedCyclicGraphReferences(t *testing.T) {
	root := contractDesign()
	service := root.Services[0]
	for index := range 128 {
		method := &expr.MethodExpr{
			Name:    fmt.Sprintf("show_%d", index),
			Service: service,
			Payload: &expr.AttributeExpr{Type: expr.String},
			Result:  &expr.AttributeExpr{Type: expr.String},
		}
		service.Methods = append(service.Methods, method)
	}
	for range 1024 {
		root.Services = append(root.Services, service)
	}

	digest, err := Digest(root, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.NotEmpty(t, digest)
}

func TestDigestDoesNotTreatEquivalentSharingAsSemantic(t *testing.T) {
	shared := &expr.AttributeExpr{Type: expr.String, Description: "value"}
	sharedRoot := contractDesign()
	sharedRoot.Services[0].Methods[0].Result = objectWithChildren(shared, shared)

	duplicatedRoot := contractDesign()
	duplicatedRoot.Services[0].Methods[0].Result = objectWithChildren(
		&expr.AttributeExpr{Type: expr.String, Description: "value"},
		&expr.AttributeExpr{Type: expr.String, Description: "value"},
	)

	sharedDigest, err := Digest(sharedRoot, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	duplicatedDigest, err := Digest(duplicatedRoot, "gen", "example.com/service/gen", 3)
	require.NoError(t, err)
	require.Equal(t, sharedDigest, duplicatedDigest)
}

func TestEncoderDoesNotTreatEquivalentCyclicSharingAsSemantic(t *testing.T) {
	sharedA := &cycleNode{Name: "A"}
	sharedB := &cycleNode{Name: "B"}
	sharedA.Next = sharedB
	sharedB.Next = sharedA

	leftA := &cycleNode{Name: "A"}
	leftB := &cycleNode{Name: "B"}
	leftA.Next = leftB
	leftB.Next = leftA
	rightA := &cycleNode{Name: "A"}
	rightB := &cycleNode{Name: "B"}
	rightA.Next = rightB
	rightB.Next = rightA

	sharedEncoding := encodeTestValue(t, cyclePair{Left: sharedA, Right: sharedB})
	duplicatedEncoding := encodeTestValue(t, cyclePair{Left: leftA, Right: rightB})
	require.Equal(t, sharedEncoding, duplicatedEncoding)
}

func TestEncoderHandlesDeepPointerChains(t *testing.T) {
	root := &cycleNode{Name: "leaf"}
	for index := range 4096 {
		root = &cycleNode{Name: fmt.Sprintf("node_%d", index), Next: root}
	}

	require.NotEmpty(t, encodeTestValue(t, root))
}

func TestEncoderHandlesLongPointerCycles(t *testing.T) {
	nodes := make([]*cycleNode, 4096)
	for index := range nodes {
		nodes[index] = &cycleNode{Name: fmt.Sprintf("node_%d", index)}
	}
	for index := range len(nodes) - 1 {
		nodes[index].Next = nodes[index+1]
	}
	nodes[len(nodes)-1].Next = nodes[0]

	require.NotEmpty(t, encodeTestValue(t, nodes[0]))
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

func sharedAttributeGraph(depth int) *expr.AttributeExpr {
	attribute := &expr.AttributeExpr{Type: expr.String, Description: "leaf"}
	for level := range depth {
		attribute = objectWithChildren(attribute, attribute)
		attribute.Description = fmt.Sprintf("level %d", level)
	}
	return attribute
}

func objectWithChildren(left, right *expr.AttributeExpr) *expr.AttributeExpr {
	object := expr.Object{
		&expr.NamedAttributeExpr{Name: "left", Attribute: left},
		&expr.NamedAttributeExpr{Name: "right", Attribute: right},
	}
	return &expr.AttributeExpr{Type: &object}
}

func encodeTestValue(t *testing.T, value any) []byte {
	t.Helper()
	enc := &encoder{
		active:        make(map[pointerIdentity]int),
		cache:         newPointerCache(nil),
		minCycleDepth: -1,
	}
	require.NoError(t, enc.encode(reflect.ValueOf(value), &pathFrame{component: "root"}))
	return enc.buffer.Bytes()
}
