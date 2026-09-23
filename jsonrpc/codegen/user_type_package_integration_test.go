package codegen

import (
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
)

// TestUserTypePackageJSONRPCModuleCompiles compiles the JSON-RPC packages
// generated for user types placed in a custom package with struct:pkg:path
// metadata. The types appear in payloads, results, errors, nested fields,
// arrays, maps, and SSE and WebSocket streams of local wrappers, next to a
// payload field named like the package.
func TestUserTypePackageJSONRPCModuleCompiles(t *testing.T) {
	root := RunJSONRPCDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/usertypepkg", root)

	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
