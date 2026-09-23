package codegen

import (
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
)

// TestUserTypePackageGRPCModuleCompiles compiles the gRPC packages generated
// for user types placed in a custom package with struct:pkg:path metadata.
// The types appear in payloads, results, errors, streams, nested fields,
// arrays, and maps, next to a payload field named like the package.
func TestUserTypePackageGRPCModuleCompiles(t *testing.T) {
	root := RunGRPCDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/usertypepkg", root, resolveGRPCLoomSource(t))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
}
