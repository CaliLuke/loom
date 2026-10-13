package testdata

import "github.com/CaliLuke/loom/dsl"

// ErrorConstructorIdentityDSL exercises distinct error wire names whose preferred
// constructor names collide. The ordinary mode is an unchanged control.
func ErrorConstructorIdentityDSL(ordinary bool) {
	names := []string{"foo_bar", "FooBar"}
	if ordinary {
		names = []string{"conflict", "invalid"}
	}
	dsl.API("errors", func() {
	})
	dsl.Service("errors", func() {
		for _, name := range names {
			dsl.Error(name, func() {
				dsl.Field(1, "msg", dsl.String)
			})
		}
		dsl.Method("show", func() {
			dsl.Payload(func() {
				dsl.Field(1, "value", dsl.String)
			})
			dsl.Result(dsl.String)
			for _, name := range names {
				dsl.Error(name)
			}
			dsl.HTTP(func() {
				dsl.POST("/show")
				dsl.Response(names[0], dsl.StatusConflict)
				dsl.Response(names[1], dsl.StatusBadRequest)
			})
		})
	})
}

// ErrorConstructorNamespaceDSL combines preferred factory names, an authored
// numeric suffix and local payload type names in one package.
func ErrorConstructorNamespaceDSL() {
	leaf := dsl.Type("MakeFooBar2", func() {
		dsl.Field(1, "value", dsl.String)
	})
	payload := dsl.Type("MakeFooBar", func() {
		dsl.Field(1, "nested", leaf)
	})
	dsl.Service("errors", func() {
		for _, name := range []string{"foo_bar", "FooBar", "foo_bar2"} {
			dsl.Error(name)
		}
		dsl.Method("show", func() {
			dsl.Payload(payload)
			dsl.Result(payload)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
		})
	})
}

// ErrorConstructorExternalDSL preserves external nominal names and custom errors
// without emitted factories alongside a default error constructor.
func ErrorConstructorExternalDSL() {
	payload := dsl.Type("MakeFooBar", func() {
		dsl.Field(1, "value", dsl.String)
		dsl.Meta("struct:pkg:path", "domain")
	})
	custom := dsl.Type("CustomFailure", func() {
		dsl.Field(1, "message", dsl.String)
	})
	dsl.Service("errors", func() {
		dsl.Error("foo_bar")
		dsl.Error("custom", custom)
		dsl.Method("show", func() {
			dsl.Payload(payload)
			dsl.Result(payload)
			dsl.HTTP(func() {
				dsl.POST("/show")
				dsl.Response("foo_bar", dsl.StatusConflict)
				dsl.Response("custom", dsl.StatusBadRequest)
			})
		})
	})
}
