package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestJSONRPCWebSocketEffectiveSecurityAdmission(t *testing.T) {
	for _, kind := range []string{"jwt", "key", "oauth", "basic", "session", "cookie", "alternative", "optional"} {
		for _, shape := range []string{"server", "client", "bidi"} {
			t.Run(kind+"/"+shape, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, testdata.JSONRPCWebSocketSecurityDSL(kind, shape, "", "service", "ws"))
				require.Error(t, err)
				require.ErrorContains(t, err, "WebSocket")
			})
		}
	}
	for _, scope := range []string{"api", "method"} {
		t.Run("scope/"+scope, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, testdata.JSONRPCWebSocketSecurityDSL("jwt", "server", "", scope, "ws"))
			require.ErrorContains(t, err, "WebSocket")
		})
	}
}

func TestJSONRPCWebSocketExplicitSecurityMappingRejection(t *testing.T) {
	for _, location := range []string{"header", "cookie", "query"} {
		t.Run(location, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, testdata.JSONRPCWebSocketSecurityDSL("jwt", "server", location, "service", "ws"))
			require.ErrorContains(t, err, "WebSocket")
		})
	}
}

func TestJSONRPCWebSocketSecurityAdmissionControls(t *testing.T) {
	for _, tc := range []struct {
		name, kind, shape, location, scope, transport string
	}{
		{"unsecured", "", "server", "", "service", "ws"},
		{"NoSecurity overrides session", "session", "server", "", "none", "ws"},
		{"anonymous overrides session", "session", "server", "", "anonymous", "ws"},
		{"anonymous", "anonymous", "bidi", "", "method", "ws"},
		{"POST JWT", "jwt", "server", "", "service", "post"},
		{"SSE JWT", "jwt", "server", "", "service", "sse"},
		{"HTTP WebSocket JWT", "jwt", "server", "", "service", "http-ws"},
		{"HTTP WebSocket Basic", "basic", "server", "", "service", "http-ws"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, testdata.JSONRPCWebSocketSecurityDSL(tc.kind, tc.shape, tc.location, tc.scope, tc.transport))
			if tc.scope == "none" || tc.scope == "anonymous" {
				method := root.Service("secured").Method("watch")
				require.Empty(t, method.EffectiveSessionAuths())
			}
		})
	}
}

func TestJSONRPCWebSocketDisplacedSecurityAdmission(t *testing.T) {
	for _, kind := range []string{"jwt", "key", "oauth"} {
		for _, shape := range []string{"bidi", "client", "bidi-retagged"} {
			t.Run(kind+"/"+shape, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, testdata.JSONRPCWebSocketSecurityDSL(kind, shape, "body", "service", "ws"))
				require.ErrorContains(t, err, "credential")
			})
		}
	}
}

func TestJSONRPCWebSocketMessageCredentialControls(t *testing.T) {
	for _, kind := range []string{"jwt", "key", "oauth"} {
		for _, shape := range []string{"server", "bidi-same"} {
			t.Run(kind+"/"+shape, func(t *testing.T) {
				root := expr.RunDSL(t, testdata.JSONRPCWebSocketSecurityDSL(kind, shape, "body", "service", "ws"))
				method := root.Service("secured").Method("watch")
				require.Len(t, method.EffectiveRequirements(), 1)
				require.Len(t, method.EffectiveRequirements()[0].Schemes, 1)
				require.NotNil(t, method.Payload.Find("token"))
			})
		}
	}
}

func TestJSONRPCWebSocketCredentialRoleSurvival(t *testing.T) {
	for _, mode := range []string{"renamed", "wrong-type", "nullable", "custom", "partial-and", "partial-or", "shared-lost"} {
		t.Run(mode, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, testdata.JSONRPCWebSocketCredentialRolesDSL(mode))
			require.ErrorContains(t, err, "credential")
		})
	}
	t.Run("named strings and named request", func(t *testing.T) {
		expr.RunDSL(t, testdata.JSONRPCWebSocketCredentialRolesDSL("named"))
	})
}
