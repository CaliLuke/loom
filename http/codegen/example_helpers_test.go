package codegen

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
)

type exampleDSLTestCase struct {
	Name string
	DSL  func()
}

func assertExampleCodeGolden(
	t *testing.T,
	cases []exampleDSLTestCase,
	build func(*ServicesData) []*codegen.File,
	expectedFiles int,
	fileLabel string,
) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			httpServices := NewServicesData(service.NewServicesData(root), root.API.HTTP)
			fs := build(httpServices)
			require.Len(t, fs, expectedFiles)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 0)
			var buf bytes.Buffer
			for _, s := range sections[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			golden := filepath.Join("testdata", "golden", fileLabel+"-"+c.Name+".golden")
			testutil.NewGoldenFile(t, ".").StringContent(code).Path(golden).CompareContent()
		})
	}
}
