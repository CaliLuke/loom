package openapiv3

import "encoding/json/v2"

type responseReference struct {
	Ref         string  `json:"$ref" yaml:"$ref"`
	Description *string `json:"description,omitzero" yaml:"description,omitempty"`
}

// MarshalJSON preserves the description override on a response reference.
func (r *ResponseRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.serialized(), json.Deterministic(true))
}

// MarshalYAML preserves the description override on a response reference.
func (r *ResponseRef) MarshalYAML() (any, error) {
	return r.serialized(), nil
}

// UnmarshalJSON decodes an inline response or a reference with its description.
func (r *ResponseRef) UnmarshalJSON(data []byte) error {
	var ref responseReference
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}
	value := ResponseRef{Ref: ref.Ref}
	if ref.Ref != "" {
		value.Description = ref.Description
	} else if err := json.Unmarshal(data, &value.Value); err != nil {
		return err
	}
	*r = value
	return nil
}

// UnmarshalYAML decodes an inline response or a reference with its description.
func (r *ResponseRef) UnmarshalYAML(unmarshal func(any) error) error {
	var ref responseReference
	if err := unmarshal(&ref); err != nil {
		return err
	}
	value := ResponseRef{Ref: ref.Ref}
	if ref.Ref != "" {
		value.Description = ref.Description
	} else if err := unmarshal(&value.Value); err != nil {
		return err
	}
	*r = value
	return nil
}

func (r *ResponseRef) serialized() any {
	if r.Ref != "" {
		return &responseReference{Ref: r.Ref, Description: r.Description}
	}
	return r.Value
}
