package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const sseRequestIDHarness = `package requestid_test

import (
 "context"
 "net/http/httptest"
 "testing"

 events "example.com/requestid/gen/events"
 server "example.com/requestid/gen/http/events/server"
 loom "github.com/CaliLuke/loom/pkg"
 loomhttp "github.com/CaliLuke/loom/http"
"github.com/gorilla/websocket"
 "github.com/stretchr/testify/require"
)

type service struct {
 seen, contextID string
}

func (s *service) Watch(ctx context.Context, p *events.WatchPayload, stream events.WatchServerStream) error {
 s.seen = VALUE
 s.contextID, _ = ctx.Value(loomhttp.LastEventIDKey).(string)
 if s.seen == "fail" {
  return loom.Fault("before stream")
 }
 return stream.Send(s.seen)
}

EXTRA

func TestHeaderBinding(t *testing.T) {
 s := &service{}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(events.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil OPTS))
 for _, header := range []string{"resume-42", "", "fail"} {
  req := httptest.NewRequest("GET", "/events?last_event_id=initial", nil)
  if header != "" {
   req.Header.Set("Last-Event-ID", header)
  }
  response := httptest.NewRecorder()
  mux.ServeHTTP(response, req)
  want := "initial"
  if header != "" {
   want = header
  }
  if header == "fail" {
   require.Equal(t, 500, response.Code, response.Body.String())
   require.Contains(t, response.Body.String(), "before stream")
   require.NotContains(t, response.Header().Get("Content-Type"), "text/event-stream")
   continue
  }
  require.Equal(t, 200, response.Code, response.Body.String())
  require.Contains(t, response.Body.String(), want)
  require.Equal(t, want, s.seen)
  require.Equal(t, header, s.contextID)
 }
}
`

// TestSSERequestIDGeneratedModule checks header binding in compiled HTTP SSE servers.
func TestSSERequestIDGeneratedModule(t *testing.T) {
	for _, tc := range []struct {
		name, field     string
		required, mixed bool
	}{
		{"snake", "LastEventID", false, false},
		{"custom", "ResumeToken", false, false},
		{"required", "ResumeToken", true, false},
		{"mixed", "ResumeToken", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("events", func() {
					Method("watch", func() {
						Payload(func() {
							Attribute("last_event_id", String, func() {
								if tc.field != "LastEventID" {
									Meta("struct:field:name", tc.field)
								}
							})
							if tc.required {
								Required("last_event_id")
							}
						})
						StreamingResult(String)
						HTTP(func() {
							GET("/events")
							Param("last_event_id")
							ServerSentEvents(func() {
								SSERequestID("last_event_id")
							})
						})
					})
					if tc.mixed {
						Method("socket", func() {
							StreamingResult(String)
							HTTP(func() {
								GET("/socket")
							})
						})
					}
				})
			})
			value := "p." + tc.field
			if !tc.required {
				value = "*p." + tc.field
			}
			extra, opts := "", ""
			if tc.mixed {
				extra = `func (s *service) Socket(ctx context.Context, stream events.SocketServerStream) error {
 return stream.Send("socket")
}`
				opts = ", &websocket.Upgrader{}, nil"
			}
			harness := strings.NewReplacer("VALUE", value, "EXTRA", extra, "OPTS", opts).Replace(sseRequestIDHarness)
			if !tc.mixed {
				harness = strings.ReplaceAll(harness, "\n\"github.com/gorilla/websocket\"", "")
			}
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/requestid", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "request_id_test.go"), []byte(harness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}
