package expr

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

// HTTPFormTaggedBranchEncoding captures the default native form representation
// of one branch. Object fields are flattened; other values use valueKey. The
// caller supplies the discriminator tag and validates collisions with its key.
// Unsupported forms return an error without affecting JSON-only occurrences.
func HTTPFormTaggedBranchEncoding(body *AttributeExpr, valueKey string) (*HTTPFormTaggedBranch, error) {
	captured := &HTTPFormTaggedBranch{Body: body}
	// Native tagged form methods have no authored Encoding Object input.
	for key := range HTTPMediaMetadata(body) {
		if strings.HasPrefix(key, "openapi:encoding:") {
			return nil, fmt.Errorf("does not support Encoding Object overrides")
		}
	}
	if IsObject(body.Type) {
		if IsNullable(body) {
			return nil, fmt.Errorf("flattened objects must be non-nullable")
		}
		fields, err := captureHTTPFormFields(&AttributeExpr{Type: body.Type})
		if err != nil {
			return nil, err
		}
		captured.Object = &HTTPFormEncoding{Body: body, ObjectOpenness: body.effectiveObjectOpenness(), Fields: fields}
		captured.AllowsEmpty = len(body.AllRequired()) == 0
		return captured, nil
	}
	codec, keys, err := httpencoding.Resolve(httpencoding.FormField, valueKey, HTTPValueShape(body), httpencoding.Serialization{})
	if err != nil {
		return nil, err
	}
	captured.Value = &httpencoding.FieldCodec{Name: valueKey, Codec: codec, Keys: keys}
	captured.AllowsEmpty = codec.Application == httpencoding.Items
	return captured, nil
}

func (form *HTTPFormEncoding) captureTaggedBranches(union *Union) error {
	for _, branch := range union.Values {
		captured, err := HTTPFormTaggedBranchEncoding(branch.Attribute, union.GetValueKey())
		if err != nil {
			return fmt.Errorf("tagged form branch %q: %w", branch.Name, err)
		}
		captured.Tag = UnionVariantTag(branch)
		if captured.Object != nil {
			for _, field := range captured.Object.Fields {
				if !field.Omitted && field.Name == union.GetTypeKey() {
					return fmt.Errorf("tagged form branch %q field conflicts with discriminator %q", branch.Name, union.GetTypeKey())
				}
			}
		}
		form.TaggedBranches = append(form.TaggedBranches, captured)
	}
	return nil
}
