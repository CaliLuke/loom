package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// primitiveUnionRoundTripTest is the test file written into the generated
// service package of primitiveUnionServiceDSL. It exercises the generated
// branch constructors, accessors, Validate, MarshalJSON and UnmarshalJSON
// methods of every constructor OneOf with bare primitive branches.
const primitiveUnionRoundTripTest = `package primunion

import (
	"testing"
)

type unionCodec interface {
	MarshalJSON() ([]byte, error)
	Validate() error
}

func TestPrimitiveUnionRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		value   unionCodec
		want    string
		decode  func([]byte) (unionCodec, error)
		matches func(unionCodec) bool
	}{
		{
			name:  "string",
			value: NewIntOrStringString("hi"),
			want:  ` + "`" + `{"type":"String","value":"hi"}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u IntOrString
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(IntOrString).AsString()
				_, other := u.(IntOrString).AsInt()
				return ok && !other && v == "hi" && u.(IntOrString).Kind() == IntOrStringKindString
			},
		},
		{
			name:  "int",
			value: NewIntOrStringInt(42),
			want:  ` + "`" + `{"type":"Int","value":42}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u IntOrString
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(IntOrString).AsInt()
				return ok && v == 42 && u.(IntOrString).Kind() == IntOrStringKindInt
			},
		},
		{
			name:  "int64",
			value: NewFloat64OrInt64Int64(9007199254740993),
			want:  ` + "`" + `{"type":"Int64","value":9007199254740993}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u Float64OrInt64
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(Float64OrInt64).AsInt64()
				return ok && v == 9007199254740993
			},
		},
		{
			name:  "float64",
			value: NewFloat64OrInt64Float64(1.5),
			want:  ` + "`" + `{"type":"Float64","value":1.5}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u Float64OrInt64
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(Float64OrInt64).AsFloat64()
				return ok && v == 1.5
			},
		},
		{
			name:  "boolean",
			value: NewBooleanOrBytesBoolean(true),
			want:  ` + "`" + `{"type":"Boolean","value":true}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u BooleanOrBytes
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(BooleanOrBytes).AsBoolean()
				return ok && v
			},
		},
		{
			name:  "bytes",
			value: NewBooleanOrBytesBytes([]byte("hi")),
			want:  ` + "`" + `{"type":"Bytes","value":"aGk="}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u BooleanOrBytes
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(BooleanOrBytes).AsBytes()
				return ok && string(v) == "hi"
			},
		},
		{
			name:  "any",
			value: NewAnyOrStringAny(AnyOrStringAny(` + "`" + `{"n":1.50}` + "`" + `)),
			want:  ` + "`" + `{"type":"Any","value":{"n":1.50}}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u AnyOrString
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(AnyOrString).AsAny()
				return ok && string(v) == ` + "`" + `{"n":1.50}` + "`" + `
			},
		},
		{
			name:  "mixed primitive",
			value: NewLeafOrStringString("x"),
			want:  ` + "`" + `{"type":"String","value":"x"}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u LeafOrString
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(LeafOrString).AsString()
				return ok && v == "x"
			},
		},
		{
			name:  "mixed user type",
			value: NewIntOrLeafLeaf(&Leaf{Name: "ab"}),
			want:  ` + "`" + `{"type":"Leaf","value":{"name":"ab"}}` + "`" + `,
			decode: func(data []byte) (unionCodec, error) {
				var u IntOrLeaf
				err := u.UnmarshalJSON(data)
				return u, err
			},
			matches: func(u unionCodec) bool {
				v, ok := u.(IntOrLeaf).AsLeaf()
				_, other := u.(IntOrLeaf).AsInt()
				return ok && !other && v != nil && v.Name == "ab"
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.value.Validate(); err != nil {
				t.Errorf("Validate: %v", err)
			}
			got, err := c.value.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("MarshalJSON = %s, want %s", got, c.want)
			}
			decoded, err := c.decode(got)
			if err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if !c.matches(decoded) {
				t.Errorf("UnmarshalJSON(%s) = %#v, want %#v", got, decoded, c.value)
			}
		})
	}
}

func TestPrimitiveUnionRejectsInvalidJSON(t *testing.T) {
	cases := map[string]string{
		"wrong value type": ` + "`" + `{"type":"Int","value":"x"}` + "`" + `,
		"unknown type":     ` + "`" + `{"type":"Float64","value":1}` + "`" + `,
		"missing value":    ` + "`" + `{"type":"String"}` + "`" + `,
		"null value":       ` + "`" + `{"type":"String","value":null}` + "`" + `,
		"malformed":        ` + "`" + `{"type":` + "`" + `,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			var u IntOrString
			if err := u.UnmarshalJSON([]byte(data)); err == nil {
				t.Errorf("UnmarshalJSON(%s) succeeded with kind %q", data, u.Kind())
			}
		})
	}
	var zero IntOrString
	if err := zero.Validate(); err == nil {
		t.Error("Validate of the zero union succeeded")
	}
	if _, err := zero.MarshalJSON(); err == nil {
		t.Error("MarshalJSON of the zero union succeeded")
	}
}

func TestPrimitiveUnionFieldRoundTrip(t *testing.T) {
	in := Envelope{Pick: NewLeafOrStringString("x")}
	opt := NewIntOrLeafInt(7)
	in.Opt = &opt
	pick, err := in.Pick.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var out Envelope
	if err := out.Pick.UnmarshalJSON(pick); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if v, ok := out.Pick.AsString(); !ok || v != "x" {
		t.Errorf("Pick = %#v, want String branch x", out.Pick)
	}
	optData, err := in.Opt.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON Opt: %v", err)
	}
	out.Opt = new(IntOrLeaf)
	if err := out.Opt.UnmarshalJSON(optData); err != nil {
		t.Fatalf("UnmarshalJSON Opt: %v", err)
	}
	if v, ok := out.Opt.AsInt(); !ok || v != 7 {
		t.Errorf("Opt = %#v, want Int branch 7", out.Opt)
	}
}
`

