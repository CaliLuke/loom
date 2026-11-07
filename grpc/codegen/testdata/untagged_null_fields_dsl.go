package testdata

import . "github.com/CaliLuke/loom/dsl"

// UntaggedNullFieldsDSL exercises native union presence across protobuf fields,
// root messages and collections. Explicit nullable branches remain unsupported.
func UntaggedNullFieldsDSL() {
	choice := Type("Choice", OneOf(Any, Boolean), Untagged)
	boxed := Type("Boxed", func() {
		Field(1, "choice", choice)
		Required("choice")
	})
	record := Type("Record", func() {
		Field(1, "required", choice)
		Field(3, "optional", choice)
		Field(5, "list", ArrayOf(boxed))
		Field(6, "map", MapOf(String, boxed))
		Required("required")
	})
	Service("nullfields", func() {
		Method("echo", func() {
			Payload(record)
			Result(record)
			GRPC(func() {
			})
		})
		Method("root", func() {
			Payload(choice)
			Result(choice)
			GRPC(func() {
			})
		})
	})
}
