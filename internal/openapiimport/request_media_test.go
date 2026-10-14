package openapiimport

import (
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

const importedRequestMediaHarness = `package imported_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	client "example.com/imported/gen/http/pet_api/client"
	server "example.com/imported/gen/http/pet_api/server"
	api "example.com/imported/gen/pet_api"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestImportedRequestMedia(t *testing.T) {
	const media = "application/vnd.example+json; charset=utf-8"
	var calls atomic.Int32
	endpoints := &api.Endpoints{PetsCreate: func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		payload := value.(*api.PetsCreatePayload)
		if payload.Body.Name != "Milo" || payload.Body.Kind != "cat" {
			t.Errorf("decoded payload: %+v", payload.Body)
		}
		return &api.Pet{ID: 1, Name: payload.Body.Name, Status: 1}, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	policy, err := loomhttp.NewRequestBodyPolicy(128)
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(policy.Handler(mux))
	defer host.Close()
	encoder := func(r *http.Request) loomhttp.Encoder {
		if got := r.Header.Get("Content-Type"); got != media {
			t.Errorf("encoder media %q, want %q", got, media)
		}
		return loomhttp.RequestEncoder(r)
	}
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), encoder, loomhttp.ResponseDecoder, false)
	_, err = c.PetsCreate()(context.Background(), &api.PetsCreatePayload{Body: &api.NewPet{Name: "Milo", Kind: "cat"}})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("valid request invokes service %d times", calls.Load())
	}
	for _, tc := range []struct {
		media  string
		body   string
		status int
		code   string
		detail string
	}{
		{"application/unknown", ` + "`" + `{"name":"Milo"}` + "`" + `, 415, "unsupported_media_type", "unsupported media type application/unknown"},
		{"application/json;bad", ` + "`" + `{"name":"Milo"}` + "`" + `, 415, "unsupported_media_type", "unsupported media type application/json;bad"},
		{media, ` + "`" + `{"name":"Milo"}` + "`" + ` + strings.Repeat(" ", 129), 413, "request_too_large", "request body too large"},
	} {
		response, err := host.Client().Post(host.URL+"/pets", tc.media, strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		wire, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
		var problem loomhttp.ProblemResponse
		if err := json.Unmarshal(wire, &problem); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != tc.status || problem.Code != tc.code || problem.Detail != tc.detail {
			t.Errorf("media %q: status/code/detail %d/%s/%s, want %d/%s/%s", tc.media, response.StatusCode, problem.Code, problem.Detail, tc.status, tc.code, tc.detail)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("rejected requests invoked service: %d", calls.Load())
	}
}
`

func TestSingleJSONVendorRequestMediaTypeIsKept(t *testing.T) {
	source := []byte(`openapi: 3.0.3
info: {title: E, version: "1"}
paths:
  /a:
    patch:
      operationId: patchA
      requestBody:
        required: true
        content:
          application/merge-patch+json:
            schema: {type: object, properties: {a: {type: string}}}
      responses:
        "204": {description: ok}
`)
	doc, diags, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diags)
	design, err := Render(doc, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `ContentType("application/merge-patch+json")`)
}

func TestRequestMediaSupportedFixtureRoundTrip(t *testing.T) {
	const media = "application/vnd.example+json; charset=utf-8"
	var source map[string]any
	require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
	operation := source["paths"].(map[string]any)["/pets"].(map[string]any)["post"].(map[string]any)
	content := operation["requestBody"].(map[string]any)["content"].(map[string]any)
	content[media] = content["application/json"]
	delete(content, "application/json")
	encoded, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(encoded)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `ContentType("`+media+`")`)
	directory := requireRenderedDesignGenerates(t, design)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "request_media_wire_test.go"), []byte(importedRequestMediaHarness), 0600))
	command := exec.Command("go", "test", "-race", "-count=1", ".")
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))

	contract := readGeneratedOpenAPIContract(t, directory)
	got := operationFromImportedSpec(t, contract, "/pets", "post")["requestBody"].(map[string]any)["content"].(map[string]any)
	require.Len(t, got, 1)
	require.Contains(t, got, media)
	wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(wire)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
}

func TestRequestMediaAdmissionUsesSharedCodec(t *testing.T) {
	for _, media := range []string{"application/json;bad", "application/*+json", "not-a-media+json", "application/unknown"} {
		t.Run(media, func(t *testing.T) {
			document := &Document{
				Title: "Media",
				Operations: []Operation{{
					Method: "POST", Path: "/send", OperationID: "send",
					RequestBody: &RequestBody{ContentTypes: []string{media}, Schema: &Schema{Type: "string"}},
					Responses:   []StatusResponse{{Status: "204", Response: Response{Description: "ok"}}},
				}},
			}
			_, err := Render(document, Options{PackageName: "design"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "not renderable")
		})
	}
}
