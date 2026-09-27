package codegen

const jsonRPCViewMarkerHarness = `package jsonrpcviewed_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	feedclient "example.com/jsonrpcviewed/gen/jsonrpc/feed/client"
	filesclient "example.com/jsonrpcviewed/gen/jsonrpc/files/client"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"
)

var viewMarkerCases = []struct {
	name, marker, body string
	fixed, valid       bool
	collection         bool
}{
	{"tiny", ` + "`" + `,"loom_view":"tiny"` + "`" + `, tinyJSON, false, true, false},
	{"default", ` + "`" + `,"loom_view":"default"` + "`" + `, fullJSON, false, true, false},
	{"legacy", "", fullJSON, false, true, false},
	{"missing-default-title", ` + "`" + `,"loom_view":"default"` + "`" + `, tinyJSON, false, false, false},
	{"legacy-missing-title", "", tinyJSON, false, false, false},
	{"unknown", ` + "`" + `,"loom_view":"bogus"` + "`" + `, fullJSON, false, false, false},
	{"null-marker", ` + "`" + `,"loom_view":null` + "`" + `, fullJSON, false, false, false},
	{"malformed", ` + "`" + `,"loom_view":123` + "`" + `, fullJSON, false, false, false},
	{"fixed-tiny", ` + "`" + `,"loom_view":"tiny"` + "`" + `, tinyJSON, true, true, false},
	{"fixed-legacy", "", tinyJSON, true, true, false},
	{"fixed-mismatch", ` + "`" + `,"loom_view":"default"` + "`" + `, fullJSON, true, false, false},
	{"collection-null", ` + "`" + `,"loom_view":"default"` + "`" + `, ` + "`" + `[null]` + "`" + `, false, false, true},
	{"collection-tiny", ` + "`" + `,"loom_view":"tiny"` + "`" + `, ` + "`" + `[` + "`" + `+tinyJSON+` + "`" + `]` + "`" + `, false, true, true},
}

func TestSSEViewMarkers(t *testing.T) {
	for _, tc := range viewMarkerCases {
		for _, event := range []string{"message", "notification", "response"} {
			t.Run(tc.name+"/"+event, func(t *testing.T) {
				hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/event-stream")
					envelope := fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","method":"feed/stream.event","params":%s%s}` + "`" + `, tc.body, tc.marker)
					if event == "response" {
						envelope = fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":"x","result":%s%s}` + "`" + `, tc.body, tc.marker)
					}
					if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, envelope); err != nil {
						t.Errorf("write frame: %v", err)
					}
				}))
				defer hs.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				c := feedclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				open := c.Follow()
				if tc.fixed {
					open = c.FollowTiny()
				} else if tc.collection {
					open = c.FollowList()
				}
				raw, err := open(ctx, "x")
				if err != nil {
					t.Fatal(err)
				}
				if tc.collection {
					stream := raw.(*feedclient.FollowListClientStream)
					_, err = stream.Recv(ctx)
					if closeErr := stream.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				} else if tc.fixed {
					stream := raw.(*feedclient.FollowTinyClientStream)
					_, err = stream.Recv(ctx)
					if closeErr := stream.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				} else {
					stream := raw.(*feedclient.FollowClientStream)
					_, err = stream.Recv(ctx)
					if closeErr := stream.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}
				if (err == nil) != tc.valid {
					t.Errorf("decode error = %v; want valid=%v", err, tc.valid)
				}
			})
		}
	}
}

func TestWebSocketViewMarkers(t *testing.T) {
	for _, tc := range viewMarkerCases {
		t.Run(tc.name, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() {
					if err := conn.Close(); err != nil {
						t.Error(err)
					}
				}()
				var request struct {
					ID any ` + "`" + `json:"id"` + "`" + `
				}
				if err := conn.ReadJSON(&request); err != nil {
					t.Error(err)
					return
				}
				id, err := json.Marshal(request.ID)
				if err != nil {
					t.Error(err)
					return
				}
				data := fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%s,"result":%s%s}` + "`" + `, id, tc.body, tc.marker)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
					t.Error(err)
				}
			}))
			defer hs.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c := filesclient.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), nil, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
			defer func() {
				if err := c.Close(); err != nil {
					t.Error(err)
				}
			}()
			var err error
			if tc.collection {
				_, err = exchange(t, ctx, c.TalkList(), "x", (*filesclient.TalkListClientStream).RecvWithContext)
			} else if tc.fixed {
				_, err = exchange(t, ctx, c.TalkTiny(), "x", (*filesclient.TalkTinyClientStream).RecvWithContext)
			} else {
				_, err = exchange(t, ctx, c.Talk(), "x", (*filesclient.TalkClientStream).RecvWithContext)
			}
			if (err == nil) != tc.valid || errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("decode error = %v; want valid=%v", err, tc.valid)
			}
		})
	}
}

func TestWebSocketChangesViewsOnOneStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := newFilesClient(t)
	raw, err := c.Talk()(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	stream := raw.(*filesclient.TalkClientStream)
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, view := range []string{"tiny", "default", "tiny"} {
		if err := stream.SendWithContext(ctx, view); err != nil {
			t.Fatal(err)
		}
		got, err := stream.RecvWithContext(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if (got.Title == nil) != (view == "tiny") {
			t.Errorf("view %q got title %v", view, got.Title)
		}
	}
}
`
