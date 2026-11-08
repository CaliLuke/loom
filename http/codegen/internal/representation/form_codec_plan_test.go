package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestFormBodyPlanCodecs checks that the runtime plan of a form or multipart
// body records the codec its captured contract selects at each node: a JSON
// content field, a JSON item below an array that keeps the body codec, a
// styled union beside the same union as JSON, and raw and omitted fields that
// keep the body codec. In a form, a custom codec also owns a JSON boundary and
// a custom descendant overrides the JSON codec of its field; a multipart body
// with an emitted replacement type is a custom whole body instead.
func TestFormBodyPlanCodecs(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		body := expr.ValueCodecForm
		if multipart {
			body = expr.ValueCodecMultipart
		}
		root := formCodecPlan(t, multipart, func() func() {
			record := dsl.Type("Record", func() {
				dsl.Attribute("x", dsl.String)
				dsl.Required("x")
			})
			choice := dsl.Type("Choice", dsl.OneOf(record, dsl.Boolean), dsl.Untagged)
			tags := dsl.Type("Tags", dsl.ArrayOf(dsl.String))
			return func() {
				dsl.Payload(func() {
					dsl.Attribute("pet", func() {
						dsl.Attribute("kind", dsl.String)
						if !multipart {
							dsl.Attribute("stamp", dsl.String, func() {
								dsl.Meta("struct:field:type", "other.Stamp", "example.com/other")
							})
						}
					})
					dsl.Attribute("jsonTags", tags)
					dsl.Attribute("textTags", tags)
					dsl.Attribute("styled", choice)
					dsl.Attribute("json", choice)
					if !multipart {
						dsl.Attribute("custom", dsl.String, func() {
							dsl.Meta("struct:field:type", "other.Custom", "example.com/other")
						})
						dsl.Required("custom")
						dsl.Meta("openapi:encoding:custom:contentType", "application/json")
					}
					dsl.Attribute("data", dsl.Bytes)
					dsl.Attribute("skipped", record, func() {
						dsl.Meta("struct:tag:form", "-")
					})
					dsl.Required("pet", "jsonTags", "textTags", "styled", "json", "data", "skipped")
					dsl.Meta("openapi:encoding:jsonTags:contentType", "application/json")
					dsl.Meta("openapi:encoding:styled:style", "form")
					dsl.Meta("openapi:encoding:styled:explode", "false")
					dsl.Meta("openapi:encoding:json:contentType", "application/json")
					dsl.Meta("openapi:encoding:data:contentType", "application/json")
				})
			}
		})
		require.Equal(t, body, root.Codec())
		pet := planMember(t, root, "pet")
		require.Equal(t, expr.ValueCodecJSON, pet.Codec(), "a default object field is JSON content")
		require.Equal(t, expr.ValueCodecJSON, planMember(t, pet, "kind").Codec())
		if !multipart {
			require.Equal(t, expr.ValueCodecCustom, planMember(t, pet, "stamp").Codec(), "a custom descendant overrides the inherited JSON codec")
			require.Equal(t, expr.ValueCodecCustom, planMember(t, root, "custom").Codec(), "a custom codec owns its JSON boundary target")
		}

		jsonTags, textTags := planMember(t, root, "jsonTags"), planMember(t, root, "textTags")
		require.Equal(t, body, jsonTags.Codec(), "item application keeps the array container physical")
		require.Equal(t, expr.ValueCodecJSON, planStructural(jsonTags).Element().Codec())
		require.Equal(t, body, textTags.Codec())
		require.Equal(t, body, planStructural(textTags).Element().Codec(), "the same named array as text items is not JSON")

		styled, json := planMember(t, root, "styled"), planMember(t, root, "json")
		require.Equal(t, body, styled.Codec())
		for _, branch := range planStructural(styled).Branches() {
			require.Equal(t, body, branch.Node.Codec(), "styled union branch %s", branch.Tag)
		}
		require.Equal(t, expr.ValueCodecJSON, json.Codec(), "the same named union as JSON content")
		for _, branch := range planStructural(json).Branches() {
			require.Equal(t, expr.ValueCodecJSON, branch.Node.Codec(), "JSON union branch %s", branch.Tag)
		}

		if multipart {
			require.Equal(t, body, planMember(t, root, "data").Codec(), "multipart bytes stay a binary part under a JSON label")
		} else {
			require.Equal(t, expr.ValueCodecJSON, planMember(t, root, "data").Codec(), "form bytes can select JSON content")
		}
		require.Equal(t, body, planMember(t, root, "skipped").Codec(), "an omitted field has no content boundary")
	}
}

