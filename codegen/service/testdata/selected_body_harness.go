package testdata

import (
	"strconv"
	"strings"
)

// SelectedBodyFieldHarness exercises generated clients, mounted servers and CLI
// builders against the service field selected by Body, including renamed fields.
func SelectedBodyFieldHarness(transport, direction, shape string, renamed bool) string {
	member := "Value"
	if renamed {
		member = "ActualValue"
	}
	values := make([]string, 0, 3)
	values = append(values, `"hello"`, `""`)
	switch shape {
	case "array":
		values = append(values[:0], `["hello"]`, `[]`)
	case "object":
		values = append(values[:0], `{"text":"hello"}`, `{}`)
	case "nullable":
		values = append(values, `null`)
	}
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	rawCLI := shape == "string" || shape == "optional" || shape == "defaulted"
	setup := `server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil).Mount(mux)`
	wire := `require.JSONEq(t, value, string(requestBytes))`
	invalid := `wire := value`
	invalidStatus := `require.Equal(t, http.StatusBadRequest, rec.Code)
 var problem loomhttp.ProblemResponse
 require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
 require.Equal(t, "decode_payload", problem.Code)
 require.NotEmpty(t, problem.Detail)`
	if transport == "jsonrpc" {
		setup = `server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil).Mount(mux)`
		wire = `var envelope struct { Params jsontext.Value ` + "`json:\"params\"`" + ` }
 require.NoError(t, json.Unmarshal(requestBytes, &envelope))
 require.JSONEq(t, value, string(envelope.Params))`
		invalid = `wire := "{\"jsonrpc\":\"2.0\",\"id\":\"bad\",\"method\":\"call\",\"params\":"+value+"}"`
		invalidStatus = `require.Equal(t, http.StatusOK, rec.Code)
 var problem struct { Error struct { Code int ` + "`json:\"code\"`" + `; Message string ` + "`json:\"message\"`" + ` } ` + "`json:\"error\"`" + ` }
 require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
 require.NotZero(t, problem.Error.Code)
 require.NotEmpty(t, problem.Error.Message)`
	}
	template := selectedRequestHarness
	if direction == "response" {
		template = selectedResponseHarness
		if transport == "jsonrpc" {
			template = selectedJSONRPCResponseCodecHarness
		}
	}
	if direction == "request" && transport == "http" && (shape == "optional" || shape == "defaulted") {
		expected := `require.Nil(t, payload.MEMBER)`
		if shape == "defaulted" {
			expected = `require.Equal(t, "fallback", payload.MEMBER)`
		}
		template += strings.ReplaceAll(selectedAbsenceHarness, "ABSENCECHECK", expected)
	}
	return strings.NewReplacer("TRANSPORT", transport, "MEMBER", member, "VALUES", strings.Join(quoted, ", "), "RAWCLI", strconv.FormatBool(rawCLI), "SETUP", setup, "WIRECHECK", wire, "INVALIDWIRE", invalid, "INVALIDSTATUS", invalidStatus, "OPTIONALCLI", strconv.FormatBool(shape == "optional")).Replace(template)
}

const selectedRequestHarness = `package bodyfields_test
import (
 "bytes"
 "context"
 json "encoding/json/v2"
 "encoding/json/jsontext"
 "io"
 "net/http"
 "net/http/httptest"
 "testing"
 "github.com/stretchr/testify/require"
 fields "example.com/bodyfields/gen/fields"
 "example.com/bodyfields/gen/TRANSPORT/fields/client"
 "example.com/bodyfields/gen/TRANSPORT/fields/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
type doerFunc func(*http.Request)(*http.Response,error)
func (f doerFunc) Do(r *http.Request)(*http.Response,error) { return f(r) }
func TestSelectedRequestRoundTrip(t *testing.T) {
 for _,value := range []string{VALUES} {
  t.Run(value,func(t *testing.T) {
   var input fields.CallPayload
   require.NoError(t,json.Unmarshal([]byte("{\"value\":"+value+"}"),&input))
   called:=false
   endpoints:=&fields.Endpoints{Call:func(_ context.Context,v any)(any,error) {
    called=true
    require.Equal(t,input.MEMBER,v.(*fields.CallPayload).MEMBER)
    return nil,nil
   }}
   mux:=loomhttp.NewMuxer()
   SETUP
   var requestBytes []byte
   doer:=doerFunc(func(r *http.Request)(*http.Response,error) {
    var err error
    requestBytes,err=io.ReadAll(r.Body)
    require.NoError(t,err)
    require.NoError(t,r.Body.Close())
    r.Body=io.NopCloser(bytes.NewReader(requestBytes))
    rec:=httptest.NewRecorder()
    mux.ServeHTTP(rec,r)
    return rec.Result(),nil
   })
   _,err:=client.NewClient("http","example.test",doer,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false).Call()(context.Background(),&input)
   require.NoError(t,err)
   require.True(t,called)
   WIRECHECK
   cliValue:=value
   if RAWCLI { require.NoError(t,json.Unmarshal([]byte(value),&cliValue)) }
   built,err:=client.BuildCallPayload(cliValue)
   require.NoError(t,err)
   if OPTIONALCLI && cliValue == "" {
    // The raw optional-string CLI argument uses empty text for absence.
    // The separate mounted wire assertion above preserves explicit empty input.
    require.Nil(t,built.MEMBER)
   } else {
    require.Equal(t,input.MEMBER,built.MEMBER)
   }
  })
 }
}
func TestSelectedRequestRejectsMalformedInput(t *testing.T) {
 for _,value:=range []string{"{", "[", "!"} {
  t.Run(value,func(t *testing.T) {
   called:=false
   endpoints:=&fields.Endpoints{Call:func(context.Context,any)(any,error) { called=true; return nil,nil }}
   mux:=loomhttp.NewMuxer()
   SETUP
   INVALIDWIRE
   req:=httptest.NewRequest(http.MethodPost,"http://example.test/",bytes.NewBufferString(wire))
   req.Header.Set("Content-Type","application/json")
   rec:=httptest.NewRecorder()
   mux.ServeHTTP(rec,req)
   require.False(t,called)
   INVALIDSTATUS
  })
 }
}
var _ jsontext.Value
`

