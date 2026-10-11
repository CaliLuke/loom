package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestEncodingMetadataAdmission(t *testing.T) {
	for _, target := range []string{"3.1", "3.2"} {
		for _, location := range []string{"request", "response", "error"} {
			for _, tc := range []struct {
				name    string
				keys    []string
				invalid bool
			}{
				{"named and prefix", []string{"encoding:id:style", "prefixEncoding:0:style"}, true},
				{"named and item", []string{"encoding:id:style", "itemEncoding:style"}, true},
				{"nested conflict", []string{"itemEncoding:encoding:id:style", "itemEncoding:prefixEncoding:0:style"}, true},
				{"deep conflict", []string{"encoding:id:prefixEncoding:0:encoding:name:style", "encoding:id:prefixEncoding:0:itemEncoding:style"}, true},
				{"positional pair", []string{"prefixEncoding:0:style", "itemEncoding:style"}, false},
				{"independent siblings", []string{"prefixEncoding:0:encoding:id:style", "itemEncoding:prefixEncoding:0:style"}, false},
			} {
				t.Run(target+"/"+location+"/"+tc.name, func(t *testing.T) {
					root := httpgen.RunHTTPDSL(t, func() {
						body := dsl.Type("Encoded", func() {
							dsl.Attribute("id", dsl.String)
						})
						metadata := func() {
							for _, key := range tc.keys {
								dsl.Meta("openapi:"+key, "form")
							}
						}
						dsl.Service("encoding", func() {
							dsl.Method("show", func() {
								switch location {
								case "request":
									dsl.Payload(body)
								case "response":
									dsl.Result(body, metadata)
								case "error":
									dsl.Error("bad", body, metadata)
								}
								dsl.HTTP(func() {
									dsl.POST("/encoded")
									if location == "request" {
										dsl.Body(func() {
											dsl.Attribute("id")
											metadata()
										})
									}
									if location == "error" {
										dsl.Response("bad", dsl.StatusBadRequest)
									}
								})
							})
						})
					})
					root.API.Meta = expr.MetaExpr{"openapi:version": {target}}
					_, err := openapiv3.Files(root)
					if tc.invalid {
						require.ErrorContains(t, err, "encoding cannot coexist with prefixEncoding or itemEncoding")
						require.ErrorContains(t, err, "encoding.show")
					} else {
						require.NoError(t, err)
					}
				})
			}
		}
	}
}
