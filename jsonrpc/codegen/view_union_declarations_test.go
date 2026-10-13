package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestViewedUnionDeclarationsJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.ViewedOmittedUnionJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/viewunion", root)

	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
}
