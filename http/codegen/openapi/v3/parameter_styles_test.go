package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParameterStylesBothTargets(t *testing.T) {
	for _, tc := range []struct {
		target, version string
		design          func()
	}{{"3.2", "3.2.0", testdata.ParameterStylesDSL}, {"3.1", "3.1.1", testdata.ParameterStyles31DSL}, {"3.2", "3.2.0", testdata.ParameterPropertyNamesDSL}} {
		t.Run(tc.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, tc.design, tc.target, tc.version)
			validateOpenAPIVersion(t, artifacts.JSON, tc.version)
			validateOpenAPIVersion(t, artifacts.YAML, tc.version)
		})
	}
}

func TestCookie31RejectsUnrestrictedCookieStyle(t *testing.T) {
	for _, name := range []string{"upload_session", "preferences"} {
		for _, style := range []string{"cookie", "form"} {
			t.Run(name+"/"+style, func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, func() {
					API("cookies", func() { Meta("openapi:version", "3.1") })
					Service("cookies", func() {
						Method("read", func() {
							Payload(func() { Attribute("value", String) })
							HTTP(func() {
								GET("/")
								Cookie("value:"+name, func() {
									if style == "form" || name == "preferences" {
										Meta("openapi:style", style)
									}
									if name == "preferences" {
										Meta("openapi:allowReserved", "true")
									}
								})
							})
						})
					})
				})
				_, err := openapiv3.Files(root)
				if style == "cookie" {
					require.ErrorContains(t, err, name)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}
