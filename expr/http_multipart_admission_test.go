package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPMultipartUnionAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selected bool
		optional bool
		required bool
	}{
		{name: "root"},
		{name: "optional root", optional: true},
		{name: "selected", selected: true},
		{name: "optional selected", selected: true, optional: true},
		{name: "required selected", selected: true, optional: true, required: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				empty := Type("EmptyVariant", func() {})
				record := Type("RecordVariant", func() {
					Attribute("value", String)
					Required("value")
				})
				choice := Type("Choice", OneOf(empty, record), func() {
					Untagged()
				})
				Service("Uploads", func() {
					Method("Submit", func() {
						if tc.selected {
							Payload(func() {
								Attribute("choice", choice)
								Attribute("trace", String)
								if tc.required {
									Required("choice")
								}
							})
						} else {
							Payload(choice)
						}
						HTTP(func() {
							POST("/")
							MultipartRequest()
							if tc.selected {
								Header("trace")
								Body("choice")
							}
							if tc.optional {
								OptionalRequestBody()
							}
						})
					})
				})
			}
			if tc.required {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), "payload attribute mapped to the request body to be optional")
			} else {
				expr.RunDSL(t, design)
			}
		})
	}
}

func TestHTTPMultipartOptionalObjectPresence(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		expr.RunDSL(t, func() {
			body := Type("Upload", func() {
				Attribute("value", String)
				Required("value")
			})
			Service("Uploads", func() {
				Method("Submit", func() {
					Payload(body)
					HTTP(func() {
						POST("/")
						MultipartRequest()
						OptionalRequestBody()
						if explicit {
							Body(body)
						}
					})
				})
			})
		})
	}
}
