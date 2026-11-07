package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func integerStorage(dt expr.DataType, goType string, imports ...string) *expr.AttributeExpr {
	return &expr.AttributeExpr{Type: dt, Meta: expr.MetaExpr{"struct:field:type": append([]string{goType}, imports...)}}
}

func TestRenderIntegerText(t *testing.T) {
	ctx := serviceContext("", codegen.NewNameScope())
	cases := []struct {
		name     string
		wire     expr.DataType
		physical *expr.AttributeExpr
		want     string
	}{
		{"native", expr.Int, &expr.AttributeExpr{Type: expr.Int}, "text = strconv.Itoa(v)\n"},
		{"narrow storage widens", expr.Int, integerStorage(expr.Int, "int8"), "text = strconv.Itoa(int(v))\n"},
		{"wider storage is checked", expr.Int, integerStorage(expr.Int, "int64"),
			"textWire := int(v)\nif int64(textWire) != v || (v < 0) != (textWire < 0) {\nreturn \"\", fmt.Errorf(\"value %v cannot be represented as int\", v)\n}\ntext = strconv.Itoa(textWire)\n"},
		{"signed storage under unsigned wire", expr.UInt, integerStorage(expr.UInt, "int8"),
			"textWire := uint(v)\nif int8(textWire) != v || (v < 0) != (textWire < 0) {\nreturn \"\", fmt.Errorf(\"value %v cannot be represented as uint\", v)\n}\ntext = strconv.FormatUint(uint64(textWire), 10)\n"},
		{"opaque storage asserts integer capability", expr.Int, integerStorage(expr.Int, "time.Duration", "time"),
			"const _ = ^time.Duration(0)\ntextWire := int(v)\nif time.Duration(textWire) != v || (v < 0) != (textWire < 0) {\nreturn \"\", fmt.Errorf(\"value %v cannot be represented as int\", v)\n}\ntext = strconv.Itoa(textWire)\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, renderIntegerText(tc.wire, tc.physical, ctx, "v", "text", "textWire", `"", `))
		})
	}
}

func TestRenderIntegerDecimal(t *testing.T) {
	cases := []struct {
		name       string
		physical   *expr.AttributeExpr
		statements string
		text       string
	}{
		{"signed storage", integerStorage(expr.Int, "int8"), "", "strconv.FormatInt(int64(v), 10)"},
		{"unsigned storage", integerStorage(expr.Int, "uint64"), "", "strconv.FormatUint(uint64(v), 10)"},
		{"named unsigned storage", &expr.AttributeExpr{Type: &expr.UserTypeExpr{AttributeExpr: integerStorage(expr.Int, "uint64"), TypeName: "Big"}}, "", "strconv.FormatUint(uint64(v), 10)"},
		{"opaque storage splits by sign", integerStorage(expr.Int, "custom.Number", "example.com/custom"),
			"const _ = ^custom.Number(0)\nvText := strconv.FormatUint(uint64(v), 10)\nif v < 0 {\nvText = strconv.FormatInt(int64(v), 10)\n}\n", "vText"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var code strings.Builder
			require.Equal(t, tc.text, renderIntegerDecimal(&code, tc.physical, "custom.Number", "v", "vText"))
			require.Equal(t, tc.statements, code.String())
		})
	}
}

func TestIntegerElementText(t *testing.T) {
	ctx := serviceContext("", codegen.NewNameScope())
	mapping := &expr.Map{KeyType: integerStorage(expr.Int, "int16"), ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: integerStorage(expr.Int, "uint64")}}}
	data := &AttributeData{Type: mapping, fieldAttribute: &expr.AttributeExpr{Type: mapping}, fieldContext: ctx}
	require.Equal(t, "keyStr := strconv.Itoa(int(key))", integerElementText(data, "key", "keyStr", "key"))
	require.Equal(t, "valStrWire := int(val)\nif uint64(valStrWire) != val || (val < 0) != (valStrWire < 0) {\nreturn fmt.Errorf(\"value %v cannot be represented as int\", val)\n}\nvalStr := strconv.Itoa(valStrWire)",
		integerElementText(data, "elem.elem", "valStr", "val"))
	native := &AttributeData{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}, fieldContext: ctx}
	native.fieldAttribute = &expr.AttributeExpr{Type: native.Type}
	require.Empty(t, integerElementText(native, "elem", "valueStr", "value"))
}

