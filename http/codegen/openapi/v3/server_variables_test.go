package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestServerVariableEnumsAreIndependent(t *testing.T) {
	for _, version := range []struct{ target, full string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, func() {
				region := Type("Region", String, func() {
					Enum("us", "eu", "ap")
				})
				API("variables", func() {
					Server("api", func() {
						Services("svc")
						Host("public", func() {
							URI("https://{region}.example.{zone}/{stage}")
							Variable("region", region, func() {
								Enum("us", "eu")
								Default("us")
							})
							Variable("zone", String, func() {
								Enum("com", "net")
								Default("com")
							})
							Variable("stage", String, func() {
								Default("prod")
							})
						})
					})
				})
				Service("svc", func() {
					Method("get", func() {
						HTTP(func() {
							GET("/")
						})
					})
				})
			}, version.target, version.full)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			servers := requireSlice(t, spec["servers"], "servers")
			variables := requireMap(t, requireMap(t, servers[0], "server")["variables"], "variables")
			for name, want := range map[string][]any{"region": {"us", "eu"}, "zone": {"com", "net"}, "stage": nil} {
				variable := requireMap(t, variables[name], name)
				if want == nil {
					require.NotContains(t, variable, "enum")
				} else {
					require.Equal(t, want, variable["enum"])
				}
			}
			require.Equal(t, "com", requireMap(t, variables["zone"], "zone")["default"])
			lintRenderedContract(t, spec)
		})
	}
}
