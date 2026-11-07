package generator

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestNativeDefaultContractArtifacts emits the bounded acceptance specimens.
// Independent go test processes supply byte-determinism evidence; compilation
// and runtime checks operate on those exact artifacts, without regeneration.
func TestNativeDefaultContractArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_NATIVE_DEFAULT_ARTIFACTS")
	if output == "" {
		t.Skip("set LOOM_NATIVE_DEFAULT_ARTIFACTS for native default qualification")
	}
	for _, transport := range []string{"http", "grpc"} {
		t.Run(transport+"-control", func(t *testing.T) {
			dir := filepath.Join(output, transport+"-control")
			require.NoError(t, os.MkdirAll(dir, 0o750))
			renderDesignModule(t, dir, "example.com/objectdefaults", httpdata.ObjectDefaultsDSL(transport))
		})
		t.Run(transport, func(t *testing.T) {
			dir := filepath.Join(output, transport)
			require.NoError(t, os.MkdirAll(dir, 0o750))
			renderDesignModule(t, dir, "example.com/nativedefaults", nativeDefaultContractDesign(transport))
			harness := nativeDefaultHTTPHarness
			if transport == "grpc" {
				harness = nativeDefaultGRPCHarness
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
		})
	}
	t.Run("opaque", func(t *testing.T) {
		dir := filepath.Join(output, "opaque")
		require.NoError(t, os.MkdirAll(dir, 0o750))
		source := "package opaque\nimport (\"reflect\"; \"testing\"; web \"net/url\")\nfunc TestOpaqueTypedLiterals(t *testing.T) {\n"
		for index, tc := range []struct {
			target string
			raw    any
			want   string
		}{
			{"web.Values", url.Values{"q": {"one"}}, `web.Values{"q": {"one"}}`},
			{"[]web.Values", []url.Values{{"q": {"one"}}}, `[]web.Values{{"q": {"one"}}}`},
			{"map[string]web.Values", map[string]url.Values{"a": {"q": {"one"}}}, `map[string]web.Values{"a": {"q": {"one"}}}`},
			{"[1]web.Values", [1]url.Values{{"q": {"one"}}}, `[1]web.Values{{"q": {"one"}}}`},
			{"[]web.Values", []url.Values{}, `[]web.Values{}`},
			{"[]web.Values", []url.Values(nil), `[]web.Values(nil)`},
			{"map[string]web.Values", map[string]url.Values{}, `map[string]web.Values{}`},
			{"map[string]web.Values", map[string]url.Values(nil), `map[string]web.Values(nil)`},
		} {
			attribute := &expr.AttributeExpr{Type: expr.Any, Meta: expr.MetaExpr{"struct:field:type": {tc.target, "net/url", "web"}}}
			context := codegen.NewAttributeContext(false, false, true, "", codegen.NewNameScope())
			plan, err := context.DefaultValue(attribute, tc.raw)
			require.NoError(t, err)
			source += fmt.Sprintf("var value%d %s = %s\nif !reflect.DeepEqual(value%d, %s) {\n t.Errorf(\"case %d: %%#v\", value%d)\n}\n", index, tc.target, plan.Expression, index, tc.want, index, index)
		}
		source += "}\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "values_test.go"), []byte(source), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/opaquedefaults\n\ngo 1.27\n"), 0o600))
	})
}

