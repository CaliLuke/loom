package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	ctestdata "github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestExampleCLIFiles(t *testing.T) {
	assertExampleCodeGolden(t, []exampleDSLTestCase{
		{Name: "no-server", DSL: ctestdata.NoServerDSL},
		{Name: "server-hosting-service-subset", DSL: ctestdata.ServerHostingServiceSubsetDSL},
		{Name: "server-hosting-multiple-services", DSL: ctestdata.ServerHostingMultipleServicesDSL},
		{Name: "streaming", DSL: testdata.StreamingResultDSL},
		{Name: "streaming-multiple-services", DSL: testdata.StreamingMultipleServicesDSL},
	}, func(httpServices *ServicesData) []*codegen.File {
		return ExampleCLIFiles("", httpServices)
	}, 1, "client")
}
