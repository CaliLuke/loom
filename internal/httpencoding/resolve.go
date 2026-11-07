package httpencoding

import "fmt"

type (
	// FieldCodec captures one form field's effective wire identity and grammar.
	FieldCodec struct {
		// Codec is the resolved value grammar; omitted fields have no grammar.
		Codec
		// Name is the effective outer form name after authored tag overrides.
		Name string
		// Omitted excludes this field from encoding, decoding and key claims.
		Omitted bool
		// Keys is the namespace claimed by this field's resolved grammar.
		Keys KeySet
	}

	// Style identifies a resolved parameter serialization grammar.
	Style uint8
	// ScalarKind identifies a scalar's JSON domain, independently of Go width.
	ScalarKind uint8
	// ValueDomain retains facts needed to compare cookie encodings.
	ValueDomain struct {
		// Unreserved reports that every emitted value uses RFC 3986 unreserved bytes.
		Unreserved bool
	}
	// KeyDomain retains the admitted text domain of dynamic object keys.
	KeyDomain struct {
		// Unreserved reports that every emitted key uses RFC 3986 unreserved bytes.
		Unreserved bool
	}
	// PropertyFact describes one fixed object's scalar property.
	PropertyFact struct {
		// WireName is the serialized property name.
		WireName string
		// Required distinguishes required members from optional members.
		Required bool
		// Kind is the property's JSON scalar domain.
		Kind ScalarKind
		// Domain records the property's value alphabet.
		Domain ValueDomain
	}
	// Application identifies the source value to which a content encoding applies.
	Application uint8
	// BranchShape describes a binding or union alternative after resolving aliases.
	BranchShape struct {
		// Binary identifies a byte value for multipart raw-part encoding.
		Binary bool
		// SourceType retains the effective source JSON Schema type before normalization.
		// Empty means native inference; "absent" preserves an omitted source type.
		SourceType string
		// Items retains the complete array item shape, including nested arrays/unions.
		Items *BranchShape
		// Shape is the value's structural category.
		Shape Shape
		// Nullable reports whether the value admits explicit JSON null.
		Nullable bool
		// Properties retains fixed scalar members in declaration order.
		Properties []PropertyFact
		// Keys describes a ScalarMap's dynamic key domain.
		Keys KeyDomain
		// Domain describes the scalar, sequence element or map value domain.
		Domain ValueDomain
		// Branches retains Union alternatives in declaration order.
		Branches []BranchShape
	}
	// Codec is the resolved wire grammar of a binding.
	Codec struct {
		// Application distinguishes a whole field from one content value per array item.
		Application Application
		// Branches retains alternative grammars when a union has a style or item codec.
		Branches []BranchCodec
		// Style identifies the grammar, including JSON content and root layouts.
		Style Style
		// Explode is effective and normalized for this value shape.
		Explode bool
		// Shape identifies the bound value's structural category.
		Shape Shape
	}
	// BranchCodec retains one resolved union alternative and its key ownership.
	BranchCodec struct {
		// Codec is the alternative's resolved grammar.
		Codec Codec
		// Keys is the alternative's wire-key claim.
		Keys KeySet
		// Inner contains alternatives of a nested Union.
		Inner []BranchCodec
	}
)

const (
	// Value applies the codec to the complete field value, or its style expansion.
	Value Application = iota
	// Items applies the content codec separately to each top-level array item.
	Items
)

const (
	// Simple uses comma-delimited lists and associative values.
	Simple Style = iota
	// Matrix prefixes path values with semicolon-delimited names.
	Matrix
	// Label prefixes path values with a dot.
	Label
	// Form uses URI query/form expansion rules.
	Form
	// SpaceDelimited separates collection members with spaces.
	SpaceDelimited
	// PipeDelimited separates collection members with vertical bars.
	PipeDelimited
	// DeepObject uses bracketed property names.
	DeepObject
	// CookieStyle uses raw cookie pairs and comma-delimited collections.
	CookieStyle
	// JSON encodes one complete JSON document under a field name.
	JSON
	// Root preserves the unwrapped root layout.
	Root
	// Content retains per-branch content grammars for a source-array union field.
	// Consumers must use Branches instead of treating this as one URI style.
	Content
)

