package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestBinaryMetadataAggregate(t *testing.T) {
	root := RunGRPCDSL(t, binaryMetadataAggregateDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_BINARY_METADATA_OUTPUT"); output != "" {
		dir = filepath.Join(output, "aggregate")
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderGRPCResponseContractModule(t, dir, "example.com/binaryaggregate", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary_test.go"), []byte(binaryMetadataAggregateHarness), 0o600))
	if os.Getenv("LOOM_BINARY_METADATA_GENERATE_ONLY") == "1" {
		return
	}
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-count=1", "-v", ".")
}

func TestBinaryMetadataConsumerAgreement(t *testing.T) {
	root := RunGRPCDSL(t, binaryMetadataAggregateDSL)
	service := CreateGRPCServices(root).Get("binary")
	for _, placement := range []string{"request", "header", "trailer"} {
		t.Run(placement, func(t *testing.T) {
			e := service.Endpoint(placement)
			fields := e.Request.Metadata
			if placement == "header" {
				fields = e.Response.Headers
			} else if placement == "trailer" {
				fields = e.Response.Trailers
			}
			require.Len(t, fields, 4)
			for _, field := range fields {
				want := "x-" + field.AttributeName + "-bin"
				if field.AttributeName == "text" {
					want = "x-text"
				}
				require.Equal(t, want, field.Name)
				require.Contains(t, renderGRPCMetadataDecode(field, "md", false), `Get("`+want+`")`)
				require.Contains(t, renderGRPCMetadataAppend(field, "payload"), `Append("`+want+`"`)
				require.Contains(t, renderGRPCMetadataEncode(field, "md"), `Append("`+want+`"`)
			}
			require.Len(t, e.ResponseContractCases, 1)
			if placement == "header" {
				require.Equal(t, []string{"x-raw-bin"}, e.ResponseContractCases[0].RequiredHeaders)
			} else if placement == "trailer" {
				require.Equal(t, []string{"x-raw-bin"}, e.ResponseContractCases[0].RequiredTrailers)
			}
		})
	}
}

func TestBinaryMetadataErrorContractAgreement(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		dsl.Service("binary", func() {
			dsl.Method("call", func() {
				dsl.Result(dsl.String)
				dsl.Error("broken", func() {
					dsl.Field(1, "head", dsl.Bytes)
					dsl.Field(2, "tail", dsl.Bytes)
					dsl.Required("head", "tail")
				})
				dsl.GRPC(func() {
					dsl.Response("broken", dsl.CodeInternal, func() {
						dsl.Headers(func() {
							dsl.Attribute("head:X-HEAD")
						})
						dsl.Trailers(func() {
							dsl.Attribute("tail:X-TAIL")
						})
					})
				})
			})
		})
	})
	endpoint := CreateGRPCServices(root).Get("binary").Endpoint("call")
	require.Len(t, endpoint.Errors, 1)
	mapping := root.API.GRPC.Service("binary").GRPCEndpoints[0].GRPCErrors[0].Response
	require.Equal(t, "x-head-bin", mapping.Headers.ElemName("head"))
	require.Equal(t, "x-tail-bin", mapping.Trailers.ElemName("tail"))
	require.Len(t, endpoint.ResponseContractCases, 2)
	contract := endpoint.ResponseContractCases[1]
	require.Equal(t, "broken", contract.ErrorName)
	require.Equal(t, []string{"x-head-bin"}, contract.RequiredHeaders)
	require.Equal(t, []string{"x-tail-bin"}, contract.RequiredTrailers)
}

func binaryMetadataAggregateDSL() {
	raw := dsl.Type("Raw", dsl.Bytes)
	list := dsl.Type("RawList", dsl.ArrayOf(raw))
	value := dsl.Type("Value", func() {
		dsl.Field(1, "body", dsl.String)
		dsl.Field(2, "raw:data", raw)
		dsl.Field(3, "optional", raw, func() {
			dsl.Default([]byte("default"))
		})
		dsl.Field(4, "items", list)
		dsl.Field(5, "text", dsl.String)
		dsl.Required("body", "raw")
	})
	mapping := func() {
		for _, name := range []string{"raw", "optional", "items", "text"} {
			dsl.Attribute(name + ":x-" + name)
		}
	}
	dsl.Service("binary", func() {
		for _, placement := range []string{"request", "header", "trailer"} {
			dsl.Method(placement, func() {
				dsl.Payload(value)
				dsl.Result(value)
				dsl.GRPC(func() {
					if placement == "request" {
						dsl.Metadata(mapping)
					}
					dsl.Response(dsl.CodeOK, func() {
						if placement == "header" {
							dsl.Headers(mapping)
						} else if placement == "trailer" {
							dsl.Trailers(mapping)
						}
					})
				})
			})
		}
	})
}
