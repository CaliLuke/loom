package service

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
)

func TestExampleInterceptorsFiles(t *testing.T) {
	cases := []struct {
		Name          string
		DSL           func()
		ExpectedFiles []string
	}{
		{
			Name: "no-interceptors",
			DSL:  testdata.NoInterceptorExampleDSL,
		},
		{
			Name: "server-interceptor",
			DSL:  testdata.ServerInterceptorExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "server_interceptor_service_server.go"),
			},
		},
		{
			Name: "client-interceptor",
			DSL:  testdata.ClientInterceptorExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "client_interceptor_service_client.go"),
			},
		},
		{
			Name: "server-interceptor-by-name",
			DSL:  testdata.ServerInterceptorByNameExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "server_interceptor_by_name_service_server.go"),
			},
		},
		{
			Name: "multiple-interceptors",
			DSL:  testdata.MultipleInterceptorsExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "multiple_interceptors_service_server.go"),
				filepath.Join("interceptors", "multiple_interceptors_service_client.go"),
			},
		},
		{
			Name: "multiple-services",
			DSL:  testdata.MultipleServicesInterceptorsExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "multiple_services_interceptors_service_server.go"),
				filepath.Join("interceptors", "multiple_services_interceptors_service_client.go"),
				filepath.Join("interceptors", "multiple_services_interceptors_service2_server.go"),
				filepath.Join("interceptors", "multiple_services_interceptors_service2_client.go"),
			},
		},
		{
			Name: "api-interceptors",
			DSL:  testdata.APIInterceptorExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "api_interceptor_service_server.go"),
				filepath.Join("interceptors", "api_interceptor_service_client.go"),
			},
		},
		{
			Name: "chained-interceptors",
			DSL:  testdata.ChainedInterceptorExampleDSL,
			ExpectedFiles: []string{
				filepath.Join("interceptors", "chained_interceptor_service_server.go"),
				filepath.Join("interceptors", "chained_interceptor_service_client.go"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// Run DSL
			root := expr.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.NotNil(t, root)

			// Generate files
			fs := ExampleInterceptorsFiles("", root, services)
			require.Len(t, fs, len(c.ExpectedFiles))

			// Verify file paths
			paths := make([]string, len(fs))
			for i, f := range fs {
				paths[i] = f.Path
				assert.True(t, f.SkipExist, "example interceptor %s must preserve user ownership", f.Path)
			}
			assert.ElementsMatch(t, c.ExpectedFiles, paths)

			// Verify file content
			for _, f := range fs {
				buf := new(bytes.Buffer)
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(buf))
				}
				bs, err := format.Source(buf.Bytes())
				require.NoError(t, err, buf.String())
				code := string(bs)

				// Remove the first line so the version of Loom is not part of the golden file
				code = strings.SplitN(code, "\n", 2)[1]

				// Use the base name of the generated file without extension for the golden file
				baseName := filepath.Base(f.Path)
				ext := filepath.Ext(baseName)
				goldenName := baseName[:len(baseName)-len(ext)] + ".golden"
				golden := filepath.Join("testdata", "example_interceptors", goldenName)
				assertGolden(t, code, golden)
			}
		})
	}
}

func TestExampleInterceptorsOwnership(t *testing.T) {
	for _, design := range []struct {
		name string
		dsl  func()
		file string
	}{
		{name: "server", dsl: testdata.ServerInterceptorExampleDSL, file: "server_interceptor_service_server.go"},
		{name: "client", dsl: testdata.ClientInterceptorExampleDSL, file: "client_interceptor_service_client.go"},
	} {
		for _, location := range []string{"output", "cwd", "fresh"} {
			t.Run(design.name+"/"+location, func(t *testing.T) {
				cwd, output := t.TempDir(), t.TempDir()
				t.Chdir(cwd)
				relative := filepath.Join("interceptors", design.file)
				destination := filepath.Join(output, relative)
				original := []byte("package interceptors // user edited\n")
				var occupied string
				var before os.FileInfo
				if location != "fresh" {
					directory := output
					if location == "cwd" {
						directory = cwd
					}
					occupied = filepath.Join(directory, relative)
					require.NoError(t, os.MkdirAll(filepath.Dir(occupied), 0o750))
					require.NoError(t, os.WriteFile(occupied, original, 0o600))
					var err error
					before, err = os.Stat(occupied)
					require.NoError(t, err)
				}
				root := expr.RunDSL(t, design.dsl)
				files := ExampleInterceptorsFiles("", root, NewServicesData(root))
				require.Len(t, files, 1, "cwd must not decide the output declaration")
				require.Equal(t, relative, files[0].Path)
				rendered, err := files[0].Render(output)
				require.NoError(t, err)
				if location == "output" {
					assert.Empty(t, rendered, "preserved entries must not be reported as rendered")
				} else {
					require.Equal(t, destination, rendered)
				}
				if occupied != "" {
					got, err := os.ReadFile(occupied)
					require.NoError(t, err)
					assert.Equal(t, original, got)
					after, err := os.Stat(occupied)
					require.NoError(t, err)
					assert.True(t, os.SameFile(before, after), "existing entry identity must survive")
				}
				generated, err := os.ReadFile(destination)
				require.NoError(t, err)
				if location != "output" {
					require.Contains(t, string(generated), "package interceptors")
					generated = append(generated, []byte("\n// user customization\n")...)
					require.NoError(t, os.WriteFile(destination, generated, 0o600))
				}
				before, err = os.Stat(destination)
				require.NoError(t, err)
				files = ExampleInterceptorsFiles("", root, NewServicesData(root))
				require.Len(t, files, 1)
				rendered, err = files[0].Render(output)
				require.NoError(t, err)
				assert.Empty(t, rendered)
				repeated, err := os.ReadFile(destination)
				require.NoError(t, err)
				assert.Equal(t, generated, repeated)
				after, err := os.Stat(destination)
				require.NoError(t, err)
				assert.True(t, os.SameFile(before, after))
			})
		}
	}
}
