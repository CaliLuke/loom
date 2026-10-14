package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestUnionPhysicalGRPCGenerated covers metadata reachable only through a
// union branch and the selected custom-to-native protobuf conversion boundary.
func TestUnionPhysicalGRPCGenerated(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := map[bool]string{false: "native", true: "custom"}[custom]
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				node := dsl.Type("Choice", func() {
					dsl.OneOf("value", func() {
						dsl.Field(1, "text", dsl.String, func() {
							if custom {
								dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
							}
						})
						dsl.Field(2, "number", dsl.Int)
					})
				})
				dsl.Service("choices", func() {
					dsl.Method("echo", func() {
						dsl.Payload(node)
						dsl.Result(node)
						dsl.GRPC(func() {})
					})
				})
			})
			dir := t.TempDir()
			renderGRPCResponseContractModule(t, dir, "example.com/customgrpc", root)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "text.go"), []byte(unionPhysicalGRPCText), 0o600))
			if custom {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(unionPhysicalGRPCHarness), 0o600))
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "test", "-race", "./gen/...", "./scalar")
			if custom {
				runGRPCGoCommand(t, dir, "test", "-race", ".")
			}
		})
	}
}

const unionPhysicalGRPCHarness = `package customgrpc_test

import (
	"errors"
	pb "example.com/customgrpc/gen/grpc/choices/pb"
	server "example.com/customgrpc/gen/grpc/choices/server"
	"example.com/customgrpc/scalar"
	"testing"
)

func TestCustomUnionText(t *testing.T) {
	for _, text := range []string{"valid", ""} {
		result, err := server.NewEchoPayload(&pb.EchoRequest{Value: &pb.EchoRequest_Text{Text: text}})
		if err != nil || result == nil {
			t.Fatalf("custom union text: %#v %v", result, err)
		}
		response, err := server.NewProtoEchoResponse(result)
		if err != nil || response.GetText() != text {
			t.Fatalf("publication: %#v %v", response, err)
		}
	}
	result, err := server.NewEchoPayload(&pb.EchoRequest{Value: &pb.EchoRequest_Text{Text: "invalid"}})
	if result != nil || !errors.Is(err, scalar.ErrInvalid) {
		t.Fatalf("authored parse failure: %#v %v", result, err)
	}
}
`

const unionPhysicalGRPCText = `package scalar

import "errors"

var ErrInvalid = errors.New("custom parse failed")

type Text struct {
	Value string
}

func (v Text) String() string {
	return v.Value
}
func (v *Text) UnmarshalText(value []byte) error {
	if string(value) == "invalid" {
		return ErrInvalid
	}
	v.Value = string(value)
	return nil
}
`
