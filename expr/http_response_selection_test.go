package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPResponseDefaultUniqueness(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responses func()
		want      string
	}{
		{"implicit", func() {
		}, ""},
		{"explicit", func() {
			Response(StatusOK)
		}, ""},
		{"tagged alternative", func() {
			Response(StatusOK)
			Response(StatusCreated, func() {
				Tag("kind", "created")
			})
		}, ""},
		{"two defaults", func() {
			Response(StatusOK)
			Response(StatusCreated)
		}, "exactly one response must define no Tag; found 2"},
		{"tag and two defaults", func() {
			Response(StatusOK)
			Response(StatusCreated, func() {
				Tag("kind", "created")
			})
			Response(StatusAccepted)
		}, "exactly one response must define no Tag; found 2"},
		{"no default", func() {
			Response(StatusCreated, func() {
				Tag("kind", "created")
			})
		}, "All responses define a Tag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				Service("selection", func() {
					Method("call", func() {
						Result(func() {
							Attribute("kind", String)
						})
						HTTP(func() {
							GET("/")
							tc.responses()
						})
					})
				})
			}
			if tc.want == "" {
				expr.RunDSL(t, design)
			} else {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, tc.want)
				require.ErrorContains(t, err, `service "selection" HTTP endpoint "call"`)
			}
		})
	}
}