func nativeDefaultContractDesign(transport string) func() {
	return func() {
		d.API("native_defaults", func() {
		})
		choice := d.Type("Choice", d.OneOf(func() {
			d.Field(1, "text", d.String)
			d.Field(2, "number", d.Int, func() {
				d.Minimum(0)
			})
		}))
		holder := d.Type("Holder", func() {
			d.Field(1, "choice", choice)
			d.Field(3, "label", d.String)
			d.Required("label")
		})
		requiredHolder := d.Type("RequiredHolder", func() {
			d.Field(1, "choice", choice)
			d.Required("choice")
		})
		outer := d.Type("Outer", d.OneOf(func() {
			d.Field(1, "nested", choice)
			d.Field(3, "flag", d.Boolean)
		}))
		nestedHolder := d.Type("NestedHolder", func() {
			d.Field(1, "outer", outer)
		})
		d.Service("defaults", func() {
			d.Method("send", func() {
				d.Payload(func() {
					d.Field(1, "value", holder, func() {
						d.Default(map[string]any{"choice": "hello", "label": "kept"})
					})
					d.Field(2, "items", d.ArrayOf(holder), func() {
						d.Default([]any{map[string]any{"choice": 2, "label": "item"}})
					})
					d.Field(3, "indexed", d.MapOf(d.String, holder), func() {
						d.Default(map[string]any{"a": map[string]any{"choice": "mapped", "label": "map"}})
					})
					d.Field(7, "omitted", holder, func() {
						d.Default(map[string]any{"label": "omitted"})
					})
					d.Field(8, "requiredValue", requiredHolder, func() {
						d.Default(map[string]any{"choice": 3})
					})
					d.Field(9, "nested", nestedHolder, func() {
						d.Default(map[string]any{"outer": "nested"})
					})
					if transport == "http" {
						d.Field(4, "choices", d.ArrayOf(choice), func() {
							d.Default([]any{"hello", 0})
						})
						d.Field(5, "maybe", holder, func() {
							d.Nullable()
							d.Default(map[string]any{"choice": 0, "label": "nullable"})
						})
						d.Field(6, "empty", d.ArrayOf(choice), func() {
							d.Default([]any{})
						})
					}
				})
				if transport == "grpc" {
					d.GRPC(func() {
					})
				} else {
					d.HTTP(func() {
						d.POST("/send")
					})
				}
			})
		})
	}
}

const nativeDefaultHTTPHarness = `package defaults_test
import (
 "context"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 svc "example.com/nativedefaults/gen/defaults"
 server "example.com/nativedefaults/gen/http/defaults/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)
func TestNativeDefaultHTTP(t *testing.T) {
 var received *svc.SendPayload
 calls := 0
 endpoint := func(_ context.Context, payload any) (any,error) {
  calls++
  received = payload.(*svc.SendPayload)
  return nil,nil
 }
 mux := loomhttp.NewMuxer()
 server.Mount(mux,server.New(&svc.Endpoints{Send:endpoint},mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
 for _, tc := range []struct{name,body string; valid bool}{
  {"absent", "{}",true},
  {"supplied", "{\"value\":{\"choice\":{\"type\":\"number\",\"value\":0},\"label\":\"\"},\"items\":[],\"indexed\":{},\"choices\":[],\"maybe\":null}",true},
  {"invalid branch", "{\"value\":{\"choice\":{\"type\":\"number\",\"value\":-1},\"label\":\"bad\"}}",false},
  {"nonnull required shape", "{\"value\":null}",false},
 } {
  t.Run(tc.name,func(t *testing.T) {
   before:=calls
   request:=httptest.NewRequest(http.MethodPost,"/send",strings.NewReader(tc.body))
   request.Header.Set("Content-Type","application/json")
   response:=httptest.NewRecorder()
   mux.ServeHTTP(response,request)
   if !tc.valid {
    require.Equal(t,before,calls)
    require.Equal(t,http.StatusBadRequest,response.Code)
    require.Contains(t,response.Body.String(),"detail")
    return
   }
   require.Equal(t,before+1,calls,response.Body.String())
   require.Equal(t,http.StatusNoContent,response.Code)
   require.NotNil(t,received.Value)
   require.NotNil(t,received.Value.Choice)
   require.NotNil(t,received.Omitted)
   require.Nil(t,received.Omitted.Choice)
   required,ok:=received.RequiredValue.Choice.AsNumber(); require.True(t,ok); require.Equal(t,3,int(required))
   nested,ok:=received.Nested.Outer.AsNested(); require.True(t,ok); require.NotNil(t,nested)
   nestedText,ok:=nested.AsText(); require.True(t,ok); require.Equal(t,"nested",string(nestedText))
   require.NotNil(t,received.Empty)
   require.Empty(t,received.Empty)
   if tc.name=="absent" {
    text,ok:=received.Value.Choice.AsText(); require.True(t,ok); require.Equal(t,"hello",string(text))
    number,ok:=received.Items[0].Choice.AsNumber(); require.True(t,ok); require.Equal(t,2,int(number))
    text,ok=received.Indexed["a"].Choice.AsText(); require.True(t,ok); require.Equal(t,"mapped",string(text))
    require.Len(t,received.Choices,2)
    text,ok=received.Choices[0].AsText()
    require.True(t,ok)
    require.Equal(t,"hello",string(text))
    number,ok=received.Choices[1].AsNumber()
    require.True(t,ok)
    require.Zero(t,number)
    maybe,ok:=received.Maybe.Value(); require.True(t,ok); require.Equal(t,"nullable",maybe.Label)
    require.NotNil(t,maybe.Choice)
    number,ok=maybe.Choice.AsNumber()
    require.True(t,ok)
    require.Zero(t,number)
   } else {
    number,ok:=received.Value.Choice.AsNumber(); require.True(t,ok); require.Zero(t,number)
    require.Empty(t,received.Value.Label)
    require.NotNil(t,received.Items); require.Empty(t,received.Items)
    require.NotNil(t,received.Indexed); require.Empty(t,received.Indexed)
    require.NotNil(t,received.Choices); require.Empty(t,received.Choices)
    require.True(t,received.Maybe.IsNull())
   }
  })
 }
}
`

