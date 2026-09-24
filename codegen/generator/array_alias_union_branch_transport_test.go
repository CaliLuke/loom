package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestArrayAliasAndUnionBranchGeneratedCodeCompiles builds and vets the
// service and gRPC packages, client CLI included, generated for named arrays
// used as message fields, collection elements and union branches, and for
// unions used as branches of other unions, including a type that reaches
// itself through them.
func TestArrayAliasAndUnionBranchGeneratedCodeCompiles(t *testing.T) {
	cases := []struct {
		name       string
		modulePath string
		dsl        func()
	}{
		{"array alias", "example.com/arrayalias", grpctestdata.ArrayAliasDSL},
		{"union branch union", "example.com/unionbranchunion", grpctestdata.UnionBranchUnionDSL},
		{"recursive array alias", "example.com/recursivearrayalias", grpctestdata.RecursiveArrayAliasDSL},
		{"recursive array message", "example.com/recursivearraymessage", grpctestdata.RecursiveArrayMessageDSL},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := buildGeneratedModule(t, c.modulePath, c.dsl)
			output, err := testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}
