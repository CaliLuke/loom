package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestClientResponseCodeProjectsSingleViewOneOfResults regresses HTTP code
// generation around single-view ResultType responses that drop a OneOf field.
// Client validation, union collection, body type emission, and decode/init
// must all derive from the same effective projected transport body.
func TestClientResponseCodeProjectsSingleViewOneOfResults(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{name: "single-result", dsl: testdata.OneOfResultSingleViewDSL},
		{name: "collection-result", dsl: testdata.OneOfResultCollectionSingleViewDSL},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			typeCode := renderClientTypesCode(t, c.dsl)
			decodeCode := renderClientDecodeCode(t, c.dsl)

			require.Contains(t, typeCode, "AnimalView")
			require.NotContains(t, typeCode, "CatDetailsResponseBody")
			require.NotContains(t, typeCode, "DogDetailsResponseBody")
			require.NotContains(t, typeCode, "BirdDetailsResponseBody")
			require.NotContains(t, typeCode, "FishDetailsResponseBody")
			require.NotContains(t, typeCode, `json:"details,omitempty"`)

			require.NotContains(t, decodeCode, "body.Details")
			require.NotContains(t, decodeCode, "CatDetailsResponseBody")
			require.NotContains(t, decodeCode, "DogDetailsResponseBody")
			require.NotContains(t, decodeCode, "BirdDetailsResponseBody")
			require.NotContains(t, decodeCode, "FishDetailsResponseBody")
		})
	}
}

// renderClientTypesCode renders the client type file for a single-service DSL.
func renderClientTypesCode(t *testing.T, dsl func()) string {
	t.Helper()

	const genpkg = "gen"

	root := RunHTTPDSL(t, dsl)
	services := CreateHTTPServices(root)
	fs := clientType(genpkg, root.API.HTTP.Services[0], make(map[string]struct{}), services)

	var buf bytes.Buffer
	for _, s := range fs.AllSections()[1:] {
		require.NoError(t, s.Write(&buf))
	}

	return codegen.FormatTestCode(t, "package foo\n"+buf.String())
}

// renderClientDecodeCode renders the client decode section for a single-service
// DSL and returns the generated code.
func renderClientDecodeCode(t *testing.T, dsl func()) string {
	t.Helper()

	const genpkg = "gen"

	root := RunHTTPDSL(t, dsl)
	services := CreateHTTPServices(root)
	fs := ClientFiles(genpkg, services)
	require.GreaterOrEqual(t, len(fs), 2)

	sections := fs[1].Section("response-decoder")
	require.NotEmpty(t, sections)

	return codegen.SectionsCode(t, sections)
}
