package service

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
)

func TestDestinationErrorRoles(t *testing.T) {
	for _, kind := range []string{"collision", "shared", "ordinary"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", kind, reverse), func(t *testing.T) {
				root := codegen.RunDSL(t, testdata.DestinationErrorRolesDSL(kind))
				services := NewServicesData(root)
				failure := services.Get("alpha").errorTypes[0]
				payload := services.Get("beta").Methods[0].Payload
				order := slices.Clone(root.Services)
				if reverse {
					slices.Reverse(order)
				}
				files := renderedDestinationFiles(t, order, services)
				var source strings.Builder
				for _, contents := range files {
					source.WriteString(contents)
				}
				for _, name := range []string{failure.VarName, payload} {
					require.Equal(t, 1, strings.Count(source.String(), "type "+name+" struct {"), "one declaration for %s", name)
				}
				require.Equal(t, 1, strings.Count(source.String(), "func (e *"+failure.VarName+") Error() string"))
				require.Equal(t, 1, strings.Count(source.String(), "func (e *"+failure.VarName+") LoomErrorName() string"))
				require.Contains(t, source.String(), `return "authored_failure"`)
			})
		}
	}
}
