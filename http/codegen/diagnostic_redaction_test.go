package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/observability/diagnostics"
)

func TestDiagnosticCredentialWireSources(t *testing.T) {
	root := RunHTTPDSL(t, diagnosticCredentialDSL)
	service := root.API.HTTP.Services[0]
	for _, endpoint := range service.HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			prepared := transportir.BuildEndpoint(endpoint)
			require.Len(t, prepared.Security.Parameters, 1)
			parameter := prepared.Security.Parameters[0]
			switch endpoint.Name() {
			case "query":
				require.Equal(t, "query", parameter.In)
				require.Equal(t, "ticket", parameter.Name)
			case "header":
				require.Equal(t, "header", parameter.In)
				require.Equal(t, "X-Permit", parameter.Name)
			case "basic":
				require.Equal(t, "header", parameter.In)
				require.Equal(t, "Authorization", parameter.Name)
				for source, wire := range map[string]string{"login": "signin", "passcode": "proof"} {
					field := prepared.Request.Body.Find(source)
					require.NotNil(t, field)
					require.Equal(t, wire, expr.JSONFieldName(source, field))
				}
			case "body":
				require.Equal(t, "body", parameter.In)
				require.Equal(t, "credential", parameter.Name)
				field := prepared.Request.Body.Find("credential")
				require.NotNil(t, field)
				require.Equal(t, "permit", expr.JSONFieldName("credential", field))
			}
		})
	}
}

func TestDiagnosticCredentialProjection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		want   diagnostics.Fields
	}{
		{"mapped roles", diagnosticCredentialDSL, diagnostics.Fields{Headers: []string{"Authorization", "X-Permit"}, Query: []string{"ticket"}, JSON: []string{"permit", "proof", "signin"}, Form: []string{"credential", "login", "passcode"}}},
		{"streaming session", streamingSessionCookiePathDSL, diagnostics.Fields{Headers: []string{"Cookie", "Set-Cookie"}}},
		{"bidirectional session", bidirectionalSessionCookiePathDSL, diagnostics.Fields{Headers: []string{"Cookie", "Set-Cookie"}}},
		{"session", sessionCookieSecurityDSL, diagnostics.Fields{Headers: []string{"Authorization", "Cookie", "Set-Cookie"}}},
		{"API inherited session", apiLevelSessionCookieSecurityDSL, diagnostics.Fields{Headers: []string{"Authorization", "Cookie", "Set-Cookie"}}},
		{"duplicated Basic roles", diagnosticDuplicatedBasicDSL, diagnostics.Fields{Headers: []string{"Authorization", "X-Proof"}, Query: []string{"signin"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			services := CreateHTTPServices(root)
			data := services.Get(root.Services[0].Name)
			require.Equal(t, tc.want, serviceDiagnosticFields(data))
			endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
			before := endpoint.CredentialBindings()
			if len(before) > 0 {
				snapshot := endpoint.CredentialBindings()
				snapshot[0].Name = "mutated"
				snapshot[0].Sources[0] = "mutated"
				require.Equal(t, before, endpoint.CredentialBindings())
			}
		})
	}
}

func TestDiagnosticCredentialBodyAndScope(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want diagnostics.Fields
	}{
		{"scalar", diagnostics.Fields{Body: true}},
		{"path", diagnostics.Fields{Headers: []string{"Authorization"}, Path: true}},
		{"inherited", diagnostics.Fields{JSON: []string{"permit"}, Form: []string{"credential"}}},
		{"disabled", diagnostics.Fields{}},
		{"custom tags", diagnostics.Fields{JSON: []string{"Permit"}, Form: []string{"proof"}}},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			root := RunHTTPDSL(t, diagnosticBodyScopeDSL(tc.mode))
			services := CreateHTTPServices(root)
			require.Equal(t, tc.want, serviceDiagnosticFields(services.Get("mapped")))
			files := DiagnosticFiles(services, "http")
			if tc.mode == "disabled" {
				require.Empty(t, files)
			} else {
				require.Len(t, files, 1)
			}
		})
	}
}