const nativeDefaultGRPCHarness = `package defaults_test
import (
 "context"
 "testing"
 svc "example.com/nativedefaults/gen/defaults"
 client "example.com/nativedefaults/gen/grpc/defaults/client"
 server "example.com/nativedefaults/gen/grpc/defaults/server"
 "github.com/stretchr/testify/require"
)
func TestNativeDefaultGRPC(t *testing.T) {
 wire,err:=client.NewProtoSendRequest(&svc.SendPayload{})
 require.NoError(t,err)
 decoded,err:=server.DecodeSendRequest(context.Background(),wire,nil)
 require.NoError(t,err)
 actual:=decoded.(*svc.SendPayload)
 require.NotNil(t,actual.Omitted)
 require.Nil(t,actual.Omitted.Choice)
 required,ok:=actual.RequiredValue.Choice.AsNumber(); require.True(t,ok); require.Equal(t,3,int(required))
 nested,ok:=actual.Nested.Outer.AsNested(); require.True(t,ok); require.NotNil(t,nested)
 nestedText,ok:=nested.AsText(); require.True(t,ok); require.Equal(t,"nested",string(nestedText))
 require.NotNil(t,actual.Value)
 require.NotNil(t,actual.Value.Choice)
 text,ok:=actual.Value.Choice.AsText(); require.True(t,ok); require.Equal(t,"hello",string(text))
 number,ok:=actual.Items[0].Choice.AsNumber(); require.True(t,ok); require.Equal(t,2,int(number))
 text,ok=actual.Indexed["a"].Choice.AsText(); require.True(t,ok); require.Equal(t,"mapped",string(text))
 // Supplied native zero must survive client/server conversion.
 actual.Value.Choice.SetNumber(0)
 actual.Value.Label=""
 wire,err=client.NewProtoSendRequest(actual)
 require.NoError(t,err)
 decoded,err=server.DecodeSendRequest(context.Background(),wire,nil)
 require.NoError(t,err)
 actual=decoded.(*svc.SendPayload)
 number,ok=actual.Value.Choice.AsNumber(); require.True(t,ok); require.Zero(t,number)
 require.Empty(t,actual.Value.Label)
 actual.Value.Choice.SetNumber(-1)
 wire,err=client.NewProtoSendRequest(actual)
 require.NoError(t,err)
 _,err=server.DecodeSendRequest(context.Background(),wire,nil)
 require.Error(t,err)
}
`
