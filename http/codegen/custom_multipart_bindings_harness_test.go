package codegen

const customMultipartBindingsHarness = `package integration

import (
 "bytes"
 "context"
 "encoding/json/v2"
 "fmt"
 "io"
 "mime/multipart"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 loom "github.com/CaliLuke/loom/pkg"
 loomhttp "github.com/CaliLuke/loom/http"
 uploads "example.com/multipart/gen/uploads"
 server "example.com/multipart/gen/http/uploads/server"
)

func TestCustomMultipartWire(t *testing.T) {
 const location = "LOCATION"
 type testCase struct {
  name, value, authorization, query, callbackError string
  malformed bool
  status int
  code, detail string
  callbacks, calls int
 }
 authorization := ""
 if location == "basic" {
  authorization = "Basic dXNlcjpwYXNz"
 } else if location == "bearer" {
  authorization = "Bearer token"
 }
 cases := []testCase{
  {name:"valid", value:"9", authorization:authorization, status:204, callbacks:1, calls:1},
  {name:"malformed multipart", value:"9", authorization:authorization, malformed:true, status:400, code:"decode_payload", detail:"invalid request body"},
  {name:"callback private error", value:"9", authorization:authorization, callbackError:"private", status:400, code:"decode_payload", detail:"invalid request body", callbacks:1},
  {name:"callback service error", value:"9", authorization:authorization, callbackError:"service", status:400, code:"missing_field", detail:"files", callbacks:1},
 }
 if location != "none" && location != "path_string" {
  cases = append(cases,
   testCase{name:"conversion", value:"bad", authorization:authorization, status:400, code:"invalid_field_type", detail:"validation error", callbacks:1},
   testCase{name:"constraint", value:"0", authorization:authorization, status:400, code:"invalid_range", detail:"validation error", callbacks:1})
  if location != "path" {
   cases = append(cases, testCase{name:"missing", authorization:authorization, status:400, code:"missing_field", detail:"value", callbacks:1})
  }
 }
 if location == "query" || location == "basic" || location == "bearer" || location == "custom_query" {
  cases = append(cases, testCase{name:"malformed query", value:"9", authorization:authorization, query:"value=%zz", status:400, code:"decode_payload", detail:"invalid query string", callbacks:1})
 }
 if location == "basic" {
  cases = append(cases,
   testCase{name:"short credentials", value:"9", authorization:"Basic YTpi", status:400, code:"invalid_length", detail:"validation error", callbacks:1},
   testCase{name:"missing credentials", value:"9", status:400, code:"missing_field", detail:"Authorization", callbacks:1})
 }
 if location == "bearer" {
  cases = append(cases, testCase{name:"wrong framing", value:"9", authorization:"Basic dXNlcjpwYXNz", status:400, code:"decode_payload", detail:"invalid Authorization bearer credential", callbacks:1})
 }
 if location == "custom_query" {
  for i := range cases {
   if cases[i].name == "constraint" {
    cases[i].code = "invalid_pattern"
   }
  }
 }
 for _, test := range cases {
  t.Run(test.name, func(t *testing.T) {
   callbacks, calls := 0, 0
   var got *uploads.SendPayload
   mux := loomhttp.NewMuxer()
   endpoints := &uploads.Endpoints{Send:func(_ context.Context, value any) (any,error) {
    calls++
    got = value.(*uploads.SendPayload)
    return nil,nil
   }}
   decoder := func(reader *multipart.Reader, payload **uploads.SendPayload) error {
    callbacks++
    if test.callbackError == "private" {
     return fmt.Errorf("private callback secret")
    }
    if test.callbackError == "service" {
     return loom.MissingFieldError("files", "body")
    }
    part, err := reader.NextPart()
    if err != nil {
     return err
    }
    data, err := io.ReadAll(part)
    if err != nil {
     return err
    }
    if err := part.Close(); err != nil {
     return err
    }
    *payload = &uploads.SendPayload{Files:[][]byte{data}}
    return nil
   }
   transport := server.New(endpoints,mux,loomhttp.RequestDecoder,loomhttp.ResponseEncoder,nil,nil,decoder)
   server.Mount(mux,transport)
   var body bytes.Buffer
   writer := multipart.NewWriter(&body)
   if err := writer.WriteField("files","contents"); err != nil {
    t.Fatal(err)
   }
   if err := writer.Close(); err != nil {
    t.Fatal(err)
   }
   target := "/send"
   if location == "path" || location == "path_string" {
    target += "/" + test.value
   }
   if location == "query" || location == "basic" || location == "bearer" || location == "custom_query" {
    target += "?value=" + test.value
    if test.query != "" {
     target = "/send?" + test.query
    }
   }
   request := httptest.NewRequest(http.MethodPost,target,&body)
   request.Header.Set("Content-Type",writer.FormDataContentType())
   if test.malformed {
    request.Header.Set("Content-Type","application/json")
   }
   if location == "header" {
    request.Header.Set("X-Value",test.value)
   }
   if location == "cookie" && test.value != "" {
    request.AddCookie(&http.Cookie{Name:"value",Value:test.value})
   }
   if test.authorization != "" {
    request.Header.Set("Authorization",test.authorization)
   }
   response := httptest.NewRecorder()
   mux.ServeHTTP(response,request)
   if response.Code != test.status || callbacks != test.callbacks || calls != test.calls {
    t.Fatalf("status/callbacks/calls = %d/%d/%d, want %d/%d/%d; %s",response.Code,callbacks,calls,test.status,test.callbacks,test.calls,response.Body.String())
   }
   if test.code != "" {
    var problem struct {
     Code string ` + "`json:\"code\"`" + `
     Detail string ` + "`json:\"detail\"`" + `
    }
    if err := json.Unmarshal(response.Body.Bytes(),&problem); err != nil {
     t.Fatal(err)
    }
    if problem.Code != test.code || !strings.Contains(problem.Detail,test.detail) {
     t.Errorf("problem = %#v, want %s containing %q",problem,test.code,test.detail)
    }
    if strings.Contains(response.Body.String(),"private callback secret") {
     t.Error("private callback error leaked")
    }
   } else {
    if got == nil || len(got.Files) != 1 || string(got.Files[0]) != "contents" {
     t.Fatalf("decoded payload = %#v",got)
    }
    VALUE_CHECK
   }
  })
 }
}
`

const customMultipartTokenSource = `package scalar

import "strconv"

// Token is a decimal token decoded through the custom text contract.
type Token string

// UnmarshalText parses a decimal integer token.
func (t *Token) UnmarshalText(text []byte) error {
 if _, err := strconv.Atoi(string(text)); err != nil {
  return err
 }
 *t = Token(text)
 return nil
}
`
