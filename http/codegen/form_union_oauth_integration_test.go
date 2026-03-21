package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cg "goa.design/goa/v3/codegen"
	servicecodegen "goa.design/goa/v3/codegen/service"
	. "goa.design/goa/v3/dsl"
)

func TestFormRequestUnionOAuthIntegration(t *testing.T) {
	const (
		modulePath = "example.com/formunionit"
		genpkg     = modulePath + "/gen"
	)

	root := RunHTTPDSL(t, oauthFormRequestUnionDSL)
	serviceData := servicecodegen.NewServicesData(root)
	httpData := CreateHTTPServices(root)

	files := servicecodegen.Files(genpkg, root.Services[0], serviceData, make(map[string][]string))
	files = append(files, PathFiles(httpData)...)
	files = append(files, ClientTypeFiles(genpkg, httpData)...)
	files = append(files, ClientFiles(genpkg, httpData)...)
	files = append(files, ServerTypeFiles(genpkg, httpData)...)
	for _, svc := range httpData.Expressions.Services {
		if file := ServerEncodeDecodeFile(genpkg, svc, httpData); file != nil {
			files = append(files, file)
		}
	}

	dir := t.TempDir()
	renderGeneratedFiles(t, dir, files)

	repoRoot := checkoutPinnedGoaModule(t, dir)

	goMod := fmt.Sprintf(`module %s

go 1.25.0

require goa.design/goa/v3 v3.0.0

replace goa.design/goa/v3 => %s
`, modulePath, repoRoot)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(oauthIntegrationHarness), 0644); err != nil {
		t.Fatalf("write integration harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func checkoutPinnedGoaModule(t *testing.T, parentDir string) string {
	t.Helper()

	if local := configuredLocalGoaModulePath(); local != "" {
		return local
	}

	commit := strings.TrimSpace(runCommand(t, "", "git", "rev-parse", "HEAD"))
	remote := strings.TrimSpace(resolveGitRemoteURL(t))
	dest := filepath.Join(parentDir, "goa-pinned")

	runCommand(t, "", "git", "init", dest)
	runCommand(t, dest, "git", "remote", "add", "origin", remote)
	runCommand(t, dest, "git", "fetch", "--depth", "1", "origin", commit)
	runCommand(t, dest, "git", "checkout", "--detach", "FETCH_HEAD")

	return dest
}

func configuredLocalGoaModulePath() string {
	if repo := os.Getenv("GOA_REPO"); repo != "" {
		return repo
	}
	root, err := runCommandAllowFailure("", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	modeFile := filepath.Join(strings.TrimSpace(root), "jsonrpc", "integration_tests", ".goa_source_mode")
	data, err := os.ReadFile(modeFile)
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 || fields[0] != "local" {
		return ""
	}
	return fields[1]
}

func resolveGitRemoteURL(t *testing.T) string {
	t.Helper()

	for _, name := range []string{"fork", "origin"} {
		if out, err := runCommandAllowFailure("", "git", "remote", "get-url", name); err == nil {
			url := strings.TrimSpace(out)
			if url != "" {
				return url
			}
		}
	}
	t.Fatal("could not resolve git remote URL from fork or origin")
	return ""
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

	out, err := runCommandAllowFailure(dir, name, args...)
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return out
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

	goahttp "goa.design/goa/v3/http"
	"golang.org/x/oauth2"

	token "example.com/formunionit/gen/token"
	tokenclient "example.com/formunionit/gen/http/token/client"
	tokenserver "example.com/formunionit/gen/http/token/server"
)

func TestXOAuth2AuthCodeRequestUsesFlatFormFields(t *testing.T) {
	mux := goahttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, goahttp.RequestDecoder)

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
	mux := goahttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, goahttp.RequestDecoder)

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
	client := tokenclient.NewClient(u.Scheme, u.Host, srv.Client(), goahttp.RequestEncoder, goahttp.ResponseDecoder, false)
	endpoint := client.Exchange()

	result, err := endpoint(context.Background(), &token.Grant{
		Grant: token.NewGrant2RefreshToken(&token.RefreshTokenGrant{}),
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
	mux := goahttp.NewMuxer()
	decode := tokenserver.DecodeExchangeRequest(mux, goahttp.RequestDecoder)

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
