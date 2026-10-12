package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBasicCLIArgumentDefaults(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.BasicCLIDefaultsDSL("http"))
	data := CreateHTTPServices(root)
	for _, tc := range []struct {
		service, method            string
		userDefault, passDefault   any
		userPointer, passPointer   bool
		userRequired, passRequired bool
	}{
		{"basic", "optional", nil, nil, true, true, false, false},
		{"basic", "user", nil, nil, false, true, true, false},
		{"basic", "pass", nil, nil, true, false, false, true},
		{"basic", "defaulted", "alice", "secret", false, false, false, false},
		{"defaults", "plain", nil, nil, true, true, false, false},
		{"defaults", "both", "alice", "secret", false, false, false, false},
		{"defaults", "user", "alice", nil, false, true, false, false},
		{"defaults", "pass", nil, "secret", true, false, false, false},
		{"defaults", "required", "alice", "secret", false, false, true, true},
		{"defaults", "empty", "", nil, false, true, false, false},
		{"defaults", "inherited", "inherited", "inherited", false, false, false, false},
		{"defaults", "override", "alice", "inherited", false, false, false, false},
	} {
		t.Run(tc.service+"/"+tc.method, func(t *testing.T) {
			endpoint := data.Get(tc.service).Endpoint(tc.method)
			args := endpoint.Payload.Request.PayloadInit.CLIArgs
			require.Len(t, args, 2)
			for i, component := range []struct {
				defaultValue      any
				pointer, required bool
			}{
				{tc.userDefault, tc.userPointer, tc.userRequired},
				{tc.passDefault, tc.passPointer, tc.passRequired},
			} {
				arg := args[i]
				require.Equal(t, component.defaultValue, arg.DefaultValue)
				require.Equal(t, component.pointer, arg.Pointer)
				require.Equal(t, component.pointer, arg.FieldPointer)
				require.Equal(t, component.required, arg.Required)
				require.Equal(t, "string", arg.TypeName)
				reference := "string"
				if component.pointer {
					reference = "*string"
				}
				require.Equal(t, reference, arg.TypeRef)
				flag := makeFlagData(endpoint, arg, "")
				require.Equal(t, component.defaultValue, flag.Default)
			}
		})
	}
}

func TestBasicCLIDefaultsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.BasicCLIDefaultsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/basic", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_defaults_test.go"), []byte(strings.NewReplacer("TRANSPORT", "http", "MULTIPARTARGS", ", nil, nil").Replace(servicedata.BasicCLIDefaultsHarness)), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
