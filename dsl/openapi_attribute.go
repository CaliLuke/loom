package dsl

// ReadOnly marks an attribute schema as read-only in generated OpenAPI output.
func ReadOnly() {
	Meta("openapi:readOnly", "true")
}

// WriteOnly marks an attribute schema as write-only in generated OpenAPI output.
func WriteOnly() {
	Meta("openapi:writeOnly", "true")
}
