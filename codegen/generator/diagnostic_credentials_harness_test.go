package generator

const diagnosticWireHarness = `package diagnostics_test

import (
 "bytes"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/observability/diagnostics"
)

type wireDoer struct {
 t *testing.T
 want string
}

// Do checks request bytes and returns the same unmodified response bytes.
func (d wireDoer) Do(r *http.Request) (*http.Response, error) {
 body, err := io.ReadAll(r.Body)
 if err != nil {
  return nil, err
 }
 if string(body) != d.want {
  d.t.Errorf("request wire changed: %s", body)
 }
 return &http.Response{Status: "200 OK", StatusCode: 200, Header: http.Header{"Content-Type": []string{r.Header.Get("Content-Type")}}, Body: io.NopCloser(strings.NewReader(d.want))}, nil
}

func assertPreview(t *testing.T, request *http.Request, want string, fields diagnostics.Fields) {
 t.Helper()
 debug := loomhttp.NewDebugDoer(wireDoer{t: t, want: want}, fields)
 response, err := debug.Do(request)
 if err != nil {
  t.Fatal(err)
 }
 body, err := io.ReadAll(response.Body)
 if err != nil {
  t.Fatal(err)
 }
 if err := response.Body.Close(); err != nil {
  t.Fatal(err)
 }
 if string(body) != want {
  t.Errorf("response wire changed: %s", body)
 }
 var output bytes.Buffer
 debug.Fprint(&output)
 if strings.Contains(output.String(), "diagnostic-secret-value") {
  t.Errorf("credential leaked: %s", output.String())
 }
 if !strings.Contains(output.String(), "visible") {
  t.Errorf("ordinary query lost: %s", output.String())
 }
}

func request(t *testing.T, body, contentType string) *http.Request {
 t.Helper()
 req := httptest.NewRequest(http.MethodPost, "http://example.test/?ordinary=visible", strings.NewReader(body))
 req.Header.Set("Content-Type", contentType)
 return req
}
`

const diagnosticMixedWireCases = `package diagnostics_test

import (
 "io"
 "testing"

 web "example.com/diagnostics/gen/web"
 rpc "example.com/diagnostics/gen/rpc"
 webclient "example.com/diagnostics/gen/http/web/client"
 rpcclient "example.com/diagnostics/gen/jsonrpc/rpc/client"
 webpolicy "example.com/diagnostics/gen/http/web"
 rpcpolicy "example.com/diagnostics/gen/jsonrpc/rpc"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedJSONRPCEnvelopePreview(t *testing.T) {
 req := request(t, "", "application/json")
 if err := rpcclient.EncodeBodyRequest(loomhttp.RequestEncoder)(req, &rpc.BodyPayload{Credential: "diagnostic-secret-value"}); err != nil {
  t.Fatal(err)
 }
 raw, err := req.GetBody()
 if err != nil {
  t.Fatal(err)
 }
 body, err := io.ReadAll(raw)
 if err != nil {
  t.Fatal(err)
 }
 if err := raw.Close(); err != nil {
  t.Fatal(err)
 }
 assertPreview(t, req, string(body), rpcpolicy.Fields())
}

func TestGeneratedFormPreview(t *testing.T) {
 value := webclient.NewFormRequestBody(&web.FormPayload{Credential: "diagnostic-secret-value"})
 values, err := loomhttp.EncodeFormValues(value)
 if err != nil {
  t.Fatal(err)
 }
 if values.Get("credential") != "diagnostic-secret-value" {
  t.Fatalf("unexpected form: %v", values)
 }
 body := values.Encode()
 assertPreview(t, request(t, body, "application/x-www-form-urlencoded"), body, webpolicy.Fields())
}
`

const diagnosticBodyWireCases = `package diagnostics_test

import (
 "encoding/json/v2"
 "testing"

 custom "example.com/diagnostics/gen/custom"
 customclient "example.com/diagnostics/gen/http/custom/client"
 custompolicy "example.com/diagnostics/gen/http/custom"
 scalarpolicy "example.com/diagnostics/gen/http/scalar"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedPhysicalBodyPreview(t *testing.T) {
 value := customclient.NewShowRequestBody(&custom.ShowPayload{Permit: "diagnostic-secret-value"})
 encoded, err := json.Marshal(value)
 if err != nil {
  t.Fatal(err)
 }
 body := string(encoded)
 assertPreview(t, request(t, body, "application/json"), body, custompolicy.Fields())
 values, err := loomhttp.EncodeFormValues(value)
 if err != nil {
  t.Fatal(err)
 }
 if values.Get("proof") != "diagnostic-secret-value" {
  t.Fatalf("unexpected custom form: %v", values)
 }
 body = values.Encode()
 assertPreview(t, request(t, body, "application/x-www-form-urlencoded"), body, custompolicy.Fields())
 body = "\"diagnostic-secret-value\""
 assertPreview(t, request(t, body, "application/json"), body, scalarpolicy.Fields())
}
`