// integerTextBoundaryDSL stores integer transport values in non-default Go
// types across headers, cookies, styled and ordinary paths and queries,
// header lists and optional values, plus named cross-sign result storage.
func integerTextBoundaryDSL() {
	API("integers", func() {
	})
	big := Type("Big", Int, func() {
		Meta("struct:field:type", "uint64")
	})
	neg := Type("Neg", UInt, func() {
		Meta("struct:field:type", "int8")
	})
	out := Type("Out", func() {
		Attribute("c64", Int, func() {
			Meta("struct:field:type", "int64")
		})
		Attribute("big", big)
		Attribute("neg", neg)
		Attribute("rc", Int, func() {
			Meta("struct:field:type", "int16")
		})
		Attribute("message", String)
		Required("c64", "big", "neg", "rc", "message")
	})
	Service("conv", func() {
		Method("encode", func() {
			Payload(func() {
				Attribute("hdr", Int, func() {
					Meta("struct:field:type", "int64")
				})
				Attribute("dur", Int, func() {
					Meta("struct:field:type", "time.Duration", "time")
				})
				Attribute("ck", Int, func() {
					Meta("struct:field:type", "int32")
				})
				Attribute("d", Int, func() {
					Meta("struct:field:type", "time.Duration", "time")
				})
				Attribute("num", Int, func() {
					Meta("struct:field:type", "custom.Number", "example.com/integers/custom")
				})
				Attribute("bp", big)
				Attribute("nums", ArrayOf(Int, func() {
					Meta("struct:field:type", "custom.Number", "example.com/integers/custom")
				}))
				Attribute("lbl", Int, func() {
					Meta("struct:field:type", "int64")
					Meta("openapi:style", "label")
				})
				Attribute("arr", ArrayOf(Int, func() {
					Meta("struct:field:type", "int32")
				}), func() {
					Meta("openapi:style", "pipeDelimited")
					Meta("openapi:explode", "false")
				})
				Attribute("tags", ArrayOf(Int, func() {
					Meta("struct:field:type", "uint64")
				}))
				Attribute("uq", Int, func() {
					Meta("struct:field:type", "uint64")
				})
				Attribute("small", Int, func() {
					Meta("struct:field:type", "int8")
				})
				Attribute("hs", ArrayOf(Int, func() {
					Meta("struct:field:type", "int16")
				}))
				Attribute("opt", Int64, func() {
					Meta("struct:field:type", "int32")
				})
				Required("hdr", "dur", "ck", "d", "lbl", "num", "bp", "nums", "arr", "tags", "uq", "small", "hs")
			})
			Result(out)
			HTTP(func() {
				GET("/{d}/{lbl}/{num}/{bp}/{nums}")
				Header("hdr:X-Hdr")
				Header("dur:X-Dur")
				Header("hs:X-Hs")
				Header("opt:X-Opt")
				Cookie("ck")
				Param("arr")
				Param("tags")
				Param("uq")
				Param("small")
				Response(StatusOK, func() {
					Header("c64:X-C64")
					Header("big:X-Big")
					Header("neg:X-Neg")
					Cookie("rc")
					Body("message")
				})
			})
		})
	})
}

