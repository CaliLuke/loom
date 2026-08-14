package openapiv3

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestBuildDocumentPreservesExternalDocs(t *testing.T) {
	root := expr.RunDSL(t, testdata.ServerCORSPolicyDSL)
	root.API.Docs = &expr.DocsExpr{
		Description: "API integration guide.",
		URL:         "https://example.com/docs/api",
	}
	service := root.API.HTTP.Services[0].ServiceExpr
	service.Docs = &expr.DocsExpr{
		Description: "Service integration guide.",
		URL:         "https://example.com/docs/service",
	}
	service.Methods[0].Docs = &expr.DocsExpr{
		Description: "Operation integration guide.",
		URL:         "https://example.com/docs/operation",
	}

	document := New(root)
	require.NotNil(t, document)

	var serviceTag *openapi.Tag
	for _, tag := range document.Tags {
		if tag.Name == service.Name {
			serviceTag = tag
			break
		}
	}
	require.NotNil(t, serviceTag)

	tests := []struct {
		name string
		got  *openapi.ExternalDocs
		want *openapi.ExternalDocs
	}{
		{
			name: "API",
			got:  document.ExternalDocs,
			want: &openapi.ExternalDocs{
				Description: "API integration guide.",
				URL:         "https://example.com/docs/api",
			},
		},
		{
			name: "service",
			got:  serviceTag.ExternalDocs,
			want: &openapi.ExternalDocs{
				Description: "Service integration guide.",
				URL:         "https://example.com/docs/service",
			},
		},
		{
			name: "method",
			got:  document.Paths["/items"].Get.ExternalDocs,
			want: &openapi.ExternalDocs{
				Description: "Operation integration guide.",
				URL:         "https://example.com/docs/operation",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.got)
		})
	}

	jsonDocument, err := json.Marshal(document)
	require.NoError(t, err)
	require.Equal(t, 3, strings.Count(string(jsonDocument), `"externalDocs"`))
	require.Contains(t, string(jsonDocument), `"url":"https://example.com/docs/api"`)

	yamlDocument, err := yaml.Marshal(document)
	require.NoError(t, err)
	require.Equal(t, 3, strings.Count(string(yamlDocument), "externalDocs:"))
	require.Contains(t, string(yamlDocument), "url: https://example.com/docs/api")
}
