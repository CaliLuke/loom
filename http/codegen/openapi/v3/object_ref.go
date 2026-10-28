package openapiv3

import "encoding/json/v2"

type objectReference struct {
	Ref         string  `json:"$ref" yaml:"$ref"`
	Description *string `json:"description,omitzero" yaml:"description,omitempty"`
	Summary     *string `json:"summary,omitzero" yaml:"summary,omitempty"`
}

func serializedObjectReference(ref string, description, summary *string, value any) any {
	if ref != "" {
		return &objectReference{Ref: ref, Description: description, Summary: summary}
	}
	return value
}

func decodeObjectReference[V any](unmarshal func(any) error) (objectReference, *V, error) {
	var ref objectReference
	if err := unmarshal(&ref); err != nil {
		return ref, nil, err
	}
	var value *V
	if ref.Ref == "" {
		if err := unmarshal(&value); err != nil {
			return ref, nil, err
		}
	}
	return ref, value, nil
}

// MarshalJSON preserves present reference annotations and inline object content.
func (r *ResponseRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), json.Deterministic(true))
}

// MarshalYAML preserves present reference annotations and inline object content.
func (r *ResponseRef) MarshalYAML() (any, error) {
	return serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), nil
}

// UnmarshalJSON decodes an inline object or a reference with occurrence annotations.
func (r *ResponseRef) UnmarshalJSON(data []byte) error {
	return r.UnmarshalYAML(func(value any) error {
		return json.Unmarshal(data, value)
	})
}

// UnmarshalYAML decodes an inline object or a reference with occurrence annotations.
func (r *ResponseRef) UnmarshalYAML(unmarshal func(any) error) error {
	ref, value, err := decodeObjectReference[Response](unmarshal)
	if err != nil {
		return err
	}
	*r = ResponseRef{Ref: ref.Ref, Value: value}
	if ref.Ref != "" {
		r.Description, r.Summary = ref.Description, ref.Summary
	}
	return nil
}

// MarshalJSON preserves present reference annotations and inline object content.
func (r *HeaderRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), json.Deterministic(true))
}

// MarshalYAML preserves present reference annotations and inline object content.
func (r *HeaderRef) MarshalYAML() (any, error) {
	return serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), nil
}

// UnmarshalJSON decodes an inline object or a reference with occurrence annotations.
func (r *HeaderRef) UnmarshalJSON(data []byte) error {
	return r.UnmarshalYAML(func(value any) error {
		return json.Unmarshal(data, value)
	})
}

// UnmarshalYAML decodes an inline object or a reference with occurrence annotations.
func (r *HeaderRef) UnmarshalYAML(unmarshal func(any) error) error {
	ref, value, err := decodeObjectReference[Header](unmarshal)
	if err != nil {
		return err
	}
	*r = HeaderRef{Ref: ref.Ref, Value: value}
	if ref.Ref != "" {
		r.Description, r.Summary = ref.Description, ref.Summary
	}
	return nil
}

// MarshalJSON preserves present reference annotations and inline object content.
func (r *RequestBodyRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), json.Deterministic(true))
}

// MarshalYAML preserves present reference annotations and inline object content.
func (r *RequestBodyRef) MarshalYAML() (any, error) {
	return serializedObjectReference(r.Ref, r.Description, r.Summary, r.Value), nil
}

// UnmarshalJSON decodes an inline object or a reference with occurrence annotations.
func (r *RequestBodyRef) UnmarshalJSON(data []byte) error {
	return r.UnmarshalYAML(func(value any) error {
		return json.Unmarshal(data, value)
	})
}

// UnmarshalYAML decodes an inline object or a reference with occurrence annotations.
func (r *RequestBodyRef) UnmarshalYAML(unmarshal func(any) error) error {
	ref, value, err := decodeObjectReference[RequestBody](unmarshal)
	if err != nil {
		return err
	}
	*r = RequestBodyRef{Ref: ref.Ref, Value: value}
	if ref.Ref != "" {
		r.Description, r.Summary = ref.Description, ref.Summary
	}
	return nil
}
