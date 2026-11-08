package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// InheritedRequiredProjectionDSL exercises composed requirements through path
// projection, multipart bodies and nested response collections.
func InheritedRequiredProjectionDSL() {
	API("uploads", func() {
		Meta("openapi:closed-objects", "true")
		Randomizer(expr.NewDeterministicRandomizer())
	})
	path := Type("Path", func() {
		Attribute("id", String)
		Required("id")
	})
	upload := Type("UploadBody", func() {
		Meta("openapi:typename", "")
		Attribute("file", Bytes)
		Attribute("filename", String)
		Required("file", "filename")
	})
	Service("uploads", func() {
		Method("create", func() {
			NoAccessCheck("Public reproduction")
			Payload(func() {
				Meta("openapi:typename", "UploadPayload")
				Extend(path)
				Attribute("id")
				Extend(upload)
				Required("id")
			})
			HTTP(func() {
				POST("/uploads/{id}")
				Param("id")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
	})
	base := Type("BaseRow", func() {
		Attribute("id", String)
		Required("id")
	})
	row := Type("Row", func() {
		Extend(base)
		Attribute("extra", String)
		Required("extra")
	})
	group := Type("Group", func() {
		Attribute("items", ArrayOf(row))
		Required("items")
	})
	derived := Type("DerivedGroup", func() {
		Extend(group)
		Attribute("label", String)
	})
	rows := Type("Rows", func() {
		Attribute("items", ArrayOf(row))
		Attribute("groups", ArrayOf(derived))
		Required("items")
	})
	Service("rows", func() {
		Method("list", func() {
			NoAccessCheck("Public reproduction")
			Result(rows)
			HTTP(func() {
				GET("/rows")
				Response(StatusOK)
			})
		})
	})
}
