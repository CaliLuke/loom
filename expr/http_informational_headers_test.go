package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestInformationalHeaderAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selector any
		header   bool
		status   bool
		want     string
	}{
		{"empty exact", StatusEarlyHints, false, false, ""},
		{"declared header without carrier", StatusEarlyHints, true, false, "response defines headers but result is empty"},
		{"class without status", Status1XX, false, false, "non-exact or status-selected responses require StatusField"},
		{"class with status only", Status1XX, false, true, ""},
		{"automatic continue", StatusContinue, false, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				hints := Type("Hints", func() {
					if tc.status {
						Attribute("status", Int)
						Required("status")
					}
				})
				Service("hints", func() {
					Method("read", func() {
						HTTP(func() {
							GET("/")
							Informational("Hints", tc.selector, hints, func() {
								Description("no response body")
								if tc.header {
									Header("missing:X-Missing", String)
								}
								if tc.status {
									StatusField("status")
								}
							})
							Response(StatusNoContent)
						})
					})
				})
			}
			if tc.want != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.want)
				return
			}
			root := expr.RunDSL(t, design)
			e := root.API.HTTP.Service("hints").Endpoint("read")
			require.Len(t, e.Informational, 1)
			info := e.Informational[0]
			require.True(t, info.Headers.IsEmpty())
			require.Nil(t, info.Body)
			require.Equal(t, "no response body", info.Description)
			require.Equal(t, expr.Empty, e.Responses[0].Body.Type)
			require.Equal(t, StatusNoContent, e.Responses[0].Status.Code)
			if tc.status {
				require.Equal(t, "status", info.StatusField)
				require.ElementsMatch(t, []string{"status"}, info.RequiredFields(""))
			} else {
				require.Empty(t, info.StatusField)
				require.Empty(t, info.RequiredFields(""))
				require.Empty(t, *expr.AsObject(info.Value.Type))
				require.Equal(t, tc.selector, info.Status.Code)
			}
		})
	}
}
