package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestHTTPWireNameValidTokens preserves the entire standard token alphabet.
func TestHTTPWireNameValidTokens(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, cookie := range []bool{false, true} {
			expr.RunDSL(t, wireBindingDSL(response, func() {
				name := "a:!#$%&'*+-.^_`|~0123456789AZaz"
				if cookie {
					Cookie(name)
				} else {
					Header(name)
				}
			}))
		}
	}
	// Query names have a URL-encoding contract, not a header-token contract.
	expr.RunDSL(t, wireBindingDSL(false, func() {
		Param("a:query name;with spaces")
	}))
}

// TestHTTPWireNameInheritedAndSession checks names after scope/credential resolution.
func TestHTTPWireNameInheritedAndSession(t *testing.T) {
	for _, source := range []string{"api", "service", "session field", "session browser"} {
		t.Run(source, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				if source == "api" {
					API("names", func() {
						HTTP(func() {
							Header("a:bad name")
						})
					})
				}
				key := APIKeySecurity("key")
				field := "a"
				if source == "session browser" {
					field = ""
				}
				session := SessionAuth("session", func() {
					CookieTransport(key, field, func() {
						CookieName("bad name")
					})
				})
				Service("names", func() {
					if source == "service" {
						HTTP(func() {
							Cookie("a:bad name")
						})
					}
					if source == "session field" || source == "session browser" {
						SessionSecurity(session)
					}
					Method("show", func() {
						Payload(func() {
							Attribute("a", String)
						})
						HTTP(func() {
							GET("/names")
						})
					})
				})
			})
			require.ErrorContains(t, err, "nonempty HTTP token")
		})
	}
}

// TestHTTPWireNamePrimitiveResponseCookie covers the non-object admission branch.
func TestHTTPWireNamePrimitiveResponseCookie(t *testing.T) {
	for _, failed := range []bool{false, true} {
		err := expr.RunInvalidDSL(t, func() {
			Service("names", func() {
				Method("show", func() {
					Result(String)
					Error("failed", String)
					HTTP(func() {
						GET("/names")
						mapping := func() {
							Cookie("bad name")
						}
						if failed {
							Response("failed", StatusBadRequest, mapping)
						} else {
							Response(StatusOK, mapping)
						}
					})
				})
			})
		})
		require.ErrorContains(t, err, "nonempty HTTP token")
	}
}

// TestHTTPWireNameColonQueries keeps distinct full suffixes in URL parameters.
func TestHTTPWireNameColonQueries(t *testing.T) {
	root := expr.RunDSL(t, wireBindingDSL(false, func() {
		Param("a:q:first")
		Param("b:q:second")
	}))
	params := root.API.HTTP.Services[0].HTTPEndpoints[0].Params
	require.Equal(t, "q:first", params.ElemName("a"))
	require.Equal(t, "q:second", params.ElemName("b"))
}

// TestHTTPWireNameColonMetadata preserves the complete suffix for gRPC consumers.
func TestHTTPWireNameColonMetadata(t *testing.T) {
	for _, placement := range []string{"request", "header", "trailer"} {
		t.Run(placement, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				metadataBindingDSL(placement, "text", "x:colon", "")
			})
			endpoint := root.API.GRPC.Service("binding").GRPCEndpoints[0]
			require.Equal(t, "x:colon", metadataBindingPlacement(endpoint, placement).ElemName("raw"))
		})
	}
}
