package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

func TestUnionResponseReusedValidator(t *testing.T) {
	for _, first := range []bool{false, true} {
		for _, root := range []bool{false, true} {
			t.Run(reuseCaseName(first, root), func(t *testing.T) {
				design := RunHTTPDSL(t, reusedUnionResponseDSL(first, root))
				services := CreateHTTPServices(design)
				var validators string
				for _, file := range ServerTypeFiles("example.com/reuse/gen", services) {
					for _, section := range file.Section("server-validate") {
						validators += codegen.SectionCode(t, section)
					}
				}
				require.Contains(t, validators, "func ValidateItemResponse(")
				require.Contains(t, validators, "loom.ValidateFormat(")
				require.Contains(t, validators, "ValidateItemResponse(e,")
			})
		}
	}
}

func TestUnionResponseReusedValidatorGenerated(t *testing.T) {
	for _, first := range []bool{false, true} {
		for _, root := range []bool{false, true} {
			t.Run(reuseCaseName(first, root), func(t *testing.T) {
				design := RunHTTPDSL(t, reusedUnionResponseDSL(first, root))
				dir := t.TempDir()
				renderHTTPModule(t, dir, "example.com/reuse", design)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "validation_test.go"), []byte(reusedUnionResponseHarness), 0o600))
				runGoCommand(t, dir, "mod", "tidy")
				runGoCommand(t, dir, "test", "./...")
			})
		}
	}
}

func reuseCaseName(first, root bool) string {
	name := "ordinary-first"
	if first {
		name = "union-first"
	}
	if root {
		return name + "/root"
	}
	return name + "/collection"
}

func reusedUnionResponseDSL(unionFirst, root bool) func() {
	return func() {
		item := Type("Item", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("asset", String, func() {
				Format(FormatURI)
			})
			Required("asset")
		})
		data := Type("Data", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("data", ArrayOf(item))
			Attribute("status", Boolean, func() {
				Enum(true)
			})
			Required("data", "status")
		})
		failure := Type("Failure", func() {
			Attribute("error", String)
			Attribute("status", Boolean, func() {
				Enum(false)
			})
			Required("error", "status")
		})
		ordinary := func() {
			Method("ListItems", func() {
				if root {
					Result(item)
				} else {
					Result(ArrayOf(item))
				}
				HTTP(func() {
					GET("/items/all")
				})
			})
		}
		union := func() {
			Method("List", func() {
				Result(OneOf(data, failure), func() {
					Untagged()
				})
				HTTP(func() {
					GET("/items")
				})
			})
		}
		Service("Repro", func() {
			if unionFirst {
				union()
				ordinary()
			} else {
				ordinary()
				union()
			}
		})
	}
}

const reusedUnionResponseHarness = `package reuse_test

import (
 "testing"
 "github.com/stretchr/testify/require"
 server "example.com/reuse/gen/http/repro/server"
)

func TestNestedValidator(t *testing.T) {
 for _, test := range []struct { name, asset string; invalid bool }{
  {"valid", "https://example.com/item", false},
  {"invalid URI", "not a URI", true},
 } {
  t.Run(test.name,func(t *testing.T) {
   err := server.ValidateDataResponse(&server.DataResponse{
    Data: []*server.ItemResponse{{Asset:test.asset}}, Status:true,
   })
   if test.invalid {
    require.Error(t,err)
   } else {
    require.NoError(t,err)
   }
  })
 }
}

func TestCollectionPresence(t *testing.T) {
 for _, test := range []struct { name string; items []*server.ItemResponse; invalid bool }{
  {"missing", nil, true},
  {"empty", []*server.ItemResponse{}, false},
  {"null element", []*server.ItemResponse{nil}, true},
 } {
  t.Run(test.name,func(t *testing.T) {
   err := server.ValidateDataResponse(&server.DataResponse{Data:test.items, Status:true})
   if test.invalid {
    require.Error(t,err)
   } else {
    require.NoError(t,err)
   }
  })
 }
}
`
