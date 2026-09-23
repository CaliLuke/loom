package codegen

import (
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
)

// TestUserTypePackageHTTPModuleCompiles compiles the HTTP packages generated
// for user types placed in a custom package with struct:pkg:path metadata.
// The types appear in payloads, results, errors, streams, nested fields,
// arrays, and maps, next to a transport param named like the package.
func TestUserTypePackageHTTPModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/usertypepkg", root)

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