const (
	// Boolean is the JSON boolean domain.
	Boolean ScalarKind = iota
	// Integer is the JSON integer domain without a carrier width.
	Integer
	// Number is the JSON number domain without a carrier width.
	Number
	// String is the JSON string domain.
	String
	// Bytes is a byte string represented as text.
	Bytes
)

// Resolve selects a codec and namespace for one named or root binding.
func Resolve(loc Location, name string, value BranchShape, source Serialization) (Codec, KeySet, error) {
	if loc == FormField && !serializationFlags(source) {
		return resolveFieldContent(name, value, source)
	}
	codec, err := effectiveCodec(loc, value, source)
	if err != nil {
		return Codec{}, KeySet{}, err
	}
	if value.Nullable && codec.Style != JSON {
		return Codec{}, KeySet{}, fmt.Errorf("nullable values require a JSON codec")
	}
	if codec.Style == JSON || codec.Style == Root {
		keys := KeySet{Literals: []string{name}}
		if codec.Style == Root {
			keys = KeySet{Residual: true}
		}
		return codec, keys, nil
	}
	if value.Shape == Union {
		branches, keys, err := resolveBranches(loc, name, value.Branches, codec)
		codec.Branches = branches
		return codec, keys, err
	}
	return resolveShape(loc, name, value, codec)
}

func effectiveCodec(loc Location, value BranchShape, source Serialization) (Codec, error) {
	codec := Codec{Shape: value.Shape}
	if loc == WholeQuery || loc == FormRoot {
		codec.Style = Root
		return codec, nil
	}
	content, err := parameterContent(loc, value, source)
	if err != nil {
		return Codec{}, err
	}
	if content {
		codec.Style = JSON
		return codec, nil
	}
	style := source.Style
	if style == "" {
		switch loc {
		case Path, Header:
			style = "simple"
		case Cookie:
			style = "cookie"
		default:
			style = "form"
		}
	}
	var allowed bool
	codec.Style, allowed = styleAtLocation(loc, style)
	if !allowed {
		return Codec{}, fmt.Errorf("style %s is not defined for this parameter location", style)
	}
	codec.Explode = codec.Style == Form || codec.Style == CookieStyle
	if source.Explode != nil {
		codec.Explode = *source.Explode
	}
	return codec, nil
}

func styleAtLocation(loc Location, name string) (Style, bool) {
	switch name {
	case "simple":
		return Simple, loc == Path || loc == Header
	case "matrix":
		return Matrix, loc == Path
	case "label":
		return Label, loc == Path
	case "form":
		return Form, loc == NamedQuery || loc == FormField || loc == Cookie
	case "spaceDelimited":
		return SpaceDelimited, loc == NamedQuery || loc == FormField
	case "pipeDelimited":
		return PipeDelimited, loc == NamedQuery || loc == FormField
	case "deepObject":
		return DeepObject, loc == NamedQuery || loc == FormField
	case "cookie":
		return CookieStyle, loc == Cookie
	default:
		return 0, false
	}
}

func resolveShape(loc Location, name string, value BranchShape, codec Codec) (Codec, KeySet, error) {
	codec.Shape = value.Shape
	if value.Shape == Nested {
		return Codec{}, KeySet{}, fmt.Errorf("nested values have no style serialization")
	}
	if value.Shape != Scalar && value.Shape != Sequence && value.Shape != Object && value.Shape != ScalarMap {
		return Codec{}, KeySet{}, fmt.Errorf("value shape has no supported parameter codec")
	}
	switch codec.Style {
	case DeepObject:
		if value.Shape != Object && value.Shape != ScalarMap {
			return Codec{}, KeySet{}, fmt.Errorf("deepObject is defined only for objects (§4.12.6)")
		}
		codec.Explode = false
	case SpaceDelimited, PipeDelimited:
		if codec.Explode {
			return Codec{}, KeySet{}, fmt.Errorf("n/a for explode: true")
		}
		if value.Shape == Scalar {
			return Codec{}, KeySet{}, fmt.Errorf("n/a for primitives")
		}
	case Form:
		if loc == Cookie && value.Shape != Scalar && codec.Explode {
			return Codec{}, KeySet{}, fmt.Errorf("form with explode uses the wrong cookie delimiter (Appendix D.1)")
		}
	}
	if value.Shape == Scalar {
		codec.Explode = codec.Style == Form || codec.Style == CookieStyle
	}
	keys, err := bindingKeys(loc, name, value, codec)
	return codec, keys, err
}

