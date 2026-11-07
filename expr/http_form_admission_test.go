package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPFormUnionAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selected bool
		optional bool
		required bool
		form     bool
		scalar   bool
		tagged   bool
		want     string
	}{
		{name: "form root", form: true},
		{name: "tagged form root", form: true, tagged: true, scalar: true},
		{name: "tagged form optional selected", form: true, tagged: true, scalar: true, selected: true, optional: true},
		{name: "form optional root", form: true, optional: true},
		{name: "form selected", form: true, selected: true},
		{name: "form optional selected", form: true, selected: true, optional: true},
		{name: "form required selected", form: true, selected: true, optional: true, required: true, want: "payload attribute mapped to the request body to be optional"},
		{name: "JSON optional root", optional: true},
		{name: "JSON scalar optional root", optional: true, scalar: true},
		{name: "JSON scalar optional selected", optional: true, scalar: true, selected: true},
		{name: "form scalar alternatives", form: true, scalar: true, want: "form body requires an object or map"},
		{name: "JSON optional selected", selected: true, optional: true},
		{name: "JSON required selected", selected: true, optional: true, required: true, want: "payload attribute mapped to the request body to be optional"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				empty := Type("EmptyVariant", func() {})
				record := Type("RecordVariant", func() {
					Attribute("value", String)
					Required("value")
				})
				alternatives := OneOf(empty, record)
				if tc.scalar {
					alternatives = OneOf(String, Int)
				}
				choice := Type("Choice", alternatives, func() {
					if !tc.tagged {
						Untagged()
					}
				})
				Service("Forms", func() {
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
							if tc.selected {
								Header("trace")
								Body("choice")
							}
							if tc.form {
								FormRequest()
							}
							if tc.optional {
								OptionalRequestBody()
							}
						})
					})
				})
			}
			if tc.want != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.want)
				return
			}
			expr.RunDSL(t, design)
		})
	}
}

func TestHTTPFormRequestRejectsScalarMetadataPayload(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("Forms", func() {
			Method("Submit", func() {
				Payload(String)
				HTTP(func() {
					POST("/")
					Header("value")
					FormRequest()
				})
			})
		})
	})
	require.ErrorContains(t, err, "form body requires an object or map")
}
