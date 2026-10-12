package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestObjectDefaultsTransport(t *testing.T) {
	root := RunGRPCDSL(t, httptestdata.ObjectDefaultsDSL("grpc"))
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/objectdefaults", root, resolveGRPCLoomSource(t))
	runGRPCGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(objectDefaultsGRPCHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", ".")
	runGRPCGoCommand(t, dir, "vet", "./...")
}

const objectDefaultsGRPCHarness = `package defaults_test

import (
 "context"
 "testing"

 svc "example.com/objectdefaults/gen/defaults"
 leaf "example.com/objectdefaults/gen/types/leaf"
 client "example.com/objectdefaults/gen/grpc/defaults/client"
 server "example.com/objectdefaults/gen/grpc/defaults/server"
 "github.com/stretchr/testify/require"
)

func TestObjectDefaults(t *testing.T) {
 for _, explicit:=range []bool{false,true} {
  source:=&svc.SendPayload{}
  if explicit {
   source.Value=&leaf.Leaf{Count:0,Label:""}
  }
  wire:=client.NewProtoSendRequest(source)
  decoded,err:=server.DecodeSendRequest(context.Background(),wire,nil)
  require.NoError(t,err)
  value:=decoded.(*svc.SendPayload)
  require.NotNil(t,value.Nested)
  require.Equal(t,0,value.Nested.Child.Count)
  require.Equal(t,"nested",value.Nested.Child.Label)
  require.Equal(t,5,value.Nested.Items[0].Count)
  require.Equal(t,2,value.Items[0].Count)
  require.Equal(t,3,value.Indexed["key"].Count)
  if explicit {
   require.Equal(t,0,value.Value.Count)
   require.Equal(t,"",value.Value.Label)
  } else {
   require.Equal(t,1,value.Value.Count)
   require.Equal(t,"nested",value.Value.Label)
  }
 }
 invalid:=client.NewProtoSendRequest(&svc.SendPayload{Value:&leaf.Leaf{Count:-1}})
 _,err:=server.DecodeSendRequest(context.Background(),invalid,nil)
 require.Error(t,err)
}
`
