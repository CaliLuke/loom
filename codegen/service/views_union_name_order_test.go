package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestViewUnionReferencesUseDeclaredNames(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		peer := dsl.Type("Peer", func() {
			dsl.OneOf("block", func() {
				dsl.Attribute("text", dsl.String)
			})
		})
		result := dsl.ResultType("application/vnd.rt", "RT", func() {
			dsl.Attribute("peer", peer)
			dsl.OneOf("block", func() {
				dsl.Attribute("number", dsl.Int)
			})
			dsl.View("default", func() {
				dsl.Attribute("peer")
				dsl.Attribute("block")
			})
		})
		dsl.Service("svc", func() {
			dsl.Method("get", func() {
				dsl.Result(result)
			})
		})
	})
	services := NewServicesData(root)
	views := renderViewsFile(t, root, services)
	require.Contains(t, views, "Block *Block2")
	service := renderServiceFile(t, root, services)
	require.Contains(t, generatedFunction(t, service, "ProjectRT"), "svcviews.Block2")
	require.Contains(t, generatedFunction(t, service, "NewRTFromRTView"), "var u Block2")
}
