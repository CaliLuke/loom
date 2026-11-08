package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestTaggedFormInheritedDiscriminatorCollision(t *testing.T) {
	for _, renamed := range []bool{false, true} {
		err := expr.RunInvalidDSL(t, func() {
			base := Type("Base", func() {
				if renamed {
					Attribute("other", String, func() {
						Meta("struct:tag:form", "kind")
					})
				} else {
					Attribute("kind", String)
				}
			})
			child := Type("Child", func() {
				Extend(base)
				Attribute("value", String)
			})
			choice := Type("Choice", OneOf(child, String), func() {
				Meta("oneof:type:field", "kind")
			})
			Service("collision", func() {
				Method("create", func() {
					Payload(choice)
					HTTP(func() {
						POST("/")
						FormRequest()
					})
				})
			})
		})
		require.ErrorContains(t, err, "conflicts with discriminator")
	}
}
