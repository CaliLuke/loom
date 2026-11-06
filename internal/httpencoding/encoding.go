// Package httpencoding owns named URL field serialization contracts.
package httpencoding

type (
	// Shape describes the value at a URL field boundary, after alias resolution.
	Shape uint8
	// Location distinguishes named values from root URL layouts.
	Location uint8
	// Serialization describes authored OpenAPI serialization fields. Pointers
	// preserve explicit presence, which changes form encoding semantics.
	Serialization struct {
		// MediaType is the parameter content or form field contentType.
		MediaType string
		// Style is the authored RFC6570 serialization style.
		Style string
		// Explode is the explicitly authored explode setting.
		Explode *bool
		// AllowReserved is the explicitly authored URI escaping setting.
		AllowReserved *bool
	}
)

const (
	// Scalar is one primitive value, including bytes.
	Scalar Shape = iota
	// Sequence is an array of scalar values.
	Sequence
	// Nested contains values beyond one scalar collection level.
	Nested
	// Object is a fixed object value.
	Object
	// UnsupportedShape is a shape without a built-in URL codec.
	UnsupportedShape
	// ScalarMap is a dynamic object with scalar values.
	ScalarMap
	// Union contains alternatives resolved under one binding style.
	Union
)
const (
	// NamedQuery is an ordinary named query parameter.
	NamedQuery Location = iota
	// FormField is a named URL-encoded form property.
	FormField
	// WholeQuery is the unprefixed MapParams layout.
	WholeQuery
	// FormRoot is the unwrapped form body layout.
	FormRoot
	// Path is one whole path-segment binding.
	Path
	// Header is a request or response header binding.
	Header
	// Cookie is a request cookie binding.
	Cookie
)