// TestFormRootPlanCodecs covers form root structures without multipart
// equivalents: the values of a map root and the branches of a tagged root,
// whose object branches flatten their fields and whose other branches use the
// value key.
func TestFormRootPlanCodecs(t *testing.T) {
	t.Run("map root", func(t *testing.T) {
		root := formCodecPlan(t, false, func() func() {
			entry := dsl.Type("Entry", func() {
				dsl.Attribute("x", dsl.String)
			})
			return func() {
				dsl.Payload(dsl.MapOf(dsl.String, entry))
			}
		})
		require.Equal(t, expr.ValueCodecForm, root.Codec())
		require.Equal(t, expr.ValueCodecJSON, planStructural(root).Element().Codec(), "each object map value is JSON content")
	})
	t.Run("tagged root", func(t *testing.T) {
		root := formCodecPlan(t, false, func() func() {
			entry := dsl.Type("Entry", func() {
				dsl.Attribute("x", dsl.String)
			})
			flat := dsl.Type("Flat", func() {
				dsl.Attribute("entry", entry)
				dsl.Attribute("name", dsl.String)
			})
			pick := dsl.Type("Pick", dsl.OneOf(func() {
				dsl.Attribute("flat", flat)
				dsl.Attribute("entries", dsl.ArrayOf(entry))
			}))
			return func() {
				dsl.Payload(pick)
			}
		})
		require.Equal(t, expr.ValueCodecForm, root.Codec())
		branches := map[string]expr.ValuePlanNode{}
		for _, branch := range planStructural(root).Branches() {
			branches[branch.Tag] = branch.Node
		}
		flat := branches["flat"]
		require.Equal(t, expr.ValueCodecForm, flat.Codec(), "an object branch flattens its fields")
		require.Equal(t, expr.ValueCodecJSON, planMember(t, flat, "entry").Codec())
		require.Equal(t, expr.ValueCodecForm, planMember(t, flat, "name").Codec())
		entries := branches["entries"]
		require.Equal(t, expr.ValueCodecForm, entries.Codec(), "a value-key array keeps the body codec")
		require.Equal(t, expr.ValueCodecJSON, planStructural(entries).Element().Codec())
	})
}

// formCodecPlan returns the structural root of the runtime body plan of a form
// or multipart request. design declares types and returns the method payload.
func formCodecPlan(t *testing.T, multipart bool, design func() func()) expr.ValuePlanNode {
	t.Helper()
	root := codegen.RunDSL(t, func() {
		payload := design()
		dsl.Service("forms", func() {
			dsl.Method("submit", func() {
				payload()
				dsl.HTTP(func() {
					dsl.POST("/")
					if multipart {
						dsl.MultipartRequest()
					} else {
						dsl.FormRequest()
					}
				})
			})
		})
	})
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	request := prepared.Endpoints[0].Request
	target := BuildValuePlan(request.BodyValue, request.Body, schemaContext(true), expr.ValuePlanRuntime)
	require.NoError(t, target.Error)
	return planStructural(target.Plan.Root())
}

// planStructural descends named-declaration edges to the structural node.
func planStructural(node expr.ValuePlanNode) expr.ValuePlanNode {
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	return node
}

func planMember(t *testing.T, node expr.ValuePlanNode, name string) expr.ValuePlanNode {
	t.Helper()
	for _, member := range planStructural(node).Members() {
		if member.Name == name {
			return member.Node
		}
	}
	require.Failf(t, "missing plan member", "%s", name)
	return expr.ValuePlanNode{}
}
