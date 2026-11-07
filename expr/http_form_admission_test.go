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

func TestHTTPOptionalObjectBodyAdmission(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		explicit, required, optional, multipart, json bool
		binding                                       string
		want                                          string
	}{
		{name: "explicit required form", explicit: true, required: true, optional: true, want: "OptionalRequestBody requires the request body to have no required attributes."},
		{name: "implicit required form", required: true, optional: true, want: "OptionalRequestBody requires the request body to have no required attributes."},
		{name: "required header excluded", required: true, optional: true, binding: "header"},
		{name: "required query excluded", required: true, optional: true, binding: "query"},
		{name: "required path excluded", required: true, optional: true, binding: "path"},
		{name: "optional members form", optional: true},
		{name: "required body form", required: true},
		{name: "explicit required multipart", explicit: true, required: true, optional: true, multipart: true},
		{name: "implicit required multipart", required: true, optional: true, multipart: true},
		{name: "implicit JSON remains missing body", required: true, optional: true, json: true, want: "HTTP endpoint uses OptionalRequestBody but does not define a request body."},
		{name: "explicit required JSON", explicit: true, required: true, optional: true, json: true, want: "OptionalRequestBody requires the request body to have no required attributes."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				Service("admission", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("token", String)
							Attribute("note", String)
							if tc.required {
								Required("token")
							}
						})
						HTTP(func() {
							if tc.binding == "path" {
								POST("/{token}")
							} else {
								POST("/")
							}
							if tc.optional {
								OptionalRequestBody()
							}
							switch {
							case tc.multipart:
								MultipartRequest()
							case !tc.json:
								FormRequest()
							}
							switch tc.binding {
							case "header":
								Header("token")
							case "query", "path":
								Param("token")
							}
							if tc.explicit {
								Body(func() {
									Attribute("token")
									Attribute("note")
								})
							}
						})
					})
				})
			}
			if tc.want != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.want)
				return
			}
			root := expr.RunDSL(t, design)
			body := root.API.HTTP.Service("admission").Endpoint("create").Body
			if tc.binding != "" {
				require.Empty(t, body.AllRequired())
				_, token := body.FindAttribute("token")
				require.Nil(t, token)
			} else if tc.required {
				require.Contains(t, body.AllRequired(), "token")
			}
		})
	}
}
