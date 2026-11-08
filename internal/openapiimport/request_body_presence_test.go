package openapiimport

import (
	"encoding/json/v2"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyPresenceEmission(t *testing.T) {
	for _, tc := range []struct {
		name, fixture, path string
		required            bool
		requiredMember      bool
		selected            bool
		json                bool
		mode                requestBodyMode
		envelope            bool
		optionalMarker      bool
	}{
		{name: "optional multipart", path: "/multipart", mode: requestBodyMultipart, optionalMarker: true},
		{name: "required multipart", path: "/multipart", required: true, mode: requestBodyMultipart},
		{name: "optional multipart required member", path: "/multipart", requiredMember: true, mode: requestBodyMultipart, optionalMarker: true},
		{name: "optional form", path: "/form", mode: requestBodyForm, optionalMarker: true},
		{name: "required form", path: "/form", required: true, mode: requestBodyForm},
		{name: "optional union", fixture: "oneof/39-multipart-empty.json", path: "/value", mode: requestBodyMultipart, optionalMarker: true},
		{name: "required union", fixture: "oneof/39-multipart-empty.json", path: "/value", required: true, mode: requestBodyMultipart},
		{name: "selected optional union", fixture: "oneof/39-multipart-empty.json", path: "/value", selected: true, mode: requestBodyMultipart, envelope: true},
		{name: "selected required union", fixture: "oneof/39-multipart-empty.json", path: "/value", required: true, selected: true, mode: requestBodyMultipart, envelope: true},
		{name: "optional raw multipart map", fixture: "map", path: "/maps", mode: requestBodyRaw},
		{name: "optional JSON", path: "/multipart", json: true, mode: requestBodyJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := tc.fixture
			if fixture == "" {
				fixture = "discriminator/media.yaml"
			}
			var raw []byte
			if fixture == "map" {
				raw = mediaRootMapSource(t, "multipart/form-data", false, 0, false, map[string]any{"type": "string"})
			} else {
				raw = readFixture(t, fixture)
			}
			var input map[string]any
			require.NoError(t, yaml.Unmarshal(raw, &input))
			item := input["paths"].(map[string]any)[tc.path].(map[string]any)
			operation := item["post"].(map[string]any)
			body := operation["requestBody"].(map[string]any)
			body["required"] = tc.required
			if tc.requiredMember {
				content := body["content"].(map[string]any)
				schema := content["multipart/form-data"].(map[string]any)["schema"].(map[string]any)
				schema["required"] = []any{"event"}
			}
			if tc.selected {
				operation["parameters"] = []any{map[string]any{
					"name": "trace", "in": "header", "schema": map[string]any{"type": "string"},
				}}
			}
			if tc.json {
				media := body["content"].(map[string]any)["multipart/form-data"].(map[string]any)
				delete(media, "encoding")
				body["content"] = map[string]any{"application/json": media}
			}
			input["paths"] = map[string]any{tc.path: item}
			source, err := json.Marshal(input, json.Deterministic(true))
			require.NoError(t, err)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			plan := planDocument(document)
			require.Empty(t, plan.diagnostics)
			require.Len(t, plan.operations, 1)
			selected := plan.operations[0].body
			require.Equal(t, tc.mode, selected.mode)
			require.Equal(t, tc.envelope, selected.envelope)
			require.Equal(t, tc.required, selected.body.Required)
			if tc.requiredMember {
				require.Contains(t, selected.body.Schema.Required, "event")
			}
			renderer := renderer{document: plan.document, schemas: plan.schemas, references: plan.references}
			require.NoError(t, renderer.operationRequestBodyHTTP(selected, "#/paths"+tc.path+"/post"))
			if tc.optionalMarker {
				require.Contains(t, renderer.builder.String(), "OptionalRequestBody()")
			} else {
				require.NotContains(t, renderer.builder.String(), "OptionalRequestBody()")
			}
			if tc.envelope {
				payload := renderer.childRenderer()
				required, err := payload.payloadBody(selected, "#/requestBody")
				require.NoError(t, err)
				require.Equal(t, tc.required, required, "selected payload member owns body presence")
			}
			if tc.mode == requestBodyRaw {
				require.Contains(t, renderer.builder.String(), "SkipRequestBodyEncodeDecode()")
				require.Contains(t, renderer.builder.String(), "OpenAPIRequestBodyTypes(")
				require.Contains(t, renderer.builder.String(), "false")
			}
		})
	}
}
