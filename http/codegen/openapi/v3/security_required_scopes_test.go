package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestRenderedOAuthRequiredScopeOwnership(t *testing.T) {
	for _, version := range []string{"3.1", "3.2"} {
		t.Run(version, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, servicetestdata.SecurityOAuthScopesDSL)
			root.API.Meta = expr.MetaExpr{"openapi:version": {version}}
			spec := openapiv3.New(root)
			require.Equal(t, []map[string][]string{{"read": {"shared", "read"}, "write": {"shared", "write"}}}, spec.Paths["/oauth"].Get.Security)
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			wantVersion := openapiv3.OpenAPIVersion
			if version == "3.1" {
				wantVersion = openapiv3.OpenAPICompatibilityVersion
			}
			for _, file := range files {
				data := renderSection(t, file.AllSections()[0])
				validateOpenAPIVersion(t, data.Bytes(), wantVersion)
			}
		})
	}
}
