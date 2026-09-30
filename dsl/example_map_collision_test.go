package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestExampleRejectsCanonicalJSONMapKeyCollision(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("svc", func() {
			Method("send", func() {
				Payload(MapOf(Any, String), func() {
					Example(map[any]any{1: "integer", "1": "text"})
				})
			})
		})
	})
	require.ErrorContains(t, err,
		`service "svc" method "send": payload - example map keys collide as JSON object member "1"`)
}
