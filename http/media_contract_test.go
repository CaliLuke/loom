package http

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMediaContractFactories(t *testing.T) {
	contracts := []string{"application/json", "text/plain", "*/*"}
	for _, tc := range []struct {
		name, media string
		index       int
		value       any
		rejected    bool
	}{
		{"json", "", 0, 42, false}, {"text", "", 1, "hello", false},
		{"range", "application/octet-stream", 2, []byte("bytes"), false},
		{"mismatch", "text/plain", 0, 42, true}, {"missing range", "", 2, []byte("bytes"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", nil)
			if tc.media != "" {
				request.Header.Set("Content-Type", tc.media)
			}
			called := false
			err := EncodeMediaRequest(request, tc.index, contracts, tc.value, nil, func(r *stdhttp.Request) Encoder {
				called = true
				return EncodingFunc(func(value any) error {
					require.Equal(t, tc.value, value)
					return nil
				})
			})
			if tc.rejected {
				require.Error(t, err)
				require.False(t, called)
				return
			}
			require.NoError(t, err)
			require.True(t, called)
			selected, err := SelectMediaType(request.Header.Get("Content-Type"), contracts)
			require.NoError(t, err)
			require.Equal(t, tc.index, selected)
		})
	}
}

func TestMediaContractBuiltinCodecs(t *testing.T) {
	contracts := []string{"application/json", "text/plain", "application/octet-stream"}
	for _, tc := range []struct {
		name   string
		index  int
		value  any
		target func() any
	}{
		{"json", 0, 42, func() any {
			return new(int)
		}},
		{"text", 1, "hello", func() any {
			return new(string)
		}},
		{"binary", 2, []byte{0, 255}, func() any {
			return new([]byte)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", nil)
			require.NoError(t, EncodeMediaRequest(request, tc.index, contracts, tc.value, nil, RequestEncoder))
			data, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			incoming := httptest.NewRequest("POST", "/", bytes.NewReader(data))
			incoming.Header.Set("Content-Type", request.Header.Get("Content-Type"))
			target := tc.target()
			require.NoError(t, DecodeMediaRequest(incoming, target, false, nil, RequestDecoder))
			writer := httptest.NewRecorder()
			encoder, err := MediaResponseEncoder(context.Background(), writer, tc.index, contracts, tc.value, nil, ResponseEncoder)
			require.NoError(t, err)
			writer.WriteHeader(201)
			require.NoError(t, encoder.Encode(nil))
			response := writer.Result()
			target = tc.target()
			require.NoError(t, DecodeMediaResponse(response, target, false, nil, ResponseDecoder))
			switch got := target.(type) {
			case *int:
				require.Equal(t, tc.value, *got)
			case *string:
				require.Equal(t, tc.value, *got)
			case *[]byte:
				require.Equal(t, tc.value, *got)
			}
		})
	}
}

func TestMediaJSONNullAndMalformed(t *testing.T) {
	for _, tc := range []struct {
		name, body      string
		nullable, valid bool
	}{
		{"zero", "0", false, true}, {"null rejected", "null", false, false},
		{"null allowed", "null", true, true}, {"empty", "", false, false},
		{"whitespace", " \n\t", false, false}, {"malformed", "!", false, false},
		{"truncated", "{", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", bytes.NewBufferString(tc.body))
			request.Header.Set("Content-Type", "application/json")
			var value *int
			err := DecodeMediaRequest(request, &value, tc.nullable, nil, RequestDecoder)
			require.Equal(t, tc.valid, err == nil, "%v", err)
			response := &stdhttp.Response{Header: stdhttp.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewBufferString(tc.body))}
			value = nil
			err = DecodeMediaResponse(response, &value, tc.nullable, nil, ResponseDecoder)
			require.Equal(t, tc.valid, err == nil, "%v", err)
		})
	}
}

func TestMediaResponseFactorySelection(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		index        int
		rejected     bool
	}{
		{"exact", "", 0, false}, {"range", "image/png", 1, false},
		{"different branch", "image/png", 0, true}, {"unselected", "", -1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			writer.Header().Set("Content-Type", tc.header)
			called := false
			encoder, err := MediaResponseEncoder(context.Background(), writer, tc.index, []string{"text/plain", "*/*"}, "selected", nil, func(ctx context.Context, w stdhttp.ResponseWriter) Encoder {
				called = true
				require.Equal(t, w.Header().Get("Content-Type"), ctx.Value(ContentTypeKey))
				require.Empty(t, writer.Body.String())
				return EncodingFunc(func(value any) error {
					require.Equal(t, "selected", value)
					return nil
				})
			})
			require.Equal(t, !tc.rejected, called)
			if tc.rejected {
				require.Error(t, err)
				require.Nil(t, encoder)
			} else {
				require.NoError(t, err)
				require.NoError(t, encoder.Encode("ignored"))
			}
		})
	}
}

func TestMediaMultipartPartTypes(t *testing.T) {
	value := struct {
		Data []byte `form:"data"`
	}{Data: []byte{0, 255}}
	request := httptest.NewRequest("POST", "/", nil)
	require.NoError(t, EncodeMediaRequest(request, 0, []string{"multipart/form-data"}, value, map[string]string{"data": "image/png"}, RequestEncoder))
	_, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	require.NoError(t, err)
	reader := multipart.NewReader(request.Body, params["boundary"])
	part, err := reader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "data", part.FormName())
	require.Equal(t, "image/png", part.Header.Get("Content-Type"))
	data, err := io.ReadAll(part)
	require.NoError(t, err)
	require.Equal(t, value.Data, data)
	require.NoError(t, part.Close())
}

func TestMediaCustomCodecsHaveNoFallback(t *testing.T) {
	for _, media := range []string{"application/xml", "application/gob", "image/png"} {
		t.Run(media, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", nil)
			require.Error(t, EncodeMediaRequest(request, 0, []string{media}, "value", nil, RequestEncoder))
			request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`"value"`))
			request.Header.Set("Content-Type", media)
			var value string
			require.Error(t, DecodeMediaRequest(request, &value, false, nil, RequestDecoder))
			writer := httptest.NewRecorder()
			_, err := MediaResponseEncoder(context.Background(), writer, 0, []string{media}, "value", nil, ResponseEncoder)
			require.Error(t, err)
			require.Empty(t, writer.Body.String())
		})
	}
}

func TestMediaEmptyFormRetainsObject(t *testing.T) {
	type form struct {
		Name *string `form:"name"`
	}
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var decoded *form
	require.NoError(t, DecodeMediaRequest(request, &decoded, false, nil, RequestDecoder))
	require.Equal(t, &form{}, decoded)
	response := &stdhttp.Response{Header: request.Header, Body: stdhttp.NoBody}
	decoded = nil
	require.NoError(t, DecodeMediaResponse(response, &decoded, false, nil, ResponseDecoder))
	require.Equal(t, &form{}, decoded)
}
