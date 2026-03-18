package codegen

import (
	"bytes"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/dsl"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
)

func TestCookieAPIKeySecurity(t *testing.T) {
	t.Run("endpoint requirement uses cookie transport", func(t *testing.T) {
		root := RunHTTPDSL(t, cookieAPIKeySecurityDSL)
		endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
		require.Len(t, endpoint.Requirements, 1)
		require.Len(t, endpoint.Requirements[0].Schemes, 1)

		scheme := endpoint.Requirements[0].Schemes[0]
		require.Equal(t, "cookie", scheme.In)
		require.Equal(t, "__Host-ak_session", scheme.Name)

		headers := expr.AsObject(endpoint.Headers.Type)
		require.Zero(t, len(*headers), "cookie-backed api key must not synthesize an Authorization header")
	})

	t.Run("openapi uses cookie security scheme", func(t *testing.T) {
		root := RunHTTPDSL(t, cookieAPIKeySecurityDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
		doc := parseOpenAPIV3Document(t, v3JSON)
		require.NotNil(t, doc.Components)
		require.NotNil(t, doc.Components.SecuritySchemes)
		require.Equal(t, 1, doc.Components.SecuritySchemes.Len())

		pathItem, ok := doc.Paths.PathItems.Get("/auth/profile")
		require.True(t, ok)
		require.NotNil(t, pathItem)
		require.NotNil(t, pathItem.Get)
		require.Len(t, pathItem.Get.Security, 1)

		for name, scheme := range doc.Components.SecuritySchemes.FromOldest() {
			require.NotNil(t, scheme, name)
			require.Equal(t, "apiKey", scheme.Type, name)
			require.Equal(t, "cookie", scheme.In, name)
			require.Equal(t, "__Host-ak_session", scheme.Name, name)

			requirement, found := pathItem.Get.Security[0].Requirements.Get(name)
			require.True(t, found, name)
			require.Empty(t, requirement, name)
		}
	})

	t.Run("http codegen does not duplicate cookie-backed auth fields", func(t *testing.T) {
		root := RunHTTPDSL(t, cookieAPIKeySecurityDSL)
		services := CreateHTTPServices(root)

		serverTypes := serverType("gen", root.API.HTTP.Services[0], services)
		var serverTypesBuf bytes.Buffer
		for _, section := range serverTypes.SectionTemplates[1:] {
			require.NoError(t, section.Write(&serverTypesBuf))
		}
		serverTypesCode := codegen.FormatTestCode(t, "package foo\n"+serverTypesBuf.String())
		require.Contains(t, serverTypesCode, "func NewProfilePayload(browserSession string)")
		require.NotContains(t, serverTypesCode, "browserSession *string, browserSession *string")
		require.NotContains(t, serverTypesCode, "browserSession string, browserSession string")

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverDecode := codegen.SectionCode(t, serverFiles[1].SectionTemplates[2])
		require.Contains(t, serverDecode, `r.Cookie("__Host-ak_session")`)
		require.NotContains(t, serverDecode, "Authorization")
		require.NotContains(t, serverDecode, "browserSession *string, browserSession *string")
		require.NotContains(t, serverDecode, "browserSession string, browserSession string")

		clientFiles := ClientFiles("", services)
		require.Len(t, clientFiles, 2)
		clientEncode := codegen.SectionCode(t, clientFiles[1].SectionTemplates[2])
		require.Contains(t, clientEncode, `req.AddCookie(&http.Cookie{`)
		require.Contains(t, clientEncode, `Name:  "__Host-ak_session"`)
		require.NotContains(t, clientEncode, "Authorization")
	})
}

func renderOpenAPIJSON(
	t *testing.T,
	build func(*expr.RootExpr) ([]*codegen.File, error),
	root *expr.RootExpr,
) []byte {
	t.Helper()

	files, err := build(root)
	require.NoError(t, err)
	for _, f := range files {
		if filepath.Ext(f.Path) != ".json" {
			continue
		}
		require.Len(t, f.SectionTemplates, 1)
		section := f.SectionTemplates[0]
		require.NotEmpty(t, section.Source)
		require.NotNil(t, section.Data)

		var buf bytes.Buffer
		tmpl := template.Must(template.New("openapi").Funcs(section.FuncMap).Parse(section.Source))
		require.NoError(t, tmpl.Execute(&buf, section.Data))
		return buf.Bytes()
	}

	t.Fatalf("no JSON OpenAPI file generated")
	return nil
}

func parseOpenAPIV3Document(t *testing.T, spec []byte) *v3.Document {
	t.Helper()

	parsed, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	require.Equal(t, openapiv3.OpenAPIVersion, parsed.GetVersion())

	model, err := parsed.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model)
	require.NotNil(t, model.Model.Paths)

	return &model.Model
}

var cookieAPIKeySecurityDSL = func() {
	var browserSessionCookie = dsl.APIKeySecurity("browser_session_cookie", func() {
		dsl.Description("Browser session cookie")
	})

	dsl.Service("cookieSecurity", func() {
		dsl.Method("profile", func() {
			dsl.Security(browserSessionCookie)
			dsl.Payload(func() {
				dsl.APIKey("browser_session_cookie", "browser_session", dsl.String, func() {
					dsl.Description("Opaque browser session cookie")
				})
				dsl.Required("browser_session")
			})
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.GET("/auth/profile")
				dsl.Cookie("browser_session:__Host-ak_session")
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}
