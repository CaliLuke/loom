package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestCollectionPayloadParameterLocations(t *testing.T) {
	for _, shape := range []string{"array", "map"} {
		for _, location := range []string{"path", "query", "header", "cookie"} {
			for _, named := range []bool{false, true} {
				for _, raw := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/named=%t/raw=%t", shape, location, named, raw), func(t *testing.T) {
						expr.RunDSL(t, collectionPayloadDesign(shape, named, raw, []string{location}))
					})
				}
			}
		}
	}
}

func TestCollectionPayloadRejectsMultipleOwners(t *testing.T) {
	for _, shape := range []string{"array", "map"} {
		for _, locations := range [][]string{{"header", "cookie"}, {"cookie", "query"}, {"cookie", "body"}, {"header", "body"}, {"header", "header"}, {"cookie", "cookie"}, {"query", "form"}, {"cookie", "form"}} {
			t.Run(shape+fmt.Sprint(locations), func(t *testing.T) {
				require.ErrorContains(t, expr.RunInvalidDSL(t, collectionPayloadDesign(shape, false, false, locations)), "at most one transport binding")
			})
		}
	}
}

func collectionPayloadDesign(shape string, named, raw bool, locations []string) func() {
	return func() {
		var typ any = MapOf(String, String)
		if shape == "array" {
			typ = ArrayOf(String)
		}
		bodyType := Type("BodyCollection", typ)
		if named {
			typ = Type("Collection", typ)
		}
		Service("collections", func() {
			Method("send", func() {
				Payload(typ)
				HTTP(func() {
					if locations[0] == "path" {
						POST("/{value}")
					} else {
						POST("/")
					}
					for index, location := range locations {
						name := "value"
						if index > 0 {
							name = "other"
						}
						switch location {
						case "path", "query":
							Param(name)
						case "header":
							Header(name)
						case "cookie":
							Cookie(name)
						case "body":
							Body(bodyType)
						case "form":
							FormRequest()
						}
					}
					if raw {
						SkipRequestBodyEncodeDecode()
					}
					Response(StatusNoContent)
				})
			})
		})
	}
}
