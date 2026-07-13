package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestServerTypesUseViewRequirednessOverrides(t *testing.T) {
	root := RunHTTPDSL(t, servicetestdata.ViewRequirednessOverridesDSL)
	services := CreateHTTPServices(root)
	fs := serverType("gen", root.API.HTTP.Services[0], services)
	var buf bytes.Buffer
	for _, section := range fs.AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
	require.Contains(t, code, "CanonicalRequired string  `form:\"canonical_required\" json:\"canonical_required\" xml:\"canonical_required\"`")
	require.Contains(t, code, "CanonicalOptional *string `form:\"canonical_optional,omitempty\" json:\"canonical_optional,omitempty\" xml:\"canonical_optional,omitempty\"`")
	require.Contains(t, code, "CanonicalRequired *string `form:\"canonical_required,omitempty\" json:\"canonical_required,omitempty\" xml:\"canonical_required,omitempty\"`")
	require.Contains(t, code, "CanonicalOptional string  `form:\"canonical_optional\" json:\"canonical_optional\" xml:\"canonical_optional\"`")
	overridden := generatedHTTPFunction(t, code, "NewShowResponseBodyOverridden")
	require.Contains(t, overridden, "CanonicalRequired: res.CanonicalRequired")
	require.Contains(t, overridden, "body.CanonicalOptional = *res.CanonicalOptional")
}

func generatedHTTPFunction(t *testing.T, code, name string) string {
	t.Helper()
	start := bytes.Index([]byte(code), []byte("func "+name+"("))
	require.NotEqual(t, -1, start)
	end := bytes.Index([]byte(code[start:]), []byte("\n}\n"))
	require.NotEqual(t, -1, end)
	return code[start : start+end+3]
}

func TestServerTypes(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"server-mixed-payload-attrs", testdata.MixedPayloadInBodyDSL},
		{"server-multiple-methods", testdata.MultipleMethodsDSL},
		{"server-payload-extend-validate", testdata.PayloadExtendedValidateDSL},
		{"server-result-type-validate", testdata.ResultTypeValidateDSL},
		{"server-with-result-collection", testdata.ResultWithResultCollectionDSL},
		{"server-with-result-view", testdata.ResultWithResultViewDSL},
		{"server-empty-error-response-body", testdata.EmptyErrorResponseBodyDSL},
		{"server-with-error-custom-pkg", testdata.WithErrorCustomPkgDSL},
		{"server-body-custom-name", testdata.PayloadBodyCustomNameDSL},
		{"server-path-custom-name", testdata.PayloadPathCustomNameDSL},
		{"server-query-custom-name", testdata.PayloadQueryCustomNameDSL},
		{"server-header-custom-name", testdata.PayloadHeaderCustomNameDSL},
		{"server-cookie-custom-name", testdata.PayloadCookieCustomNameDSL},
		{"server-payload-with-validated-alias", testdata.PayloadWithValidatedAliasDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := serverType(genpkg, root.API.HTTP.Services[0], services)
			var buf bytes.Buffer
			for _, s := range fs.AllSections()[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			testutil.AssertGo(t, "testdata/golden/server_types_"+c.Name+".go.golden", code)
		})
	}
}
