package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/encodingmeta"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	// HTTPMultipartEncoding captures one multipart body occurrence. Untagged
	// branches retain authored order; object fields retain binding identity.
	HTTPMultipartEncoding struct {
		// Body retains the occurrence and its local constraints.
		Body *AttributeExpr
		// ObjectOpenness governs undeclared physical part names.
		ObjectOpenness ObjectOpenness
		// Metadata retains detached occurrence metadata and inherited authored slots.
		Metadata MetaExpr
		// Fields maps full authored field identities to their part contracts.
		Fields map[string]HTTPMultipartFieldEncoding
		// Branches retains recursive untagged root alternatives in authored order.
		Branches []*HTTPMultipartEncoding
		// Custom selects the documented whole-body user codec for an ordinary
		// payload. Fields is nil in this case; new untagged occurrences never
		// select a custom codec implicitly.
		Custom bool
	}
	// HTTPMultipartFieldEncoding captures one physical part binding.
	HTTPMultipartFieldEncoding struct {
		// Name is the effective form field name, independently of JSON tags.
		Name string
		// Omitted reports an authored tag that excludes the binding.
		Omitted bool
		// Codec determines the value grammar and effective Content-Type label.
		Codec httpencoding.MultipartCodec
		// Derived supplies uploaded file metadata only when this part is absent.
		Derived *HTTPMultipartMetadata
	}
	// HTTPMultipartMetadata retains a semantic string default from one file binding.
	HTTPMultipartMetadata struct {
		// File is the full authored identity of the sole scalar byte field.
		File string
		// Kind selects the filename or Content-Type header.
		Kind HTTPMultipartMetadataKind
	}
	// HTTPMultipartMetadataKind identifies an uploaded file metadata source.
	HTTPMultipartMetadataKind uint8
)

const (
	// HTTPMultipartFilename derives the nonempty uploaded filename.
	HTTPMultipartFilename HTTPMultipartMetadataKind = iota
	// HTTPMultipartContentType derives the nonempty uploaded media label.
	HTTPMultipartContentType
)

// HTTPMultipartBodyEncoding resolves multipart representation and admission.
// New untagged roots contain only non-nullable object alternatives, and all new
// union fields must have a generated part codec. Ordinary unsupported payloads
// retain the explicit custom whole-body codec seam.
func HTTPMultipartBodyEncoding(body *AttributeExpr) (*HTTPMultipartEncoding, error) {
	if err := validateHTTPMediaSourceTypes(body, make(map[*AttributeExpr]bool)); err != nil {
		return nil, err
	}
	if containsTypedObject(body, make(map[*AttributeExpr]bool)) {
		return nil, fmt.Errorf("HTTP form and multipart bodies do not support objects with AdditionalAttributes")
	}
	return captureHTTPMultipartBody(body, nil, make(map[DataType]bool), containsUntaggedUnion(body))
}

func captureHTTPMultipartBody(
	body *AttributeExpr,
	inherited MetaExpr,
	active map[DataType]bool,
	generated bool,
) (*HTTPMultipartEncoding, error) {
	if body == nil || body.Type == nil || body.Type == Empty {
		return nil, fmt.Errorf("multipart body requires an object")
	}
	body = UnwrapInlineHTTPBody(body)
	typ := httpParameterBaseType(body.Type)
	if typ == nil || active[typ] {
		return nil, fmt.Errorf("multipart body has a recursive root alternative")
	}
	if IsNullable(body) {
		return nil, fmt.Errorf("multipart body alternatives must be non-nullable")
	}
	active[typ] = true
	defer delete(active, typ)
	metadata, err := captureHTTPMediaMetadata(body, inherited)
	if err != nil {
		return nil, err
	}
	result := &HTTPMultipartEncoding{Body: body, Metadata: metadata, ObjectOpenness: body.effectiveObjectOpenness()}
	if union, ok := typ.(*Union); ok {
		if !union.Untagged {
			return nil, fmt.Errorf("MultipartRequest requires an object payload, constructor unions are not supported")
		}
		if len(union.Values) == 0 {
			return nil, fmt.Errorf("multipart union body requires at least one alternative")
		}
		for _, branch := range union.Values {
			child, err := captureHTTPMultipartBody(branch.Attribute, metadata, active, true)
			if err != nil {
				return nil, fmt.Errorf("multipart branch %q: %w", branch.Name, err)
			}
			result.Branches = append(result.Branches, child)
		}
		return result, nil
	}
	object, isObject := typ.(*Object)
	if generated && !isObject {
		return nil, fmt.Errorf("multipart root alternatives require object bodies")
	}
	if !isObject || !multipartGeneratedObject(body) {
		if generated {
			return nil, fmt.Errorf("multipart union bodies require generated codecs for every field")
		}
		if containsNullable(body) {
			return nil, fmt.Errorf("custom multipart bodies do not support nullable attributes")
		}
		result.Custom = true
		return result, nil
	}
	scoped := *body
	scoped.Meta = metadata
	result.Fields, err = captureHTTPMultipartFields(&scoped, object)
	if err != nil {
		return nil, err
	}
	captureMultipartMetadata(result, object)
	return result, nil
}

