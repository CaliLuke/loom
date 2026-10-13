package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestSharedModuleIncludesViews(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mixed, viewed bool
	}{
		{"jsonrpc", false, true},
		{"mixed", true, true},
		{"ordinary", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				if !tc.viewed {
					jsonrpcMixedServicePackageAliasDSL()
					return
				}
				jsonrpcViewedUnaryResultDSL()
				if tc.mixed {
					dsl.Service("plainhttp", func() {
						dsl.Method("ping", func() {
							dsl.Result(dsl.String)
							dsl.HTTP(func() {
								dsl.GET("/ping")
							})
						})
					})
				}
			})
			dir := t.TempDir()
			if base := os.Getenv("LOOM_MODULE_VIEWS_ARTIFACTS"); base != "" {
				dir = filepath.Join(base, tc.name)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			const module = "example.com/moduleviews"
			if tc.mixed {
				renderMixedHTTPJSONRPCModule(t, dir, module, root)
			} else {
				renderJSONRPCModule(t, dir, module, root)
			}
			if tc.viewed {
				require.FileExists(t, filepath.Join(dir, "gen", "notes", "views", "view.go"))
			} else {
				require.NoDirExists(t, filepath.Join(dir, "gen", "foo", "views"))
				require.NoDirExists(t, filepath.Join(dir, "gen", "foojssvr", "views"))
			}
			if os.Getenv("LOOM_MODULE_VIEWS_GENERATE_ONLY") != "" {
				return
			}
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "build", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
		})
	}
}
