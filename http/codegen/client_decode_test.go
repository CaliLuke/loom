package codegen

import (
	"github.com/CaliLuke/loom/codegen/testutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientDecode(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"empty-body", testdata.EmptyServerResponseDSL},
		{"body-result-multiple-views", testdata.ResultBodyMultipleViewsDSL},
		{"empty-body-result-multiple-views", testdata.EmptyBodyResultMultipleViewsDSL},
		{"explicit-body-primitive-result", testdata.ExplicitBodyPrimitiveResultMultipleViewsDSL},
		{"explicit-body-result-multiple-views", testdata.ExplicitBodyUserResultMultipleViewsDSL},
		{"explicit-body-result-collection", testdata.ExplicitBodyResultCollectionDSL},
		{"tag-result-multiple-views", testdata.ResultMultipleViewsTagDSL},
		{"empty-server-response-with-tags", testdata.EmptyServerResponseWithTagsDSL},
		{"header-string-implicit", testdata.ResultHeaderStringImplicitDSL},
		{"header-string-array", testdata.ResultHeaderStringArrayDSL},
		{"header-string-array-validate", testdata.ResultHeaderStringArrayValidateDSL},
		{"header-array", testdata.ResultHeaderArrayDSL},
		{"header-array-validate", testdata.ResultHeaderArrayValidateDSL},
		{"with-headers-dsl", testdata.WithHeadersBlockDSL},
		{"with-headers-dsl-viewed-result", testdata.WithHeadersBlockViewedResultDSL},
		{"validate-error-response-type", testdata.ValidateErrorResponseTypeDSL},
		{"empty-error-response-body", testdata.EmptyErrorResponseBodyDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ClientFiles("", services)
			require.Len(t, fs, 2)
			sections := fs[1].AllSections()
			require.Greater(t, len(sections), 2)
			code := codegen.SectionCode(t, sections[2])
			testutil.AssertGo(t, "testdata/golden/client_decode_"+c.Name+".go.golden", code)
		})
	}
}

func TestClientDecodeUnexpectedResponseBodyIsBounded(t *testing.T) {
	code := clientDecodeSectionCode(t, testdata.EmptyServerResponseDSL)

	require.Contains(t, code, "body, err := loomhttp.ReadUnexpectedResponseBody(resp)")
	require.Contains(t, code, "if err != nil {")
	require.NotContains(t, code, "body, _ := io.ReadAll(resp.Body)")
}

func TestClientDecodeRestoreBodyIsBounded(t *testing.T) {
	code := clientDecodeSectionCode(t, testdata.EmptyServerResponseDSL)

	require.Contains(t, code, "b, err := loomhttp.ReadResponseBody(resp)")
	require.NotContains(t, code, "b, err := io.ReadAll(resp.Body)")
}

func TestClientDecodePropagatesViewedResultConstructorError(t *testing.T) {
	code := clientDecodeSectionCode(t, testdata.ResultBodyMultipleViewsDSL)

	require.Contains(t, code, `res, err := servicebodymultipleview.NewResulttypemultipleviews(vres)`)
	require.Contains(t, code, `return nil, loomhttp.ErrValidationError("ServiceBodyMultipleView", "MethodBodyMultipleView", err)`)
	require.NotContains(t, code, `res := servicebodymultipleview.NewResulttypemultipleviews(vres)`)
}

func clientDecodeSectionCode(t *testing.T, dsl func()) string {
	t.Helper()
	root := RunHTTPDSL(t, dsl)
	services := CreateHTTPServices(root)
	fs := ClientFiles("", services)
	require.Len(t, fs, 2)
	sections := fs[1].AllSections()
	require.Greater(t, len(sections), 2)
	return codegen.SectionCode(t, sections[2])
}