func captureMultipartMetadata(contract *HTTPMultipartEncoding, object *Object) {
	file := ""
	for _, field := range *object {
		captured := contract.Fields[field.Name]
		if captured.Omitted || captured.Codec.Kind != httpencoding.BinaryPart || captured.Codec.Application != httpencoding.Value {
			continue
		}
		if file != "" {
			return
		}
		file = field.Name
	}
	if file == "" {
		return
	}
	for _, field := range *object {
		captured := contract.Fields[field.Name]
		if captured.Omitted || httpParameterBaseType(field.Attribute.Type) != String {
			continue
		}
		var kind HTTPMultipartMetadataKind
		switch AttributeName(field.Name) {
		case "filename":
			kind = HTTPMultipartFilename
		case "content_type":
			kind = HTTPMultipartContentType
		default:
			continue
		}
		captured.Derived = &HTTPMultipartMetadata{File: file, Kind: kind}
		contract.Fields[field.Name] = captured
	}
}

func captureHTTPMultipartFields(body *AttributeExpr, object *Object) (map[string]HTTPMultipartFieldEncoding, error) {
	fields := make(map[string]HTTPMultipartFieldEncoding, len(*object))
	var claims []httpencoding.Binding
	for _, field := range *object {
		element := ElementName(field.Name)
		name, visible := encodingmeta.FormFieldName(field.Attribute.Meta, element)
		captured := HTTPMultipartFieldEncoding{Name: name, Omitted: !visible}
		if visible {
			shape := HTTPValueShape(field.Attribute)
			// Composite JSON values exceed the text shape vocabulary. JSON
			// projection owns their complete nested shape and consuming cycles.
			if shape.Shape == httpencoding.UnsupportedShape && multipartJSONValue(field.Attribute, make(map[DataType]int), 0) {
				shape.Shape = httpencoding.Nested
			}
			source := HTTPParameterSerialization(body, "openapi:encoding:"+element+":")
			codec, err := httpencoding.ResolveMultipart(name, shape, source)
			if err != nil {
				return nil, fmt.Errorf("multipart field %q: %w", field.Name, err)
			}
			captured.Codec = codec
			claims = append(claims, httpencoding.Binding{Name: field.Name, Keys: codec.Keys})
		}
		fields[field.Name] = captured
	}
	if err := httpencoding.CheckNamespaces(httpencoding.FormField, claims); err != nil {
		return nil, err
	}
	return fields, nil
}

func multipartGeneratedObject(body *AttributeExpr) bool {
	if encodingmeta.Replacement(HTTPMediaMetadata(body)) != "" && !legacyNullableWrapper(body) {
		return false
	}
	for _, field := range *AsObject(body.Type) {
		if _, visible := encodingmeta.FormFieldName(field.Attribute.Meta, ElementName(field.Name)); !visible {
			continue
		}
		if !multipartJSONValue(field.Attribute, make(map[DataType]int), 0) {
			return false
		}
	}
	return true
}

func multipartJSONValue(attribute *AttributeExpr, active map[DataType]int, depth int) bool {
	if encodingmeta.Replacement(HTTPMediaMetadata(attribute)) != "" && !legacyNullableWrapper(attribute) {
		return false
	}
	typ := httpParameterBaseType(attribute.Type)
	if typ == nil {
		return false
	}
	if previous, seen := active[typ]; seen {
		return depth > previous
	}
	active[typ] = depth
	defer delete(active, typ)
	switch actual := typ.(type) {
	case Primitive:
		return true
	case *Object:
		for _, field := range *actual {
			if !multipartJSONValue(field.Attribute, active, depth+1) {
				return false
			}
		}
		return true
	case *Array:
		return multipartJSONValue(actual.ElemType, active, depth+1)
	case *Map:
		return multipartScalarKey(actual.KeyType) && multipartJSONValue(actual.ElemType, active, depth+1)
	case *Union:
		for _, branch := range actual.Values {
			if !multipartJSONValue(branch.Attribute, active, depth) {
				return false
			}
		}
		return len(actual.Values) > 0
	default:
		return false
	}
}

func multipartScalarKey(attribute *AttributeExpr) bool {
	primitive, ok := httpParameterBaseType(attribute.Type).(Primitive)
	return ok && primitive != Any && primitive != Bytes && !IsNullable(attribute)
}
