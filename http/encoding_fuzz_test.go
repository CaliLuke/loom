package http

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type (
	// decodeFuzzBody is a representative generated request body type.
	decodeFuzzBody struct {
		Name  *string           `json:"name,omitempty" xml:"name,omitempty"`
		Count *int              `json:"count,omitempty" xml:"count,omitempty"`
		Tags  []string          `json:"tags,omitempty" xml:"tags,omitempty"`
		Attrs map[string]string `json:"attrs,omitempty" xml:"-"`
	}

	// decodeOutcome is how a generated server classifies a body decode.
	decodeOutcome string
)

const (
	decodeOK       decodeOutcome = "ok"
	decodeMissing  decodeOutcome = "missing"
	decodeInvalid  decodeOutcome = loom.DecodePayload
	decodeTooLarge decodeOutcome = loom.RequestBodyTooLarge
)

var decodeFuzzContentTypes = []string{
	"",
	"application/json",
	"application/json; charset=utf-8",
	"APPLICATION/JSON",
	"application/xml",
	"application/gob",
	"text/plain",
	"text/html; charset=utf-8",
	"application/x-unknown",
	"application/json;;",
}

// FuzzRequestDecoder decodes arbitrary bodies through RequestDecoder behind a
// RequestBodyPolicy, the path generated servers use, and checks that decoding
// never panics and that empty, JSON-whitespace, malformed, truncated, valid,
// and oversized bodies are classified the way generated code relies on.
func FuzzRequestDecoder(f *testing.F) {
	bodies := []string{
		``, ` `, " \t\r\n", `{}`, `{"name":"a","count":1}`, `{"name":"a","count":`,
		`{"name":"a"`, `{"name":`, `{`, `[`, `"`, `nul`, `null`, `true`, `1e999`,
		`{"name":1}`, `{"tags":["a",1]}`, `{"attrs":{"k":"v"}}`, `{"name":"a","name":"b"}`,
		"{\"name\":\"\xff\"}", `{} {}`, `{}x`, "\xef\xbb\xbf{}", `<decodeFuzzBody><name>a</name></decodeFuzzBody>`,
		`<a>`, `<?xml version="1.0"?>`, `<!-- only a comment -->`, "\x0c\xff\x81\x03\x01\x01",
	}
	for i, body := range bodies {
		f.Add([]byte(body), uint8(i%len(decodeFuzzContentTypes)), uint8(i%4), uint16(64))
	}
	f.Add([]byte(`{"name":"too long"}`), uint8(1), uint8(0), uint16(4))

	f.Fuzz(func(t *testing.T, body []byte, ctIndex, target uint8, limit uint16) {
		contentType := decodeFuzzContentTypes[int(ctIndex)%len(decodeFuzzContentTypes)]
		maxBytes := int64(limit%1024) + 1
		err := decodeFuzzRequest(t, body, contentType, target%4, maxBytes)
		for _, required := range []bool{true, false} {
			got := classifyGeneratedDecode(err, required)
			if int64(len(body)) > maxBytes && decodeFuzzReadsBody(contentType) {
				require.Equal(t, decodeTooLarge, got, "oversized body (%d > %d)", len(body), maxBytes)
				continue
			}
			require.NotEqual(t, decodeTooLarge, got, "body within limit reported too large")
			if !decodeFuzzIsJSON(contentType) {
				continue
			}
			switch {
			case jsonWhitespaceOnly(body):
				want := decodeMissing
				if !required {
					want = decodeOK
				}
				require.Equal(t, want, got, "whitespace-only JSON body %q", body)
			case target%4 == 0 && decodeFuzzValidJSON(body):
				// Syntactically valid JSON decodes into any unless a value is
				// out of range for its Go type, which is a semantic error.
				var semanticErr *json.SemanticError
				if got != decodeOK {
					require.ErrorAs(t, err, &semanticErr, "valid JSON body %q into any", body)
					require.Equal(t, decodeInvalid, got)
				}
			default:
				if got != decodeOK {
					require.Equal(t, decodeInvalid, got, "JSON body %q: %v", body, err)
				}
				if !decodeFuzzValidJSON(body) {
					require.Equal(t, decodeInvalid, got, "invalid JSON body %q accepted", body)
				}
			}
		}

		// The client-side response decoder shares the JSON seam and must never
		// report a non-empty body as missing.
		resp := &http.Response{
			Header: http.Header{"Content-Type": []string{contentType}},
			Body:   io.NopCloser(bytes.NewReader(body)),
		}
		var v any
		respErr := ResponseDecoder(resp).Decode(&v)
		if errors.Is(respErr, io.EOF) && decodeFuzzIsJSON(contentType) {
			require.True(t, jsonWhitespaceOnly(body), "response body %q decoded as EOF", body)
		}
	})
}

// decodeFuzzRequest runs RequestDecoder inside a RequestBodyPolicy handler
// and returns the Decode error.
func decodeFuzzRequest(t *testing.T, body []byte, contentType string, target uint8, maxBytes int64) error {
	t.Helper()
	policy, err := NewRequestBodyPolicy(maxBytes)
	require.NoError(t, err)
	var decodeErr error
	called := false
	handler := policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		dec := RequestDecoder(r)
		switch target {
		case 0:
			var v any
			decodeErr = dec.Decode(&v)
		case 1:
			var v decodeFuzzBody
			decodeErr = dec.Decode(&v)
		case 2:
			var v map[string]any
			decodeErr = dec.Decode(&v)
		default:
			var v string
			decodeErr = dec.Decode(&v)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
	require.True(t, called)
	return decodeErr
}

// classifyGeneratedDecode mirrors the generated server request decoder: io.EOF
// is a missing body (an error only when the body is required), Loom service
// errors pass through by name, and every other error is a decode failure.
func classifyGeneratedDecode(err error, required bool) decodeOutcome {
	if err == nil {
		return decodeOK
	}
	if errors.Is(err, io.EOF) {
		if required {
			return decodeMissing
		}
		return decodeOK
	}
	var serviceErr *loom.ServiceError
	if errors.As(err, &serviceErr) {
		return decodeOutcome(serviceErr.Name)
	}
	return decodeInvalid
}

func decodeFuzzIsJSON(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "", "application/json", "application/json; charset=utf-8":
		return true
	}
	return false
}

// decodeFuzzReadsBody reports whether RequestDecoder reads the body for
// contentType; unsupported media types fail without reading.
func decodeFuzzReadsBody(contentType string) bool {
	switch contentType {
	case "application/x-unknown", "application/json;;":
		return false
	}
	return true
}

// decodeFuzzValidJSON reports whether body holds exactly one valid JSON value
// under the RFC 7493 rules encoding/json/v2 enforces by default.
func decodeFuzzValidJSON(body []byte) bool {
	dec := jsontext.NewDecoder(bytes.NewReader(body))
	if _, err := dec.ReadValue(); err != nil {
		return false
	}
	_, err := dec.ReadToken()
	return errors.Is(err, io.EOF)
}
