package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestParameterCodecExport(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ParameterStylesDSL)
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	for _, tc := range []struct {
		path, location, style string
		explode               *bool
	}{
		{"/PathSimple/{value}", "path", "", nil},
		{"/PathMatrix/{value}", "path", "matrix", nil},
		{"/PathLabel/{value}", "path", "label", nil},
		{"/PathMatrixList/{value}", "path", "matrix", boolPtr(true)},
		{"/QueryList", "query", "", boolPtr(false)},
		{"/QueryObject", "query", "deepObject", nil},
		{"/HeaderObject", "header", "", boolPtr(true)},
		{"/CookieScalar", "cookie", "cookie", nil},
		{"/CookieList", "cookie", "cookie", boolPtr(false)},
		{"/CookieForm", "cookie", "", boolPtr(false)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ref := doc.Paths[tc.path].Operations["GET"].Parameters[0]
			if ref.Ref != "" {
				ref = doc.Components.Parameters[strings.TrimPrefix(ref.Ref, "#/components/parameters/")]
			}
			require.Equal(t, tc.location, ref.Value.In)
			require.Equal(t, tc.style, ref.Value.Style)
			require.Equal(t, tc.explode, ref.Value.Explode)
		})
	}
	jsonParam := doc.Paths["/QueryJSON"].Operations["GET"].Parameters[0]
	if jsonParam.Ref != "" {
		jsonParam = doc.Components.Parameters[strings.TrimPrefix(jsonParam.Ref, "#/components/parameters/")]
	}
	require.Contains(t, jsonParam.Value.Content, "application/json")
	require.Nil(t, jsonParam.Value.Schema)
	form := doc.Paths["/FormFields"].Operations["POST"].RequestBody.Value.Content["application/x-www-form-urlencoded"]
	require.Equal(t, []string{"form"}, form.Metadata["openapi:encoding:facets:style"])
	require.Equal(t, []string{"form"}, form.Metadata["openapi:encoding:flags:style"])
	require.Equal(t, []string{"form"}, form.Metadata["openapi:encoding:attributes:style"])
	require.NotContains(t, form.Metadata, "openapi:encoding:attributes:contentType")
	require.Equal(t, []string{"application/json"}, form.Metadata["openapi:encoding:jsonattrs:contentType"])
	response := doc.Paths["/ResponseMap"].Operations["GET"].Responses["204"].Value
	require.NotNil(t, response)
	require.Equal(t, "", response.Headers["X-Labels"].Value.Style)
	require.Equal(t, boolPtr(true), response.Headers["X-Labels"].Value.Explode)
}

func TestCookieFormEquivalence31(t *testing.T) {
	unreserved := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"compact", "full"}}}
	integer := &expr.AttributeExpr{Type: expr.Int}
	unrestricted := &expr.AttributeExpr{Type: expr.String}
	for _, tc := range []struct {
		name          string
		att           *expr.AttributeExpr
		explode, want bool
	}{
		{"integer", integer, true, true},
		{"enum", unreserved, true, true},
		{"uuid", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Format: "uuid"}}, true, true},
		{"date", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Format: "date"}}, true, true},
		{"string", unrestricted, true, false},
		{"integer map keys", &expr.AttributeExpr{Type: &expr.Map{KeyType: integer, ElemType: unreserved}}, false, true},
		{"unrestricted keys", &expr.AttributeExpr{Type: &expr.Map{KeyType: unrestricted, ElemType: integer}}, false, false},
		{"object", &expr.AttributeExpr{Type: &expr.Object{{Name: "count", Attribute: integer}}}, false, true},
		{"reserved property", &expr.AttributeExpr{Type: &expr.Object{{Name: "a/b", Attribute: integer}}}, false, false},
		{"exploded collection", &expr.AttributeExpr{Type: &expr.Array{ElemType: integer}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shape := expr.HTTPValueShape(tc.att)
			codec := httpencoding.Codec{Style: httpencoding.CookieStyle, Shape: shape.Shape, Explode: tc.explode}
			require.Equal(t, tc.want, cookieFormEquivalent31(codec, shape))
		})
	}
}

func TestValidateURLFieldEncodingAllLocations(t *testing.T) {
	for _, location := range []string{"path", "query", "header", "cookie"} {
		t.Run(location, func(t *testing.T) {
			parameter := &transportir.Parameter{In: location, HTTPName: "value", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"openapi:style": {"invalid"}}}}
			request := &transportir.Request{}
			switch location {
			case "path":
				request.PathParams = []*transportir.Parameter{parameter}
			case "query":
				request.QueryParams = []*transportir.Parameter{parameter}
			case "header":
				request.Headers = []*transportir.Parameter{parameter}
			case "cookie":
				request.Cookies = []*transportir.Parameter{parameter}
				parameter.Attribute.Meta["openapi:style"] = []string{"matrix"}
			}
			err := validateURLFieldEncoding(&transportir.Endpoint{Request: request})
			require.ErrorContains(t, err, location+" parameter \"value\"")
		})
	}
}
