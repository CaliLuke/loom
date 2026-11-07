package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestOptionalRootObjectBodyPresence(t *testing.T) {
	cases := []struct {
		name, media                                                    string
		required, selected, bindings, requiredMember, optionalBindings bool
	}{
		{name: "json", media: "json"},
		{name: "form", media: "form"},
		{name: "multipart", media: "multipart"},
		{name: "multipart_required_member", media: "multipart", requiredMember: true},
		{name: "json_bound_inputs", media: "json", bindings: true},
		{name: "form_bound_inputs", media: "form", bindings: true},
		{name: "multipart_bound_inputs", media: "multipart", bindings: true},
		{name: "multipart_optional_bound_inputs", media: "multipart", bindings: true, optionalBindings: true},
		{name: "required_json", media: "json", required: true},
		{name: "required_multipart", media: "multipart", required: true},
		{name: "selected_json", media: "json", selected: true, bindings: true, requiredMember: true},
		{name: "selected_multipart", media: "multipart", selected: true, bindings: true, requiredMember: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				fields := Type("Fields", func() {
					Attribute("name", String)
					Attribute("count", Int, func() {
						Default(7)
					})
					if tc.requiredMember {
						Required("name")
					}
				})
				Service("presence", func() {
					Method("find", func() {
						Payload(func() {
							if tc.selected {
								Attribute("doc", fields)
							} else {
								Extend(fields)
							}
							if tc.bindings {
								Attribute("query", String)
								Attribute("token", String)
								if !tc.optionalBindings {
									Required("query", "token")
								}
							}
						})
						HTTP(func() {
							POST("/presence")
							if tc.selected {
								Body("doc")
							} else if tc.media == "json" {
								Body(fields)
							}
							if tc.bindings {
								Param("query")
								Header("token:X-Token")
							}
							if !tc.required {
								OptionalRequestBody()
							}
							switch tc.media {
							case "form":
								FormRequest()
							case "multipart":
								MultipartRequest()
							}
						})
					})
				})
			})
			services := CreateHTTPServices(root)
			request := services.Get("presence").Endpoint("find").Payload.Request
			require.Equal(t, !tc.required, request.OptionalObjectBody, "object carrier must retain optional body absence independently of selected member identity")
			require.Equal(t, tc.selected, request.OptionalBodyAttribute, "client selected-member identity must remain separate")
			decoder := codegen.SectionCode(t, findFileWithSection(t, ServerFiles("gen", services), "request-decoder").Section("request-decoder")[0])
			constructor := codegen.SectionCode(t, findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init").Section("server-payload-init")[0])
			if tc.media == "json" {
				require.Contains(t, decoder, "loomhttp.WithNonNullableBody(r)")
			}
			if tc.required {
				require.NotContains(t, decoder, "body = nil")
				return
			}
			require.Contains(t, decoder, "body = nil")
			require.Contains(t, decoder, request.PayloadInit.Name+"(body")
			require.Contains(t, constructor, "if body != nil", "absent body must not be converted or defaulted")
			if tc.bindings {
				require.Contains(t, decoder, "url.ParseQuery(r.URL.RawQuery)")
				require.Contains(t, decoder, "X-Token")
				require.Contains(t, constructor, ".Query = query")
				require.Contains(t, constructor, ".Token = token")
			}
		})
	}
}

func TestOptionalObjectCarrierBoundaries(t *testing.T) {
	object := &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	for _, tc := range []struct {
		name           string
		attribute      *expr.AttributeExpr
		required, want bool
	}{
		{name: "native", attribute: &expr.AttributeExpr{Type: object}, want: true},
		{name: "required", attribute: &expr.AttributeExpr{Type: object}, required: true},
		{name: "nullable", attribute: &expr.AttributeExpr{Type: object, Nullable: true}},
		{name: "opaque", attribute: &expr.AttributeExpr{Type: object, Meta: expr.MetaExpr{"struct:field:type": {"custom.Value"}}}},
		{name: "raw JSON", attribute: &expr.AttributeExpr{Type: expr.Any}},
		{name: "union", attribute: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice"}}},
		{name: "empty marker", attribute: &expr.AttributeExpr{Type: expr.Empty}},
		{name: "no body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &transportir.Request{Body: tc.attribute, MustHaveBody: tc.required}
			require.Equal(t, tc.want, isOptionalObjectBody(request))
		})
	}
}