func diagnosticBodyScopeDSL(mode string) func() {
	return func() {
		key := dsl.APIKeySecurity("mapped_key")
		if mode == "path" {
			key = dsl.BasicAuthSecurity("mapped_basic")
		}
		payload := dsl.Type("MappedPayload", func() {
			attribute := func() {
				if mode == "path" {
					dsl.Meta("security:username")
				} else if mode != "disabled" {
					dsl.Meta("security:apikey:mapped_key")
				}

				if mode != "custom tags" {
					dsl.Meta("struct:tag:json:name", "permit")
				}
				if mode == "custom tags" {
					dsl.Meta("struct:field:name", "Permit")
					dsl.Meta("struct:tag:form", "proof")
				}
			}
			dsl.Attribute("credential", dsl.String, attribute)
			if mode == "path" {
				dsl.Password("passcode", dsl.String)
				dsl.Required("passcode")
			}
			dsl.Required("credential")
		})
		dsl.Service("mapped", func() {
			dsl.Security(key)
			dsl.Method("show", func() {
				if mode == "disabled" {
					dsl.NoSecurity()
				}
				dsl.Payload(payload)
				dsl.HTTP(func() {
					if mode == "path" {
						dsl.GET("/{ticket}")
						dsl.Param("credential:ticket")
					} else {
						dsl.POST("/")
						if mode == "scalar" {
							dsl.Body("credential")
						} else {
							dsl.Body(func() {
								dsl.Attribute("credential")
							})
						}
					}
				})
			})
		})
	}
}

func diagnosticDuplicatedBasicDSL() {
	basic := dsl.BasicAuthSecurity("basic")
	dsl.Service("duplicated", func() {
		dsl.Method("show", func() {
			dsl.Security(basic)
			dsl.Payload(func() {
				dsl.Username("login", dsl.String)
				dsl.Password("passcode", dsl.String)
				dsl.Required("login", "passcode")
			})
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.Param("login:signin")
				dsl.Header("passcode:X-Proof")
			})
		})
	})
}

func diagnosticCredentialDSL() {
	key := dsl.APIKeySecurity("diagnostic_key")
	basic := dsl.BasicAuthSecurity("diagnostic_basic")
	dsl.API("diagnostic", func() {
		dsl.Server("diagnostic", func() {
			dsl.Services("diagnostic")
			dsl.Host("local", func() {
				dsl.URI("http://localhost")
			})
		})
	})
	dsl.Service("diagnostic", func() {
		dsl.Method("basic", func() {
			dsl.Security(basic)
			dsl.Payload(func() {
				dsl.Username("login", dsl.String, func() {
					dsl.Meta("struct:tag:json:name", "signin")
				})
				dsl.Password("passcode", dsl.String, func() {
					dsl.Meta("struct:tag:json:name", "proof")
				})
				dsl.Required("login", "passcode")
			})
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.POST("/basic")
				dsl.Body(func() {
					dsl.Attribute("login")
					dsl.Attribute("passcode")
				})
				dsl.Response(dsl.StatusOK)
			})
		})
		for _, location := range []string{"query", "header", "body"} {
			dsl.Method(location, func() {
				dsl.Security(key)
				dsl.Payload(func() {
					dsl.APIKey("diagnostic_key", "credential", dsl.String, func() {
						dsl.Meta("struct:tag:json:name", "permit")
					})
					dsl.Required("credential")
				})
				dsl.Result(dsl.Empty)
				dsl.HTTP(func() {
					dsl.POST("/" + location)
					switch location {
					case "query":
						dsl.Param("credential:ticket")
					case "header":
						dsl.Header("credential:X-Permit")
					case "body":
						dsl.Body(func() {
							dsl.Attribute("credential")
						})
					}
					dsl.Response(dsl.StatusOK)
				})
			})
		}
	})
}
