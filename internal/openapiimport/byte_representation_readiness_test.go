package openapiimport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const importedByteRepresentationSource = `openapi: 3.1.1
info: {title: Encoded, version: "1"}
paths:
  /data:
    post:
      operationId: data
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Payload'}
      responses: {"204": {description: accepted}}
components:
  schemas:
    Encoded: {type: string, format: byte}
    Payload:
      type: object
      properties:
        lower: {type: string, format: byte, minLength: 4}
        upper: {type: string, format: byte, maxLength: 8}
        fallback: {type: string, format: byte, default: "aGk="}
        choice: {type: string, format: byte, enum: ["aGk="]}
        sample: {type: string, format: byte, example: "aGk="}
        encoded: {$ref: '#/components/schemas/Encoded'}
        nullable: {type: [string, "null"], format: byte}
        sequence: {type: array, items: {$ref: '#/components/schemas/Encoded'}}
        mapping: {type: object, additionalProperties: {$ref: '#/components/schemas/Encoded'}}
        patterned: {type: string, format: byte, pattern: "^aGk=$"}
        ordinary: {type: string}
`

const importedByteRepresentationHarness = `package imported_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svc "example.com/imported/gen/encoded"
	server "example.com/imported/gen/http/encoded/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestImportedByteWireContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
		code   string
		detail string
	}{
		{"encoded minimum", "{\"lower\":\"aGk=\"}", http.StatusNoContent, "", ""},
		{"below minimum", "{\"lower\":\"\"}", http.StatusBadRequest, "invalid_length", "lower"},
		{"encoded maximum", "{\"upper\":\"aGhpamtsbw==\"}", http.StatusBadRequest, "invalid_length", "upper"},
		{"encoded enum", "{\"choice\":\"aGk=\"}", http.StatusNoContent, "", ""},
		{"enum exact spelling", "{\"choice\":\"aGl=\"}", http.StatusBadRequest, "invalid_enum_value", "choice"},
		{"default", "{}", http.StatusNoContent, "", ""},
		{"supplied default empty", "{\"fallback\":\"\"}", http.StatusNoContent, "", ""},
		{"supplied default alias", "{\"fallback\":\"aGl=\"}", http.StatusNoContent, "", ""},
		{"empty", "{\"encoded\":\"\"}", http.StatusNoContent, "", ""},
		{"one byte", "{\"encoded\":\"Zg==\"}", http.StatusNoContent, "", ""},
		{"complete quantum", "{\"encoded\":\"Zm9v\"}", http.StatusNoContent, "", ""},
		{"alternate pad bits", "{\"encoded\":\"aGl=\"}", http.StatusNoContent, "", ""},
		{"missing padding", "{\"encoded\":\"aGk\"}", http.StatusBadRequest, "invalid_format", "encoded"},
		{"extra padding", "{\"encoded\":\"aGk==\"}", http.StatusBadRequest, "invalid_format", "encoded"},
		{"newline", "{\"encoded\":\"aGk=\\n\"}", http.StatusBadRequest, "invalid_format", "encoded"},
		{"URL alphabet", "{\"encoded\":\"__8=\"}", http.StatusBadRequest, "invalid_format", "encoded"},
		{"nullable null", "{\"nullable\":null}", http.StatusNoContent, "", ""},
		{"nullable string", "{\"nullable\":\"aGl=\"}", http.StatusNoContent, "", ""},
		{"nullable invalid", "{\"nullable\":\"?\"}", http.StatusBadRequest, "invalid_format", "nullable"},
		{"nested values", "{\"sequence\":[\"aGk=\",\"aGl=\"],\"mapping\":{\"key\":\"Zg==\"}}", http.StatusNoContent, "", ""},
		{"nested empty", "{\"sequence\":[],\"mapping\":{}}", http.StatusNoContent, "", ""},
		{"array invalid", "{\"sequence\":[\"?\"]}", http.StatusBadRequest, "invalid_format", "sequence"},
		{"map invalid", "{\"mapping\":{\"key\":\"?\"}}", http.StatusBadRequest, "invalid_format", "mapping"},
		{"pattern accepted", "{\"patterned\":\"aGk=\"}", http.StatusNoContent, "", ""},
		{"pattern rejected", "{\"patterned\":\"aGl=\"}", http.StatusBadRequest, "invalid_pattern", "patterned"},
		{"ordinary string", "{\"ordinary\":\"café\"}", http.StatusNoContent, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requestValue any
			mux := loomhttp.NewMuxer()
			endpoints := &svc.Endpoints{Data: func(_ context.Context, request any) (any, error) {
				requestValue = request
				return nil, nil
			}}
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			req := httptest.NewRequest(http.MethodPost, "/data", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			if tc.status == http.StatusBadRequest {
				require.Nil(t, requestValue)
				var problem map[string]any
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
				require.Equal(t, tc.code, problem["code"])
				if tc.code == "invalid_enum_value" {
					require.Contains(t, problem["detail"], tc.detail)
				} else {
					require.Equal(t, "validation error", problem["detail"])
				}
				return
			}
			require.NotNil(t, requestValue)
			wire, err := json.Marshal(requestValue)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(wire, &payload))
			body, ok := payload["body"].(map[string]any)
			require.True(t, ok, string(wire))
			var supplied map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.body), &supplied))
			if _, present := supplied["fallback"]; !present {
				require.Equal(t, "aGk=", body["fallback"], string(wire))
			}
			for name, value := range supplied {
				require.Equal(t, value, body[name], name)
			}
		})
	}
}
`

// TestImportedByteRepresentationWire checks source-string bounds and literals
// against a generated server without prescribing its internal Go carrier.
func TestImportedByteRepresentationWire(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(importedByteRepresentationSource))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	dir := requireRenderedDesignGenerates(t, design)
	requireImportedByteSpec(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "byte_contract_test.go"), []byte(importedByteRepresentationHarness), 0600))
	if output := os.Getenv("LOOM_IMPORTED_BYTE_OUTPUT"); output != "" {
		require.NoError(t, os.CopyFS(output, os.DirFS(dir)))
	}
	command := exec.Command("go", "test", "-mod=mod", "-race", "./...")
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

// TestImportedByteRepresentationPattern checks an encoded string constraint
// without prescribing the representation chosen by the importer.
func TestImportedByteRepresentationPattern(t *testing.T) {
	source := strings.Replace(importedByteRepresentationSource,
		"lower: {type: string, format: byte, minLength: 4}",
		`lower: {type: string, format: byte, pattern: "^aGk=$", example: "aGk="}`, 1)
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, design, 1)
}
