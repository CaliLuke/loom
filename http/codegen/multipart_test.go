package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedMultipartServerCompiles(t *testing.T) {
	repoRoot := runCommand(t, "", "git", "rev-parse", "--show-toplevel")
	t.Setenv("LOOM_DIR", filepath.Clean(strings.TrimSpace(repoRoot)))
	tests := map[string]func(){
		"optional fields": testdata.PayloadMultipartObjectGeneratedOptionalDSL,
		"path parameter":  testdata.PayloadMultipartCompilePathParamDSL,
		"required file":   testdata.PayloadMultipartCompileRequiredFileDSL,
		"two file fields": testdata.PayloadMultipartCompileTwoFilesDSL,
		"validation":      testdata.PayloadMultipartCompileValidationDSL,
	}
	for name, dsl := range tests {
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/multipartcompile", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

func TestServerMultipartFuncType(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"multipart-body-primitive", testdata.PayloadMultipartPrimitiveDSL},
		{"multipart-body-array-type", testdata.PayloadMultipartArrayTypeDSL},
		{"multipart-body-map-type", testdata.PayloadMultipartMapTypeDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			require.Len(t, fs, 2)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 5)
			code := codegen.SectionCode(t, sections[3])
			testutil.AssertGo(t, "testdata/golden/server_multipart_"+c.Name+".go.golden", code)
		})
	}
}

func TestClientMultipartFuncType(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"multipart-body-primitive", testdata.PayloadMultipartPrimitiveDSL},
		{"multipart-body-user-type", testdata.PayloadMultipartUserTypeDSL},
		{"multipart-body-array-type", testdata.PayloadMultipartArrayTypeDSL},
		{"multipart-body-map-type", testdata.PayloadMultipartMapTypeDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ClientFiles(genpkg, services)
			require.Len(t, fs, 2)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 4)
			code := codegen.SectionCode(t, sections[2])
			testutil.AssertGo(t, "testdata/golden/client_multipart_"+c.Name+".go.golden", code)
		})
	}
}

func TestServerMultipartNewFunc(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"server-multipart-body-primitive", testdata.PayloadMultipartPrimitiveDSL},
		{"server-multipart-body-array-type", testdata.PayloadMultipartArrayTypeDSL},
		{"server-multipart-body-map-type", testdata.PayloadMultipartMapTypeDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			require.Len(t, fs, 2)
			sections := fs[1].AllSections()
			require.Greater(t, len(sections), 3)
			code := codegen.SectionCode(t, sections[3])
			testutil.AssertGo(t, "testdata/golden/server_multipart_"+c.Name+".go.golden", code)
		})
	}
}

func TestClientMultipartNewFunc(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"client-multipart-body-primitive", testdata.PayloadMultipartPrimitiveDSL},
		{"client-multipart-body-user-type", testdata.PayloadMultipartUserTypeDSL},
		{"client-multipart-body-array-type", testdata.PayloadMultipartArrayTypeDSL},
		{"client-multipart-body-map-type", testdata.PayloadMultipartMapTypeDSL},
		{"client-multipart-with-param", testdata.PayloadMultipartWithParamDSL},
		{"client-multipart-with-params-and-headers", testdata.PayloadMultipartWithParamsAndHeadersDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ClientFiles(genpkg, services)
			require.Len(t, fs, 2)
			sections := fs[1].AllSections()
			require.Greater(t, len(sections), 3)
			code := codegen.SectionCode(t, sections[3])
			testutil.AssertGo(t, "testdata/golden/client_multipart_"+c.Name+".go.golden", code)
		})
	}
}

func TestServerMultipartObjectUsesGeneratedDecoder(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"generated object", testdata.PayloadMultipartObjectGeneratedDSL},
		{"generated object with param", testdata.PayloadMultipartWithParamDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			require.Len(t, fs, 2)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 3)
			require.Equal(t, "server-init", sections[3].SectionName())
			code := codegen.SectionCode(t, sections[3])
			require.NotContains(t, code, "DecoderFunc")
			require.NotContains(t, code, "DecoderFn")
		})
	}
}