func resolveBranches(loc Location, name string, branches []BranchShape, codec Codec) ([]BranchCodec, KeySet, error) {
	if len(branches) == 0 {
		return nil, KeySet{}, fmt.Errorf("union requires at least one branch")
	}
	resolved := make([]BranchCodec, 0, len(branches))
	var keys KeySet
	for i, branch := range branches {
		if branch.Nullable && codec.Style != JSON {
			return nil, KeySet{}, fmt.Errorf("branch %d: nullable values require a JSON codec", i)
		}
		current := BranchCodec{Codec: codec}
		current.Codec.Shape = branch.Shape
		var err error
		switch {
		case branch.Shape == Union:
			current.Inner, current.Keys, err = resolveBranches(loc, name, branch.Branches, codec)
			current.Codec.Branches = current.Inner
		case codec.Style == JSON || codec.Style == Root:
			current.Keys = KeySet{Literals: []string{name}}
			if codec.Style == Root {
				current.Keys = KeySet{Residual: true}
			}
		default:
			current.Codec, current.Keys, err = resolveShape(loc, name, branch, codec)
		}
		if err != nil {
			return nil, KeySet{}, fmt.Errorf("branch %d: %w", i, err)
		}
		resolved = append(resolved, current)
		keys = mergeKeys(keys, current.Keys)
	}
	return resolved, keys, nil
}

func parameterContent(loc Location, value BranchShape, source Serialization) (bool, error) {
	shape := value.Shape
	flags := serializationFlags(source)
	switch loc {
	case FormField:
		// Content-based fields are resolved before style selection.
	case NamedQuery:
		content := source.MediaType != "" || (shape == ScalarMap || shape == Nested) && source.Style == "" && source.Explode == nil
		if content {
			if flags {
				return false, fmt.Errorf("content-based parameters cannot declare style, explode or allowReserved")
			}
			if (source.MediaType == "" || source.MediaType == "application/json") && (shape == ScalarMap || shape == Nested || shape == Union) {
				return true, nil
			}
			return false, fmt.Errorf("field media type %q has no equivalent query codec", source.MediaType)
		}
	case Path, Header, Cookie:
		if source.MediaType != "" {
			return false, fmt.Errorf("content serialization is query-only")
		}
	default:
		return false, fmt.Errorf("unsupported parameter location %d", loc)
	}
	return false, nil
}

func serializationFlags(source Serialization) bool {
	return source.Style != "" || source.Explode != nil || source.AllowReserved != nil
}

// textFieldShape checks the complete union grammar before choosing one shared
// text codec. Repeated multipart framing may additionally admit scalar lists.
func textFieldShape(value BranchShape, sequences bool) error {
	if value.Nullable {
		return fmt.Errorf("nullable values require a JSON codec")
	}
	if value.Shape == Scalar || sequences && value.Shape == Sequence {
		return nil
	}
	if value.Shape != Union {
		return fmt.Errorf("text/plain fields require scalar alternatives")
	}
	if len(value.Branches) == 0 {
		return fmt.Errorf("union requires at least one branch")
	}
	for i, branch := range value.Branches {
		if err := textFieldShape(branch, sequences); err != nil {
			return fmt.Errorf("branch %d: %w", i, err)
		}
	}
	return nil
}