// TestPrimitiveUnionServiceTypesRoundTrip builds, vets and tests the service
// package generated for constructor OneOf unions whose branches are bare
// primitives (String, Int, Int64, Float64, Boolean, Bytes and Any) or mix a
// primitive with a user type, used as payloads, results, fields and fields of
// a result type with views.
func TestPrimitiveUnionServiceTypesRoundTrip(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/primunion", primitiveUnionServiceDSL)
	output, err := testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
	testFile := filepath.Join(dir, "gen", "primunion", "primitive_union_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte(primitiveUnionRoundTripTest), 0o600))
	output, err = testingx.RunCmd(dir, "go", "test", "./gen/primunion/")
	require.NoError(t, err, output)
}

// TestPrimitiveUnionTransportsGeneratedCodeCompiles builds and vets the
// service and transport packages generated for constructor OneOf fields with
// bare primitive branches in a design exposed on HTTP, gRPC and JSON-RPC.
func TestPrimitiveUnionTransportsGeneratedCodeCompiles(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/primuniontransport", primitiveUnionTransportsDSL)
	output, err := testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

func primitiveUnionServiceDSL() {
	dsl.API("primunion", func() {})
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	var Envelope = dsl.Type("Envelope", func() {
		dsl.Attribute("pick", dsl.OneOf(dsl.String, Leaf))
		dsl.Attribute("opt", dsl.OneOf(dsl.Int, Leaf))
		dsl.Required("pick")
	})
	var Pick = dsl.ResultType("application/vnd.pick", func() {
		dsl.Attributes(func() {
			dsl.Attribute("id", dsl.String)
			dsl.Attribute("pick", dsl.OneOf(dsl.Boolean, dsl.Int64))
		})
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("pick")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("primunion", func() {
		dsl.Method("viewed", func() {
			dsl.Result(Pick)
		})
		dsl.Method("text", func() {
			dsl.Payload(dsl.OneOf(dsl.String, dsl.Int))
			dsl.Result(dsl.OneOf(dsl.String, dsl.Int))
		})
		dsl.Method("wide", func() {
			dsl.Payload(dsl.OneOf(dsl.Int64, dsl.Float64))
		})
		dsl.Method("flag", func() {
			dsl.Payload(dsl.OneOf(dsl.Boolean, dsl.Bytes))
		})
		dsl.Method("raw", func() {
			dsl.Payload(dsl.OneOf(dsl.Any, dsl.String))
		})
		dsl.Method("mixed", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
		})
	})
}

func primitiveUnionTransportsDSL() {
	dsl.API("primuniontransport", func() {
		dsl.JSONRPC(func() {})
	})
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Field(1, "name", dsl.String)
	})
	var Envelope = dsl.Type("Envelope", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "pick", dsl.OneOf(dsl.String, dsl.Int))
		dsl.Field(4, "wide", dsl.OneOf(dsl.Int64, dsl.Float64))
		dsl.Field(6, "flag", dsl.OneOf(dsl.Boolean, dsl.Bytes))
		dsl.Field(8, "mixed", dsl.OneOf(dsl.Int32, Leaf))
		dsl.Required("pick")
	})
	dsl.Service("primunion", func() {
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
			dsl.HTTP(func() {
				dsl.POST("/echo")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("primunionrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
			dsl.JSONRPC(func() {})
		})
	})
}
