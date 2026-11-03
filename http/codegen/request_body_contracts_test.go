package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRequestBodyContractsGenerated compiles the body contracts in one module.
// Each design keeps its own generated namespace and harness package, so type
// names and global CLI flags stay isolated. One tidy, vet and race-test run
// covers all packages, including every generated client, server and CLI.
func TestRequestBodyContractsGenerated(t *testing.T) {
	const modulePath = "example.com/bodycontracts"
	dir := t.TempDir()
	writeHTTPModule(t, dir, modulePath)
	for _, tc := range []struct {
		name, oldModule string
		design          func()
		harnesses       []string
	}{
		{"optional_object", "example.com/optobjectbody", optionalObjectRequestBodyIntegrationDSL, []string{optionalObjectRequestBodyHarness}},
		{"optional_value", "example.com/optvaluebody", optionalValueBodyModuleDSL("v"), []string{optionalValueRequestBodyHarness, optionalValueRequestBodyCLIHarness}},
		{"nullable", "example.com/nullbody", func() {
			nullableRequestBodyIntegrationDSL("")
		}, []string{nullableRequestBodyHarness}},
		{"union", "example.com/unionrequestbody", unionRequestBodyIntegrationDSL, []string{unionRequestBodyHarness}},
		{"named_collection", "example.com/namedcollection", namedCollectionPayloadDSL, []string{namedCollectionPayloadHarness}},
		{"nested_collection", "example.com/nestedpayload", nestedCollectionPayloadDSL, []string{nestedCollectionPayloadHarness}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			packagePath := modulePath + "/" + tc.name
			packageDir := filepath.Join(dir, tc.name)
			renderHTTPModuleFiles(t, packageDir, packagePath+"/gen", root)
			for i, harness := range tc.harnesses {
				path := filepath.Join(packageDir, fmt.Sprintf("contract_%d_test.go", i))
				source := strings.ReplaceAll(harness, tc.oldModule+"/", packagePath+"/")
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			}
		})
	}
	if t.Failed() {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-race", "-count=1", "-v", "./..."))
}
