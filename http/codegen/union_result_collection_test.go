package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
)

// TestResultCollectionUnionTypes checks that the response bodies of a result
// type that holds a named union, returned by one endpoint and as a
// collection by another, refer to union types that the transport package
// declares with the branch types of the same body. The body types of the
// result type and of the collection element share the identifier of the
// result type but not their branch types.
func TestResultCollectionUnionTypes(t *testing.T) {
	root := RunHTTPDSL(t, resultCollectionUnionDSL)
	sd := CreateHTTPServices(root).Get("svc")
	branches := make(map[string][]string, len(sd.UnionTypes))
	for _, union := range sd.UnionTypes {
		for _, field := range union.Fields {
			branches[union.Name] = append(branches[union.Name], field.FieldType)
		}
	}
	types := make(map[string]string)
	for _, data := range append(sd.ServerBodyAttributeTypes, sd.ClientBodyAttributeTypes...) {
		types[data.Name] = data.Def
	}
	for _, endpoint := range sd.Endpoints {
		for _, response := range endpoint.Result.Responses {
			for _, data := range response.ServerBody {
				types[data.Name] = data.Def
			}
			if response.ClientBody != nil {
				types[response.ClientBody.Name] = response.ClientBody.Def
			}
		}
	}
	cases := []struct {
		holder string
		branch string
	}{
		{"RtResponseBody", "*LeafResponseBody"},
		{"RTResponse", "*LeafResponse"},
		{"VrtResponseBody", "*LeafResponseBody"},
		{"VRTResponse", "*LeafResponse"},
	}
	for _, c := range cases {
		def, ok := types[c.holder]
		if !assert.True(t, ok, "no %s type", c.holder) {
			continue
		}
		fields := nestedUnionFieldTypes(def)
		for _, field := range []string{"C", "D"} {
			union := fields[field]
			assert.Contains(t, branches, union, "%s.%s refers to undeclared union %q", c.holder, field, union)
			assert.Contains(t, branches[union], c.branch, "%s.%s union %q has branches %v", c.holder, field, union, branches[union])
		}
	}
}

func resultCollectionUnionDSL() {
	leaf, other := unionBodyBranchTypes()
	choice := Type("Choice", OneOf(leaf, other))
	rt := ResultType("application/vnd.rt", "RT", func() {
		Attribute("id", String)
		Attribute("c", choice)
		Attribute("d", choice)
		Required("d")
	})
	vrt := ResultType("application/vnd.vrt", "VRT", func() {
		Attribute("id", String)
		Attribute("c", choice)
		Attribute("d", choice)
		Required("d")
		View("default", func() {
			Attribute("id")
			Attribute("c")
			Attribute("d")
		})
		View("tiny", func() {
			Attribute("id")
		})
	})
	Service("svc", func() {
		Method("rt", func() {
			Result(rt)
			HTTP(func() {
				GET("/rt")
			})
		})
		Method("rts", func() {
			Result(CollectionOf(rt))
			HTTP(func() {
				GET("/rts")
			})
		})
		Method("vrt", func() {
			Result(vrt)
			HTTP(func() {
				GET("/vrt")
			})
		})
		Method("vrts", func() {
			Result(CollectionOf(vrt))
			HTTP(func() {
				GET("/vrts")
			})
		})
	})
}
