package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

var (
	testString = "test string"
)

func TestRequestEncoder(t *testing.T) {
	const (
		ct      = "Content-Type"
		ctJSON  = "application/json"
		ctOther = "<other>"
		wantT   = "*json.Encoder"
	)
	cases := []struct {
		name      string
		requestCT string
		wantCT    string
	}{
		{"no ct", "", ctJSON},
		{"json ct", ctJSON, ctJSON},
		{"other ct", ctOther, ctOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if c.requestCT != "" {
				r.Header.Set(ct, c.requestCT)
			}

			_ = RequestEncoder(r)

			assert.Equal(t, c.wantCT, r.Header.Get(ct))
		})
	}
}

func TestRequestEncoderGetBody(t *testing.T) {
	r := &http.Request{Header: http.Header{}}
	encoder := RequestEncoder(r)

	_, err := r.Body.Read(nil)
	assert.Error(t, err, "request Body should error (but not panic) if read before data is encoded")

	_, err = r.GetBody()
	assert.Error(t, err, "request GetBody should error (but not panic) if read before data is encoded")

	err = encoder.Encode("body")
	require.NoError(t, err)

	bodyContents, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, `"body"`, string(bodyContents))

	newBody, err := r.GetBody()
	require.NoError(t, err)

	newBodyContents, err := io.ReadAll(newBody)
	require.NoError(t, err)
	assert.Equal(t, bodyContents, newBodyContents)
}

func TestRequestDecoder(t *testing.T) {
	const (
		ct           = "Content-Type"
		ctJSON       = "application/json"
		ctXML        = "application/xml"
		ctGob        = "application/gob"
		unsupportedT = "*http.unsupportedDecoder"
		jsonT        = "*json.Decoder"
		xmlT         = "*xml.Decoder"
		gobT         = "*gob.Decoder"
	)
	cases := []struct {
		name      string
		requestCT string
		wantCT    string
	}{
		{"no ct", "", jsonT},
		{"unsupported ct", "application/foo", unsupportedT},
		{"json ct", ctJSON, jsonT},
		{"xml ct", ctXML, xmlT},
		{"gob ct", ctGob, gobT},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if c.requestCT != "" {
				r.Header.Set(ct, c.requestCT)
			}

			decoder := RequestDecoder(r)

			assert.Equal(t, c.wantCT, fmt.Sprintf("%T", decoder))
		})
	}
}

func TestUnsupportedDecoder(t *testing.T) {
	// Write the response produced when writing the error returned the
	// unsupported decoder to validate the response status code.
	w := httptest.NewRecorder()
	decoder := &unsupportedDecoder{"application/foo"}
	err := decoder.Decode(nil)
	require.Error(t, err)
	encoder := ErrorEncoder(ResponseEncoder, nil)

	err = encoder(context.Background(), w, err)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestResponseEncoder(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		acceptType  string
		encoderType string
	}{
		{"no ct, no at", "", "", "*json.Encoder"},
		{"no ct, at json", "", "application/json", "*json.Encoder"},
		{"no ct, at xml", "", "application/xml", "*xml.Encoder"},
		{"no ct, at gob", "", "application/gob", "*gob.Encoder"},
		{"no ct, at html", "", "text/html", "*http.textEncoder"},
		{"no ct, at plain", "", "text/plain", "*http.textEncoder"},
		{"ct json", "application/json", "application/gob", "*json.Encoder"},
		{"ct +json", "+json", "application/gob", "*json.Encoder"},
		{"ct xml", "application/xml", "application/gob", "*xml.Encoder"},
		{"ct +xml", "+xml", "application/gob", "*xml.Encoder"},
		{"ct gob", "application/gob", "application/xml", "*gob.Encoder"},
		{"ct +gob", "+gob", "application/xml", "*gob.Encoder"},
		{"ct html", "text/html", "application/gob", "*http.textEncoder"},
		{"ct +html", "+html", "application/gob", "*http.textEncoder"},
		{"ct plain", "text/plain", "application/gob", "*http.textEncoder"},
		{"ct +txt", "+txt", "application/gob", "*http.textEncoder"},
		{"no ct, at json with params", "", "application/json; charset=utf-8", "*json.Encoder"},
		{"no ct, at xml with params", "", "application/xml; charset=utf-8", "*xml.Encoder"},
		{"no ct, at gob with params", "", "application/gob; charset=utf-8", "*gob.Encoder"},
		{"no ct, at html with params", "", "text/html; charset=utf-8", "*http.textEncoder"},
		{"no ct, at plain with params", "", "text/plain; charset=utf-8", "*http.textEncoder"},
		{"ct json with params", "application/json; charset=utf-8", "application/gob", "*json.Encoder"},
		{"ct +json with params", "+json; charset=utf-8", "application/gob", "*json.Encoder"},
		{"ct xml with params", "application/xml; charset=utf-8", "application/gob", "*xml.Encoder"},
		{"ct +xml with params", "+xml; charset=utf-8", "application/gob", "*xml.Encoder"},
		{"ct gob with params", "application/gob; charset=utf-8", "application/xml", "*gob.Encoder"},
		{"ct +gob with params", "+gob; charset=utf-8", "application/xml", "*gob.Encoder"},
		{"ct html with params", "text/html; charset=utf-8", "application/gob", "*http.textEncoder"},
		{"ct +html with params", "+html; charset=utf-8", "application/gob", "*http.textEncoder"},
		{"ct plain with params", "text/plain; charset=utf-8", "application/gob", "*http.textEncoder"},
		{"ct +txt with params", "+txt; charset=utf-8", "application/gob", "*http.textEncoder"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, AcceptTypeKey, c.acceptType)
			ctx = context.WithValue(ctx, ContentTypeKey, c.contentType)
			w := httptest.NewRecorder()

			encoder := ResponseEncoder(ctx, w)

			assert.Equal(t, c.encoderType, fmt.Sprintf("%T", encoder))
		})
	}
}

