package example

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
)

func TestServerFilesOmitUnusedHostOverrideFlags(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		API("Quickstart", func() {
			Server("orchestrator", func() {
				Services("orchestrator")
			})
		})
		Service("orchestrator", func() {
			Method("run", func() {
				HTTP(func() {
					GET("/")
				})
			})
		})
	})
	services := service.NewServicesData(root)
	files := ServerFiles(filepath.Join("example.com", "quickstart"), root, services)
	require.Len(t, files, 1)

	var buf bytes.Buffer
	for _, section := range files[0].AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package foo\n"+buf.String())

	if strings.Contains(code, "domainF = flag.String") {
		t.Errorf("expected generated server main to omit domain flag when no server URIs are rendered")
	}
	if strings.Contains(code, "secureF = flag.Bool") {
		t.Errorf("expected generated server main to omit secure flag when no server URIs are rendered")
	}
}

func TestServerFilesDoNotUseLegacyClueEndpointMiddleware(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		API("LegacyMiddleware", func() {
			Server("orchestrator", func() {
				Services("orchestrator")
				Host("dev", func() {
					URI("http://localhost:8080")
				})
			})
		})
		Service("orchestrator", func() {
			Method("run", func() {
				HTTP(func() {
					POST("/rpc")
				})
			})
		})
	})
	services := service.NewServicesData(root)
	files := ServerFiles(filepath.Join("example.com", "legacy"), root, services)
	require.Len(t, files, 1)

	var buf bytes.Buffer
	for _, section := range files[0].AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package foo\n"+buf.String())

	require.NotContains(t, code, "LogPayloads()")
	require.NotContains(t, code, "log.Endpoint")
}
