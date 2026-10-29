package expr

import (
	"fmt"
	"mime"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

// HTTPMediaPartTypes returns the effective multipart Content-Type of each
// authored object field. It validates metadata against the generated part codec.
func HTTPMediaPartTypes(body *AttributeExpr) (map[string]string, error) {
	object := AsObject(body.Type)
	if object == nil || containsNullable(body) || containsUntaggedUnion(body) {
		return nil, fmt.Errorf("multipart media requires a non-nullable object without untagged unions")
	}
	result := make(map[string]string, len(*object))
	for _, field := range *object {
		name := ElementName(field.Name)
		authored, _ := HTTPMediaMetadata(body).Last("openapi:encoding:" + name + ":contentType")
		media, err := httpencoding.MultipartContentType(httpMediaFieldShape(field.Attribute), httpMediaBaseType(field.Attribute.Type) == Bytes, authored)
		if err != nil {
			return nil, fmt.Errorf("multipart field %q: %w", name, err)
		}
		result[name] = media
	}
	return result, nil
}

func validateMediaFields(body *AttributeExpr, media string) error {
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return err
	}
	if base != "application/x-www-form-urlencoded" && base != "multipart/form-data" {
		return nil
	}
	object := AsObject(body.Type)
	if object == nil || containsNullable(body) || containsUntaggedUnion(body) {
		return fmt.Errorf("form and multipart media require non-nullable objects without untagged unions")
	}
	meta := HTTPMediaMetadata(body)
	if err := validateMediaEncodingKeys(meta, object); err != nil {
		return err
	}
	for _, field := range *object {
		prefix := "openapi:encoding:" + ElementName(field.Name) + ":"
		value := httpencoding.Serialization{}
		value.MediaType, _ = meta.Last(prefix + "contentType")
		value.Style, _ = meta.Last(prefix + "style")
		if raw, ok := meta.Last(prefix + "explode"); ok {
			explode := raw != "false"
			value.Explode = &explode
		}
		if raw, ok := meta.Last(prefix + "allowReserved"); ok {
			reserved := raw != "false"
			value.AllowReserved = &reserved
		}
		if base == "multipart/form-data" {
			value.MediaType = "" // Part media is checked against the field codec below.
		}
		if err := httpencoding.Match(httpencoding.FormField, httpMediaFieldShape(field.Attribute), value); err != nil {
			return fmt.Errorf("media field %q: %w", field.Name, err)
		}
	}
	if base == "multipart/form-data" {
		_, err := HTTPMediaPartTypes(body)
		return err
	}
	return nil
}

func httpMediaFieldShape(attribute *AttributeExpr) httpencoding.Shape {
	switch {
	case IsMap(attribute.Type):
		return httpencoding.Map
	case IsObject(attribute.Type), IsUnion(attribute.Type):
		return httpencoding.Object
	case IsArray(attribute.Type):
		if element := AsArray(attribute.Type).ElemType; IsPrimitive(element.Type) && !IsAny(element.Type) {
			return httpencoding.Sequence
		}
		return httpencoding.UnsupportedShape
	default:
		return httpencoding.Scalar
	}
}

func httpMediaBaseType(typ DataType) DataType {
	for {
		named, ok := typ.(UserType)
		if !ok {
			return typ
		}
		typ = named.Attribute().Type
	}
}

// HTTPMediaMetadata resolves occurrence-first metadata through body aliases.
// It returns a detached map so transport owners can retain the effective contract.
func HTTPMediaMetadata(attribute *AttributeExpr) MetaExpr {
	result := MetaExpr{}
	seen := make(map[*AttributeExpr]bool)
	for attribute != nil && !seen[attribute] {
		seen[attribute] = true
		for name, values := range attribute.Meta {
			if _, found := result[name]; !found {
				result[name] = append([]string(nil), values...)
			}
		}
		named, ok := attribute.Type.(UserType)
		if !ok {
			return result
		}
		attribute = named.Attribute()
	}
	return result
}

func validateMediaEncodingKeys(meta MetaExpr, object *Object) error {
	allowed := make(map[string]bool)
	for _, field := range *object {
		for _, suffix := range []string{"contentType", "style", "explode", "allowReserved"} {
			allowed["openapi:encoding:"+ElementName(field.Name)+":"+suffix] = true
		}
	}
	for key := range meta {
		if strings.HasPrefix(key, "openapi:encoding:") && !allowed[key] || strings.HasPrefix(key, "openapi:itemEncoding:") || strings.HasPrefix(key, "openapi:prefixEncoding:") {
			return fmt.Errorf("media encoding %q has no generated field contract", key)
		}
	}
	return nil
}
