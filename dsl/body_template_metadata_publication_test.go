package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestBodyTemplateMemberMetadataPublication(t *testing.T) {
	for _, tc := range []struct {
		name   string
		remove bool
		readd  bool
	}{
		{name: "append control"},
		{name: "removed member stays removed", remove: true},
		{name: "remove then readd replaces inheritance", remove: true, readd: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				extra := Type("Extra", func() {
					Attribute("value", String, func() {
						Meta("trace", "base")
					})
				})
				Service("publication", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("object", func() {
								Attribute("value", String, func() {
									Meta("trace", "source")
								})
							})
						})
						HTTP(func() {
							POST("/")
							Body(func() {
								Attribute("object", func() {
									Extend(extra)
									Attribute("value", func() {
										if tc.remove {
											RemoveMeta("trace")
										}
										if tc.readd {
											Meta("trace", "replacement")
										}
									})
								})
							})
						})
					})
				})
			})
			endpoint := root.API.HTTP.Service("publication").Endpoint("create")
			body := bodyChild(t, bodyChild(t, endpoint.Body, "object"), "value")
			source := bodyChild(t, bodyChild(t, endpoint.MethodExpr.Payload, "object"), "value")
			t.Logf("source trace=%v published Body trace=%v", source.Meta["trace"], body.Meta["trace"])
			assert.Equal(t, []string{"source"}, source.Meta["trace"])
			switch {
			case tc.readd:
				assert.Equal(t, []string{"replacement"}, body.Meta["trace"])
			case tc.remove:
				assert.NotContains(t, body.Meta, "trace")
			default:
				assert.Equal(t, []string{"base", "source"}, body.Meta["trace"])
			}
			copied := expr.DupAtt(body)
			copied.DeleteMeta("trace")
			assert.NotContains(t, copied.Meta, "trace")
			assert.Equal(t, []string{"source"}, source.Meta["trace"])
			if tc.readd {
				assert.Equal(t, []string{"replacement"}, body.Meta["trace"])
			} else if !tc.remove {
				assert.Equal(t, []string{"base", "source"}, body.Meta["trace"])
			}
		})
	}
}
