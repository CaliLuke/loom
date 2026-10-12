package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestObjectDefaultAdmission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value map[string]any
	}{
		{name: "closed unknown member", value: map[string]any{"x": 1, "zzz": 1}},
		{name: "invalid member", value: map[string]any{"x": "wrong"}},
		{name: "missing required member", value: map[string]any{}},
		{name: "violated member constraint", value: map[string]any{"x": -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Type("Container", func() {
					Attribute("value", func() {
						if tc.name == "closed unknown member" {
							Meta("openapi:additionalProperties", "false")
						}
						Attribute("x", Int, func() {
							Minimum(0)
						})
						Required("x")
						Default(tc.value)
					})
				})
			})
			require.Error(t, err)
		})
	}
}

// TestObjectDefaultOpenAdmission records the source contract separately from
// representability in a generated Go struct: open objects retain extra members.
func TestObjectDefaultOpenAdmission(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Container", func() {
			Attribute("value", func() {
				Attribute("x", Int)
				Required("x")
				Default(map[string]any{"x": 1, "zzz": 1})
			})
		})
	})
	require.Equal(t, map[string]any{"x": 1, "zzz": 1}, root.Types[0].Attribute().Find("value").DefaultValue)
}
