package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

const jsonTagSkipHarness = `package tags_test

import (
 "encoding/json/v2"
 "testing"

 "example.com/tags/gen/storage"
 "github.com/stretchr/testify/require"
)

func TestGeneratedServiceSkip(t *testing.T) {
 private := "private"
 value := &storage.State{Secret: &private, Full: &private}
 data, err := json.Marshal(value)
 require.NoError(t, err)
 require.JSONEq(t, "{}", string(data))
}
`

func TestJSONTagSkipGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/tags", jsonTagSkipDSL, jsonTagSkipHarness)
}

func jsonTagSkipDSL() {
	state := dsl.Type("State", func() {
		dsl.Attribute("secret", dsl.String, func() {
			dsl.Meta("struct:tag:json:name", "-")
		})
		dsl.Attribute("full", dsl.String, func() {
			dsl.Meta("struct:tag:json", "-")
		})
	})
	dsl.Service("storage", func() {
		dsl.Method("store", func() {
			dsl.Payload(state)
		})
	})
}
