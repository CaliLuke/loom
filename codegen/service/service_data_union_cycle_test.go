package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestCollectServiceUnionsOnRecursiveInlineObjectCycle(t *testing.T) {
	cases := []struct {
		Name   string
		Method func(ut expr.DataType)
	}{
		{"payload-and-result", func(ut expr.DataType) {
			dsl.Payload(ut)
			dsl.Result(ut)
		}},
		{"payload", func(ut expr.DataType) {
			dsl.Payload(ut)
		}},
		{"result", func(ut expr.DataType) {
			dsl.Result(ut)
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				node := dsl.Type("CycleNode", func() {
					dsl.Field(1, "holder", func() {
						dsl.OneOf("choice", func() {
							dsl.Field(1, "br", func() {
								dsl.Field(1, "branches", dsl.ArrayOf("CycleNode"))
							})
						})
					})
				})
				dsl.Service("CycleService", func() {
					dsl.Method("Walk", func() {
						c.Method(node)
					})
				})
			})

			data := NewServicesData(root).Get("CycleService")
			names := make([]string, 0, len(data.unions))
			for _, union := range data.unions {
				names = append(names, union.Name)
			}
			require.Equal(t, []string{"Choice"}, names)
			typeNames := make([]string, 0, len(data.userTypes))
			for _, userType := range data.userTypes {
				typeNames = append(typeNames, userType.Name)
			}
			require.ElementsMatch(t, []string{"CycleNode", "choiceBr"}, typeNames)
		})
	}
}
