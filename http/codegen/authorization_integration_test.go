package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationIntegration(t *testing.T) {
	root := RunHTTPDSL(t, authorizationIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/accessit", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authorization_test.go"), []byte(authorizationHarness+authorizationAdversarialHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func authorizationIntegrationDSL() {
	ref := Type("DocumentRef", func() {
		Attribute("id", String, func() {
			MinLength(1)
		})
		Required("id")
	})
	edit := Authorization("document.edit", ref)
	audit := Authorization("document.audit", Empty)
	Interceptor("AccessProbe")
	readRef := Type("ReadRef", func() {
		Attribute("id", String)
	})
	jwt := JWTSecurity("jwt")
	Service("documents", func() {
		StrictAuthorization()
		Security(jwt)
		HTTP(func() {
			AuthErrorResponses()
		})
		Method("update", func() {
			ServerInterceptor("AccessProbe")
			Payload(func() {
				Token("token", String)
				Attribute("document_id", String, func() {
					MinLength(1)
				})
				Required("document_id", "token")
			})
			Authorize(edit, func() {
				Bind("id", "document_id")
			})
			Authorize(audit)
			Result(String)
			HTTP(func() {
				POST("/documents")
				Response(StatusOK)
			})
		})
		Method("dispatch", func() {
			Payload(func() {
				Token("token", String)
				Attribute("id", String)
				Attribute("action", String, func() {
					Enum("read", "write")
				})
				Required("token", "id")
			})
			AuthorizeBy("action", func() {
				AuthorizationCase("read", func() {
					NoAccessCheck("Returns the caller's own profile")
				})
				AuthorizationCase("write", func() {
					Authorize(edit, func() {
						Bind("id", "id")
					})
				})
			})
			Result(String)
			HTTP(func() {
				POST("/dispatch")
				Response(StatusOK)
			})
		})

		Method("variant", func() {
			Payload(func() {
				Token("token", String)
				OneOf("value", func() {
					Attribute("edit", ref)
					Attribute("read", readRef)
				})
				Required("token", "value")
			})
			AuthorizeBy("value", func() {
				AuthorizationCase("edit", func() {
					Authorize(edit, func() {
						Bind("id", "value.id")
					})
				})
				AuthorizationCase("read", func() {
					NoAccessCheck("Own profile")
				})
			})
			Result(String)
			HTTP(func() {
				POST("/variant")
				Response(StatusOK)
			})
		})
		Method("health", func() {
			NoSecurity()
			NoAccessCheck("Public health probe")
			Result(String)
			HTTP(func() {
				GET("/health")
				Response(StatusOK)
			})
		})
	})
}

const authorizationHarness = `package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/accessit/gen/documents"
	docsvr "example.com/accessit/gen/http/documents/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/stretchr/testify/require"
)

type actorKey struct {
}
type documentService struct {
	calls int
}

func (s *documentService) Update(_ context.Context, p *documents.UpdatePayload) (string, error) {
	s.calls++
	return p.DocumentID, nil
}
func (s *documentService) Dispatch(_ context.Context, p *documents.DispatchPayload) (string, error) {
	s.calls++
	return p.ID, nil
}
func (s *documentService) Variant(_ context.Context, _ *documents.VariantPayload) (string, error) {
	s.calls++
	return "ok", nil
}
func (*documentService) Health(context.Context) (string, error) {
	return "healthy", nil
}
func (*documentService) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	if token != "valid" {
		return nil, loom.PermanentError("unauthorized", "Authentication required")
	}
	return context.WithValue(ctx, actorKey{}, true), nil
}

type accessEvaluator struct {
	audits      int
	auditDenied bool
	denied      bool
	err         error
	checks      int
}

func (a *accessEvaluator) AuthorizeDocumentAudit(context.Context) error {
	a.audits++
	if a.auditDenied {
		return loom.PermanentError("forbidden", "Audit access denied")
	}
	return nil
}

type documentInterceptors struct {
	apply func(context.Context, any, loom.Endpoint) (any, error)
}

func (i *documentInterceptors) AccessProbe(ctx context.Context, info *documents.AccessProbeInfo, next loom.Endpoint) (any, error) {
	if i.apply != nil {
		return i.apply(ctx, info.RawPayload(), next)
	}
	return next(ctx, info.RawPayload())
}
func (a *accessEvaluator) AuthorizeDocumentEdit(ctx context.Context, input *documents.DocumentRef) error {
	a.checks++
	if ctx.Value(actorKey{}) != true {
		return errors.New("authorization preceded authentication")
	}
	if a.err != nil {
		return a.err
	}
	if a.denied || input.ID != "allowed" {
		return loom.PermanentError("forbidden", "Access denied")
	}
	return nil
}

func TestAuthorizationHTTP(t *testing.T) {
	cases := []struct {
		name, token, body string
		status, calls     int
	}{
		{"allowed", "valid", "allowed", 200, 1},
		{"denied", "valid", "denied", 403, 0},
		{"unauthenticated", "bad", "allowed", 401, 0},
		{"invalid", "valid", "", 400, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &documentService{}
			a := &accessEvaluator{}
			endpoints := documents.NewEndpoints(s, a, &documentInterceptors{})
			mux := loomhttp.NewMuxer()
			server := docsvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			docsvr.Mount(mux, server)
			req := httptest.NewRequest(http.MethodPost, "/documents", strings.NewReader("{\"document_id\":\""+tc.body+"\"}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tc.token)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.Equal(t, tc.calls, s.calls)
			if tc.status == 403 {
				require.Contains(t, response.Body.String(), "forbidden")
			}
		})
	}
}

func TestAuthorizationDirectAndMiddleware(t *testing.T) {
	s := &documentService{}
	a := &accessEvaluator{}
	e := documents.NewEndpoints(s, a, &documentInterceptors{})
	ctx := context.Background()
	_, err := e.Update(ctx, &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
	require.NoError(t, err)
	require.Equal(t, 1, s.calls)
	for _, invalid := range []any{nil, (*documents.UpdatePayload)(nil), "wrong type", &documents.UpdatePayload{Token: "valid"}} {
		_, err = e.Update(ctx, invalid)
		require.Error(t, err)
	}
	a.denied = true
	e.Use(func(_ loom.Endpoint) loom.Endpoint {
		return func(context.Context, any) (any, error) {
			return "cached", nil
		}
	})
	_, err = e.Update(ctx, &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
	require.Error(t, err)
	require.Equal(t, 1, s.calls)
	a.denied = false
	e = documents.NewEndpoints(s, a, &documentInterceptors{})
	e.Use(func(next loom.Endpoint) loom.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			req.(*documents.UpdatePayload).DocumentID = "denied"
			return next(ctx, req)
		}
	})
	_, err = e.Update(ctx, &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
	require.Error(t, err)
	require.Equal(t, 1, s.calls)
}

func TestAuthorizationUnion(t *testing.T) {
	s := &documentService{}
	a := &accessEvaluator{}
	e := documents.NewEndpoints(s, a, &documentInterceptors{})
	for _, tc := range []struct {
		id      string
		allowed bool
	}{{"allowed", true}, {"denied", false}, {"", false}} {
		_, err := e.Variant(context.Background(), &documents.VariantPayload{Token: "valid", Value: documents.NewValueEdit(&documents.DocumentRef{ID: tc.id})})
		require.Equal(t, tc.allowed, err == nil)
	}
	_, err := e.Variant(context.Background(), &documents.VariantPayload{Token: "valid", Value: documents.NewValueRead(nil)})
	require.Error(t, err)
	_, err = e.Variant(context.Background(), &documents.VariantPayload{Token: "valid"})
	require.Error(t, err)
}

func TestAuthorizationVariantsAndDependencies(t *testing.T) {
	s := &documentService{}
	a := &accessEvaluator{}
	e := documents.NewEndpoints(s, a, &documentInterceptors{})
	for _, tc := range []struct {
		action, id, token string
		allowed           bool
	}{
		{"read", "denied", "valid", true},
		{"write", "allowed", "valid", true},
		{"write", "denied", "valid", false},
		{"unknown", "allowed", "valid", false},
		{"read", "allowed", "bad", false},
	} {
		_, err := e.Dispatch(context.Background(), &documents.DispatchPayload{Action: &tc.action, ID: tc.id, Token: tc.token})
		require.Equal(t, tc.allowed, err == nil)
	}
	require.Panics(t, func() {
		documents.NewEndpoints(s, nil, &documentInterceptors{})
	})
	var missing *accessEvaluator
	require.Panics(t, func() {
		documents.NewEndpoints(s, missing, &documentInterceptors{})
	})
	a.err = errors.New("policy storage unavailable")
	_, err := documents.NewUpdateEndpoint(s, a, s.JWTAuth)(context.Background(), &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
	require.ErrorIs(t, err, a.err)
}
`
