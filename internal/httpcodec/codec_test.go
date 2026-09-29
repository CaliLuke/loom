package httpcodec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectors(t *testing.T) {
	for _, test := range []struct {
		media                             string
		request, response, accept, decode Kind
	}{
		{"", JSON, Invalid, JSON, JSON},
		{"application/json", JSON, JSON, JSON, JSON},
		{"APPLICATION/JSON; charset=utf-8", JSON, JSON, JSON, JSON},
		{"application/xml", XML, XML, XML, XML},
		{"application/gob", GOB, GOB, GOB, GOB},
		{"text/plain", Text, Text, Text, Text},
		{"text/html", Text, Text, Text, Text},
		{"application/x+json", Unsupported, JSON, JSON, JSON},
		{"application/x+xml", Unsupported, XML, JSON, XML},
		{"application/x+gob", Unsupported, GOB, JSON, GOB},
		{"application/x+html", Unsupported, Text, JSON, Text},
		{"application/x+txt", Unsupported, Text, JSON, Text},
		{"application/unknown", Unsupported, JSON, JSON, JSON},
		{"application/json;bad", Unsupported, Invalid, JSON, JSON},
		{"application/xml;bad", Unsupported, Invalid, JSON, JSON},
		{"invalid value+xml", Unsupported, Invalid, JSON, XML},
		{"invalid value+gob", Unsupported, Invalid, JSON, GOB},
		{"invalid value+txt", Unsupported, Invalid, JSON, Text},
		{"text/plain,application/json", Unsupported, Invalid, JSON, JSON},
	} {
		t.Run(test.media, func(t *testing.T) {
			require.Equal(t, test.request, RequestContentType(test.media).Kind)
			require.Equal(t, test.response, ResponseContentType(test.media).Kind)
			require.Equal(t, test.accept, ResponseAccept(test.media).Kind)
			require.Equal(t, test.decode, ResponseDecode(test.media).Kind)
		})
	}
}
