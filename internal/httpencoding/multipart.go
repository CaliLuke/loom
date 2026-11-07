package httpencoding

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/httpcodec"
)

type (
	// PartKind identifies a multipart field's concrete value representation.
	PartKind uint8
	// MultipartCodec captures the grammar and media label of one multipart field.
	// Consumers must use Kind rather than infer the grammar from the field shape
	// or ContentType: binary fields retain raw bytes under custom media labels.
	MultipartCodec struct {
		// Keys is the complete physical namespace claimed by this field.
		Keys KeySet
		// StyleCodec retains explicit RFC6570 framing; nil selects content encoding.
		StyleCodec *Codec
		// Branches retains ordered item codecs for a source-array union.
		Branches []MultipartCodec
		// Application distinguishes one whole value from one part per array item.
		Application Application
		// ContentType is the concrete media label written on each part.
		ContentType string
		// Kind selects text, JSON, or raw byte encoding and decoding.
		Kind PartKind
	}
)

const (
	// TextPart encodes scalars as text and scalar lists as repeated text parts.
	TextPart PartKind = iota
	// JSONPart encodes one complete JSON document in a single part.
	JSONPart
	// BinaryPart encodes raw bytes with the authored concrete media label.
	BinaryPart
	// BranchPart selects a codec from Branches using original union membership.
	BranchPart
)

// ResolveMultipart captures named multipart encoding from source type facts.
// Content encoding applies once per item of an explicitly array-valued property.
// Typeless unions retain whole-value application. Explicit serialization flags
// select RFC6570 framing and suppress authored contentType, as in OpenAPI 3.2.
// Binary values remain raw parts; their identity follows the applied value,
// including array items. Branches retain heterogeneous source-array codecs.
func ResolveMultipart(name string, value BranchShape, source Serialization) (MultipartCodec, error) {
	if serializationFlags(source) {
		style, keys, err := Resolve(FormField, name, value, source)
		if err != nil {
			return MultipartCodec{}, err
		}
		kind, media := TextPart, "text/plain"
		if value.Binary {
			kind, media = BinaryPart, "application/octet-stream"
		}
		return MultipartCodec{Kind: kind, ContentType: media, StyleCodec: &style, Keys: keys}, nil
	}
	application := Value
	if fieldSourceType(value) == "array" {
		application = Items
	}
	return multipartAppliedCodec(name, value, source, application)
}

func multipartAppliedCodec(name string, value BranchShape, source Serialization, application Application) (MultipartCodec, error) {
	if application == Items {
		if value.Nullable {
			return MultipartCodec{}, fmt.Errorf("nullable array fields have no distinct item-encoding spelling for null")
		}
		if value.Shape == Union {
			if len(value.Branches) == 0 {
				return MultipartCodec{}, fmt.Errorf("union requires at least one branch")
			}
			result := MultipartCodec{Kind: BranchPart, Application: Items}
			for index, branch := range value.Branches {
				child, err := multipartAppliedCodec(name, branch, source, Items)
				if err != nil {
					return MultipartCodec{}, fmt.Errorf("branch %d: %w", index, err)
				}
				result.Branches = append(result.Branches, child)
				result.Keys = mergeKeys(result.Keys, child.Keys)
			}
			return result, nil
		}
		item, err := fieldArrayItem(value)
		if err != nil {
			return MultipartCodec{}, err
		}
		codec, err := multipartAppliedCodec(name, item, source, Value)
		codec.Application = Items
		return codec, err
	}
	media := source.MediaType
	if media == "" {
		media = fieldDefaultMedia(value)
		if value.Binary {
			media = "application/octet-stream"
		}
	}
	kind, err := fieldMediaKind(media)
	if err != nil {
		return MultipartCodec{}, err
	}
	if value.Binary {
		if value.Shape != Scalar || value.Nullable {
			return MultipartCodec{}, fmt.Errorf("binary multipart fields require non-nullable scalar bytes")
		}
		return MultipartCodec{ContentType: media, Kind: BinaryPart, Keys: KeySet{Literals: []string{name}}}, nil
	}
	switch kind {
	case httpcodec.JSON:
		if value.Shape == UnsupportedShape && value.Items == nil {
			return MultipartCodec{}, fmt.Errorf("value shape has no supported multipart JSON codec")
		}
		return MultipartCodec{ContentType: media, Kind: JSONPart, Keys: KeySet{Literals: []string{name}}}, nil
	case httpcodec.Text:
		if err := textFieldShape(value, false); err != nil {
			return MultipartCodec{}, err
		}
		return MultipartCodec{ContentType: media, Kind: TextPart, Keys: KeySet{Literals: []string{name}}}, nil
	default:
		return MultipartCodec{}, fmt.Errorf("multipart content type %q has no equivalent codec; declare contentType or style", media)
	}
}
