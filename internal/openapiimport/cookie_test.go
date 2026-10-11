package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportCookieShapes(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		for _, test := range []struct {
			name, schema string
			accepted     bool
		}{
			{"string", `{"type":"string"}`, true},
			{"integer", `{"type":"integer"}`, true},
			{"array", `{"type":"array","items":{"type":"string"}}`, false},
			{"object", `{"type":"object","properties":{"value":{"type":"string"}}}`, false},
		} {
			for _, reference := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/reference=%t", version, test.name, reference), func(t *testing.T) {
					schema := test.schema
					components := ""
					if reference {
						schema = `{"$ref":"#/components/schemas/CookieValue"}`
						components = `,"components":{"schemas":{"CookieValue":` + test.schema + `}}`
					}
					source := []byte(fmt.Sprintf(`{"openapi":%q,"info":{"title":"Cookies","version":"1"},"paths":{"/items":{"get":{"operationId":"listItems","parameters":[{"name":"filter","in":"cookie","schema":%s}],"responses":{"204":{"description":"done"}}}}}%s}`, version, schema, components))
					document, diagnostics, err := Analyze(source)
					require.NoError(t, err)
					if !test.accepted {
						require.Contains(t, diagnostics.Error(), "cookie-parameter-shape")
						require.Contains(t, diagnostics.Error(), "#/operations/0/parameters/0/schema")
						fatal, _ := diagnostics.Classify(true)
						require.NotEmpty(t, fatal)
						_, err = Render(document, Options{PackageName: "design"})
						require.Error(t, err)
						return
					}
					require.Empty(t, diagnostics)
					rendered, err := Render(document, Options{PackageName: "design"})
					require.NoError(t, err)
					requireRenderedDesignEvaluates(t, rendered, 1)
					if reference && test.name == "string" {
						requireRenderedDesignGenerates(t, rendered)
					}
				})
			}
		}
	}
}
