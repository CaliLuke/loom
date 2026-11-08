package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestMultipartJSONQueryConversionReturn requires the JSON query conversion
// rendered inside a custom multipart decoder, whose function returns only an
// error, to leave its return to that section and to classify a failed
// conversion, an authored default that cannot be materialized, as the shared
// parameter default fault.
func TestMultipartJSONQueryConversionReturn(t *testing.T) {
	root := RunHTTPDSL(t, multipartJSONQueryConversionDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/uploads/gen", root)
	path := filepath.Join(dir, "gen", "http", "uploads", "server", "encode_decode.go")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	server := string(data)
	require.Empty(t, returnArityMismatches(t, path))
	for _, pkg := range []string{"server", "client"} {
		require.Empty(t, undefinedPackageIdentifiers(t, filepath.Join(dir, "gen", "http", "uploads", pkg)), "%s constructors copy the service-owned value", pkg)
	}
	for _, decoder := range []string{"func NewUploadsUploadDecoder(", "func NewUploadsBrokenDecoder("} {
		start := strings.Index(server, decoder)
		require.NotEqual(t, -1, start, decoder)
		body := server[start:]
		if end := strings.Index(body[1:], "\nfunc "); end >= 0 {
			body = body[:end+1]
		}
		require.Contains(t, body, "transformErr := new(error)", "%s converts with a fallible plan", decoder)
		require.Contains(t, body, `return loom.Fault("%s", "parameter pets: invalid default")`, decoder)
	}
}

// TestJSONQueryCustomMemberConstructorClosure requires constructors of a JSON
// query map of service objects with custom text members to copy the
// service-owned value instead of referencing an unpublished object helper.
func TestJSONQueryCustomMemberConstructorClosure(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		pet := Type("Pet", func() {
			Attribute("name", String)
			Attribute("tag", String, func() {
				Meta("struct:field:type", "label.Label", "example.com/pets/label")
			})
			Required("name")
		})
		Service("pets", func() {
			Method("list", func() {
				Payload(func() {
					Attribute("pets", MapOf(String, pet))
				})
				HTTP(func() {
					GET("/pets")
					Param("pets")
					Response(StatusNoContent)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/pets/gen", root)
	for _, pkg := range []string{"server", "client"} {
		require.Empty(t, undefinedPackageIdentifiers(t, filepath.Join(dir, "gen", "http", "pets", pkg)), pkg)
	}
}

// TestMultipartJSONQueryConversionGenerated compiles and runs a custom
// multipart endpoint whose JSON query map applies an authored custom-text
// member default during conversion.
func TestMultipartJSONQueryConversionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, multipartJSONQueryConversionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/uploads", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "label"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "label", "label.go"), []byte(jsonQueryLabelSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "uploads_test.go"), []byte(multipartJSONQueryConversionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-count=1", "-v", "./..."))
}

// multipartJSONQueryConversionDSL selects the custom multipart codec through a
// custom text body field, so the multipart decoder renders the request
// elements, and maps a JSON query map whose object members default.
func multipartJSONQueryConversionDSL() {
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/uploads/label")
	}
	pet := Type("Pet", func() {
		Attribute("name", String)
		Attribute("tag", String, func() {
			label()
			Default("label:pet")
		})
		Required("name")
	})
	faultyPet := Type("FaultyPet", func() {
		Attribute("name", String)
		Attribute("tag", String, func() {
			label()
			Default("oops")
		})
		Required("name")
	})
	Service("uploads", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("note", String, label)
				Attribute("pets", MapOf(String, pet))
			})
			HTTP(func() {
				POST("/upload")
				Param("pets")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("broken", func() {
			Payload(func() {
				Attribute("note", String, label)
				Attribute("pets", MapOf(String, faultyPet))
			})
			HTTP(func() {
				POST("/broken")
				Param("pets")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
	})
}

const multipartJSONQueryConversionHarness = `package uploads_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"testing"

	server "example.com/uploads/gen/http/uploads/server"
	service "example.com/uploads/gen/uploads"
	"example.com/uploads/label"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, path, pets string) (*httptest.ResponseRecorder, []any) {
	t.Helper()
	var received []any
	endpoint := func(_ context.Context, payload any) (any, error) {
		received = append(received, payload)
		return nil, nil
	}
	upload := func(_ *multipart.Reader, p **service.UploadPayload) error {
		*p = &service.UploadPayload{}
		return nil
	}
	broken := func(_ *multipart.Reader, p **service.BrokenPayload) error {
		*p = &service.BrokenPayload{}
		return nil
	}
	mux := loomhttp.NewMuxer()
	endpoints := &service.Endpoints{Upload: endpoint, Broken: endpoint}
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, upload, broken))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("note", "label:n"))
	require.NoError(t, writer.Close())
	target := path
	if pets != "" {
		target += "?" + url.Values{"pets": {pets}}.Encode()
	}
	request := httptest.NewRequest("POST", target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response, received
}

func TestUploadConversion(t *testing.T) {
	for _, test := range []struct {
		name string
		pets string
		want map[string]*service.Pet
	}{
		{"absent map stays nil", "", nil},
		{"absent member uses its default", "{\"a\":{\"name\":\"x\"}}", map[string]*service.Pet{"a": {Name: "x", Tag: label.Label{Text: "pet"}}}},
		{"supplied member is kept", "{\"a\":{\"name\":\"x\",\"tag\":\"label:t\"}}", map[string]*service.Pet{"a": {Name: "x", Tag: label.Label{Text: "t"}}}},
		{"empty map stays empty", "{}", map[string]*service.Pet{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, received := serve(t, "/upload", test.pets)
			require.Equal(t, 204, response.Code, response.Body.String())
			require.Len(t, received, 1)
			assert.Equal(t, test.want, received[0].(*service.UploadPayload).Pets)
		})
	}
}

func TestConversionErrors(t *testing.T) {
	for _, test := range []struct {
		name, path, pets string
		status           int
		code             string
	}{
		{"invalid supplied member is a client error", "/upload", "{\"a\":{\"name\":\"x\",\"tag\":\"bad\"}}", 400, "decode_payload"},
		{"faulty authored default is a server fault", "/broken", "{\"a\":{\"name\":\"x\"}}", 500, "fault"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, received := serve(t, test.path, test.pets)
			require.Equal(t, test.status, response.Code, response.Body.String())
			require.Empty(t, received)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			assert.Equal(t, test.code, problem.Code, response.Body.String())
			assert.Equal(t, test.status == 500, problem.Fault, response.Body.String())
		})
	}
}

func TestFaultyDefaultOnlyWhenNeeded(t *testing.T) {
	for _, pets := range []string{"", "{\"a\":{\"name\":\"x\",\"tag\":\"label:t\"}}"} {
		response, received := serve(t, "/broken", pets)
		require.Equal(t, 204, response.Code, response.Body.String())
		require.Len(t, received, 1)
	}
}
`
