package openapiv3_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorHintBothTargets(t *testing.T) {
	for _, target := range []string{"3.1", "3.2"} {
		for _, fallback := range []bool{false, true} {
			t.Run(target+"/fallback="+map[bool]string{false: "false", true: "true"}[fallback], func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, discriminatorHintTargetDSL(fallback))
				root.API.Meta["openapi:version"] = []string{target}
				files, err := openapiv3.Files(root)
				if target == "3.2" && !fallback {
					require.ErrorContains(t, err, "defaultMapping")
					return
				}
				require.NoError(t, err)
				for _, file := range files {
					buffer := renderSection(t, file.AllSections()[0])
					raw := buffer.Bytes()
					version := "3.2.0"
					if target == "3.1" {
						version = "3.1.1"
					}
					validateOpenAPIVersion(t, raw, version)
					if file.Path[len(file.Path)-5:] != ".json" {
						continue
					}
					spec := decodeOpenAPIJSON(t, raw)
					media := requireResponseMediaType(t, requireOperation(t, spec, "/hint", "get"), "application/json")
					choice := resolveCardinalitySchema(t, spec, requireMap(t, media["schema"], "hint response schema"))
					require.Empty(t, choice["required"])
					hint := requireMap(t, choice["discriminator"], "hint")
					require.Equal(t, "kind", hint["propertyName"])
					require.Equal(t, map[string]any{"cat": "#/components/schemas/HintCat", "kitty": "#/components/schemas/HintCat"}, hint["mapping"])
					require.Equal(t, map[string]any{"v": float64(1)}, hint["x-audit"])
					if target == "3.1" {
						require.NotContains(t, hint, "defaultMapping")
					} else {
						require.Equal(t, "#/components/schemas/HintCat", hint["defaultMapping"])
					}
				}
			})
		}
	}
}

func discriminatorHintTargetDSL(fallback bool) func() {
	return func() {
		API("hints", func() {
			Meta("openapi:example", "false")
		})
		cat := Type("HintCat", func() {
			Attribute("kind", String, func() {
				Enum("cat")
			})
			Required("kind")
		})
		dog := Type("HintDog", func() {
			Attribute("kind", String, func() {
				Enum("dog")
			})
		})
		choice := Type("HintChoice", OneOf(func() {
			Untagged()
			Attribute("Cat", cat, func() {
				Meta("openapi:discriminator:mapping", "cat", "kitty")
			})
			Attribute("Dog", dog)
		}), func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:extension:x-audit", `{"v":1}`)
			if fallback {
				Meta("openapi:discriminator:defaultMapping", "HintCat")
			}
		})
		Service("hints", func() {
			Method("show", func() {
				Result(choice)
				HTTP(func() {
					GET("/hint")
					Response(StatusOK)
				})
			})
		})
	}
}

func TestDiscriminatorHintFingerprintIsolation(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		discriminatorHintTargetDSL(true)()
		base := expr.Root.UserType("HintChoice")
		Service("other", func() {
			Method("show", func() {
				Result(base, func() {
					Meta("openapi:discriminator:extension:x-audit", `{"v":2}`)
				})
				HTTP(func() {
					GET("/other")
					Response(StatusOK)
				})
			})
		})
	})
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	// Both occurrence hints must survive component allocation and reuse.
	buffer := renderSection(t, files[0].AllSections()[0])
	raw := buffer.String()
	require.Contains(t, raw, `"v": 1`)
	require.Contains(t, raw, `"v": 2`)
}
