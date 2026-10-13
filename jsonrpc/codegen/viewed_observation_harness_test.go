package codegen

const jsonRPCViewedWSObservationHarness = `package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	files "example.com/jsonrpcviewed/gen/files"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

// TestRetainedWrapperIdentity checks both dynamic and fixed view wrappers after
// their request observer has finished and with unrelated per-call contexts.
func TestRetainedWrapperIdentity(t *testing.T) {
	var events []loomtransport.Event
	ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		events = append(events, event)
	}))
	req := httptest.NewRequest("GET", "/files", nil).WithContext(ctx)
	observer, _ := loomtransport.BeginJSONRPCRequest(ctx, httptest.NewRecorder(), "files", req)
	ctx = loomtransport.WithRequestObserver(ctx, observer)
	binding := loomtransport.NewStreamObserver(ctx, loomtransport.TransportJSONRPC)
	first := &talkStreamWrapper{view: "tiny", observation: binding.JSONRPC("talk", "", true)}
	second := &talkTinyStreamWrapper{observation: binding.JSONRPC("talkTiny", "", true)}
	observer.End()
	if err := first.SendResponse(context.Background(), &files.Note{ID: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := second.SendResponse(context.Background(), &files.Note{ID: "second"}); err != nil {
		t.Fatal(err)
	}
	if err := first.SendError(context.Background(), errors.New("late")); err != nil {
		t.Fatal(err)
	}
	var methods []string
	for _, event := range events {
		if event.Reason != loomtransport.ReasonStreamFinalResponseSuppressed {
			continue
		}
		if event.Service != "files" || !event.Notification || event.HTTPMethod != "GET" {
			t.Errorf("identity: %+v", event)
		}
		methods = append(methods, event.Method)
	}
	if len(methods) != 3 || methods[0] != "talk" || methods[1] != "talkTiny" || methods[2] != "talk" {
		t.Errorf("retained methods: %v", methods)
	}
}
`

const jsonRPCViewedSSEObservationHarness = `package jsonrpcviewed_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	feed "example.com/jsonrpcviewed/gen/feed"
	feedserver "example.com/jsonrpcviewed/gen/jsonrpc/feed/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

// TestMountedSSEObservation uses the same mounted viewed-result fixture as the
// wire checks; normal, notification and pre-event conversion failure retain
// operation identity and one physical lifecycle pair.
func TestMountedSSEObservation(t *testing.T) {
	for _, tc := range []struct {
		name string
		request string
		notification bool
	}{
		{name: "normal", request: "{\"jsonrpc\":\"2.0\",\"method\":\"follow\",\"id\":\"r\",\"params\":\"x\"}"},
		{name: "notification", notification: true, request: "{\"jsonrpc\":\"2.0\",\"method\":\"follow\",\"params\":\"x\"}"},
		{name: "invalid-view", request: "{\"jsonrpc\":\"2.0\",\"method\":\"follow\",\"id\":\"r\",\"params\":\"bogus\"}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []loomtransport.Event
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				events = append(events, event)
			}))
			mux := loomhttp.NewMuxer()
			feedserver.Mount(mux, feedserver.New(feed.NewEndpoints(feedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			req := httptest.NewRequest("POST", "/feed", strings.NewReader(tc.request)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			opened, closed, suppressed := 0, 0, 0
			for _, event := range events {
				if event.Kind != loomtransport.EventKindStreamOpen && event.Kind != loomtransport.EventKindStreamClose {
					continue
				}
				if event.Service != "feed" || event.Method != "follow" || event.HTTPMethod != "POST" || event.Notification != tc.notification {
					t.Errorf("identity: %+v", event)
				}
				if event.Reason == loomtransport.ReasonStreamFinalResponseSuppressed {
					suppressed++
				} else if event.Kind == loomtransport.EventKindStreamOpen {
					opened++
				} else {
					closed++
				}
			}
			if opened != 1 || closed != 1 || (suppressed == 1) != tc.notification {
				t.Errorf("events open=%d close=%d suppressed=%d: %s", opened, closed, suppressed, rec.Body.String())
			}
		})
	}
}
`
