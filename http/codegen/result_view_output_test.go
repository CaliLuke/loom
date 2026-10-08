package codegen

// resultViewOutputHarness exercises the shared constructor through generated
// endpoints and a WebSocket sender. Transport classification consumes the same
// returned fault; client-side validation remains covered by the presence harness.
const resultViewOutputHarness = `package presence_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	client "example.com/viewpresence/gen/http/presence/client"
	server "example.com/viewpresence/gen/http/presence/server"
	presence "example.com/viewpresence/gen/presence"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loomhttp "github.com/CaliLuke/loom/http"
	loomjsonrpc "github.com/CaliLuke/loom/jsonrpc"
	loom "github.com/CaliLuke/loom/pkg"
)

type outputService struct {
	result  *presence.Record
	view    string
	failure error
	sent    chan error
}

func (s outputService) Show(context.Context) (*presence.Record, string, error) {
	return s.result, s.view, s.failure
}

func (s outputService) Raw(_ context.Context, body io.ReadCloser) (*presence.Record, string, error) {
	if err := body.Close(); err != nil {
		return nil, "", err
	}
	return s.result, s.view, s.failure
}

func (s outputService) Tiny(context.Context) (*presence.Record, error) {
	return s.result, s.failure
}

func (s outputService) List(context.Context) (presence.RecordCollection, string, error) {
	return presence.RecordCollection{s.result}, s.view, s.failure
}

func (s outputService) Watch(ctx context.Context, stream presence.WatchServerStream) error {
	stream.SetView(s.view)
	err := stream.SendWithContext(ctx, s.result)
	s.sent <- err
	return errors.Join(err, stream.Close())
}

func validOutput(t *testing.T) *presence.Record {
	t.Helper()
	var result presence.Record
	if err := json.Unmarshal([]byte(responseCases(t)[1].body), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}

func requireOutputFault(t *testing.T, err error, causeName string) {
	t.Helper()
	var fault *loom.ServiceError
	if !errors.As(err, &fault) || !fault.Fault || fault.Name != "fault" {
		t.Errorf("error=%v, want output fault", err)
		return
	}
	if causeName != "" {
		var cause *loom.ServiceError
		if !errors.As(errors.Unwrap(fault), &cause) || cause.Name != causeName {
			t.Errorf("cause=%v, want %s", errors.Unwrap(fault), causeName)
		}
	}
	if code := status.Code(loomgrpc.EncodeServerError(err, nil)); code != codes.Internal {
		t.Errorf("gRPC code=%s, want Internal", code)
	}
	if code := loomjsonrpc.CodeForServiceError(fault); code != loomjsonrpc.InternalError {
		t.Errorf("JSON-RPC code=%d, want internal error", code)
	}
}

func TestOutputContract(t *testing.T) {
	for _, c := range []struct {
		name, view, cause string
		mutate            func(*presence.Record)
		absent            bool
	}{
		{name: "valid", view: "default"},
		{name: "implicit_default"},
		{name: "tiny", view: "tiny", mutate: func(r *presence.Record) {
			r.Profile.Name = "x"
		}},
		{name: "unknown", view: "bad-view", cause: loom.InvalidEnumValue},
		{name: "length", view: "default", cause: loom.InvalidLength, mutate: func(r *presence.Record) {
			r.ID = "x"
		}},
		{name: "nested_length", view: "default", cause: loom.InvalidLength, mutate: func(r *presence.Record) {
			r.Profile.Name = "x"
		}},
		{name: "missing_object", view: "default", cause: loom.MissingField, mutate: func(r *presence.Record) {
			r.Profile = nil
		}},
		{name: "missing_array", view: "default", cause: loom.MissingField, mutate: func(r *presence.Record) {
			r.Tags = nil
		}},
		{name: "nil", view: "default", absent: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			result := validOutput(t)
			if c.mutate != nil {
				c.mutate(result)
			}
			if c.absent {
				result = nil
			}
			wantFault := c.cause != "" || c.absent
			viewed, err := presence.NewViewedRecord(result, c.view)
			if wantFault {
				requireOutputFault(t, err, c.cause)
				if viewed != nil {
					t.Errorf("invalid output escaped: %#v", viewed)
				}
			} else if err != nil || viewed == nil {
				t.Errorf("result=%#v error=%v", viewed, err)
			}

			mux := loomhttp.NewMuxer()
			endpoints := presence.NewEndpoints(outputService{result: result, view: c.view}, nil)
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
			for _, request := range []struct{ method, path string }{{"GET", "/show"}, {"POST", "/raw"}} {
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, httptest.NewRequest(request.method, request.path, strings.NewReader("opaque")))
				wantStatus := http.StatusOK
				if wantFault {
					wantStatus = http.StatusInternalServerError
				}
				if response.Code != wantStatus {
					t.Errorf("HTTP status=%d body=%s", response.Code, response.Body.String())
				}
				if wantFault && !strings.Contains(response.Body.String(), "fault") {
					t.Errorf("HTTP fault classification lost: %s", response.Body.String())
				}
				if !wantFault {
					decoded, err := client.DecodeShowResponse(loomhttp.ResponseDecoder, false)(response.Result())
					if err != nil || decoded == nil {
						t.Errorf("valid output decode=%#v err=%v", decoded, err)
					}
				}
			}
		})
	}
}

func TestOutputCollectionContract(t *testing.T) {
	for _, c := range []struct {
		name   string
		result presence.RecordCollection
		bad    bool
	}{
		{name: "nil"},
		{name: "empty", result: presence.RecordCollection{}},
		{name: "valid", result: presence.RecordCollection{validOutput(t)}},
		{name: "nil_element", result: presence.RecordCollection{nil}, bad: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("projection panicked: %v", p)
				}
			}()
			viewed, err := presence.NewViewedRecordCollection(c.result, "default")
			if c.bad {
				requireOutputFault(t, err, loom.InvalidFieldType)
			} else if err != nil || len(viewed.Projected) != len(c.result) {
				t.Errorf("collection=%#v err=%v", viewed, err)
			}
		})
	}
}

func TestOutputFixedViewAndServiceError(t *testing.T) {
	result := validOutput(t)
	result.Profile.Name = "x"
	endpoints := presence.NewEndpoints(outputService{result: result}, nil)
	if _, err := endpoints.Tiny(context.Background(), nil); err != nil {
		t.Errorf("fixed tiny view validated excluded field: %v", err)
	}
	original := loom.MissingFieldError("caller_field", "payload")
	endpoints = presence.NewEndpoints(outputService{result: result, view: "bad-view", failure: original}, nil)
	if _, err := endpoints.Show(context.Background(), nil); err != original {
		t.Errorf("service error replaced: %v", err)
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/show", nil))
	if response.Code != http.StatusBadRequest {
		t.Errorf("service error status=%d", response.Code)
	}
}

type outputEdit struct{ id string }

func (e outputEdit) Edit(ctx context.Context, info *presence.EditInfo, next loom.Endpoint) (any, error) {
	result, err := next(ctx, info.RawPayload())
	if err != nil {
		return nil, err
	}
	result.(*presence.Record).ID = e.id
	return result, nil
}

func TestInterceptorOutputValidation(t *testing.T) {
	for _, id := range []string{"edited", "x"} {
		endpoint := presence.NewEndpoints(outputService{result: validOutput(t), view: "tiny"}, outputEdit{id: id}).Show
		result, err := endpoint(context.Background(), nil)
		if id == "x" {
			requireOutputFault(t, err, loom.InvalidLength)
		} else if err != nil || result == nil {
			t.Errorf("valid edited result=%#v err=%v", result, err)
		}
	}
}

func TestStreamOutputValidation(t *testing.T) {
	result := validOutput(t)
	result.ID = "x"
	sent := make(chan error, 1)
	mux := loomhttp.NewMuxer()
	endpoints := presence.NewEndpoints(outputService{result: result, view: "default", sent: sent}, nil)
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl := client.NewClient("ws", strings.TrimPrefix(srv.URL, "http://"), srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	raw, err := cl.Watch()(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	stream := raw.(*client.WatchClientStream)
	stream.SetView("default")
	if value, err := stream.RecvWithContext(ctx); err == nil {
		t.Errorf("stream emitted invalid output: %#v", value)
	}
	select {
	case err := <-sent:
		requireOutputFault(t, err, loom.InvalidLength)
	case <-ctx.Done():
		t.Errorf("sender did not complete: %v", ctx.Err())
	}
}
`
