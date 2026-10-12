package codegen

import (
	"testing"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestViewedUnionDeclarationsJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.ViewedOmittedUnionJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/viewunion", root)
	data := CreateJSONRPCServices(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile("example.com/viewunion/gen", svc, data.ServicesData); views != nil {
			renderCodegenFiles(t, dir, []*cg.File{views})
		}
	}
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
}