func TestIntegerTextBoundaryGenerated(t *testing.T) {
	root := RunHTTPDSL(t, integerTextBoundaryDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/integers", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "integers_test.go"), []byte(integerTextBoundaryHarness), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "custom"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "custom", "number.go"), []byte(integerTextBoundaryNumber), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

// integerTextBoundaryNumber is opaque integer storage whose fmt.Formatter and
// fmt.Stringer text must never reach the wire.
const integerTextBoundaryNumber = `package custom

import "fmt"

type Number int64

func (n Number) Format(state fmt.State, _ rune) {
	fmt.Fprint(state, "formatted")
}

func (n Number) String() string {
	return "stringer"
}
`

const integerTextBoundaryHarness = `package integers_test

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"example.com/integers/custom"
	conv "example.com/integers/gen/conv"
	client "example.com/integers/gen/http/conv/client"
	server "example.com/integers/gen/http/conv/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	got *conv.EncodePayload
	out *conv.Out
}

func (s *service) Encode(_ context.Context, p *conv.EncodePayload) (*conv.Out, error) {
	s.got = p
	return s.out, nil
}

type countingDoer struct{ calls int }

func (d *countingDoer) Do(r *http.Request) (*http.Response, error) {
	d.calls++
	return http.DefaultClient.Do(r)
}

func start(t *testing.T, svc *service) *httptest.Server {
	mux := loomhttp.NewMuxer()
	server.New(conv.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil).Mount(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestIntegerTextBoundary(t *testing.T) {
	svc := &service{out: &conv.Out{C64: math.MaxInt64, Big: 42, Neg: 127, Rc: -7, Message: "m"}}
	srv := start(t, svc)
	host := strings.TrimPrefix(srv.URL, "http://")
	doer := &countingDoer{}
	encode := client.NewClient("http", host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Encode()
	opt := int32(-9)
	payload := &conv.EncodePayload{Hdr: math.MinInt64, Dur: 5 * time.Second, Ck: math.MaxInt32, D: -5, Lbl: -3,
		Num: math.MinInt64, Bp: 42, Nums: []custom.Number{math.MinInt64, math.MaxInt64},
		Arr: []int32{-1, 2}, Tags: []uint64{0, math.MaxInt64}, Uq: 7, Small: 127, Hs: []int16{math.MinInt16}, Opt: &opt}
	res, err := encode(context.Background(), payload)
	if err != nil {
		t.Fatalf("accepted round trip: %v", err)
	}
	if out := res.(*conv.Out); *out != *svc.out {
		t.Errorf("result = %+v, want %+v", out, svc.out)
	}
	got := svc.got
	if got.Num != payload.Num || got.Bp != payload.Bp || len(got.Nums) != 2 || got.Nums[0] != math.MinInt64 || got.Nums[1] != math.MaxInt64 {
		t.Errorf("decoded path storage = %v %v %v, want %v %v %v", got.Num, got.Bp, got.Nums, payload.Num, payload.Bp, payload.Nums)
	}
	if got.Hdr != payload.Hdr || got.Dur != payload.Dur || got.Ck != payload.Ck || got.D != payload.D || got.Lbl != payload.Lbl ||
		got.Uq != payload.Uq || got.Small != payload.Small || got.Opt == nil || *got.Opt != opt ||
		len(got.Arr) != 2 || got.Arr[0] != -1 || len(got.Tags) != 2 || got.Tags[1] != math.MaxInt64 || len(got.Hs) != 1 || got.Hs[0] != math.MinInt16 {
		t.Errorf("decoded payload = %+v, want %+v", got, payload)
	}
	text, err := client.EncodeConvPath(5*time.Second, -3, math.MinInt64, math.MaxUint64, []custom.Number{math.MinInt64, -1, 0, math.MaxInt64})
	if want := "/5000000000/.-3/-9223372036854775808/18446744073709551615/-9223372036854775808,-1,0,9223372036854775807"; err != nil || text != want {
		t.Errorf("path builder = %q, %v; want faithful decimal %q", text, err, want)
	}

	calls := doer.calls
	for name, bad := range map[string]func(*conv.EncodePayload){
		"query uint64 max":      func(p *conv.EncodePayload) {
			p.Uq = math.MaxUint64
		},
		"named path uint64 max": func(p *conv.EncodePayload) {
			p.Bp = math.MaxUint64
		},
		"slice element":         func(p *conv.EncodePayload) {
			p.Tags = []uint64{math.MaxUint64}
		},
	} {
		p := *payload
		bad(&p)
		if _, err := encode(context.Background(), &p); err == nil || !strings.Contains(err.Error(), "cannot be represented as int") {
			t.Errorf("%s: err = %v, want representability error", name, err)
		}
	}
	if doer.calls != calls {
		t.Errorf("rejected requests reached Do: %d calls", doer.calls-calls)
	}

	valid := url.Values{"arr": {"1"}, "tags": {"1"}, "uq": {"1"}, "small": {"1"}}
	for name, mutate := range map[string]func(url.Values, http.Header){
		"narrow query 200":    func(q url.Values, _ http.Header) {
			q.Set("small", "200")
		},
		"unsigned query -1":   func(q url.Values, _ http.Header) {
			q.Set("uq", "-1")
		},
		"slice query -1":      func(q url.Values, _ http.Header) {
			q.Set("tags", "-1")
		},
		"header list 40000":   func(_ url.Values, h http.Header) {
			h.Set("X-Hs", "40000")
		},
		"optional 2147483648": func(_ url.Values, h http.Header) {
			h.Set("X-Opt", "2147483648")
		},
	} {
		q := url.Values{}
		for k, v := range valid {
			q[k] = append([]string(nil), v...)
		}
		h := http.Header{"X-Hdr": {"1"}, "X-Dur": {"1"}, "X-Hs": {"1"}, "Cookie": {"ck=1"}}
		mutate(q, h)
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/1/.1/1/1/1?"+q.Encode(), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header = h
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, resp.StatusCode)
		}
	}

	svc.out = &conv.Out{C64: 5, Big: math.MaxUint64, Neg: 1, Rc: 1, Message: "m"}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/1/.1/1/1/1?"+valid.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header = http.Header{"X-Hdr": {"1"}, "X-Dur": {"1"}, "X-Hs": {"1"}, "Cookie": {"ck=1"}}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError || resp.Header.Get("X-C64") != "" || resp.Header.Get("X-Big") != "" {
		t.Errorf("unrepresentable result published: status %d X-C64=%q X-Big=%q", resp.StatusCode, resp.Header.Get("X-C64"), resp.Header.Get("X-Big"))
	}
}
`
