package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestWebSocketClientViewComments names the received result and explains how
// the client uses its view, for streams with and without outgoing payloads.
func TestWebSocketClientViewComments(t *testing.T) {
	cases := []struct {
		name   string
		design func()
		result string
	}{
		{"StreamingResultWithViews", testdata.StreamingResultWithViewsDSL, "streamingresultwithviewsservice.Usertype"},
		{"StreamingResultCollectionWithViews", testdata.StreamingResultCollectionWithViewsDSL, "streamingresultcollectionwithviewsservice.UsertypeCollection"},
		{"StreamingPayloadResultWithViews", testdata.StreamingPayloadResultWithViewsDSL, "streamingpayloadresultwithviewsservice.Usertype"},
		{"StreamingPayloadResultCollectionWithViews", testdata.StreamingPayloadResultCollectionWithViewsDSL, "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection"},
		{"BidirectionalStreamingResultWithViews", testdata.BidirectionalStreamingResultWithViewsDSL, "bidirectionalstreamingresultwithviewsservice.Usertype"},
		{"BidirectionalStreamingResultCollectionWithViews", testdata.BidirectionalStreamingResultCollectionWithViewsDSL, "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.design)
			service := CreateHTTPServices(root).Get(c.name + "Service")
			ws := service.Endpoints[0].ClientWebSocket
			require.NotNil(t, ws)
			var field, method bytes.Buffer
			require.NoError(t, websocketStructTypeSection(ws).Write(&field))
			require.NoError(t, websocketSetViewSection(ws).Write(&method))
			for name, text := range map[string]string{"field": field.String(), "SetView": method.String()} {
				t.Run(name, func(t *testing.T) {
					comments := []string{}
					for _, line := range strings.Split(text, "\n") {
						if _, rest, ok := strings.Cut(line, "//"); ok {
							comments = append(comments, strings.TrimSpace(rest))
						}
					}
					doc := strings.Join(comments, " ")
					require.Contains(t, doc, c.result)
					require.Contains(t, doc, "validate")
					require.Contains(t, doc, "received")
					require.NotContains(t, doc, "render  ")
					require.NotContains(t, doc, "sending to")
				})
			}
		})
	}
}
