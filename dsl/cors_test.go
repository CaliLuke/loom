package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestCORSDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("cors_api", func() {
			HTTP(func() {
				CORS(func() {
					Origin("https://example.com", func() {
						Methods("GET", "POST")
						Headers("X-Api-Key")
						ExposeHeaders("X-Request-ID", "X-RateLimit-Remaining")
						MaxAge(600)
						Credentials()
					})
					OriginRegex(`https://.*\.example\.com`)
				})
			})
			JSONRPC(func() {
				CORS(func() {
					Origin("*")
				})
			})
		})

		Service("widgets", func() {
			HTTP(func() {
				CORS(func() {
					Origin("https://widgets.example.com", func() {
						Methods("GET")
					})
				})
			})
			Method("list", func() {
				Result(func() {
					Attribute("names", String)
				})
				HTTP(func() {
					GET("/widgets")
				})
			})
		})
	})

	apiCORS := root.API.HTTP.CORS
	require.NotNil(t, apiCORS)
	require.Len(t, apiCORS.Origins, 2)

	exact := apiCORS.Origins[0]
	require.Equal(t, "https://example.com", exact.Pattern)
	require.False(t, exact.Regex)
	require.Equal(t, []string{"GET", "POST"}, exact.Methods)
	require.Equal(t, []string{"X-Api-Key"}, exact.Headers)
	require.Equal(t, []string{"X-Request-ID", "X-RateLimit-Remaining"}, exact.Expose)
	require.Equal(t, 600, exact.MaxAge)
	require.True(t, exact.Credentials)

	regex := apiCORS.Origins[1]
	require.Equal(t, `https://.*\.example\.com`, regex.Pattern)
	require.True(t, regex.Regex)
	require.False(t, regex.Credentials)

	jsonrpcCORS := root.API.JSONRPC.CORS
	require.NotNil(t, jsonrpcCORS)
	require.Len(t, jsonrpcCORS.Origins, 1)
	require.Equal(t, "*", jsonrpcCORS.Origins[0].Pattern)

	svc := root.API.HTTP.Service("widgets")
	require.NotNil(t, svc)
	require.NotNil(t, svc.CORS)
	require.Len(t, svc.CORS.Origins, 1)
	require.Equal(t, "https://widgets.example.com", svc.CORS.Origins[0].Pattern)
	require.Equal(t, []string{"GET"}, svc.CORS.Origins[0].Methods)
}

func TestCORSDSLErrors(t *testing.T) {
	cases := []struct {
		name    string
		dsl     func()
		wantErr string
	}{
		{
			name: "cors without origins",
			dsl: func() {
				API("cors_api", func() {
					HTTP(func() {
						CORS(func() {})
					})
				})
				Service("svc", func() {
					Method("list", func() {
						HTTP(func() {
							GET("/list")
						})
					})
				})
			},
			wantErr: "CORS must define at least one origin",
		},
		{
			name: "credentials with wildcard origin",
			dsl: func() {
				API("cors_api", func() {
					HTTP(func() {
						CORS(func() {
							Origin("*", func() {
								Credentials()
							})
						})
					})
				})
				Service("svc", func() {
					Method("list", func() {
						HTTP(func() {
							GET("/list")
						})
					})
				})
			},
			wantErr: "CORS credentials are incompatible with wildcard origin",
		},
		{
			name: "invalid origin regex",
			dsl: func() {
				API("cors_api", func() {
					HTTP(func() {
						CORS(func() {
							OriginRegex("[")
						})
					})
				})
				Service("svc", func() {
					Method("list", func() {
						HTTP(func() {
							GET("/list")
						})
					})
				})
			},
			wantErr: `CORS origin regex "[" is invalid`,
		},
		{
			name: "negative max age",
			dsl: func() {
				API("cors_api", func() {
					HTTP(func() {
						CORS(func() {
							Origin("https://example.com", func() {
								MaxAge(-1)
							})
						})
					})
				})
				Service("svc", func() {
					Method("list", func() {
						HTTP(func() {
							GET("/list")
						})
					})
				})
			},
			wantErr: "CORS max age cannot be negative",
		},
		{
			name: "origin outside cors",
			dsl: func() {
				Origin("https://example.com")
			},
			wantErr: "invalid use of Origin",
		},
		{
			name: "cors in method http expression",
			dsl: func() {
				Service("svc", func() {
					Method("list", func() {
						HTTP(func() {
							GET("/list")
							CORS(func() {
								Origin("https://example.com")
							})
						})
					})
				})
			},
			wantErr: "invalid use of CORS",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
