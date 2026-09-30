package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestEffectiveConstraintsGeneratedHTTPModule(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		base := Type("DefaultBase", String, func() {
			Default("base")
			MinLength(2)
		})
		inherited := Type("InheritedDefault", base)
		overridden := Type("OverriddenDefault", base, func() {
			Default("override")
		})
		numberBase := Type("NumberBase", Int, func() { Default(7) })
		inheritedNumber := Type("InheritedNumber", numberBase)
		overriddenNumber := Type("OverriddenNumber", inheritedNumber, func() { Default(9) })
		bytesBase := Type("BytesBase", Bytes, func() { Default([]byte("base")) })
		inheritedBytes := Type("InheritedBytes", bytesBase)
		overriddenBytes := Type("OverriddenBytes", inheritedBytes, func() { Default([]byte("override")) })
		defaults := Type("Defaults", func() {
			Attribute("inherited", inherited)
			Attribute("overridden", overridden)
			Attribute("nullable", inherited, func() {
				Nullable()
			})
		})
		transportDefaults := Type("TransportDefaults", func() {
			Attribute("inherited_query", inherited)
			Attribute("overridden_query", overridden)
			Attribute("inherited_header", inherited)
			Attribute("overridden_header", overridden)
			Attribute("inherited_cookie", inherited)
			Attribute("overridden_cookie", overridden)
			Attribute("inherited_number_query", inheritedNumber)
			Attribute("overridden_number_header", overriddenNumber)
			Attribute("inherited_bytes_header", inheritedBytes)
			Attribute("overridden_bytes_cookie", overriddenBytes)
		})
		transportResponse := Type("TransportResponse", func() {
			Attribute("inherited_header", inherited)
			Attribute("overridden_number_header", overriddenNumber)
			Attribute("inherited_bytes_cookie", inheritedBytes)
		})
		baseRequired := Type("BaseRequired", func() {
			Attribute("base", String)
			Attribute("derived", String)
			Required("base")
		})
		derivedRequired := Type("DerivedRequired", baseRequired, func() {
			Required("derived")
		})
		patternBase := Type("PatternBase", String, func() { Pattern("^a") })
		patternDerived := Type("PatternDerived", patternBase, func() { Pattern("b$") })
		formatBase := Type("FormatBase", String, func() { Format(FormatIP) })
		formatDerived := Type("FormatDerived", formatBase, func() { Format(FormatIPv4) })
		clausePayload := Type("ClausePayload", func() {
			Attribute("pattern", patternDerived)
			Attribute("ip", formatDerived)
			Required("pattern", "ip")
		})
		Service("constraints", func() {
			Method("defaults", func() {
				Payload(defaults)
				HTTP(func() {
					POST("/defaults")
					Body(defaults)
				})
			})
			Method("required", func() {
				Payload(derivedRequired)
				HTTP(func() {
					POST("/required")
					Body(derivedRequired)
				})
			})
			Method("parameters", func() {
				Payload(transportDefaults)
				HTTP(func() {
					GET("/parameters")
					Param("inherited_query")
					Param("overridden_query")
					Header("inherited_header:X-Inherited")
					Header("overridden_header:X-Overridden")
					Cookie("inherited_cookie:inherited")
					Cookie("overridden_cookie:overridden")
					Param("inherited_number_query")
					Header("overridden_number_header:X-Number")
					Header("inherited_bytes_header:X-Bytes")
					Cookie("overridden_bytes_cookie:bytes")
				})
			})
			Method("response", func() {
				Result(transportResponse)
				HTTP(func() {
					GET("/response")
					Response(StatusOK, func() {
						Header("inherited_header:X-Result")
						Header("overridden_number_header:X-Result-Number")
						Cookie("inherited_bytes_cookie:result_bytes")
					})
				})
			})
			Method("clauses", func() {
				Payload(clausePayload)
				HTTP(func() {
					POST("/clauses")
					Body(clausePayload)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/effectiveconstraints", root)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "effective_constraints_test.go"),
		[]byte(effectiveConstraintsHTTPHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const effectiveConstraintsHTTPHarness = `package effectiveconstraints_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	constraints "example.com/effectiveconstraints/gen/constraints"
	server "example.com/effectiveconstraints/gen/http/constraints/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	defaults []byte
	required *constraints.DerivedRequired
	parameters []byte
	clauses int
	err error
}

func (s *service) Defaults(_ context.Context, payload *constraints.Defaults2) error {
	s.defaults, s.err = json.Marshal(payload, json.Deterministic(true))
	return s.err
}

func (s *service) Required(_ context.Context, payload *constraints.DerivedRequired) error {
	s.required = payload
	return nil
}

func (s *service) Parameters(_ context.Context, payload *constraints.TransportDefaults) error {
	s.parameters, s.err = json.Marshal(payload, json.Deterministic(true))
	return s.err
}

func (s *service) Response(context.Context) (*constraints.TransportResponse, error) {
	return &constraints.TransportResponse{
		InheritedHeader: constraints.InheritedDefault("response"),
		OverriddenNumberHeader: constraints.OverriddenNumber(11),
		InheritedBytesCookie: constraints.InheritedBytes("response-bytes"),
	}, nil
}

func (s *service) Clauses(context.Context, *constraints.ClausePayload) error {
	s.clauses++
	return nil
}

func TestEffectiveDefaultsAndRequiredFields(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(constraints.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	post := func(path, body string) (int, []byte) {
		response, err := httpServer.Client().Post(httpServer.URL+path, "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, raw
	}

	status, raw := post("/defaults", "{}")
	if status != http.StatusNoContent {
		t.Fatalf("defaults status %d: %s", status, raw)
	}
	if svc.err != nil {
		t.Fatalf("marshal defaults: %v", svc.err)
	}
	if got, want := string(svc.defaults), "{\"inherited\":\"base\",\"overridden\":\"override\",\"nullable\":\"base\"}"; got != want {
		t.Errorf("defaults %s, want %s", got, want)
	}

	status, raw = post("/defaults", "{\"nullable\":null}")
	if status != http.StatusNoContent {
		t.Fatalf("nullable status %d: %s", status, raw)
	}
	if got, want := string(svc.defaults), "{\"inherited\":\"base\",\"overridden\":\"override\",\"nullable\":null}"; got != want {
		t.Errorf("nullable defaults %s, want %s", got, want)
	}

	status, _ = post("/required", "{\"base\":\"one\"}")
	if status != http.StatusBadRequest || svc.required != nil {
		t.Errorf("missing derived required: status %d, payload %#v", status, svc.required)
	}
	status, raw = post("/required", "{\"base\":\"one\",\"derived\":\"two\"}")
	if status != http.StatusNoContent {
		t.Fatalf("required status %d: %s", status, raw)
	}
	if svc.required == nil || svc.required.Base != "one" || svc.required.Derived != "two" {
		t.Errorf("required payload %#v", svc.required)
	}

	status, raw = post("/clauses", "{\"pattern\":\"ab\",\"ip\":\"192.0.2.1\"}")
	if status != http.StatusNoContent || svc.clauses != 1 {
		t.Fatalf("valid clauses status %d calls %d: %s", status, svc.clauses, raw)
	}
	for _, body := range []string{
		"{\"pattern\":\"xb\",\"ip\":\"192.0.2.1\"}",
		"{\"pattern\":\"ax\",\"ip\":\"192.0.2.1\"}",
		"{\"pattern\":\"ab\",\"ip\":\"2001:db8::1\"}",
	} {
		status, _ = post("/clauses", body)
		if status != http.StatusBadRequest || svc.clauses != 1 {
			t.Errorf("invalid clauses %s: status %d calls %d", body, status, svc.clauses)
		}
	}

	request, err := http.NewRequest(http.MethodGet, httpServer.URL+"/parameters", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("parameters status %d", response.StatusCode)
	}
	if svc.err != nil {
		t.Fatalf("marshal parameters: %v", svc.err)
	}
	if got, want := string(svc.parameters), "{\"inherited_query\":\"base\",\"overridden_query\":\"override\",\"inherited_header\":\"base\",\"overridden_header\":\"override\",\"inherited_cookie\":\"base\",\"overridden_cookie\":\"override\",\"inherited_number_query\":7,\"overridden_number_header\":9,\"inherited_bytes_header\":\"YmFzZQ==\",\"overridden_bytes_cookie\":\"b3ZlcnJpZGU=\"}"; got != want {
		t.Errorf("parameter defaults %s, want %s", got, want)
	}

	request, err = http.NewRequest(http.MethodGet, httpServer.URL+"/parameters?inherited_query=query-one&overridden_query=query-two&inherited_number_query=17", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Inherited", "header-one")
	request.Header.Set("X-Overridden", "header-two")
	request.Header.Set("X-Number", "19")
	request.Header.Set("X-Bytes", "bytes-header")
	request.AddCookie(&http.Cookie{Name: "inherited", Value: "cookie-one"})
	request.AddCookie(&http.Cookie{Name: "overridden", Value: "cookie-two"})
	request.AddCookie(&http.Cookie{Name: "bytes", Value: "bytes-cookie"})
	response, err = httpServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("explicit parameters status %d", response.StatusCode)
	}
	if got, want := string(svc.parameters), "{\"inherited_query\":\"query-one\",\"overridden_query\":\"query-two\",\"inherited_header\":\"header-one\",\"overridden_header\":\"header-two\",\"inherited_cookie\":\"cookie-one\",\"overridden_cookie\":\"cookie-two\",\"inherited_number_query\":17,\"overridden_number_header\":19,\"inherited_bytes_header\":\"Ynl0ZXMtaGVhZGVy\",\"overridden_bytes_cookie\":\"Ynl0ZXMtY29va2ll\"}"; got != want {
		t.Errorf("explicit parameters %s, want %s", got, want)
	}

	response, err = httpServer.Client().Get(httpServer.URL + "/response")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("response status %d", response.StatusCode)
	}
	if got := response.Header.Get("X-Result"); got != "response" {
		t.Errorf("response header %q", got)
	}
	if got := response.Header.Get("X-Result-Number"); got != "11" {
		t.Errorf("response number header %q", got)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "result_bytes" && cookie.Value == "response-bytes" {
			return
		}
	}
	t.Error("response bytes cookie missing")
}
`
