package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestMergeJSONOmitOption preserves the exact ignore marker while retaining
// ordinary field names and options, including a literal dash field name.
func TestMergeJSONOmitOption(t *testing.T) {
	cases := []struct {
		tag, option, want string
	}{
		{"-", "", "-"},
		{"-", "omitempty", "-"},
		{"-", "omitzero", "-"},
		{"-,", "omitempty", "-,,omitempty"},
		{"-,omitempty", "omitzero", "-,omitzero"},
		{"secret", "omitempty", "secret,omitempty"},
		{"secret,omitzero", "omitempty", "secret,omitempty"},
		{"secret,omitempty,string", "omitzero", "secret,omitzero,string"},
		{"secret,string", "omitzero", "secret,string,omitzero"},
		{"-name", "omitzero", "-name,omitzero"},
	}
	for _, c := range cases {
		t.Run(c.tag+"/"+c.option, func(t *testing.T) {
			require.Equal(t, c.want, mergeJSONOmitOption(c.tag, c.option))
		})
	}
}

// TestRequiredJSONIgnoredField checks the supported required-field tag policy
// against generated client/server types and an HTTP request containing "-".
func TestRequiredJSONIgnoredField(t *testing.T) {
	for _, key := range []string{"struct:tag:json", "struct:tag:json:name"} {
		t.Run(key, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("sender", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("secret", String, func() {
								Meta(key, "-")
							})
							Attribute("visible", String)
							Required("secret", "visible")
						})
						HTTP(func() {
							POST("/")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/ignored", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored_test.go"), []byte(requiredJSONIgnoredFieldHarness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-count=1", "./...")
		})
	}
}

const requiredJSONIgnoredFieldHarness = `package ignored_test

import (
 "context"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "testing"

 client "example.com/ignored/gen/http/sender/client"
 server "example.com/ignored/gen/http/sender/server"
 "example.com/ignored/gen/sender"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestIgnoredField(t *testing.T) {
 for _, typ := range []reflect.Type{reflect.TypeFor[client.SendRequestBody](), reflect.TypeFor[server.SendRequestBody]()} {
  t.Run(typ.PkgPath(), func(t *testing.T) {
   field, ok := typ.FieldByName("Secret")
   if !ok || field.Tag.Get("json") != "-" {
    t.Errorf("secret field=%+v exists=%v, want exact ignore tag",field,ok)
   }
   value := reflect.New(typ)
   setString(value.Elem().FieldByName("Secret"),"hidden")
   setString(value.Elem().FieldByName("Visible"),"ok")
   data, err := json.Marshal(value.Interface())
   if err != nil || string(data) != ` + "`{\"visible\":\"ok\"}`" + ` {
    t.Errorf("encoded=%s err=%v",data,err)
   }
   decoded:=reflect.New(typ)
   if err:=json.Unmarshal([]byte(` + "`{\"visible\":\"ok\",\"-\":\"injected\",\"secret\":\"injected\"}`" + `),decoded.Interface());err!=nil {
    t.Fatal(err)
   }
   if !decoded.Elem().FieldByName("Secret").IsZero() {
    t.Error("ignored field accepted a wire value")
   }
  })
 }
}

func TestIgnoredFieldCannotSatisfyRequiredValidation(t *testing.T) {
 for _, body:=range []string{` + "`{\"visible\":\"ok\"}`,`{\"visible\":\"ok\",\"-\":\"injected\"}`" + `} {
  t.Run(body,func(t *testing.T){
   called:=false
   endpoints:=&sender.Endpoints{Send:func(_ context.Context,_ any)(any,error){
    called=true
    return nil,nil
   }}
   mux:=loomhttp.NewMuxer()
   server.Mount(mux,server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil))
   req:=httptest.NewRequest(http.MethodPost,"/",strings.NewReader(body))
   req.Header.Set("Content-Type","application/json")
   recorder:=httptest.NewRecorder()
   mux.ServeHTTP(recorder,req)
   var problem loomhttp.ProblemResponse
   if err:=json.Unmarshal(recorder.Body.Bytes(),&problem);err!=nil {
    t.Fatal(err)
   }
   if recorder.Code!=http.StatusBadRequest || problem.Code!="missing_field" || problem.Detail!="Missing required field: secret" || called {
    t.Errorf("status=%d problem=%+v serviceCalled=%v",recorder.Code,problem,called)
   }
  })
 }
}

func setString(field reflect.Value, value string) {
 if field.Kind()==reflect.Pointer {
  field.Set(reflect.New(field.Type().Elem()))
  field=field.Elem()
 }
 field.SetString(value)
}
`
