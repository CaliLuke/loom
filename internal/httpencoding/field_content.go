package httpencoding

import (
	"fmt"
	"mime"
	"strings"

	"github.com/CaliLuke/loom/internal/httpcodec"
)

// resolveFieldContent applies named Encoding Objects to source values before
// normalization can turn an explicitly array-valued property into a typeless union.
func resolveFieldContent(name string, value BranchShape, source Serialization) (Codec, KeySet, error) {
	application := Value
	if fieldSourceType(value) == "array" {
		application = Items
	}
	codec, err := fieldContentCodec(name, value, source, application)
	return codec, KeySet{Literals: []string{name}}, err
}

func fieldContentCodec(name string, value BranchShape, source Serialization, application Application) (Codec, error) {
	codec := Codec{Shape: value.Shape, Application: application}
	if application == Items {
		if value.Nullable {
			return Codec{}, fmt.Errorf("nullable array fields have no distinct item-encoding spelling for null")
		}
		if value.Shape == Union {
			if len(value.Branches) == 0 {
				return Codec{}, fmt.Errorf("union requires at least one branch")
			}
			codec.Style = Content
			for index, branch := range value.Branches {
				child, err := fieldContentCodec(name, branch, source, Items)
				if err != nil {
					return Codec{}, fmt.Errorf("branch %d: %w", index, err)
				}
				codec.Branches = append(codec.Branches, BranchCodec{Codec: child, Keys: KeySet{Literals: []string{name}}, Inner: child.Branches})
			}
			return codec, nil
		}
		item, err := fieldArrayItem(value)
		if err != nil {
			return Codec{}, err
		}
		resolved, err := fieldContentCodec(name, item, source, Value)
		if err != nil {
			return Codec{}, err
		}
		codec.Style, codec.Explode = resolved.Style, true
		return codec, nil
	}
	media := source.MediaType
	if media == "" {
		media = fieldDefaultMedia(value)
	}
	kind, err := fieldMediaKind(media)
	if err != nil {
		return Codec{}, err
	}
	switch kind {
	case httpcodec.JSON:
		if value.Shape == UnsupportedShape && value.Items == nil {
			return Codec{}, fmt.Errorf("field media type %q has no equivalent codec", media)
		}
		codec.Style = JSON
	case httpcodec.Text:
		if err := textFieldShape(value, false); err != nil {
			return Codec{}, err
		}
		codec.Style, codec.Explode = Form, true
		if value.Shape == Union {
			branches, _, err := resolveBranches(FormField, name, value.Branches, codec)
			if err != nil {
				return Codec{}, err
			}
			codec.Branches = branches
		}
	default:
		return Codec{}, fmt.Errorf("field media type %q has no equivalent codec; declare contentType or style", media)
	}
	return codec, nil
}

// fieldMediaKind validates a concrete Encoding Object media label and selects
// the shared content codec. Multipart preserves the authored label separately.
func fieldMediaKind(media string) (httpcodec.Kind, error) {
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return httpcodec.Unsupported, fmt.Errorf("invalid field content type %q", media)
	}
	if !strings.Contains(base, "/") || strings.Contains(base, "*") {
		return httpcodec.Unsupported, fmt.Errorf("field content type %q must be a concrete type/subtype", media)
	}
	return httpcodec.RequestContentType(base).Kind, nil
}

func fieldArrayItem(value BranchShape) (BranchShape, error) {
	if value.Items != nil {
		return *value.Items, nil
	}
	// Sequence already denotes a scalar collection. Other array shapes must
	// retain their item edge; a source type marker cannot invent that schema.
	if value.Shape == Sequence {
		return BranchShape{Shape: Scalar}, nil
	}
	return BranchShape{}, fmt.Errorf("array content application requires an array item schema")
}

func fieldSourceType(value BranchShape) string {
	if value.SourceType != "" {
		return value.SourceType
	}
	if value.Items != nil || value.Shape == Sequence {
		return "array"
	}
	switch value.Shape {
	case Scalar:
		return "string"
	case Object, ScalarMap, Nested:
		return "object"
	default:
		return "absent"
	}
}

func fieldDefaultMedia(value BranchShape) string {
	switch fieldSourceType(value) {
	case "string", "integer", "number", "boolean":
		return "text/plain"
	case "array", "object":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