func TestResponseEncoder_ContentTypeHeaderPreservesCharset(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContentTypeKey, "application/json; charset=utf-8")
	w := httptest.NewRecorder()

	encoder := ResponseEncoder(ctx, w)

	require.Equal(t, "*json.Encoder", fmt.Sprintf("%T", encoder))
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestResponseEncoder_Encode_ErrorResponse(t *testing.T) {
	serviceError := loom.NewServiceError(errors.New("foo"), "foo", false, false, false)

	cases := []struct {
		name       string
		acceptType string
		encoded    string
		wantCT     string
	}{
		{
			name:       "generic-json",
			acceptType: "application/json",
			encoded:    fmt.Sprintf(`{"type":"https://github.com/CaliLuke/loom/problems/foo","title":"Foo","status":400,"detail":"foo","instance":"urn:loom:error:%s","code":"foo"}`, serviceError.ID),
			wantCT:     ProblemJSONContentType,
		},
		{
			name:       "explicit-xml",
			acceptType: "application/xml",
			encoded:    fmt.Sprintf(`<ProblemResponse><type>https://github.com/CaliLuke/loom/problems/foo</type><title>Foo</title><status>400</status><detail>foo</detail><instance>urn:loom:error:%s</instance><code>foo</code></ProblemResponse>`, serviceError.ID),
			wantCT:     "application/xml",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			w := httptest.NewRecorder()
			var err error
			if c.name == "generic-json" {
				encoder := ErrorEncoder(ResponseEncoder, nil)
				err = encoder(ctx, w, serviceError)
			} else {
				ctx = context.WithValue(ctx, ContentTypeKey, c.acceptType)
				encoder := ResponseEncoder(ctx, w)
				err = encoder.Encode(NewProblemResponse(ctx, serviceError, http.StatusBadRequest, "", ""))
			}

			assert.NoError(t, err)
			body := strings.TrimSpace(w.Body.String())
			assert.Equal(t, c.encoded, body)
			assert.Equal(t, c.wantCT, w.Header().Get("Content-Type"))
		})
	}
}

