package expr_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPCORSDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("cors", func() {
			HTTP(func() {
				CORS(func() {
					Origin("https://app.example.com", func() {
						Methods("GET", "POST")
						Headers("Authorization", "Content-Type")
						ExposeHeaders("X-Request-Id")
						MaxAge(600)
						Credentials()
					})
					OriginRegex(`^https://preview-[^.]+\.example\.com$`)
				})
			})
		})
		Service("svc", func() {
			Method("show", func() {
				HTTP(func() {
					GET("/")
				})
			})
		})
	})

	cors := root.API.HTTP.CORS
	if cors == nil || len(cors.Origins) != 2 {
		t.Fatalf("expected two CORS origins, got %#v", cors)
	}
	if got := cors.Origins[0].Headers; len(got) != 2 || got[0] != "Authorization" || got[1] != "Content-Type" {
		t.Fatalf("unexpected CORS headers: %#v", got)
	}
	if !cors.Origins[1].Regex {
		t.Fatalf("expected regex origin")
	}
}

func TestHTTPCORSValidation(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "wildcard credentials",
			dsl: func() {
				API("cors", func() {
					HTTP(func() {
						CORS(func() {
							Origin("*", func() {
								Credentials()
							})
						})
					})
				})
			},
			want: "CORS credentials are incompatible with wildcard origin",
		},
		{
			name: "invalid regex",
			dsl: func() {
				API("cors", func() {
					HTTP(func() {
						CORS(func() {
							OriginRegex("[")
						})
					})
				})
			},
			want: "CORS origin regex",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
