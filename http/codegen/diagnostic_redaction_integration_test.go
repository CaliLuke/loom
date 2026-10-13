package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const diagnosticCredentialHarness = `package diagnostic_test

import (
 "bytes"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 svc "example.com/diagnostic/gen/diagnostic"
 client "example.com/diagnostic/gen/http/diagnostic/client"
 policy "example.com/diagnostic/gen/http/diagnostic"
 loomhttp "github.com/CaliLuke/loom/http"
)

type credentialDoer struct {
 t *testing.T
 location string
}

func TestGeneratedCredentialDiagnostics(t *testing.T) {
 const secret = "diagnostic-secret-value"
 for _, tc := range []struct {
  name string
  payload any
  encode func(*http.Request, any) error
 }{
  {"query", &svc.QueryPayload{Credential: secret}, client.EncodeQueryRequest(loomhttp.RequestEncoder)},
  {"header", &svc.HeaderPayload{Credential: secret}, client.EncodeHeaderRequest(loomhttp.RequestEncoder)},
  {"basic", &svc.BasicPayload{Login: secret, Passcode: secret}, client.EncodeBasicRequest(loomhttp.RequestEncoder)},
  {"body", &svc.BodyPayload{Credential: secret}, client.EncodeBodyRequest(loomhttp.RequestEncoder)},
 } {
  t.Run(tc.name, func(t *testing.T) {
   req := httptest.NewRequest(http.MethodPost, "http://example.test/"+tc.name+"?ordinary=visible", nil)
   if err := tc.encode(req, tc.payload); err != nil {
    t.Fatal(err)
   }
   debug := loomhttp.NewDebugDoer(credentialDoer{t: t, location: tc.name}, policy.Fields())
   response, err := debug.Do(req)
   if err != nil {
    t.Fatal(err)
   }
   if err := response.Body.Close(); err != nil {
    t.Fatal(err)
   }
   var output bytes.Buffer
   debug.Fprint(&output)
   if strings.Contains(output.String(), secret) {
    t.Errorf("generated %s credential disclosed: %s", tc.name, output.String())
   }
   if !strings.Contains(output.String(), "visible") {
    t.Error("ordinary diagnostic value lost")
   }
  })
 }
}

// Do checks application wire values and returns an empty response.
func (d credentialDoer) Do(r *http.Request) (*http.Response, error) {
 const secret = "diagnostic-secret-value"
 switch d.location {
 case "query":
  if got := r.URL.Query().Get("ticket"); got != secret {
   d.t.Errorf("query wire value = %q", got)
  }
 case "header":
  if got := r.Header.Get("X-Permit"); got != secret {
   d.t.Errorf("header wire value = %q", got)
  }
 case "body", "basic":
  body, err := io.ReadAll(r.Body)
  if err != nil {
   return nil, err
  }
  fields := []string{"permit"}
  if d.location == "basic" {
   fields = []string{"signin", "proof"}
  }
  for _, field := range fields {
   if !strings.Contains(string(body), "\""+field+"\":\""+secret+"\"") {
    d.t.Errorf("body wire value = %s", body)
   }
  }
 }
 return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Header: make(http.Header), Body: http.NoBody}, nil
}
`

func TestGeneratedCredentialDiagnostics(t *testing.T) {
	root := RunHTTPDSL(t, diagnosticCredentialDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/diagnostic", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "diagnostics_test.go"), []byte(diagnosticCredentialHarness), 0o600))
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261013051947-ee4552dc94ee")
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}
