package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPHeaderCollectionDefaultAdmission(t *testing.T) {
	for _, side := range []string{"request", "response"} {
		for _, tc := range []struct {
			name    string
			values  []string
			invalid bool
		}{
			{"ordinary", []string{"a", "b"}, false},
			{"present-empty", []string{}, false},
			{"comma", []string{"a,b"}, true},
			{"empty-item", []string{""}, true},
			{"outer-space", []string{" a"}, true},
		} {
			t.Run(side+"/"+tc.name, func(t *testing.T) {
				design := func() {
					Service("headers", func() {
						Method("echo", func() {
							values := func() {
								Attribute("tags", ArrayOf(String), func() {
									Default(tc.values)
								})
							}
							Payload(values)
							Result(values)
							HTTP(func() {
								GET("/echo")
								if side == "request" {
									Header("tags:X-Tags")
								}
								Response(StatusOK, func() {
									if side == "response" {
										Header("tags:X-Tags")
									}
								})
							})
						})
					})
				}
				if tc.invalid {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, "header")
				} else {
					expr.RunDSL(t, design)
				}
			})
		}
	}
}

func TestHTTPHeaderCollectionAuthoredConstraints(t *testing.T) {
	for _, side := range []string{"request", "response", "error"} {
		for _, source := range []string{"default", "example", "enum", "element-enum", "named-default", "set-cookie"} {
			t.Run(side+"/"+source, func(t *testing.T) {
				design := func() {
					values := ArrayOf(String)
					var valueType any = values
					if source == "named-default" {
						valueType = Type("Tags", values, func() {
							Default([]string{"private,comma"})
						})
					}
					if source == "element-enum" {
						valueType = ArrayOf(String, func() {
							Enum("private,comma")
						})
					}
					fields := func() {
						Attribute("tags", valueType, func() {
							switch source {
							case "default":
								Default([]string{"private,comma"})
							case "example":
								Example([]string{"private,comma"})
							case "enum":
								Enum([]string{"private,comma"})
							}
						})
					}
					Service("headers", func() {
						Method("echo", func() {
							Payload(fields)
							Result(fields)
							Error("bad", fields)
							HTTP(func() {
								GET("/echo")
								mapping := "tags:X-Tags"
								if source == "set-cookie" {
									mapping = "tags:Set-Cookie"
								}
								if side == "request" {
									Header(mapping)
								}
								Response(StatusOK, func() {
									if side == "response" {
										Header(mapping)
									}
								})
								Response("bad", StatusBadRequest, func() {
									if side == "error" {
										Header(mapping)
									}
								})
							})
						})
					})
				}
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, "header")
				require.NotContains(t, err.Error(), "private")
			})
		}
	}
}
