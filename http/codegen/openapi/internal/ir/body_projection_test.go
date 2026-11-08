package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestBodyProjectionParentHint renders a form body that is the hinted Parent
// itself. The hint binds to its authored member before the form remaps names:
// a retained member renders its form name, and its explicit mapping target is
// projected into the same body context; a member the body removes suppresses
// the hint on that body only. The canonical JSON Parent keeps its hint.
func TestBodyProjectionParentHint(t *testing.T) {
	for _, tc := range []struct {
		name    string
		removed bool
	}{
		{name: "renamed"},
		{name: "removed", removed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("parent body", func() {})
				parent := dsl.Type("Parent", func() {
					dsl.Meta("openapi:discriminator:propertyName", "kind")
					dsl.Meta("openapi:discriminator:mapping", "cat=Cat")
					dsl.Attribute("kind", dsl.String, func() {
						dsl.Meta("struct:tag:form", "k")
					})
					dsl.Attribute("name", dsl.String)
					dsl.Required("kind")
				})
				dsl.Type("Cat", func() {
					dsl.Extend(parent)
					dsl.Meta("openapi:allOf:reference", "true")
					dsl.Attribute("lives", dsl.Int)
				})
				dsl.Service("pets", func() {
					dsl.Method("submit", func() {
						dsl.Payload(parent)
						dsl.HTTP(func() {
							dsl.POST("/submit")
							if tc.removed {
								dsl.Header("kind")
							}
							dsl.FormRequest()
						})
					})
					dsl.Method("show", func() {
						dsl.Result(parent)
						dsl.HTTP(func() {
							dsl.GET("/show")
						})
					})
				})
			})
			document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			canonical := representationSchema(t, document, document.Components.Schemas["Parent"])
			require.Equal(t, "kind", canonical.Discriminator.PropertyName, "the canonical declaration keeps its hint")
			form := document.Paths["/submit"].Operations["POST"].RequestBody.Value.Content["application/x-www-form-urlencoded"]
			body := representationSchema(t, document, form.Schema)
			if tc.removed {
				require.Nil(t, body.Discriminator, "a removed bound member suppresses the projected hint")
				require.NotContains(t, body.Properties, "k")
				return
			}
			require.NotNil(t, body.Discriminator)
			require.Equal(t, "k", body.Discriminator.PropertyName)
			require.Contains(t, body.Properties, "k")
			cat := representationSchema(t, document, &Schema{Ref: body.Discriminator.Mapping["cat"]})
			require.Len(t, cat.AllOf, 2)
			require.Contains(t, representationSchema(t, document, cat.AllOf[1]).Properties, "k", "the mapping target is projected into the body context")
		})
	}
}
