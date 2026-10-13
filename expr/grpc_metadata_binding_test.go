package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestGRPCMetadataEvaluatedKeys(t *testing.T) {
	cases := []struct {
		name  string
		shape string
		key   string
		want  string
	}{
		{"native", "bytes", "x-raw", "x-raw-bin"},
		{"alias", "alias", "x-raw", "x-raw-bin"},
		{"chain", "chain", "x-raw", "x-raw-bin"},
		{"array", "array", "x-raw", "x-raw-bin"},
		{"named-array", "named-array", "x-raw", "x-raw-bin"},
		{"named-element", "named-element", "x-raw", "x-raw-bin"},
		{"explicit", "bytes", "x-raw-bin", "x-raw-bin"},
		{"case", "bytes", "X-RAW-BIN", "x-raw-bin"},
		{"text", "text", "x-raw", "x-raw"},
		{"explicit-text", "text", "x-raw-bin", "x-raw-bin"},
	}
	for _, placement := range []string{"request", "header", "trailer"} {
		for _, tc := range cases {
			t.Run(placement+"/"+tc.name, func(t *testing.T) {
				root := expr.RunDSL(t, func() {
					metadataBindingDSL(placement, tc.shape, tc.key, "")
				})
				e := root.API.GRPC.Service("binding").GRPCEndpoints[0]
				mapped := metadataBindingPlacement(e, placement)
				require.Equal(t, tc.want, mapped.ElemName("raw"))
				require.Equal(t, "raw", mapped.KeyName(tc.want))
				if tc.key != tc.want {
					require.Panics(t, func() {
						mapped.KeyName(tc.key)
					}, "obsolete wire aliases must not survive")
				}
				require.True(t, mapped.IsRequired("raw"))
				require.NotNil(t, expr.AsObject(e.MethodExpr.Payload.Type).Attribute("raw"))
				require.NotNil(t, expr.AsObject(e.MethodExpr.Result.Type).Attribute("raw"))
			})
		}
	}
}

func TestGRPCMetadataRejectsCanonicalCollisions(t *testing.T) {
	for _, placement := range []string{"request", "header", "trailer"} {
		for _, keys := range [][2]string{{"x-raw", "x-raw-bin"}, {"X-RAW-BIN", "x-raw-bin"}} {
			t.Run(placement+"/"+keys[0], func(t *testing.T) {
				expr.SetupTestDSL(t)
				require.True(t, eval.Execute(func() {
					metadataBindingDSL(placement, "bytes", keys[0], keys[1])
				}, nil))
				err := eval.RunDSL()
				require.Error(t, err, "distinct logical fields must not share a canonical metadata key")
				require.Contains(t, err.Error(), "x-raw-bin")
				require.Contains(t, err.Error(), "raw")
				require.Contains(t, err.Error(), "other")
			})
		}
	}
}

func metadataBindingDSL(placement, shape, key, collision string) {
	var raw any = Bytes
	switch shape {
	case "alias":
		raw = Type("Raw", Bytes)
	case "chain":
		raw = Type("Outer", Type("Inner", Bytes))
	case "array":
		raw = ArrayOf(Bytes)
	case "named-array":
		raw = Type("RawList", ArrayOf(Bytes))
	case "named-element":
		raw = ArrayOf(Type("Raw", Bytes))
	case "text":
		raw = String
	}
	value := Type("Value", func() {
		Field(1, "raw", raw)
		Field(2, "other", String)
		Required("raw")
	})
	mapping := func() {
		Attribute("raw:"+key, String)
		if collision != "" {
			Attribute("other:" + collision)
		}
	}
	Service("binding", func() {
		Method("echo", func() {
			Payload(value)
			Result(value)
			GRPC(func() {
				if placement == "request" {
					Metadata(mapping)
				}
				Response(CodeOK, func() {
					if placement == "header" {
						Headers(mapping)
					} else if placement == "trailer" {
						Trailers(mapping)
					}
				})
			})
		})
	})
}

func metadataBindingPlacement(e *expr.GRPCEndpointExpr, placement string) *expr.MappedAttributeExpr {
	switch placement {
	case "request":
		return e.Metadata
	case "header":
		return e.Response.Headers
	default:
		return e.Response.Trailers
	}
}
