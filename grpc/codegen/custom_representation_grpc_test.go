package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestCustomStringGRPCGenerated(t *testing.T) {
	root := RunGRPCDSL(t, customStringGRPCDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/customgrpc", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "text.go"), []byte(customGRPCStringSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "contract_test.go"), []byte(customGRPCStringHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-race", ".", "./gen/...", "./scalar")
}

func customStringGRPCDSL() {
	state := dsl.Type("State", func() {
		dsl.Field(1, "native", dsl.String)
		dsl.Field(2, "text", dsl.String, func() {
			dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
			dsl.Default("fallback")
		})
		dsl.Field(3, "child", "State")
		dsl.Field(4, "children", dsl.ArrayOf("State"))
		dsl.Field(5, "by_name", dsl.MapOf(dsl.String, "State"))
		dsl.Field(6, "texts", dsl.ArrayOf(dsl.String, func() {
			dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
		}))
		dsl.Field(7, "labels", dsl.MapOf(dsl.String, dsl.String, func() {
			dsl.Elem(func() {
				dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
			})
		}))
	})
	bad := dsl.Type("BadState", func() {
		dsl.Field(1, "text", dsl.String, func() {
			dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
			dsl.Default("invalid")
		})
	})
	dsl.Service("values", func() {
		dsl.Method("echo", func() {
			dsl.Payload(state)
			dsl.Result(state)
			dsl.GRPC(func() {})
		})
		dsl.Method("bad", func() {
			dsl.Payload(bad)
			dsl.Result(bad)
			dsl.GRPC(func() {})
		})
	})
}

const customGRPCStringSource = `package scalar

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

const customGRPCStringHarness = `package customgrpc_test

import (
 "errors"
 "testing"

 "google.golang.org/protobuf/proto"
 "github.com/stretchr/testify/require"

 "example.com/customgrpc/scalar"
 "example.com/customgrpc/gen/values"
 "example.com/customgrpc/gen/grpc/values/client"
 "example.com/customgrpc/gen/grpc/values/server"
 pb "example.com/customgrpc/gen/grpc/values/pb"
)

func TestCustomStringRoundTrip(t *testing.T) {
 for _, value := range []string{"", "ordinary", "é"} {
  t.Run(value, func(t *testing.T) {
   input := &values.State{Text: scalar.Text{Value: value}, Texts: []scalar.Text{{Value: value}}, Labels: map[string]scalar.Text{"key": {Value: value}}}
   request, err := client.NewProtoEchoRequest(input)
   require.NoError(t, err)
   require.Equal(t, value, *request.Text)
   wire, err := proto.Marshal(request)
   require.NoError(t, err)
   decoded := new(pb.EchoRequest)
   require.NoError(t, proto.Unmarshal(wire, decoded))
   payload, err := server.NewEchoPayload(decoded)
   require.NoError(t, err)
   require.Equal(t, value, payload.Text.Value)
   require.Equal(t, value, payload.Texts[0].Value)
   require.Equal(t, value, payload.Labels["key"].Value)
   response, err := server.NewProtoEchoResponse(input)
   require.NoError(t, err)
   result, err := client.NewEchoResult(response)
   require.NoError(t, err)
   require.Equal(t, value, result.Text.Value)
   require.Equal(t, value, result.Texts[0].Value)
   require.Equal(t, value, result.Labels["key"].Value)
  })
 }
 payload, err := server.NewEchoPayload(new(pb.EchoRequest))
 require.NoError(t, err)
 require.Equal(t, "fallback", payload.Text.Value)
 result, err := client.NewEchoResult(new(pb.EchoResponse))
 require.NoError(t, err)
 require.Equal(t, "fallback", result.Text.Value)
}

func TestCustomStringFailureDoesNotPublish(t *testing.T) {
 invalid := "invalid"
 for _, input := range []*pb.EchoRequest{
  {Text: &invalid},
  {Child: &pb.State{Text: &invalid}},
  {Children: []*pb.State{{Text: &invalid}}},
  {ByName: map[string]*pb.State{"bad": {Text: &invalid}}},
  {Texts: []string{invalid}},
  {Labels: map[string]string{"bad": invalid}},
 } {
  payload, err := server.NewEchoPayload(input)
  require.True(t, errors.Is(err, scalar.ErrInvalid), "%v", err)
  require.Nil(t, payload)
 }
 payload, err := server.NewBadPayload(new(pb.BadRequest))
 require.ErrorIs(t, err, scalar.ErrInvalid)
 require.Nil(t, payload)
 result, err := client.NewBadResult(new(pb.BadResponse))
 require.ErrorIs(t, err, scalar.ErrInvalid)
 require.Nil(t, result)
 empty := ""
 payload, err = server.NewBadPayload(&pb.BadRequest{Text: &empty})
 require.NoError(t, err)
 require.Empty(t, payload.Text.Value)
}
`