const selectedResponseHarness = `package bodyfields_test
import (
 "context"
 json "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "testing"
 "github.com/stretchr/testify/require"
 fields "example.com/bodyfields/gen/fields"
 "example.com/bodyfields/gen/TRANSPORT/fields/client"
 "example.com/bodyfields/gen/TRANSPORT/fields/server"
 loomhttp "github.com/CaliLuke/loom/http"
)
type doerFunc func(*http.Request)(*http.Response,error)
func (f doerFunc) Do(r *http.Request)(*http.Response,error) { return f(r) }
func TestSelectedResponseRoundTrip(t *testing.T) {
 for _,value:=range []string{VALUES} {
  t.Run(value,func(t *testing.T) {
   var output fields.CallResult
   require.NoError(t,json.Unmarshal([]byte("{\"value\":"+value+"}"),&output))
   endpoints:=&fields.Endpoints{Call:func(context.Context,any)(any,error) { return &output,nil }}
   mux:=loomhttp.NewMuxer()
   SETUP
   doer:=doerFunc(func(r *http.Request)(*http.Response,error) {
    rec:=httptest.NewRecorder()
    mux.ServeHTTP(rec,r)
    return rec.Result(),nil
   })
   result,err:=client.NewClient("http","example.test",doer,loomhttp.RequestEncoder,loomhttp.ResponseDecoder,false).Call()(context.Background(),nil)
   require.NoError(t,err)
   require.Equal(t,output.MEMBER,result.(*fields.CallResult).MEMBER)
  })
 }
}
`

const selectedAbsenceHarness = `
func TestSelectedBodyAbsence(t *testing.T) {
 for _,body:=range []string{"", " \n\t"} {
  req:=httptest.NewRequest(http.MethodPost,"http://example.test/",bytes.NewBufferString(body))
  req.Header.Set("Content-Type","application/json")
  payload,err:=server.DecodeCallRequest(loomhttp.NewMuxer(),loomhttp.RequestDecoder)(req)
  require.NoError(t,err)
  require.NotNil(t,payload)
  ABSENCECHECK
 }
}
`

// The JSON-RPC mounted handler has a separately reproduced selected-result
// bypass. Exercise its generated encoder/decoder owners here without claiming
// that the independent handler defect is repaired by member naming.
const selectedJSONRPCResponseCodecHarness = `package bodyfields_test
import (
 "context"
 json "encoding/json/v2"
 "encoding/json/jsontext"
 "net/http/httptest"
 "testing"
 "github.com/stretchr/testify/require"
 fields "example.com/bodyfields/gen/fields"
 "example.com/bodyfields/gen/jsonrpc/fields/client"
 "example.com/bodyfields/gen/jsonrpc/fields/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/jsonrpc"
)
func TestSelectedResponseCodecRoundTrip(t *testing.T) {
 for _,value:=range []string{VALUES} {
  t.Run(value,func(t *testing.T) {
   var output fields.CallResult
   require.NoError(t,json.Unmarshal([]byte("{\"value\":"+value+"}"),&output))
   body:=httptest.NewRecorder()
   require.NoError(t,server.EncodeCallResponse(loomhttp.ResponseEncoder)(context.Background(),body,&output))
   require.JSONEq(t,value,body.Body.String())
   envelope:=jsonrpc.MakeSuccessResponse("fixed",jsontext.Value(body.Body.Bytes()))
   response:=httptest.NewRecorder()
   require.NoError(t,loomhttp.ResponseEncoder(context.Background(),response).Encode(envelope))
   result,err:=client.DecodeCallResponse(loomhttp.ResponseDecoder,false)(response.Result())
   require.NoError(t,err)
   require.Equal(t,output.MEMBER,result.(*fields.CallResult).MEMBER)
  })
 }
}
`
