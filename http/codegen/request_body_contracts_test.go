package codegen

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/internal/testingx"
)

// TestRequestBodyContractsGenerated compiles the body contracts in one module.
// Each design keeps its own generated namespace and harness package, so type
// names and global CLI flags stay isolated. One tidy, vet and race-test run
// covers all packages, including every generated client, server and CLI.
func TestRequestBodyContractsGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), "")
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
			c := batch.Case(t, tc.name, tc.oldModule)
			renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			for i, harness := range tc.harnesses {
				c.WriteFile(t, fmt.Sprintf("contract_%d_test.go", i), harness)
			}
		})
	}
	batch.Check(t)
}
