package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestJSONFieldNumericAlias(t *testing.T) {
	repo, err := filepath.Abs("../..")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repo)
	for _, test := range []struct {
		name      string
		dsl       func()
		validated bool
	}{
		{"service_query_map_alias", testdata.QueryMapAliasDSL, false},
		{"service_query_map_alias_validate", testdata.QueryMapAliasValidateDSL, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, test.dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/alias", root)
			runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012215933-22e452f21fb3")
			harness := fmt.Sprintf(jsonFieldAliasHarness, test.name, test.name, test.validated)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

const jsonFieldAliasHarness = `package alias_test
import (
 "net/http/httptest"
 "net/url"
 "testing"
 "github.com/stretchr/testify/require"
 client "example.com/alias/gen/http/%s/client"
 server "example.com/alias/gen/http/%s/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
func TestAliasWire(t *testing.T) {
 const valid="{\"1\":true,\"2\":false,\"3\":true,\"4\":false,\"5\":true}"
 values:=url.Values{"map":{valid}}
 request:=httptest.NewRequest("POST","/?"+values.Encode(),nil)
 payload,err:=server.DecodeMethodARequest(loomhttp.NewMuxer(),nil)(request)
 require.NoError(t,err)
 require.Len(t,payload.Map,5)
 encoded:=httptest.NewRequest("POST","/",nil)
 require.NoError(t,client.EncodeMethodARequest(nil)(encoded,payload))
 require.Equal(t,values.Encode(),encoded.URL.RawQuery)
 request.URL.RawQuery=url.Values{"map":{"{\"1\":true}"}}.Encode()
 _,err=server.DecodeMethodARequest(loomhttp.NewMuxer(),nil)(request)
 if %t {
  require.Error(t,err,"named map validation must run after JSON decoding")
 } else {
  require.NoError(t,err)
 }
}
`
