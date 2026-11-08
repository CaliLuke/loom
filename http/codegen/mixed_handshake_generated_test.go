package codegen

const mixedHandshakeHarness = `package handshake_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/handshake/gen/http/mixed/client"
	server "example.com/handshake/gen/http/mixed/server"
	svc "example.com/handshake/gen/mixed"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type mixedImplementation struct {
	before  string
	missing bool
	fail    bool
	calls   int
	status  int
	event   bool
	view    string
}

func (s *mixedImplementation) Read(ctx context.Context, stream svc.ReadServerStream) (*svc.MixedResult, error) {
	s.calls++
	switch s.before {
	case "send":
		if err := stream.SendWithContext(ctx, "before"); err != nil {
			return nil, err
		}
	case "close":
		if err := stream.Close(); err != nil {
			return nil, err
		}
	}
	if s.fail {
		return nil, errors.New("endpoint failed")
	}
	if !s.missing {
		status := s.status
		if status == 0 {
			status = 202
		}
		if err := stream.SetResponse(&svc.MixedOpening{Status: status, Session: "metadata"}); err != nil {
			return nil, err
		}
	}
	if s.event {
		if err := stream.SendWithContext(ctx, "event"); err != nil {
			return nil, err
		}
	}
	return &svc.MixedResult{Status: "body-status", Session: "body-session"}, nil
}

func TestMixedWholeResultAndMetadata(t *testing.T) {
	for _, accept := range []string{"application/json", "text/event-stream"} {
		for _, before := range []string{"", "send", "close"} {
			t.Run(accept+"/"+before, func(t *testing.T) {
				implementation := &mixedImplementation{before: before}
				mux := loomhttp.NewMuxer()
				transport := server.New(svc.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
				server.Mount(mux, transport)
				request := httptest.NewRequest(http.MethodGet, "/mixed", nil)
				request.Header.Set("Accept", accept)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				want := 202
				if accept == "text/event-stream" && before != "" {
					want = 500
				}
				if response.Code != want || implementation.calls != 1 {
					t.Fatalf("status=%d calls=%d body=%s", response.Code, implementation.calls, response.Body.String())
				}
				if want != 202 {
					if response.Header().Get("X-Session") != "" {
						t.Fatal("failed preparation leaked metadata")
					}
					return
				}
				if response.Header().Get("X-Session") != "metadata" {
					t.Fatal("missing shared metadata", response.Header())
				}
				if accept == "application/json" && strings.TrimSpace(response.Body.String()) != "{\"status\":\"body-status\",\"session\":\"body-session\"}" {
					t.Fatal("whole Result was not retained", response.Body.String())
				}
				if accept == "text/event-stream" && response.Body.Len() != 0 {
					t.Fatal("zero-event stream emitted unary result", response.Body.String())
				}
			})
		}
	}
	for _, fail := range []bool{false, true} {
		implementation := &mixedImplementation{missing: true, fail: fail}
		mux := loomhttp.NewMuxer()
		transport := server.New(svc.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
		server.Mount(mux, transport)
		request := httptest.NewRequest(http.MethodGet, "/mixed", nil)
		request.Header.Set("Accept", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != 500 || response.Header().Get("X-Session") != "" || implementation.calls != 1 {
			t.Fatalf("failed metadata or handler published success: %d %v", response.Code, response.Header())
		}
	}
}

type metadataAudit struct {
	calls int
}

func (a *metadataAudit) MetadataAudit(ctx context.Context, info *svc.MetadataAuditInfo, next loom.Endpoint) (any, error) {
	res, err := next(ctx, info.RawPayload())
	if err != nil {
		return nil, err
	}
	if response, ok := res.(*svc.ReadResponseData); ok {
		a.calls++
		if response.Result.Session != "body-session" || response.Metadata.Session != "metadata" {
			return nil, errors.New("interceptor lost response tuple ownership")
		}
		response.Result.Session = "intercepted-body"
	}
	return res, nil
}

func TestMixedClientMetadataTuple(t *testing.T) {
	implementation := &mixedImplementation{}
	mux := loomhttp.NewMuxer()
	transport := server.New(svc.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, transport)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	httpClient := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	audit := &metadataAudit{}
	serviceClient := newMixedClient(httpClient, audit)
	result, metadata, err := serviceClient.Read(context.Background())
	if err != nil || result == nil || metadata == nil {
		t.Fatalf("unary tuple: %v %v %v", result, metadata, err)
	}
	if result.Status != "body-status" || result.Session != "intercepted-body" || metadata.Status != 202 || metadata.Session != "metadata" || audit.calls != 1 {
		t.Fatalf("tuple ownership: %v %v audit=%d", result, metadata, audit.calls)
	}
	stream, err := serviceClient.ReadStream(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stream.Response().Status != metadata.Status || stream.Response().Session != metadata.Session {
		t.Fatal("stream and unary metadata contracts differ", stream.Response(), metadata)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatal("zero event stream", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if implementation.calls != 2 {
		t.Fatal("endpoint was reinvoked", implementation.calls)
	}
}

type mixedDoer func(*http.Request) (*http.Response, error)

func (d mixedDoer) Do(request *http.Request) (*http.Response, error) {
	return d(request)
}

func TestMixedClientRejectsIncompleteTuple(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		body   string
	}{
		{"missing metadata", "", "{\"status\":\"body-status\",\"session\":\"body-session\"}"},
		{"invalid body", "metadata", "{}"},
		{"malformed body", "metadata", "{"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doer := mixedDoer(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 202, Header: http.Header{"Content-Type": {"application/json"}, "X-Session": {tc.header}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: request}, nil
			})
			httpClient := client.NewClient("http", "example.test", doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			serviceClient := newMixedClient(httpClient, &metadataAudit{})
			result, metadata, err := serviceClient.Read(context.Background())
			if err == nil || result != nil || metadata != nil {
				t.Fatalf("partial tuple escaped: %v %v %v", result, metadata, err)
			}
		})
	}
}

func (s *mixedImplementation) Primitive(ctx context.Context, stream svc.PrimitiveServerStream) (string, error) {
	err := stream.SetResponse(&svc.MixedOpening{Status: 202, Session: "metadata"})
	return "whole primitive", err
}

func (s *mixedImplementation) Collection(ctx context.Context, stream svc.CollectionServerStream) ([]string, error) {
	err := stream.SetResponse(&svc.MixedOpening{Status: 202, Session: "metadata"})
	return []string{"whole", "collection"}, err
}

func (s *mixedImplementation) Viewed(ctx context.Context, stream svc.ViewedServerStream) (*svc.MixedView, error) {
	err := stream.SetResponse(&svc.MixedOpening{Status: 202, Session: "metadata"})
	return &svc.MixedView{Status: "view-status", Session: "view-session"}, err
}

func (s *mixedImplementation) Dynamic(ctx context.Context, stream svc.DynamicServerStream) (*svc.MixedView, string, error) {
	err := stream.SetResponse(&svc.MixedOpening{Status: 202, Session: "metadata"})
	hidden := "full only"
	return &svc.MixedView{Status: "view-status", Session: "view-session", Hidden: &hidden}, s.view, err
}

func newMixedClient(c *client.Client, audit *metadataAudit) *svc.Client {
	return svc.NewClient(c.Read(), c.ReadStream(), c.Primitive(), c.PrimitiveStream(), c.Collection(), c.CollectionStream(), c.Viewed(), c.ViewedStream(), c.Event(), c.EventStream(), c.Dynamic(), c.DynamicStream(), audit)
}

func TestMixedDynamicView(t *testing.T) {
	for view, hidden := range map[string]string{"full": "full only", "default": ""} {
		t.Run(view, func(t *testing.T) {
			mux := loomhttp.NewMuxer()
			transport := server.New(svc.NewEndpoints(&mixedImplementation{view: view}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			server.Mount(mux, transport)
			hs := httptest.NewServer(mux)
			defer hs.Close()
			var wireView string
			decoder := func(response *http.Response) loomhttp.Decoder {
				wireView = response.Header.Get("loom-view")
				return loomhttp.ResponseDecoder(response)
			}
			c := newMixedClient(client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, decoder, false), &metadataAudit{})
			result, metadata, err := c.Dynamic(context.Background())
			if err != nil || result == nil || metadata == nil || metadata.Status != 202 || metadata.Session != "metadata" || wireView != view {
				t.Fatalf("dynamic view tuple: %v %v %v loom-view=%q", result, metadata, err, wireView)
			}
			got := ""
			if result.Hidden != nil {
				got = *result.Hidden
			}
			if result.Status != "view-status" || result.Session != "view-session" || got != hidden {
				t.Fatalf("selected view %q lost: %#v hidden=%q", view, result, got)
			}
		})
	}
}

func TestMixedWholeResultShapes(t *testing.T) {
	mux := loomhttp.NewMuxer()
	transport := server.New(svc.NewEndpoints(&mixedImplementation{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, transport)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := newMixedClient(client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false), &metadataAudit{})
	primitive, metadata, err := c.Primitive(context.Background())
	if err != nil || primitive != "whole primitive" || metadata == nil || metadata.Status != 202 || metadata.Session != "metadata" {
		t.Fatalf("primitive ownership: %v %v %v", primitive, metadata, err)
	}
	collection, metadata, err := c.Collection(context.Background())
	if err != nil || strings.Join(collection, ",") != "whole,collection" || metadata == nil || metadata.Status != 202 {
		t.Fatalf("collection ownership: %v %v %v", collection, metadata, err)
	}
	viewed, metadata, err := c.Viewed(context.Background())
	if err != nil || viewed == nil || viewed.Status != "view-status" || viewed.Session != "view-session" || metadata == nil || metadata.Status != 202 || metadata.Session != "metadata" {
		t.Fatalf("view ownership: %v %v %v", viewed, metadata, err)
	}
}

func TestMixedStreamSelectorAndEvent(t *testing.T) {
	for _, status := range []int{201, 202, 299, 303, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			implementation := &mixedImplementation{status: status, event: true}
			mux := loomhttp.NewMuxer()
			transport := server.New(svc.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			server.Mount(mux, transport)
			hs := httptest.NewServer(mux)
			defer hs.Close()
			var decodedStatuses []int
			decoder := func(response *http.Response) loomhttp.Decoder {
				decodedStatuses = append(decodedStatuses, response.StatusCode)
				return loomhttp.ResponseDecoder(response)
			}
			c := newMixedClient(client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, decoder, false), &metadataAudit{})
			stream, err := c.ReadStream(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if stream.Response().Status != status || stream.Response().Session != "metadata" {
				t.Fatal("metadata", stream.Response())
			}
			event, err := stream.Recv()
			if err != nil || event != "event" {
				t.Fatalf("event=%q err=%v", event, err)
			}
			if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
				t.Fatal("completion", err)
			}
			if err := stream.Close(); err != nil {
				t.Fatal(err)
			}
			if len(decodedStatuses) != 1 || decodedStatuses[0] != status || implementation.calls != 1 {
				t.Fatalf("actual handshake status lost or service repeated: %v calls=%d", decodedStatuses, implementation.calls)
			}
		})
	}
}

func (s *mixedImplementation) Event(ctx context.Context, stream svc.EventServerStream) (string, error) {
	if err := stream.SetResponse(&svc.MixedOpening{Status: 202, Session: "metadata"}); err != nil {
		return "", err
	}
	if err := stream.SendWithContext(ctx, &svc.MixedEvent{Status: "event-status", Session: "event-session", Hidden: "must not leak"}); err != nil {
		return "", err
	}
	return "whole unary", nil
}

func TestMixedViewedEvent(t *testing.T) {
	mux := loomhttp.NewMuxer()
	transport := server.New(svc.NewEndpoints(&mixedImplementation{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, transport)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := newMixedClient(client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false), &metadataAudit{})
	result, metadata, err := c.Event(context.Background())
	if err != nil || result != "whole unary" || metadata == nil || metadata.Status != 202 {
		t.Fatalf("unary/event separation: %v %v %v", result, metadata, err)
	}
	stream, err := c.EventStream(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()
	event, err := stream.Recv()
	if err != nil || event == nil || event.Status != "event-status" || event.Session != "event-session" || event.Hidden != "" {
		t.Fatalf("event view not applied: %#v %v", event, err)
	}
}
`
