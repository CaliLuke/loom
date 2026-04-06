package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	ctestdata "github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestExampleServerFiles(t *testing.T) {
	t.Run("package name check", func(t *testing.T) {
		cases := []struct {
			Name     string
			DSL      func()
			Expected string
		}{
			{
				Name:     "conflict with API name and service names including multipart",
				DSL:      ctestdata.ConflictWithAPINameAndServiceNamesIncludingMultipartDSL,
				Expected: "package alohaapi2",
			},
		}
		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				root := codegen.RunDSL(t, c.DSL)
				require.Len(t, root.Services, 3)
				httpServices := NewServicesData(service.NewServicesData(root), root.API.HTTP)
				fs := ExampleServerFiles("", httpServices)
				require.Len(t, fs, 2)
				for i, f := range fs {
					if i < len(fs)-1 {
						// Skip example http server.
						continue
					}
					sections := f.AllSections()
					require.Greater(t, len(sections), 0)
					var b bytes.Buffer
					require.NoError(t, sections[0].Write(&b))
					line, err := b.ReadBytes('\n')
					assert.NoError(t, err)
					got := string(bytes.TrimRight(line, "\n"))
					assert.Equal(t, c.Expected, got)
				}
			})
		}
	})

	t.Run("code check", func(t *testing.T) {
		assertExampleCodeGolden(t, []exampleDSLTestCase{
			{Name: "no-server", DSL: ctestdata.NoServerDSL},
			{Name: "server-hosting-service-with-file-server", DSL: ctestdata.ServerHostingServiceWithFileServerDSL},
			{Name: "server-hosting-service-subset", DSL: ctestdata.ServerHostingServiceSubsetDSL},
			{Name: "server-hosting-multiple-services", DSL: ctestdata.ServerHostingMultipleServicesDSL},
			{Name: "streaming", DSL: testdata.StreamingMultipleServicesDSL},
		}, func(httpServices *ServicesData) []*codegen.File {
			return ExampleServerFiles("", httpServices)
		}, 1, "server")
	})
}
