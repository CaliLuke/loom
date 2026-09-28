package testdata

import . "github.com/CaliLuke/loom/dsl"

// GRPCServiceImportAliasesDSL exercises framework import names and an alias
// suffix that is also the name of another service.
var GRPCServiceImportAliasesDSL = func() {
	GRPCServiceImportAliases("ordinary", "protojson", "protojsonsvc", "strconv", "fmt", "json", "metadata", "context", "io", "log", "loom")
}

// GRPCServiceImportAliases declares services in the given order with a shared
// union payload and result and a server-streaming result.
func GRPCServiceImportAliases(names ...string) {
	API("importaliases", func() {})
	leaf := Type("Leaf", func() {
		Field(1, "name", String)
	})
	other := Type("Other", func() {
		Field(1, "count", Int)
	})
	choice := Type("Choice", OneOf(leaf, other))
	for _, name := range names {
		Service(name, func() {
			Method("send", func() {
				Payload(choice)
				Result(choice)
				GRPC(func() {})
			})
			Method("watch", func() {
				StreamingResult(choice)
				GRPC(func() {})
			})
		})
	}
}

// GRPCServiceImportAliasCustomTypeDSL places a union branch in a package whose
// name equals the alias allocated for the service package.
var GRPCServiceImportAliasCustomTypeDSL = func() {
	API("customaliases", func() {})
	leaf := Type("Leaf", func() {
		Meta("struct:pkg:path", "protojsonsvc")
		Field(1, "name", String)
	})
	other := Type("Other", func() {
		Field(1, "count", Int)
	})
	choice := Type("Choice", OneOf(leaf, other))
	Service("protojson", func() {
		Method("send", func() {
			Payload(choice)
			Result(choice)
			GRPC(func() {})
		})
	})
}

// GRPCMetadataImportAliasesDSL exercises metadata locals next to transport
// aliases, numeric suffixes, and a custom body type's package alias.
var GRPCMetadataImportAliasesDSL = func() {
	API("metadataaliases", func() {})
	body := Type("Body", func() {
		Meta("struct:pkg:path", "protojsonsvc")
		Field(1, "value", String)
	})
	names := []string{"protojsonsvc", "protojsonsvc2", "protojsonsvc3", "loompb2", "loompb22", "loompb3"}
	value := Type("Value", func() {
		Field(1, "body", body)
		for _, name := range names {
			Attribute(name, String)
		}
	})
	for _, name := range []string{"protojson", "loom", "ordinary"} {
		Service(name, func() {
			Method("send", func() {
				Payload(value)
				Result(value)
				GRPC(func() {
					Metadata(func() {
						for _, name := range names {
							Attribute(name)
						}
					})
					Response(CodeOK, func() {
						Headers(func() {
							for _, name := range names[:3] {
								Attribute(name)
							}
						})
						Trailers(func() {
							for _, name := range names[3:] {
								Attribute(name)
							}
						})
					})
				})
			})
		})
	}
}
