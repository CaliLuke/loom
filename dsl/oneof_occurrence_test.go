package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedMarksOneOfTypeConstructor(t *testing.T) {
	root := expr.RunDSL(t, func() {
		ok := Type("UntaggedOK", func() {
			Attribute("data", String)
			Required("data")
		})
		failure := Type("UntaggedFailure", func() {
			Attribute("error", String)
			Required("error")
		})
		Service("untagged", func() {
			Method("show", func() {
				Result(OneOf(ok, failure), func() {
					Untagged()
				})
			})
		})
	})

	union := expr.AsUnion(root.Services[0].Methods[0].Result.Type)
	require.NotNil(t, union)
	require.True(t, union.Untagged)
}

func TestUntaggedDoesNotMutateReusedOneOfTypeConstructor(t *testing.T) {
	var tagged, untagged *expr.Union
	expr.RunDSL(t, func() {
		start := Type("IsolatedStart", func() {
			Attribute("start", String)
		})
		stop := Type("IsolatedStop", func() {
			Attribute("stop", String)
		})
		base := OneOf(start, stop)
		Service("isolated", func() {
			Method("tagged", func() {
				Result(base)
			})
			Method("untagged", func() {
				Result(base, func() {
					Untagged()
				})
			})
		})
	})

	tagged = expr.AsUnion(expr.Root.Services[0].Methods[0].Result.Type)
	untagged = expr.AsUnion(expr.Root.Services[0].Methods[1].Result.Type)
	require.NotSame(t, tagged, untagged)
	require.False(t, tagged.Untagged)
	require.True(t, untagged.Untagged)
}
