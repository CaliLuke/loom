package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestCollectionDeclarationIdentityGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		API("a", func() {})
		item := Type("item", func() {
			Field(1, "a", String)
		})
		itemUpper := Type("Item", func() {
			Field(1, "a", String)
		})
		Service("s", func() {
			Method("m", func() {
				Payload(func() {
					Field(1, "x", itemUpper)
					Field(2, "xs", ArrayOf(itemUpper))
					Field(3, "ys", ArrayOf(item))
					Field(4, "zs", MapOf(String, item))
					Field(5, "ws", MapOf(String, itemUpper))
				})
				Result(String)
				HTTP(func() {
					POST("/m")
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/collectionidentity", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}

func TestCollectionDeclarationIdentityRoundTripHTTP(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		name := "collision"
		if reserved {
			name = "reserved"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.CollectionDeclarationIdentityDSL(reserved))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/collectionidentity", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "identity_test.go"), []byte(testdata.CollectionDeclarationIdentityHarness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}
