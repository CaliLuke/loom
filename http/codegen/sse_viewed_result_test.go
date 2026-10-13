package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSSEViewedResultBodySelection(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEViewedResultDSL)
	data := CreateHTTPServices(root).Get("ViewedSSE")
	for _, method := range []string{"Watch", "List", "Mapped", "Number", "Required", "Defaulted", "Full"} {
		t.Run(method, func(t *testing.T) {
			ed := data.Endpoint(method)
			code := renderSection(t, serverSSESection(ed))
			full := viewedServerBody(ed.Result.Responses[0].ServerBody, "full")
			require.Contains(t, code, full.Init.Name+"(res.Projected)")
			if method != "Full" {
				require.Contains(t, code, "switch res.View")
				normal := viewedServerBody(ed.Result.Responses[0].ServerBody, "default")
				require.Contains(t, code, normal.Init.Name+"(res.Projected)")
			} else {
				require.NotContains(t, code, "switch res.View")
			}
		})
	}
}

func TestSSEViewedResultGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEViewedResultDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/viewedsse", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "viewed_sse_test.go"), []byte(sseViewedResultHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

const sseViewedResultHarness = `package viewedsse_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	client "example.com/viewedsse/gen/http/viewed_sse/client"
	server "example.com/viewedsse/gen/http/viewed_sse/server"
	svc "example.com/viewedsse/gen/viewed_sse"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	views []string
	nilValue bool
	sendError error
}

func item() *svc.Item {
	name, secret := "name", "secret"
	return &svc.Item{ID: "a", Name: &name, Secret: &secret, Required: "required", Defaulted: "authored"}
}

func (s *service) Watch(ctx context.Context, stream svc.WatchServerStream) error {
	for _, view := range s.views {
		stream.SetView(view)
		value := item()
		if s.nilValue {
			value = nil
		}
		if err := stream.SendWithContext(ctx, value); err != nil {
			s.sendError = err
			return err
		}
	}
	return nil
}

func (s *service) List(ctx context.Context, stream svc.ListServerStream) error {
	stream.SetView("full")
	return stream.SendWithContext(ctx, svc.ItemCollection{item()})
}

type mappedStream interface {
	SetView(string)
	SendWithContext(context.Context, *svc.Item) error
}

func (s *service) sendMapped(ctx context.Context, stream mappedStream) error {
	for _, view := range s.views {
		stream.SetView(view)
		value := item()
		count := 5
		value.Count = &count
		if err := stream.SendWithContext(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) Mapped(ctx context.Context, stream svc.MappedServerStream) error {
	return s.sendMapped(ctx, stream)
}

func (s *service) Number(ctx context.Context, stream svc.NumberServerStream) error {
	return s.sendMapped(ctx, stream)
}

func (s *service) Required(ctx context.Context, stream svc.RequiredServerStream) error {
	return s.sendMapped(ctx, stream)
}

func (s *service) Defaulted(ctx context.Context, stream svc.DefaultedServerStream) error {
	return s.sendMapped(ctx, stream)
}

func (s *service) Full(ctx context.Context, stream svc.FullServerStream) error {
	return stream.SendWithContext(ctx, item())
}

func response(t *testing.T, impl *service, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := loomhttp.NewMuxer()
	errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
		t.Errorf("unexpected transport error: %v", err)
	}
	server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler, nil))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestSelectedViewWireAndClient(t *testing.T) {
	views := []string{"default", "full", "", "full"}
	rec := response(t, &service{views: views}, "/watch")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Errorf("content type: %q", rec.Header().Get("Content-Type"))
	}
	frames := strings.Split(strings.TrimSuffix(rec.Body.String(), "\n\n"), "\n\n")
	if len(frames) != len(views) {
		t.Fatalf("want %d frames, got %q", len(views), frames)
	}
	stream := client.NewWatchStream(&http.Response{Body: io.NopCloser(strings.NewReader(rec.Body.String()))}, loomhttp.ResponseDecoder)
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	})
	for i, view := range views {
		want := map[string]string{"id": "a", "name": "name", "required": "required", "defaulted": "authored"}
		if view == "full" {
			want["secret"] = "secret"
		}
		var wire map[string]string
		if err := json.Unmarshal([]byte(strings.TrimPrefix(frames[i], "data: ")), &wire); err != nil {
			t.Fatalf("frame %d decode: %v (%q)", i, err, frames[i])
		}
		if !reflect.DeepEqual(want, wire) {
			t.Errorf("frame %d view %q: want %#v, got %#v", i, view, want, wire)
		}
		value, err := stream.Recv(context.Background())
		if err != nil {
			t.Fatalf("recv %d: %v", i, err)
		}
		if value.ID != "a" || value.Name == nil || *value.Name != "name" {
			t.Errorf("recv %d: %#v", i, value)
		}
		if view == "full" {
			if value.Secret == nil || *value.Secret != "secret" {
				t.Errorf("recv %d full view lost secret: %#v", i, value)
			}
		} else if value.Secret != nil {
			t.Errorf("recv %d default view leaked secret: %#v", i, value)
		}
	}
	if _, err := stream.Recv(context.Background()); err != io.EOF {
		t.Errorf("want EOF, got %v", err)
	}
}

func TestFixedFullView(t *testing.T) {
	rec := response(t, &service{}, "/full")
	stream := client.NewFullStream(&http.Response{Body: io.NopCloser(strings.NewReader(rec.Body.String()))}, loomhttp.ResponseDecoder)
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	})
	value, err := stream.Recv(context.Background())
	if err != nil || value == nil || value.Secret == nil || *value.Secret != "secret" {
		t.Errorf("fixed full: %#v, %v; wire %q", value, err, rec.Body.String())
	}
}

func TestCollectionFullView(t *testing.T) {
	rec := response(t, &service{}, "/list")
	stream := client.NewListStream(&http.Response{Body: io.NopCloser(strings.NewReader(rec.Body.String()))}, loomhttp.ResponseDecoder)
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	})
	value, err := stream.Recv(context.Background())
	if err != nil || len(value) != 1 || value[0].Secret == nil || *value[0].Secret != "secret" {
		t.Errorf("collection full: %#v, %v; wire %q", value, err, rec.Body.String())
	}
}

func TestMappedDataOmittedByView(t *testing.T) {
	for _, tc := range []struct {
		path, want string
	}{
		{"/mapped", strings.Repeat("data: name\n\n", 3)},
		{"/number", strings.Repeat("data: 5\n\n", 3)+"data: null\n\n"},
		{"/required", strings.Repeat("data: required\n\n", 3)},
		{"/defaulted", strings.Repeat("data: authored\n\n", 3)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := response(t, &service{views: []string{"default", "full", "", "tiny"}}, tc.path)
			if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
				t.Errorf("mapped data: status %d, want %q, got %q", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestInvalidSelectionDoesNotStartSSE(t *testing.T) {
	for _, tc := range []struct {
		name string
		view string
		nilValue bool
	}{
		{name: "unknown view", view: "missing"},
		{name: "nil result", view: "full", nilValue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			impl := &service{views: []string{tc.view}, nilValue: tc.nilValue}
			rec := response(t, impl, "/watch")
			if impl.sendError == nil || rec.Code < 400 {
				t.Errorf("want rejected send and HTTP error, got %v, %d", impl.sendError, rec.Code)
			}
			if strings.Contains(rec.Header().Get("Content-Type"), "text/event-stream") || strings.HasPrefix(rec.Body.String(), "data:") {
				t.Errorf("invalid selection committed SSE: %q", rec.Body.String())
			}
		})
	}
}
`
