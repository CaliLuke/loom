// Package testutil provides testing utilities for the Loom code generation framework.
//
// # Golden File Testing
//
// The package provides utilities for golden file testing, a technique where
// expected outputs are stored in files and compared against actual outputs
// during tests. This is particularly useful for testing code generation where
// outputs can be large and complex.
//
// Basic Usage:
//
//	func TestCodeGeneration(t *testing.T) {
//		// Create a golden file manager
//		gf := testutil.NewGoldenFile(t, "testdata/golden")
//
//		// Generate code
//		code := generateCode()
//
//		// Compare with golden file
//		gf.StringContent(code).Path("mytest.golden").CompareContent()
//	}
//
// Updating Golden Files:
//
// To update golden files when the expected output changes, run tests with
// the -update flag:
//
//	go test ./... -update
//
// Advanced Usage:
//
// The GoldenFile type provides additional methods for more complex scenarios:
//
//	// Check if a golden file exists
//	if gf.Exists("optional.golden") {
//		gf.StringContent(code).Path("optional.golden").CompareContent()
//	}
//
// Organization:
//
// Golden files are typically organized under a testdata/golden directory
// within each package. This keeps test data close to the tests while
// maintaining a clean structure.
package testutil
