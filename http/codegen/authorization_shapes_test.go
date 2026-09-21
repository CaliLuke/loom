package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationInputShapesCompile(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		ref := Type("Ref", func() {
			Meta("struct:pkg:path", "types")
			Attribute("id", String, func() {
				MinLength(1)
			})
			Required("id")
		})
		key := Type("DocumentKey", String, func() {
			MinLength(1)
		})
		keyRef := Type("KeyRef", func() {
			Attribute("id", key)
			Required("id")
		})
		keyAccess := Authorization("key.edit", keyRef)
		collision := Type("ValidateRef", func() {
			Attribute("ok", Boolean)
		})
		localRef := Type("LocalRef", func() {
			Attribute("id", String)
			Required("id")
		})
		rootInput := Type("RootInput", func() {
			Attribute("branch", localRef)
			Required("branch")
		})
		rootAccess := Authorization("root.edit", rootInput)
		action := Type("Action", String)
		inputs := Type("Inputs", func() {
			Attribute("source", localRef)
			Attribute("destination", localRef)
			Attribute("note", String)
			Attribute("ids", ArrayOf(String))
			Required("source", "destination")
		})
		custom := Authorization("resource.edit", ref)
		transfer := Authorization("resource.transfer", inputs)
		contextOnly := Authorization("account.create", Empty)
		Service("shapes", func() {
			StrictAuthorization()
			Method("custom", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				Authorize(custom, func() {
					Bind("id", "id")
				})
				Result(collision)
				HTTP(func() {
					POST("/custom")
				})
			})
			Method("transfer", func() {
				Payload(func() {
					Attribute("from", localRef)
					Attribute("to", localRef)
					Attribute("note", String)
					Attribute("ids", ArrayOf(String))
					Required("note")
				})
				Authorize(transfer, func() {
					Bind("source", "from")
					Bind("destination", "to")
					Bind("note", "note")
					Bind("ids", "ids")
				})
				Authorize(contextOnly)
				HTTP(func() {
					POST("/transfer")
				})
			})
			Method("key", func() {
				Payload(key)
				Authorize(keyAccess, func() {
					Bind("id", "")
				})
				HTTP(func() {
					POST("/key")
				})
			})
			Method("root", func() {
				Payload(OneOf(localRef, ref))
				AuthorizeBy("", func() {
					AuthorizationCase("LocalRef", func() {
						Authorize(rootAccess, func() {
							Bind("branch", "")
						})
					})
					AuthorizationCase("Ref", func() {
						NoAccessCheck("Public reference")
					})
				})
			})
			Method("alias", func() {
				Payload(func() {
					Attribute("action", action, func() {
						Enum("create", "read")
					})
				})
				AuthorizeBy("action", func() {
					AuthorizationCase("create", func() {
						Authorize(contextOnly)
					})
					AuthorizationCase("read", func() {
						NoAccessCheck("Public reference")
					})
				})
				HTTP(func() {
					POST("/alias")
				})
			})
			Method("create", func() {
				Authorize(contextOnly)
				HTTP(func() {
					POST("/create")
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/accessshapes", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./gen/...")
}
