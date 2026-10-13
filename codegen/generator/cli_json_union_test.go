package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

const cliJSONUntaggedHarness = `package cliunion_test

import (
 "testing"

 pc "example.com/cliunion/gen/http/plain/client"
 rc "example.com/cliunion/gen/jsonrpc/rpc/client"
 "github.com/stretchr/testify/require"
)

func TestUncheckedBranch(t *testing.T) {
 h, err := pc.BuildChoosePayload("{}")
 require.NoError(t, err)
 require.Equal(t, "Unchecked", string(h.Kind()))
 r, err := rc.BuildChoosePayload("{}")
 require.NoError(t, err)
 require.Equal(t, "Unchecked", string(r.Kind()))
}
`

func TestCLIJSONUntaggedInput(t *testing.T) {
	runDesignHarness(t, "example.com/cliunion", cliJSONUntaggedDSL, cliJSONUntaggedHarness)
}

func cliJSONUntaggedDSL() {
	dsl.API("cliunion", func() {
		dsl.JSONRPC(func() {
		})
	})
	unchecked := dsl.Type("Unchecked", func() {
		dsl.Attribute("flag", dsl.Boolean)
	})
	checked := dsl.Type("Checked", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	selection := dsl.Type("Selection", dsl.OneOf(unchecked, checked), func() {
		dsl.Untagged()
	})
	for _, name := range []string{"plain", "rpc"} {
		dsl.Service(name, func() {
			if name == "rpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			dsl.Method("choose", func() {
				dsl.Payload(selection)
				if name == "rpc" {
					dsl.JSONRPC(func() {
					})
				} else {
					dsl.HTTP(func() {
						dsl.POST("/choose")
					})
				}
			})
		})
	}
}
