package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

type nullableNamedBodyCase struct {
	name       string
	primitive  expr.Primitive
	definition string
	value      string
}

var nullableNamedBodyCases = []nullableNamedBodyCase{
	{"string", String, "string", `"éé"`},
	{"bool", Boolean, "bool", "true"},
	{"int", Int, "int", "7"},
	{"int32", Int32, "int32", "7"},
	{"int64", Int64, "int64", "7"},
	{"uint", UInt, "uint", "7"},
	{"uint32", UInt32, "uint32", "7"},
	{"uint64", UInt64, "uint64", "7"},
	{"float32", Float32, "float32", "1.5"},
	{"float64", Float64, "float64", "1.5"},
	{"bytes", Bytes, "[]byte", `"YWI="`},
}

// TestNullableNamedPrimitiveBodyDeclaration checks that every named scalar body
// has a declaration and retains its nullable value and requiredness.
func TestNullableNamedPrimitiveBodyDeclaration(t *testing.T) {
	for _, c := range nullableNamedBodyCases {
		for _, required := range []bool{false, true} {
			t.Run(c.name+"/required="+strconv.FormatBool(required), func(t *testing.T) {
				root := RunHTTPDSL(t, nullableNamedBodyDSL(c.primitive, required))
				request := CreateHTTPServices(root).Get("sender").Endpoint("send").Payload.Request
				require.Equal(t, c.definition, request.ServerBody.Def)
				require.Equal(t, c.definition, request.ClientBody.Def)
				require.Equal(t, "SendRequestBody", request.ServerBody.VarName)
				require.Equal(t, "loom.Nullable[SendRequestBody]", request.ServerBody.ValueRef)
				require.Equal(t, "loom.Nullable[SendRequestBody]", request.ClientBody.ValueRef)
				require.Equal(t, !required, request.OptionalBodyAttribute)
			})
		}
	}
}

// TestNullableNamedPrimitiveBodyRoundTrip exercises all scalar body types through
// generated clients and servers, including absent, null, value, and invalid input.
func TestNullableNamedPrimitiveBodyRoundTrip(t *testing.T) {
	for _, c := range nullableNamedBodyCases {
		for _, required := range []bool{false, true} {
			t.Run(c.name+"/required="+strconv.FormatBool(required), func(t *testing.T) {
				root := RunHTTPDSL(t, nullableNamedBodyDSL(c.primitive, required))
				dir := t.TempDir()
				renderHTTPModule(t, dir, "example.com/namedbody", root)
				invalid, code, detail := "0", "invalid_range", "validation error"
				switch c.primitive {
				case String:
					invalid, code = `"é"`, "invalid_length"
				case Bytes:
					invalid, code = `"YQ=="`, "invalid_length"
				case Boolean:
					invalid, code = "false", "invalid_enum_value"
					detail = `invalid value for "body": got false, expected one of true`
				}
				harness := strings.NewReplacer(
					"REQUIRED", strconv.FormatBool(required),
					"VALID_JSON", strconv.Quote(c.value),
					"INVALID_JSON", strconv.Quote(invalid),
					"VALIDATION_CODE", strconv.Quote(code),
					"VALIDATION_DETAIL", strconv.Quote(detail),
				).Replace(nullableNamedBodyHarness)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "named_body_test.go"), []byte(harness), 0o600))
				runGoCommand(t, dir, "mod", "tidy")
				runGoCommand(t, dir, "vet", "./...")
				runGoCommand(t, dir, "test", "-count=1", "./...")
			})
		}
	}
}

func nullableNamedBodyDSL(typ expr.DataType, required bool) func() {
	return func() {
		value := Type("Value", typ, func() {
			Nullable()
			switch typ {
			case String, Bytes:
				MinLength(2)
			case Boolean:
				Enum(true)
			default:
				if expr.IsPrimitive(typ) {
					Minimum(1)
				}
			}
		})
		Service("sender", func() {
			Method("send", func() {
				Payload(func() {
					Attribute("b", value)
					if required {
						Required("b")
					}
				})
				HTTP(func() {
					POST("/")
					Body("b")
				})
			})
		})
	}
}

