package service

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestRecursiveProjectionExcludesFields(t *testing.T) {
	root := codegen.RunDSL(t, testdata.RecursiveProjectionDSL)
	data := NewServicesData(root).Get("projection")
	var code strings.Builder
	for _, projected := range data.projectedTypes {
		for _, init := range projected.Projections {
			if init.Name != "ProjectWrap" {
				continue
			}
			code.WriteString(init.Code)
			fullCall := regexp.MustCompile(`FullOwner = ([A-Za-z0-9_]+)\(`).FindStringSubmatch(init.Code)
			require.Len(t, fullCall, 2)
			for _, helper := range init.Helpers {
				if helper.Name != fullCall[1] {
					code.WriteString(helper.Code)
				}
			}
		}
	}
	// Only the explicitly full direct child may use the full projection;
	// nested generic helpers must never copy the secret from canonical Owner.
	require.NotEmpty(t, code.String())
	require.NotContains(t, code.String(), "Secret:", "container conversions must honor selected child views")
}