func TestResponseEncoder_Encode_ErrorResponseWithRemedy(t *testing.T) {
	err := loom.WithErrorRemedy(
		loom.NewServiceError(errors.New("internal detail"), "bad_request", false, false, false),
		&loom.ErrorRemedy{
			Code:        "bad_request.fix",
			SafeMessage: "Retry with a valid request.",
			RetryHint:   "Correct the payload and retry.",
		},
	)

	cases := []struct {
		name     string
		wantBody string
	}{
		{
			name:     "problem-json",
			wantBody: fmt.Sprintf(`{"type":"about:blank","title":"Bad Request","status":400,"detail":"Retry with a valid request.","instance":"urn:loom:error:%s","code":"bad_request","retry_hint":"Correct the payload and retry."}`, err.ID),
		},
		{
			name:     "problem-helper",
			wantBody: fmt.Sprintf(`{"type":"about:blank","title":"Bad Request","status":400,"detail":"Retry with a valid request.","instance":"urn:loom:error:%s","code":"bad_request","retry_hint":"Correct the payload and retry."}`, err.ID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			w := httptest.NewRecorder()
			if tc.name == "problem-json" {
				encoder := ErrorEncoder(ResponseEncoder, nil)
				require.NoError(t, encoder(ctx, w, err))
			} else {
				ctx = context.WithValue(ctx, ContentTypeKey, ProblemJSONContentType)
				encoder := ResponseEncoder(ctx, w)
				require.NoError(t, encoder.Encode(NewProblemResponse(ctx, err, http.StatusBadRequest, "", "")))
			}
			assert.Equal(t, tc.wantBody, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestResponseDecoder(t *testing.T) {
	cases := []struct {
		contentType string
		decoderType string
	}{
		{"application/json", "*json.Decoder"},
		{"+json", "*json.Decoder"},
		{"application/xml", "*xml.Decoder"},
		{"+xml", "*xml.Decoder"},
		{"application/gob", "*gob.Decoder"},
		{"+gob", "*gob.Decoder"},
		{"text/html", "*http.textDecoder"},
		{"+html", "*http.textDecoder"},
		{"text/plain", "*http.textDecoder"},
		{"+txt", "*http.textDecoder"},
		{"application/json; charset=utf-8", "*json.Decoder"},
		{"+json; charset=utf-8", "*json.Decoder"},
		{"application/xml; charset=utf-8", "*xml.Decoder"},
		{"+xml; charset=utf-8", "*xml.Decoder"},
		{"application/gob; charset=utf-8", "*gob.Decoder"},
		{"+gob; charset=utf-8", "*gob.Decoder"},
		{"text/html; charset=utf-8", "*http.textDecoder"},
		{"+html; charset=utf-8", "*http.textDecoder"},
		{"text/plain; charset=utf-8", "*http.textDecoder"},
		{"+txt; charset=utf-8", "*http.textDecoder"},
	}

	for _, c := range cases {
		t.Run(c.contentType, func(t *testing.T) {
			r := &http.Response{
				Header: map[string][]string{
					"Content-Type": {c.contentType},
				},
			}
			decoder := ResponseDecoder(r)

			assert.Equal(t, c.decoderType, fmt.Sprintf("%T", decoder))
		})
	}
}

func TestResolveProblemTypeAndTitle(t *testing.T) {
	t.Run("generic status uses about blank", func(t *testing.T) {
		problemType, title := ResolveProblemTypeAndTitle("bad_request", http.StatusBadRequest, "", "")
		require.Equal(t, "about:blank", problemType)
		require.Equal(t, "Bad Request", title)
	})

	t.Run("specialized code gets deterministic uri", func(t *testing.T) {
		problemType, title := ResolveProblemTypeAndTitle("wrong_token_type", http.StatusUnauthorized, "", "")
		require.Equal(t, "https://github.com/CaliLuke/loom/problems/wrong-token-type", problemType)
		require.Equal(t, "Wrong Token Type", title)
	})

	t.Run("explicit overrides win", func(t *testing.T) {
		problemType, title := ResolveProblemTypeAndTitle(
			"wrong_token_type",
			http.StatusUnauthorized,
			"https://api.example.com/problems/wrong-token-type",
			"Wrong token type",
		)
		require.Equal(t, "https://api.example.com/problems/wrong-token-type", problemType)
		require.Equal(t, "Wrong token type", title)
	})
}

func TestTextEncoder_Encode(t *testing.T) {
	cases := []struct {
		name  string
		value any
		error error
	}{
		{"string", testString, nil},
		{"*string", &testString, nil},
		{"[]byte", []byte(testString), nil},
		{"other", 123, fmt.Errorf("can't encode int as content/type")},
	}

	buffer := bytes.Buffer{}
	encoder := newTextEncoder(&buffer, "content/type")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buffer.Reset()
			err := encoder.Encode(c.value)
			if c.error != nil {
				assert.Error(t, err, c.error)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testString, buffer.String())
		})
	}
}

func TestTextPlainDecoder_Decode_String(t *testing.T) {
	decoder := makeTextDecoder()
	var value string

	err := decoder.Decode(&value)

	assert.NoError(t, err)
	assert.Equal(t, testString, value)
}

func TestTextPlainDecoder_Decode_Bytes(t *testing.T) {
	decoder := makeTextDecoder()
	var value []byte

	err := decoder.Decode(&value)

	assert.NoError(t, err)
	assert.Equal(t, testString, string(value))
}

func TestTextPlainDecoder_Decode_Other(t *testing.T) {
	decoder := makeTextDecoder()
	expectedErr := fmt.Errorf("can't decode content/type to *int")
	var value int

	err := decoder.Decode(&value)

	assert.Error(t, err, expectedErr)
}

func makeTextDecoder() Decoder {
	buffer := bytes.Buffer{}
	buffer.WriteString(testString)
	return newTextDecoder(&buffer, "content/type")
}
