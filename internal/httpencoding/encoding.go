// Package httpencoding owns named URL field serialization contracts.
package httpencoding

import "fmt"

type (
	// Shape describes the value at a URL field boundary, after alias resolution.
	Shape uint8
	// Location distinguishes named values from root URL layouts.
	Location uint8
	// Mode is the wire representation selected for a value.
	Mode uint8
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
	// Map is a dynamic object, including nested values.
	Map
	// Object is a fixed object value.
	Object
	// UnsupportedShape is a shape without a built-in URL codec.
	UnsupportedShape
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
)
const (
	// Unsupported reports no equivalent built-in codec.
	Unsupported Mode = iota
	// Text encodes one scalar string.
	Text
	// Repeated encodes scalar array elements under a repeated field name.
	Repeated
	// JSON encodes one complete JSON document under a field name.
	JSON
	// Root preserves the existing unwrapped root layout.
	Root
)

// Select returns the single built-in representation for this boundary.
func Select(location Location, shape Shape) Mode {
	if location == WholeQuery || location == FormRoot {
		return Root
	}
	switch shape {
	case Scalar:
		return Text
	case Sequence:
		return Repeated
	case Map:
		return JSON
	case Object:
		if location == FormField {
			return JSON
		}
	}
	return Unsupported
}

// Match verifies that authored OpenAPI serialization is equivalent to the
// built-in codec. It never rewrites a mismatched format as a lossy import.
func Match(location Location, shape Shape, source Serialization) error {
	expected := Select(location, shape)
	if expected == Unsupported || expected == Root {
		return fmt.Errorf("value shape has no supported named field codec")
	}
	explicitStyle := source.Style != "" || source.Explode != nil || source.AllowReserved != nil
	if location == NamedQuery && source.MediaType != "" && explicitStyle {
		return fmt.Errorf("content-based parameters cannot declare style, explode or allowReserved")
	}
	actual := Unsupported
	switch {
	case source.MediaType == "application/json" && !explicitStyle:
		actual = JSON
	case source.MediaType != "" && source.MediaType != "application/json":
		return fmt.Errorf("field media type %q is unsupported", source.MediaType)
	case location == FormField && !explicitStyle && (shape == Map || shape == Object):
		actual = JSON
	case source.Style == "" || source.Style == "form":
		switch shape {
		case Scalar:
			actual = Text
		case Sequence:
			if source.Explode == nil || *source.Explode {
				actual = Repeated
			}
		}
	}
	if actual != expected {
		return fmt.Errorf("authored serialization does not match the built-in field codec")
	}
	return nil
}
