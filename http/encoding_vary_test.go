package http

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseEncoderVariesNegotiatedRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name, accept, contentType, wantType string
		vary, wantVary                      []string
	}{
		{"missing Accept", "", "", "application/json", nil, []string{"Accept"}},
		{"JSON", "application/json", "", "application/json", nil, []string{"Accept"}},
		{"HTML", "text/html", "", "text/html", nil, []string{"Accept"}},
		{"text", "text/plain", "", "text/plain", nil, []string{"Accept"}},
		{"XML", "application/xml", "", "application/xml", nil, []string{"Accept"}},
		{"Gob", "application/gob", "", "application/gob", nil, []string{"Accept"}},
		{"unsupported Accept", "image/png", "", "application/json", nil, []string{"Accept"}},
		{"browser alternatives", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "", "application/json", nil, []string{"Accept"}},
		{"preserve existing Vary", "text/html", "", "text/html", []string{"Origin", "Accept-Encoding"}, []string{"Origin", "Accept-Encoding", "Accept"}},
		{"existing Accept in list", "text/html", "", "text/html", []string{"Origin, aCcEpT"}, []string{"Origin, aCcEpT"}},
		{"explicit JSON", "text/html", "application/json", "application/json", []string{"Origin"}, []string{"Origin"}},
		{"explicit HTML", "application/json", "text/html", "text/html", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), AcceptTypeKey, tc.accept)
			ctx = context.WithValue(ctx, ContentTypeKey, tc.contentType)
			writer := httptest.NewRecorder()
			for _, value := range tc.vary {
				writer.Header().Add("Vary", value)
			}
			encoder := ResponseEncoder(ctx, writer)
			require.NoError(t, encoder.Encode("<script>alert(1)</script>"))
			response := writer.Result()
			require.NoError(t, response.Body.Close())
			require.Equal(t, tc.wantType, response.Header.Get("Content-Type"))
			require.Equal(t, tc.wantVary, response.Header.Values("Vary"))
		})
	}
}
