package expr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestDeclarationMetadataValidatedOnce(t *testing.T) {
	for _, tc := range []struct {
		name    string
		key     string
		value   string
		message string
	}{
		{"openness", "openapi:additionalProperties", "invalid", `must be "true" or "false"`},
		{"tag", "struct:tag:bad key", "value", `struct tag key "bad key"`},
		{"union target", "openapi:unevaluatedProperties", "false", "requires a union"},
		{"union value", "openapi:unevaluatedProperties", "true", "supports only false"},
		{"view", expr.ViewMetaKey, "missing", "is not a result type"},
		{"json option", "struct:tag:json", "value,string", "JSON ,string is not supported"},
		{"package", "struct:pkg:path", "../types", "is not a valid relative Go import path"},
		{"proto name", "struct:name:proto", "bad-name", "is not a valid protocol buffer message name"},
	} {
		for _, kind := range []string{"type", "alias", "result"} {
			for _, uses := range []int{0, 1, 2} {
				t.Run(fmt.Sprintf("%s/%s/uses=%d", tc.name, kind, uses), func(t *testing.T) {
					err := expr.RunInvalidDSL(t, func() {
						body := func() {
							Attribute("id", String)
							Meta(tc.key, tc.value)
						}
						var typ expr.DataType
						if kind == "result" {
							typ = ResultType("application/vnd.broken", func() {
								TypeName("Broken")
								body()
							})
						} else {
							typ = Type("Broken", body)
							if kind == "alias" {
								typ = Type("Alias", typ)
							}
						}
						if uses > 0 {
							Service("records", func() {
								Method("show", func() {
									Payload(typ)
									if uses == 2 {
										Result(typ)
									}
								})
							})
						}
					})
					message := tc.message
					if tc.key == expr.ViewMetaKey && kind == "result" {
						message = `unknown view "missing"`
					}
					require.ErrorContains(t, err, message)
					require.Equal(t, 1, strings.Count(err.Error(), message), err.Error())
					require.Contains(t, err.Error(), `type "Broken"`)
					require.NotContains(t, err.Error(), `method "show"`)
				})
			}
		}
	}
}

func TestOccurrenceMetadataRemainsIndependent(t *testing.T) {
	for _, policy := range []string{"true", "false"} {
		t.Run(policy, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				typ := Type("Record", func() {
					Attribute("id", String)
					Meta("openapi:additionalProperties", policy)
				})
				Service("records", func() {
					Method("show", func() {
						Payload(typ, func() {
							Meta("openapi:additionalProperties", "invalid")
						})
						Result(typ, func() {
							Meta("openapi:additionalProperties", "invalid")
						})
					})
				})
			})
			require.Equal(t, 2, strings.Count(err.Error(), `must be "true" or "false"`), err.Error())
			require.NotContains(t, err.Error(), `type "Record":`)
		})
	}
}

func TestResultDeclarationViewMetadata(t *testing.T) {
	for _, view := range []string{"default", "summary"} {
		for _, uses := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/uses=%d", view, uses), func(t *testing.T) {
				var result *expr.ResultTypeExpr
				expr.RunDSL(t, func() {
					result = ResultType("application/vnd.selected", func() {
						Attribute("id", String)
						Attribute("detail", String)
						View("summary", func() {
							Attribute("id")
						})
						Meta(expr.ViewMetaKey, view)
					})
					if uses > 0 {
						Service("records", func() {
							Method("show", func() {
								Payload(result)
								if uses == 2 {
									Result(result)
								}
							})
						})
					}
				})
				require.NotNil(t, result.Find("id"))
				require.Equal(t, view == "default", result.Find("detail") != nil)
			})
		}
	}
}
