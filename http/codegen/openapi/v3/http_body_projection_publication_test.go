package openapiv3_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/stretchr/testify/require"
)

func TestHTTPBodyContributorPublishesCompletedJSONChildren(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("JSON publication", func() {
			Meta("openapi:version", "3.1")
			Meta("openapi:example", "false")
		})
		parent := Type("Parent", func() {
			Attribute("kind", String)
			Required("kind")
			Meta("openapi:discriminator:propertyName", "kind")
		})
		Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("thumb", Bytes)
		})
		Service("publication", func() {
			Method("parent", func() {
				Result(parent)
				HTTP(func() {
					GET("/parent")
				})
			})
		})
	})
	spec := bodyProjectionSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	operation := requireOperation(t, spec, "/parent", "get")
	responses := requireMap(t, operation["responses"], "responses")
	content := requireMap(t, requireMap(t, responses["200"], "response")["content"], "content")
	parent := requireMap(t, requireMap(t, content["application/json"], "JSON")["schema"], "Parent reference")
	parentRef := requireString(t, parent["$ref"], "Parent reference")
	children := 0
	for name, raw := range schemas {
		schema := requireMap(t, raw, name)
		composition, ok := schema["allOf"].([]any)
		if !ok || len(composition) != 2 {
			continue
		}
		base := requireMap(t, composition[0], name+" base")
		if base["$ref"] != parentRef {
			continue
		}
		properties := requireMap(t, requireMap(t, composition[1], name+" own schema")["properties"], name+" own properties")
		thumb, found := properties["thumb"]
		if !found {
			continue
		}
		children++
		field := requireMap(t, thumb, name+" thumb")
		require.Equal(t, "base64", field["contentEncoding"], "published JSON child must use completed byte projection")
		require.NotEqual(t, "binary", field["format"])
	}
	require.Positive(t, children, "hinted parent must publish its child")
}
