package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestGRPCMapKeyTypes(t *testing.T) {
	for _, tc := range []struct {
		key   expr.DataType
		valid bool
	}{
		{Boolean, true}, {String, true}, {Int, true}, {Int32, true}, {Int64, true},
		{UInt, true}, {UInt32, true}, {UInt64, true},
		{Any, false}, {Bytes, false}, {Float32, false}, {Float64, false},
	} {
		for _, alias := range []bool{false, true} {
			name := tc.key.Name()
			if alias {
				name += " alias"
			}
			t.Run(name, func(t *testing.T) {
				design := func() {
					key := tc.key
					if alias {
						key = Type("Key", Type("BaseKey", key))
					}
					Service("maps", func() {
						Method("echo", func() {
							Payload(MapOf(key, Any))
							GRPC(func() {})
						})
					})
				}
				if tc.valid {
					expr.RunDSL(t, design)
				} else {
					require.ErrorContains(t, expr.RunInvalidDSL(t, design), "gRPC map keys must be Boolean, String, or integer types")
				}
			})
		}
	}
}

func TestGRPCMapKeyShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  func() expr.DataType
	}{
		{"object", func() expr.DataType {
			return Type("Key", func() {
				Field(1, "value", String)
			})
		}},
		{"array", func() expr.DataType {
			return ArrayOf(String)
		}},
		{"union", func() expr.DataType {
			return OneOf(String, Int)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				key := tc.key()
				Service("maps", func() {
					Method("echo", func() {
						Payload(MapOf(key, Any))
						GRPC(func() {})
					})
				})
			})
			require.ErrorContains(t, err, "gRPC map keys must be Boolean, String, or integer types")
			require.NotContains(t, err.Error(), "use that type as the key")
		})
	}
}

func TestGRPCMapKeyPositions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method func(expr.DataType)
	}{
		{"payload", func(dt expr.DataType) {
			Payload(dt)
		}},
		{"result", func(dt expr.DataType) {
			Result(dt)
		}},
		{"streaming payload", func(dt expr.DataType) {
			StreamingPayload(dt)
		}},
		{"streaming result", func(dt expr.DataType) {
			StreamingResult(dt)
		}},
		{"error", func(dt expr.DataType) {
			Error("bad", dt)
			GRPC(func() {
				Response("bad", CodeInvalidArgument)
			})
		}},
		{"union branch", func(dt expr.DataType) {
			Payload(OneOf(dt, String))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				node := Type("Node", func() {
					Field(1, "next", "Node")
					Field(2, "values", ArrayOf(MapOf(String, MapOf(Any, String))))
				})
				Service("maps", func() {
					Method("echo", func() {
						tc.method(node)
						GRPC(func() {})
					})
				})
			})
			require.ErrorContains(t, err, "gRPC map keys must be Boolean, String, or integer types")
		})
	}
}

func TestGRPCMapKeysInInheritedErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scope    string
		override bool
		mapped   bool
		valid    bool
	}{
		{"unmapped method error", "method", false, false, true},
		{"unmapped service error", "service", false, false, true},
		{"service mapping", "service", false, true, false},
		{"service mapping with local override", "service", true, true, false},
		{"API mapping with local override", "API", true, true, false},
		{"unused service error overridden locally", "service", true, false, true},
		{"unused API error", "API", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				bad := Type("Bad", func() {
					Field(1, "values", MapOf(Any, String))
				})
				declareError := func() {
					Error("bad", bad)
					if tc.mapped {
						GRPC(func() {
							Response("bad", CodeInvalidArgument)
						})
					}
				}
				API("maps", func() {
					if tc.scope == "API" {
						declareError()
					}
				})
				Service("maps", func() {
					if tc.scope == "service" {
						declareError()
					}
					Method("echo", func() {
						if tc.scope == "method" {
							declareError()
						}
						if tc.override {
							Error("bad")
						}
						GRPC(func() {})
					})
				})
			}
			if tc.valid {
				expr.RunDSL(t, design)
			} else {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), "gRPC map keys must be Boolean, String, or integer types")
			}
		})
	}
}
