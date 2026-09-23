package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestGeneratedInlineBodySelectionCompiles compiles and vets the generated
// code of designs with inline request Body DSLs. The generated body type, the
// constructor that builds it and every use site must agree on pointer
// semantics.
func TestGeneratedInlineBodySelectionCompiles(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{"bodyinlineobject", testdata.PayloadBodyInlineObjectDSL},
		{"bodyobject", testdata.PayloadBodyObjectDSL},
		{"bodyobjectrequired", testdata.PayloadBodyObjectRequiredDSL},
		{"bodyobjectvalidate", testdata.PayloadBodyObjectValidateDSL},
		{"sharedmethodname", testdata.InlineBodySharedMethodNameDSL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/"+tc.name, root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

// TestGeneratedInlineBodySelectionRoundTrip covers inline Body DSLs that select
// payload and result attributes, including user-type and collection
// attributes, while other attributes map to params and headers. The generated
// client and server must compile and round-trip the values.
func TestGeneratedInlineBodySelectionRoundTrip(t *testing.T) {
	root := RunHTTPDSL(t, testdata.InlineBodySelectionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/inlinebodyit", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inline_body_test.go"), []byte(inlineBodySelectionHarness), 0o644))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

const inlineBodySelectionHarness = `package inlinebodyit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	inlinebody "example.com/inlinebodyit/gen/inline_body"
	client "example.com/inlinebodyit/gen/http/inline_body/client"
	server "example.com/inlinebodyit/gen/http/inline_body/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type inlineY = struct {
	Y *string ` + "`" + `json:"y,omitempty"` + "`" + `
}

func str(s string) *string {
	return &s
}

func newClient(t *testing.T) (*client.Client, *httptest.Server) {
	t.Helper()
	endpoints := &inlinebody.Endpoints{
		Primitive: func(_ context.Context, p any) (any, error) {
			payload := p.(*inlinebody.PrimitivePayload)
			return &inlinebody.PrimitiveResult{Name: &payload.Name, H: payload.Q}, nil
		},
		Single: func(_ context.Context, p any) (any, error) {
			payload := p.(*inlinebody.SinglePayload)
			return &inlinebody.SingleResult{Item: payload.Item, H: payload.Q}, nil
		},
		Required: func(_ context.Context, p any) (any, error) {
			payload := p.(*inlinebody.RequiredPayload)
			return &inlinebody.RequiredResult{Item: payload.Item, H: payload.Q}, nil
		},
		Several: func(_ context.Context, p any) (any, error) {
			payload := p.(*inlinebody.SeveralPayload)
			h := payload.Q
			if payload.Count != nil {
				h = str(*payload.Q + "-count")
			}
			return &inlinebody.SeveralResult{Item: payload.Item, Items: payload.Items, Name: payload.Name, H: h}, nil
		},
		InlineAttribute: func(_ context.Context, p any) (any, error) {
			payload := p.(*inlinebody.InlineAttributePayload)
			result := &inlinebody.InlineAttributeResult{H: payload.Q}
			if payload.Inline != nil {
				result.Inline = &inlineY{Y: payload.Inline.Y}
			}
			return result, nil
		},
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	serverURL, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	return client.NewClient(serverURL.Scheme, serverURL.Host, httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false), httpServer
}

func TestInlineBodyRoundTrip(t *testing.T) {
	c, _ := newClient(t)
	ctx := t.Context()

	res, err := c.Primitive()(ctx, &inlinebody.PrimitivePayload{Name: "n", Q: str("q")})
	if err != nil {
		t.Fatalf("primitive: %v", err)
	}
	if r := res.(*inlinebody.PrimitiveResult); r.Name == nil || *r.Name != "n" || r.H == nil || *r.H != "q" {
		t.Errorf("primitive result = %+v", r)
	}

	res, err = c.Single()(ctx, &inlinebody.SinglePayload{Item: &inlinebody.Item{X: str("a")}, Q: str("q")})
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	if r := res.(*inlinebody.SingleResult); r.Item == nil || r.Item.X == nil || *r.Item.X != "a" || *r.H != "q" {
		t.Errorf("single result = %+v", r)
	}
	res, err = c.Single()(ctx, &inlinebody.SinglePayload{})
	if err != nil {
		t.Fatalf("single empty: %v", err)
	}
	if r := res.(*inlinebody.SingleResult); r.Item != nil || r.H != nil {
		t.Errorf("single empty result = %+v", r)
	}

	res, err = c.Required()(ctx, &inlinebody.RequiredPayload{Item: &inlinebody.Item{X: str("b")}, Q: "q"})
	if err != nil {
		t.Fatalf("required: %v", err)
	}
	if r := res.(*inlinebody.RequiredResult); r.Item == nil || *r.Item.X != "b" || r.H != "q" {
		t.Errorf("required result = %+v", r)
	}

	res, err = c.Several()(ctx, &inlinebody.SeveralPayload{
		Item:  &inlinebody.Item{X: str("c")},
		Items: []*inlinebody.Item{{X: str("d")}, {X: str("e")}},
		Name:  "n",
		Count: new(int),
		Q:     str("q"),
	})
	if err != nil {
		t.Fatalf("several: %v", err)
	}
	r := res.(*inlinebody.SeveralResult)
	if r.Item == nil || *r.Item.X != "c" || len(r.Items) != 2 || *r.Items[1].X != "e" || r.Name != "n" || *r.H != "q-count" {
		t.Errorf("several result = %+v", r)
	}

	res, err = c.InlineAttribute()(ctx, &inlinebody.InlineAttributePayload{
		Inline: &inlineY{Y: str("y")},
		Q: str("q"),
	})
	if err != nil {
		t.Fatalf("inline attribute: %v", err)
	}
	if r := res.(*inlinebody.InlineAttributeResult); r.Inline == nil || *r.Inline.Y != "y" || *r.H != "q" {
		t.Errorf("inline attribute result = %+v", r)
	}
}

func TestInlineBodyWire(t *testing.T) {
	_, httpServer := newClient(t)
	cases := []struct {
		path, body string
		status     int
		wantBody   string
	}{
		{"/several?q=v", ` + "`" + `{"item":{"x":"a"},"name":"n"}` + "`" + `, http.StatusOK, ` + "`" + `{"item":{"x":"a"},"name":"n"}` + "`" + `},
		{"/several?q=v", ` + "`" + `{"item":{"x":"a"}}` + "`" + `, http.StatusBadRequest, "name"},
		{"/required?q=v", ` + "`" + `{}` + "`" + `, http.StatusBadRequest, "item"},
		{"/required?q=v", ` + "`" + `{"item":{"x":"z"}}` + "`" + `, http.StatusOK, ` + "`" + `{"item":{"x":"z"}}` + "`" + `},
		{"/single", ` + "`" + `{"item":null}` + "`" + `, http.StatusBadRequest, "decode_payload"},
	}
	for _, tc := range cases {
		resp, err := httpServer.Client().Post(httpServer.URL+tc.path, "application/json", strings.NewReader(tc.body))
		if err != nil {
			t.Fatalf("post %s: %v", tc.path, err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close %s: %v", tc.path, err)
		}
		if resp.StatusCode != tc.status {
			t.Errorf("%s %s: status = %d, want %d (%s)", tc.path, tc.body, resp.StatusCode, tc.status, body)
		}
		if !strings.Contains(string(body), tc.wantBody) {
			t.Errorf("%s %s: body = %s, want it to contain %s", tc.path, tc.body, body, tc.wantBody)
		}
	}
}
`
