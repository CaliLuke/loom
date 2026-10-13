package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationStreamAdmission(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		ref := Type("Ref", func() {
			Attribute("id", String)
			Required("id")
		})
		event := Type("Event", func() {
			Attribute("data", String)
			Required("data")
		})
		access := Authorization("watch", ref)
		Service("feed", func() {
			StrictAuthorization()
			for _, method := range []string{"watch", "socket"} {
				Method(method, func() {
					Payload(ref)
					Authorize(access, func() {
						Bind("id", "id")
					})
					Error("forbidden")
					StreamingResult(event)
					HTTP(func() {
						GET("/" + method)
						Param("id")
						Response("forbidden", StatusForbidden)
						if method == "watch" {
							ServerSentEvents()
						}
					})
				})
			}
		})
	})
	dir := t.TempDir()
	if base := os.Getenv("LOOM_OBSERVATION_ARTIFACTS"); base != "" {
		dir = filepath.Join(base, "http")
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderHTTPModule(t, dir, "example.com/accessstream", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authorization_test.go"), []byte(authorizationStreamHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}

const authorizationStreamHarness = `package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	feed "example.com/accessstream/gen/feed"
	feedsvr "example.com/accessstream/gen/http/feed/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	observation "github.com/CaliLuke/loom/observability/transport"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type service struct {
	calls atomic.Int32
}

func (s *service) Watch(ctx context.Context, p *feed.Ref, stream feed.WatchServerStream) error {
	s.calls.Add(1)
	if p.ID == "lazy" {
		return nil
	}
	return stream.Send(&feed.Event{Data: p.ID})
}
func (s *service) Socket(ctx context.Context, p *feed.Ref, stream feed.SocketServerStream) error {
	s.calls.Add(1)
	if p.ID == "lazy" {
		return nil
	}
	return stream.Send(&feed.Event{Data: p.ID})
}

type access struct {
}

func (*access) AuthorizeWatch(_ context.Context, p *feed.Ref) error {
	if p.ID != "allowed" && p.ID != "lazy" {
		return loom.PermanentError("forbidden", "Access denied")
	}
	return nil
}
func TestStreamAdmission(t *testing.T) {
	var mu sync.Mutex
	var events []observation.Event
	ctx := observation.WithObserver(context.Background(), observation.ObserverFunc(func(_ context.Context, event observation.Event) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}))
	s := &service{}
	e := feed.NewEndpoints(s, &access{})
	mux := loomhttp.NewMuxer()
	server := feedsvr.New(e, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil)
	feedsvr.Mount(mux, server)
	for _, path := range []string{"/watch", "/socket"} {
		req := httptest.NewRequest("GET", path+"?id=denied", nil)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req.WithContext(ctx))
		require.Equal(t, 403, res.Code, res.Body.String())
		require.Contains(t, res.Body.String(), "forbidden")
		require.Zero(t, s.calls.Load())
	}
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest("GET", "/watch?id=allowed", nil).WithContext(ctx))
	require.Equal(t, 200, res.Code, res.Body.String())
	require.Contains(t, res.Body.String(), "allowed")
	require.Equal(t, int32(1), s.calls.Load())
	s.calls.Store(0)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(ctx))
	}))
	defer live.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(live.URL, "http")+"/socket?id=allowed", nil)
	require.NoError(t, err)
	if response != nil && response.Body != nil {
		defer func() {
			require.NoError(t, response.Body.Close())
		}()
	}
	defer func() {
		require.NoError(t, conn.Close())
	}()
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(message), "allowed")
	// Closing the httptest server waits for its handler before examining state.
	live.Close()
	require.Equal(t, int32(1), s.calls.Load())
	// Unopened service streams do not invent successful negotiation events.
	for _, path := range []string{"/watch", "/socket"} {
		mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path+"?id=lazy", nil).WithContext(ctx))
	}
	mu.Lock()
	counts := make(map[string]int)
	for _, event := range events {
		if event.Kind == observation.EventKindStreamOpen || event.Kind == observation.EventKindStreamClose {
			require.Equal(t, "feed", event.Service)
			require.Equal(t, http.MethodGet, event.HTTPMethod)
			require.NotEmpty(t, event.Route)
			counts[event.Method]++
		}
	}
	require.Equal(t, map[string]int{"watch": 2, "socket": 2}, counts)
	mu.Unlock()
	s.calls.Store(0)
	for _, req := range []any{nil, (*feed.WatchEndpointInput)(nil), &feed.WatchEndpointInput{}} {
		_, err := e.Watch(context.Background(), req)
		require.Error(t, err)
		require.Zero(t, s.calls.Load())
	}
}
`
