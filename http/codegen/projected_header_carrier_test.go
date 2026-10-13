package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	base "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestProjectedHeaderCarrierBinding retains the actual declared field carrier.
func TestProjectedHeaderCarrierBinding(t *testing.T) {
	array := &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
	alias := &expr.UserTypeExpr{TypeName: "Tags", AttributeExpr: &expr.AttributeExpr{Type: array}}
	for _, tc := range []struct {
		name     string
		dataType expr.DataType
		viewed   bool
		pointer  bool
	}{
		{"named array view", alias, true, false},
		{"plain array view", array, true, false},
		{"non-view alias", alias, false, false},
		{"view scalar", expr.String, true, true},
		{"service optional scalar", expr.String, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: tc.dataType}
			parent := &expr.AttributeExpr{Type: &expr.Object{&expr.NamedAttributeExpr{Name: "tags", Attribute: field}}}
			context := base.NewAttributeContext(tc.viewed, false, true, "", base.NewNameScope())
			name, dataType, pointer := transportFieldBinding("tags", field, parent, context)
			require.Equal(t, "Tags", name)
			require.Equal(t, tc.dataType, dataType)
			require.Equal(t, tc.pointer, pointer)
		})
	}
}

// TestProjectedHeaderCarrierWire checks the original named-view header through HTTP.
func TestProjectedHeaderCarrierWire(t *testing.T) {
	root := RunHTTPDSL(t, projectedHeaderCarrierDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/viewheader", root)
	harness, err := os.ReadFile("testdata/projected_header_carrier.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "carrier_test.go"), harness, 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func projectedHeaderCarrierDSL() {
	tags := dsl.Type("Tags", dsl.ArrayOf(dsl.String))
	card := dsl.ResultType("application/vnd.card", func() {
		dsl.TypeName("Card")
		dsl.Attribute("tags", tags)
		dsl.View("default", func() {
			dsl.Attribute("tags")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("tags")
		})
	})
	dsl.Service("probe", func() {
		dsl.Method("show", func() {
			dsl.Result(card)
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.Response(dsl.StatusOK, func() {
					dsl.Header("tags:X-Tags")
				})
			})
		})
	})
}
