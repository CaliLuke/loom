package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestInformationalContinueAdmission(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selector  any
		field     bool
		value     int
		shadow    bool
		wantError string
	}{
		{name: "headerless exact", selector: 100},
		{name: "exact status field", selector: 100, field: true, value: 100},
		{name: "unshadowed class observation", selector: Status1XX, field: true, value: 100},
		{name: "exact excludes other value", selector: 100, field: true, value: 103, wantError: "constraints exclude every owned status"},
		{name: "exact shadows class", selector: Status1XX, field: true, value: 100, shadow: true, wantError: "constraints exclude every owned status"},
		{name: "upgrade still rejected", selector: 101, wantError: "Informational requires HTTP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				value := Type("ContinueMetadata", func() {
					if tc.field {
						Attribute("status", Int, func() {
							Enum(tc.value)
						})
						Required("status")
					}
				})
				exact := Type("Exact", func() {
				})
				Service("continue", func() {
					Method("read", func() {
						HTTP(func() {
							GET("/read")
							Informational("continue", tc.selector, value, func() {
								if tc.field {
									StatusField("status")
								}
							})
							if tc.shadow {
								Informational("exact", 100, exact)
							}
							Response(StatusNoContent)
						})
					})
				})
			}
			if tc.wantError != "" {
				err := expr.RunInvalidDSL(t, design)
				require.Contains(t, err.Error(), tc.wantError)
				return
			}
			root := expr.RunDSL(t, design)
			endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
			selected, ok := expr.SelectInformational(endpoint.InformationalStatusKeys(), 100)
			require.True(t, ok)
			require.Equal(t, endpoint.Informational[0].Status, selected)
			if selected.IsExact() {
				require.Empty(t, endpoint.Informational[0].StatusDomain(), "declaration does not authorize emission")
			}
		})
	}
}
