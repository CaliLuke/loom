package codegen

import (
	"github.com/CaliLuke/loom/v3/codegen/testutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/http/codegen/testdata"
)

func TestClientInit(t *testing.T) {
	cases := []struct {
		Name       string
		DSL        func()
		FileCount  int
		SectionNum int
	}{
		{"multiple endpoints", testdata.ServerMultiEndpointsDSL, 2, 2},
		{"streaming", testdata.StreamingResultDSL, 3, 2},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ClientFiles("", services)
			require.Len(t, fs, c.FileCount)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), c.SectionNum)
			code := codegen.SectionCode(t, sections[c.SectionNum])
			testutil.AssertGo(t, "testdata/golden/client_init_"+c.Name+".go.golden", code)
		})
	}
}
