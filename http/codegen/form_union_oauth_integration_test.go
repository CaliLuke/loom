package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestFormRequestUnionOAuthIntegration(t *testing.T) {
	const (
		modulePath = "example.com/formunionit"
		genpkg     = modulePath + "/gen"
	)

	root := RunHTTPDSL(t, oauthFormRequestUnionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(oauthIntegrationHarness), 0644); err != nil {
		t.Fatalf("write integration harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestLargeErrorSetHTTPClientIntegration(t *testing.T) {
	const modulePath = "example.com/largeerrorsit"

	root := RunHTTPDSL(t, testdata.LargeErrorSetHTTPClientDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func renderHTTPModule(t *testing.T, dir, modulePath string, root *expr.RootExpr) {
	t.Helper()

	genpkg := modulePath + "/gen"
	serviceData := servicecodegen.NewServicesData(root)
	httpData := CreateHTTPServices(root)

	files := make([]*cg.File, 0, len(root.Services)*2+5)
	userTypePkgs := make(map[string][]string)
	for _, service := range root.Services {
		files = append(files, servicecodegen.Files(genpkg, service, serviceData, userTypePkgs)...)
		if views := servicecodegen.ViewsFile(genpkg, service, serviceData); views != nil {
			files = append(files, views)
		}
		files = append(files, servicecodegen.EndpointFile(genpkg, service, serviceData))
	}
	files = append(files, PathFiles(httpData)...)
	files = append(files, ClientTypeFiles(genpkg, httpData)...)
	files = append(files, ClientFiles(genpkg, httpData)...)
	files = append(files, ClientCLIFiles(genpkg, httpData)...)
	files = append(files, ServerTypeFiles(genpkg, httpData)...)
	files = append(files, ServerFiles(genpkg, httpData)...)

	renderGeneratedFiles(t, dir, files)

	repoRoot := checkoutPinnedLoomModule(t)
	goMod := fmt.Sprintf(`module %s

go 1.27rc3

require github.com/CaliLuke/loom v1.0.0

replace github.com/CaliLuke/loom => %s
`, modulePath, repoRoot)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

// pinnedLoom memoizes the pinned loom module checkout so every module-compile
// test in the package shares one git fetch instead of paying for its own.
// TestMain removes the checkout after the package's tests finish.
var pinnedLoom struct {
	once sync.Once
	root string
	err  error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if pinnedLoom.root != "" {
		if err := os.RemoveAll(pinnedLoom.root); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove pinned loom checkout: %v\n", err)
		}
	}
	os.Exit(code)
}

func checkoutPinnedLoomModule(t *testing.T) string {
	t.Helper()

	pinnedLoom.once.Do(func() {
		pinnedLoom.root, pinnedLoom.err = os.MkdirTemp("", "loom-source-")
	})
	if pinnedLoom.err != nil {
		t.Fatalf("create Loom source checkout directory: %v", pinnedLoom.err)
	}

	repoRoot, err := loomsource.RepositoryRoot(".")
	if err != nil {
		t.Fatalf("resolve Loom repository root: %v", err)
	}
	path, err := loomsource.Resolve(repoRoot, filepath.Join(pinnedLoom.root, "loom-pinned"))
	if err != nil {
		t.Fatalf("resolve Loom source: %v", err)
	}
	t.Logf("using Loom source: %s", path)
	return path
}

func renderGeneratedFiles(t *testing.T, dir string, files []*cg.File) {
	t.Helper()

	for _, file := range files {
		if _, err := file.Render(dir); err != nil {
			t.Fatalf("render %s: %v", file.Path, err)
		}
	}
}

func runGoCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	runCommand(t, dir, "go", args...)
}

func runCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	out, err := runLoggedCommandAllowFailure(t, dir, name, args...)
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return out
}

func runLoggedCommandAllowFailure(t *testing.T, dir, name string, args ...string) (string, error) {
	t.Helper()

	cmdText := strings.Join(append([]string{name}, args...), " ")
	if dir == "" {
		t.Logf("running command: %s", cmdText)
	} else {
		t.Logf("running command in %s: %s", dir, cmdText)
	}

	start := time.Now()
	out, err := runCommandAllowFailure(dir, name, args...)
	duration := time.Since(start)
	if err != nil {
		t.Logf("command failed after %s: %s: %v", duration.Round(time.Millisecond), cmdText, err)
	} else {
		t.Logf("command completed in %s: %s", duration.Round(time.Millisecond), cmdText)
	}
	return out, err
}

func runCommandAllowFailure(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func oauthFormRequestUnionDSL() {
	var AuthorizationCodeGrant = Type("AuthorizationCodeGrant", func() {
		Attribute("client_id", String)
		Attribute("code", String)
		Attribute("redirect_uri", String)
		Required("client_id", "code", "redirect_uri")
	})
	var RefreshTokenGrant = Type("RefreshTokenGrant", func() {
	})
	var Grant = Type("Grant", func() {
		OneOf("Grant", func() {
			Meta("oneof:type:field", "grant_type")
			Attribute("AuthorizationCode", AuthorizationCodeGrant, func() {
				Meta("oneof:type:tag", "authorization_code")
			})
			Attribute("RefreshToken", RefreshTokenGrant, func() {
				Meta("oneof:type:tag", "refresh_token")
			})
		})
	})
	var TokenResponse = Type("TokenResponse", func() {
		Attribute("access_token", String)
		Attribute("token_type", String)
		Attribute("refresh_token", String)
		Required("access_token", "token_type")
	})

	Service("Token", func() {
		Method("Exchange", func() {
			Payload(Grant)
			Result(TokenResponse)
			HTTP(func() {
				POST("/token")
				FormRequest()
				Response(StatusOK)
			})
		})
	})
}

const oauthIntegrationHarness = `package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	"golang.org/x/oauth2"

	token "example.com/formunionit/gen/token"
	tokenclient "example.com/formunionit/gen/http/token/client"
	tokenserver "example.com/formunionit/gen/http/token/server"
)

func TestXOAuth2AuthCodeRequestUsesFlatFormFields(t *testing.T) {
	mux := loomhttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, loomhttp.RequestDecoder)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))

		form, err := url.ParseQuery(string(raw))
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := form.Get("grant_type"); got != "authorization_code" {
			t.Fatalf("grant_type = %q, want authorization_code", got)
		}
		if got := form.Get("client_id"); got != "client-123" {
			t.Fatalf("client_id = %q, want client-123", got)
		}
		if got := form.Get("code"); got != "code-123" {
			t.Fatalf("code = %q, want code-123", got)
		}
		if got := form.Get("redirect_uri"); got != "https://client.example/callback" {
			t.Fatalf("redirect_uri = %q", got)
		}
		for key := range form {
			if key == "value" || strings.HasPrefix(key, "value[") {
				t.Fatalf("unexpected canonical union form key %q in %v", key, form)
			}
		}

		payload, err := decode(r)
		if err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Grant.Kind() != token.Grant2KindAuthorizationCode {
			t.Fatalf("decoded kind = %q, want %q", payload.Grant.Kind(), token.Grant2KindAuthorizationCode)
		}
		grant, ok := payload.Grant.AsAuthorizationCode()
		if !ok || grant == nil {
			t.Fatalf("decoded payload missing authorization_code branch: %#v", payload)
		}
		if grant.ClientID != "client-123" || grant.Code != "code-123" || grant.RedirectURI != "https://client.example/callback" {
			t.Fatalf("decoded auth code grant = %#v", grant)
		}

		w.Header().Set("Content-Type", "application/json")
		refreshToken := "refresh-123"
		if err := json.NewEncoder(w).Encode(&token.TokenResponse{
			AccessToken:  "access-123",
			TokenType:    "Bearer",
			RefreshToken: &refreshToken,
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := &oauth2.Config{
		ClientID:    "client-123",
		RedirectURL: "https://client.example/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://issuer.example/auth",
			TokenURL:  srv.URL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	tok, err := cfg.Exchange(context.Background(), "code-123")
	if err != nil {
		t.Fatalf("oauth2 exchange: %v", err)
	}
	if tok.AccessToken != "access-123" {
		t.Fatalf("access token = %q, want access-123", tok.AccessToken)
	}
	if tok.RefreshToken != "refresh-123" {
		t.Fatalf("refresh token = %q, want refresh-123", tok.RefreshToken)
	}
}

func TestGeneratedClientUsesFlatFormFieldsForRefreshToken(t *testing.T) {
	mux := loomhttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, loomhttp.RequestDecoder)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))

		form, err := url.ParseQuery(string(raw))
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("grant_type = %q, want refresh_token", got)
		}
		if len(form) != 1 {
			t.Fatalf("form = %v, want only grant_type", form)
		}
		for key := range form {
			if key == "value" || strings.HasPrefix(key, "value[") {
				t.Fatalf("unexpected canonical union form key %q in %v", key, form)
			}
		}

		payload, err := decode(r)
		if err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Grant.Kind() != token.Grant2KindRefreshToken {
			t.Fatalf("decoded kind = %q, want %q", payload.Grant.Kind(), token.Grant2KindRefreshToken)
		}
		grant, ok := payload.Grant.AsRefreshToken()
		if !ok || grant == nil {
			t.Fatalf("decoded payload missing refresh_token branch: %#v", payload)
		}
		if *grant != (token.RefreshTokenGrant{}) {
			t.Fatalf("decoded refresh grant = %#v, want zero-value object", *grant)
		}

		w.Header().Set("Content-Type", "application/json")
		refreshToken := "refresh-456"
		if err := json.NewEncoder(w).Encode(&token.TokenResponse{
			AccessToken:  "access-456",
			TokenType:    "Bearer",
			RefreshToken: &refreshToken,
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := tokenclient.NewClient(u.Scheme, u.Host, srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	endpoint := client.Exchange()

	result, err := endpoint(context.Background(), &token.Grant{
		Grant: func() *token.Grant2 {
			grant := token.NewGrant2RefreshToken(&token.RefreshTokenGrant{})
			return &grant
		}(),
	})
	if err != nil {
		t.Fatalf("generated client exchange: %v", err)
	}
	tokenRes, ok := result.(*token.TokenResponse)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}
	if tokenRes.AccessToken != "access-456" {
		t.Fatalf("access token = %q, want access-456", tokenRes.AccessToken)
	}
	if tokenRes.RefreshToken == nil || *tokenRes.RefreshToken != "refresh-456" {
		t.Fatalf("refresh token = %#v, want refresh-456", tokenRes.RefreshToken)
	}
}

func TestZeroFieldRefreshGrantDecodesFromDiscriminatorAndCookie(t *testing.T) {
	mux := loomhttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, loomhttp.RequestDecoder)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))

		form, err := url.ParseQuery(string(raw))
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("grant_type = %q, want refresh_token", got)
		}
		if len(form) != 1 {
			t.Fatalf("form = %v, want only grant_type", form)
		}
		if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "session-cookie" {
			t.Fatalf("session cookie = (%v, %v), want session-cookie", cookie, err)
		}

		payload, err := decode(r)
		if err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Grant.Kind() != token.Grant2KindRefreshToken {
			t.Fatalf("decoded kind = %q, want %q", payload.Grant.Kind(), token.Grant2KindRefreshToken)
		}
		grant, ok := payload.Grant.AsRefreshToken()
		if !ok || grant == nil {
			t.Fatalf("decoded payload missing refresh_token branch: %#v", payload)
		}
		if *grant != (token.RefreshTokenGrant{}) {
			t.Fatalf("decoded refresh grant = %#v, want zero-value object", *grant)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&token.TokenResponse{
			AccessToken: "access-cookie",
			TokenType:   "Bearer",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	form := url.Values{}
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-cookie"})

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post refresh grant: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
`