const nullableNamedBodyHarness = `package namedbody_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/namedbody/gen/http/sender/client"
	server "example.com/namedbody/gen/http/sender/server"
	"example.com/namedbody/gen/sender"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestNamedBodyStates(t *testing.T) {
	const required = REQUIRED
	const valid = VALID_JSON
	seen := make(chan *sender.SendPayload, 1)
	endpoints := &sender.Endpoints{Send: func(_ context.Context, p any) (any, error) {
		seen <- p.(*sender.SendPayload)
		return nil, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	var value sender.Value
	if err := json.Unmarshal([]byte(valid), &value); err != nil {
		t.Fatal(err)
	}
	states := []struct {
		name  string
		value loom.Nullable[sender.Value]
	}{
		{"absent", loom.Nullable[sender.Value]{}},
		{"null", loom.NullValue[sender.Value]()},
		{"value", loom.NullableValue(value)},
	}
	for _, state := range states {
		t.Run("client/"+state.name, func(t *testing.T) {
			_, err := c.Send()(t.Context(), &sender.SendPayload{B: state.value})
			if state.name == "absent" && required {
				if err == nil {
					t.Error("required absent body accepted")
				}
				assertNoCall(t, seen)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertState(t, seen, state.name, valid)
		})
	}
	cases := []struct {
		name, body, state, code, detail string
		status                          int
	}{
		{"empty", "", "absent", "", "", http.StatusNoContent},
		{"whitespace", " \n\t", "absent", "", "", http.StatusNoContent},
		{"null", "null", "null", "", "", http.StatusNoContent},
		{"value", valid, "value", "", "", http.StatusNoContent},
		{"malformed", "?", "", "decode_payload", "invalid request body", http.StatusBadRequest},
		{"truncated", "[", "", "decode_payload", "invalid request body", http.StatusBadRequest},
		{"invalid value", INVALID_JSON, "", VALIDATION_CODE, VALIDATION_DETAIL, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run("server/"+tc.name, func(t *testing.T) {
			if required && tc.state == "absent" {
				tc.status, tc.code, tc.detail = http.StatusBadRequest, "missing_payload", "validation error"
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, hs.URL, strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := hs.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read=%v close=%v", readErr, closeErr)
			}
			if resp.StatusCode != tc.status {
				t.Errorf("status=%d want=%d body=%s", resp.StatusCode, tc.status, body)
			}
			if tc.code != "" {
				var problem loomhttp.ProblemResponse
				if err := json.Unmarshal(body, &problem); err != nil {
					t.Fatal(err)
				}
				if problem.Code != tc.code || problem.Detail != tc.detail {
					t.Errorf("problem=%+v want code=%s detail=%s", problem, tc.code, tc.detail)
				}
				assertNoCall(t, seen)
				return
			}
			assertState(t, seen, tc.state, valid)
		})
	}
}

func assertNoCall(t *testing.T, seen <-chan *sender.SendPayload) {
	t.Helper()
	select {
	case p := <-seen:
		t.Errorf("unexpected service call: %+v", p)
	default:
	}
}

func assertState(t *testing.T, seen <-chan *sender.SendPayload, want, value string) {
	t.Helper()
	select {
	case p := <-seen:
		got := "absent"
		if p.B.IsNull() {
			got = "null"
		} else if p.B.Present() {
			got = "value"
		}
		if got != want {
			t.Errorf("state=%s want=%s", got, want)
		}
		if got == "value" {
			v, ok := p.B.Value()
			b, err := json.Marshal(v)
			if !ok || err != nil || string(b) != value {
				t.Errorf("value=%s present=%v err=%v want=%s", b, ok, err, value)
			}
		}
	default:
		t.Error("service was not invoked")
	}
}
`
