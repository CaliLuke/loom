package testdata

import "github.com/CaliLuke/loom/dsl"

// DestinationErrorRolesDSL shares error and payload declarations in common.
// The collision case gives the error an allocated suffix; shared reuses one
// type in both roles; ordinary preserves the non-conflicting Foo/foo control.
func DestinationErrorRolesDSL(kind string) func() {
	return func() {
		errorName, payloadName := "FooBar", "Foo-Bar"
		if kind == "ordinary" {
			errorName, payloadName = "Foo", "foo"
		}
		if kind == "shared" {
			errorName = "foo_bar"
		}
		failure := dsl.Type(errorName, func() {
			dsl.Meta("struct:pkg:path", "common")
			dsl.Attribute("message", dsl.String)
		})
		payload := failure
		if kind != "shared" {
			payload = dsl.Type(payloadName, func() {
				dsl.Meta("struct:pkg:path", "common")
				dsl.Attribute("count", dsl.Int)
			})
		}
		dsl.Service("alpha", func() {
			dsl.Method("use", func() {
				dsl.Error("authored_failure", failure)
			})
		})
		dsl.Service("beta", func() {
			dsl.Method("use", func() {
				dsl.Payload(payload)
			})
		})
	}
}
