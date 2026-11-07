package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestGeneratedResponseBodyNullAdmission exercises generated HTTP clients
// against raw success and error bodies, and against the generated server.
func TestGeneratedResponseBodyNullAdmission(t *testing.T) {
	root := RunHTTPDSL(t, responseBodyNullDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/responsenull", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_null_test.go"), []byte(responseBodyNullHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

func responseBodyNullDSL() {
	widget := Type("Widget", func() {
		Attribute("name", String)
		Required("name")
	})
	Service("receiver", func() {
		Method("scalar", func() {
			Result(String)
			Error("strict", String)
			HTTP(func() {
				GET("/scalar")
				Response(StatusOK)
				Response("strict", StatusBadRequest)
			})
		})
		Method("nullable_scalar", func() {
			Result(String, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/nullable_scalar")
			})
		})
		Method("object", func() {
			Result(widget)
			HTTP(func() {
				GET("/object")
			})
		})
		for _, child := range []struct {
			name     string
			nullable bool
			required bool
		}{
			{"child", false, true},
			{"nullable_child", true, true},
			{"optional_child", false, false},
		} {
			Method(child.name, func() {
				Result(func() {
					Attribute("value", String, func() {
						if child.nullable {
							Nullable()
						}
					})
					if child.required {
						Required("value")
					}
				})
				HTTP(func() {
					GET("/" + child.name)
					Response(StatusOK, func() {
						Body("value")
					})
				})
			})
		}
	})
}

const responseBodyNullHarness = `package responsenull_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/responsenull/gen/http/receiver/client"
	server "example.com/responsenull/gen/http/receiver/server"
	"example.com/responsenull/gen/receiver"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type outcome int

const (
	accepted outcome = iota
	rejectedNull
	rejectedEOF
	rejectedSyntax
)

var wire struct {
	status int
	body   string
}

func newClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()
	hs := httptest.NewServer(handler)
	t.Cleanup(hs.Close)
	return client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
}

func TestRawResponseBodies(t *testing.T) {
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wire.status)
		if _, err := io.WriteString(w, wire.body); err != nil {
			panic(err)
		}
	}))
	endpoints := map[string]func(context.Context, any) (any, error){
		"scalar": c.Scalar(), "nullable_scalar": c.NullableScalar(), "object": c.Object(),
		"child": c.Child(), "nullable_child": c.NullableChild(), "optional_child": c.OptionalChild(),
	}
	strict := map[string]outcome{"null": rejectedNull, "padded null": rejectedNull}
	permissive := map[string]outcome{"null": accepted, "padded null": accepted}
	bodies := []struct {
		name, body string
	}{
		{"null", "null"}, {"padded null", " \n null\t"}, {"empty", ""}, {"whitespace", " \n\t"},
		{"malformed", "!"}, {"truncated", "{"},
	}
	for _, tc := range []struct {
		endpoint string
		status   int
		null     map[string]outcome
		valid    string
		check    func(t *testing.T, res any, err error)
	}{
		{"scalar", http.StatusOK, strict, ` + "`" + `"ok"` + "`" + `, func(t *testing.T, res any, err error) {
			if err != nil || res.(string) != "ok" {
				t.Errorf("scalar = %#v, %v", res, err)
			}
		}},
		{"scalar", http.StatusBadRequest, strict, ` + "`" + `"bad"` + "`" + `, func(t *testing.T, res any, err error) {
			var strictError receiver.Strict
			if res != nil || !errors.As(err, &strictError) || strictError != "bad" {
				t.Errorf("strict error = %#v, %#v", res, err)
			}
		}},
		{"object", http.StatusOK, strict, ` + "`" + `{"name":"ok"}` + "`" + `, func(t *testing.T, res any, err error) {
			if err != nil || res.(*receiver.Widget).Name != "ok" {
				t.Errorf("object = %#v, %v", res, err)
			}
		}},
		{"child", http.StatusOK, strict, ` + "`" + `"ok"` + "`" + `, func(t *testing.T, res any, err error) {
			if err != nil || res.(*receiver.ChildResult).Value != "ok" {
				t.Errorf("child = %#v, %v", res, err)
			}
		}},
		{"nullable_scalar", http.StatusOK, permissive, ` + "`" + `"ok"` + "`" + `, func(t *testing.T, res any, err error) {
			value, ok := res.(loom.Nullable[string]).Value()
			if err != nil || !ok || value != "ok" {
				t.Errorf("nullable scalar = %#v, %v", res, err)
			}
		}},
		{"nullable_child", http.StatusOK, permissive, ` + "`" + `"ok"` + "`" + `, func(t *testing.T, res any, err error) {
			value, ok := res.(*receiver.NullableChildResult).Value.Value()
			if err != nil || !ok || value != "ok" {
				t.Errorf("nullable child = %#v, %v", res, err)
			}
		}},
		{"optional_child", http.StatusOK, strict, ` + "`" + `"ok"` + "`" + `, func(t *testing.T, res any, err error) {
			value := res.(*receiver.OptionalChildResult).Value
			if err != nil || value == nil || *value != "ok" {
				t.Errorf("optional child = %#v, %v", res, err)
			}
		}},
	} {
		for _, body := range bodies {
			t.Run(tc.endpoint+"/"+http.StatusText(tc.status)+"/"+body.name, func(t *testing.T) {
				wire.status, wire.body = tc.status, body.body
				res, err := endpoints[tc.endpoint](t.Context(), nil)
				want, ok := tc.null[body.name]
				switch {
				case ok:
				case body.name == "empty" || body.name == "whitespace":
					want = rejectedEOF
				default:
					want = rejectedSyntax
				}
				assertOutcome(t, res, err, want)
				if want == accepted {
					assertNull(t, tc.endpoint, res)
				}
			})
		}
		t.Run(tc.endpoint+"/"+http.StatusText(tc.status)+"/valid", func(t *testing.T) {
			wire.status, wire.body = tc.status, tc.valid
			res, err := endpoints[tc.endpoint](t.Context(), nil)
			tc.check(t, res, err)
		})
	}
	t.Run("unexpected status", func(t *testing.T) {
		wire.status, wire.body = http.StatusTeapot, "null"
		_, err := c.Scalar()(t.Context(), nil)
		var clientError *loomhttp.ClientError
		if !errors.As(err, &clientError) || clientError.Name != "invalid_response" {
			t.Errorf("unexpected status error = %v", err)
		}
	})
}

func TestGeneratedServerRoundTrip(t *testing.T) {
	endpoints := &receiver.Endpoints{
		Scalar: func(context.Context, any) (any, error) {
			return "", nil
		},
		NullableScalar: func(context.Context, any) (any, error) {
			return loom.NullValue[string](), nil
		},
		Object: func(context.Context, any) (any, error) {
			return &receiver.Widget{}, nil
		},
		Child: func(context.Context, any) (any, error) {
			return &receiver.ChildResult{}, nil
		},
		NullableChild: func(context.Context, any) (any, error) {
			return &receiver.NullableChildResult{Value: loom.NullValue[string]()}, nil
		},
		OptionalChild: func(context.Context, any) (any, error) {
			value := ""
			return &receiver.OptionalChildResult{Value: &value}, nil
		},
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	c := newClient(t, mux)
	if res, err := c.Scalar()(t.Context(), nil); err != nil || res.(string) != "" {
		t.Errorf("scalar zero = %#v, %v", res, err)
	}
	if res, err := c.NullableScalar()(t.Context(), nil); err != nil || !res.(loom.Nullable[string]).IsNull() {
		t.Errorf("nullable scalar null = %#v, %v", res, err)
	}
	if res, err := c.Object()(t.Context(), nil); err != nil || res.(*receiver.Widget).Name != "" {
		t.Errorf("object zero = %#v, %v", res, err)
	}
	if res, err := c.Child()(t.Context(), nil); err != nil || res.(*receiver.ChildResult).Value != "" {
		t.Errorf("child zero = %#v, %v", res, err)
	}
	if res, err := c.NullableChild()(t.Context(), nil); err != nil || !res.(*receiver.NullableChildResult).Value.IsNull() {
		t.Errorf("nullable child null = %#v, %v", res, err)
	}
	if res, err := c.OptionalChild()(t.Context(), nil); err != nil || *res.(*receiver.OptionalChildResult).Value != "" {
		t.Errorf("optional child zero = %#v, %v", res, err)
	}
}

func assertOutcome(t *testing.T, res any, err error, want outcome) {
	t.Helper()
	if want == accepted {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}
	var clientError *loomhttp.ClientError
	if res != nil || !errors.As(err, &clientError) || clientError.Name != "decoding_error" {
		t.Fatalf("want decoding error, got %#v, %v", res, err)
	}
	isNull := strings.Contains(err.Error(), "JSON null is not allowed")
	isEOF := errors.Is(err, io.EOF)
	switch want {
	case rejectedNull:
		if !isNull || isEOF || !strings.Contains(err.Error(), "JSON null is not allowed for this response body") {
			t.Errorf("want null rejection, got %v", err)
		}
	case rejectedEOF:
		if !isEOF || isNull {
			t.Errorf("want EOF, got %v", err)
		}
	case rejectedSyntax:
		if isEOF || isNull {
			t.Errorf("want syntax error, got %v", err)
		}
	}
}

func assertNull(t *testing.T, endpoint string, res any) {
	t.Helper()
	switch endpoint {
	case "nullable_scalar":
		if !res.(loom.Nullable[string]).IsNull() {
			t.Errorf("nullable scalar = %#v, want null", res)
		}
	case "nullable_child":
		if !res.(*receiver.NullableChildResult).Value.IsNull() {
			t.Errorf("nullable child = %#v, want null", res)
		}
	}
}
`
