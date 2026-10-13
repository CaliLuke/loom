package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestConstructorOwnershipGenerated(t *testing.T) {
	for _, mode := range []string{"handler", "response", "shared", "distinct", "ordinary", "selector", "same"} {
		for _, reverse := range []bool{false, true} {
			if reverse && mode != "shared" {
				continue
			}
			name := mode
			if reverse {
				name += "-reverse"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, func() {
					testdata.ConstructorOwnership(mode, false, reverse)
				})
				dir := t.TempDir()
				if output := os.Getenv("LOOM_CONSTRUCTOR_OWNERSHIP_OUTPUT"); output != "" {
					dir = filepath.Join(output, "http", name)
					require.NoError(t, os.MkdirAll(dir, 0o750))
				}
				renderHTTPModule(t, dir, "example.com/constructors", root)
				files, err := OpenAPIFiles(root)
				require.NoError(t, err)
				for _, file := range files {
					_, err := file.Render(dir)
					require.NoError(t, err)
				}
				if mode == "shared" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "source_test.go"), []byte(fmt.Sprintf(testdata.ConstructorOwnershipSharedHarness, "http", ",nil", false, false)), 0o600))
				}
				if mode == "selector" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "source_test.go"), []byte(fmt.Sprintf(testdata.ConstructorOwnershipWireHarness, "http")), 0o600))
				}
				if os.Getenv("LOOM_CONSTRUCTOR_GENERATE_ONLY") != "" {
					return
				}
				runGoCommand(t, dir, "mod", "tidy")
				runGoCommand(t, dir, "build", "./...")
				if mode == "selector" || mode == "shared" {
					runGoCommand(t, dir, "test", "-run", "^Test(SelectedConstructorSource|MountedSharedConstructor)$", ".")
				}
			})
		}
	}
}

func TestConstructorOwnershipPlans(t *testing.T) {
	for _, mode := range []string{"handler", "response", "shared", "selector"} {
		t.Run(mode, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				testdata.ConstructorOwnership(mode, false, false)
			})
			sd := CreateHTTPServices(root).Get("constructors")
			first := sd.Endpoints[0]
			switch mode {
			case "handler":
				require.NotEqual(t, first.HandlerInit, first.Payload.Request.PayloadInit.Name)
			case "response":
				require.NotEqual(t, first.Result.Responses[0].ServerBody[0].Init.Name, first.Payload.Request.PayloadInit.Name)
			default:
				a := first.Result.Responses[0].ServerBody[0].Init
				b := sd.Endpoints[1].Result.Responses[0].ServerBody[0].Init
				if mode == "shared" {
					require.NotEqual(t, a.ServerArgs[0].TypeRef, b.ServerArgs[0].TypeRef)
				} else {
					require.Equal(t, a.ServerArgs[0].TypeRef, b.ServerArgs[0].TypeRef)
					require.NotEqual(t, a.ServerCode, b.ServerCode)
				}
				require.NotEqual(t, a.Name, b.Name, "distinct conversion contracts require distinct declarations")
				for _, endpoint := range sd.Endpoints {
					response := endpoint.Result.Responses[0]
					body := response.ServerBody[0]
					require.Contains(t, response.EncodePlan.BodyValues[body], body.Init.Name+"(")
				}
			}
		})
	}
}

// TestConstructorConsumerArtifacts captures an existing mixed-shape design with
// relocated types, errors, unions and stream constructors for bounded comparison.
func TestConstructorConsumerArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_CONSTRUCTOR_OWNERSHIP_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_CONSTRUCTOR_OWNERSHIP_OUTPUT for artifact comparison")
	}
	root := RunHTTPDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	dir := filepath.Join(output, "http", "consumers")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	renderHTTPModule(t, dir, "example.com/consumers", root)
	files, err := OpenAPIFiles(root)
	require.NoError(t, err)
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	if os.Getenv("LOOM_CONSTRUCTOR_GENERATE_ONLY") == "" {
		runGoCommand(t, dir, "mod", "tidy")
		runGoCommand(t, dir, "build", "./...")
	}
}
