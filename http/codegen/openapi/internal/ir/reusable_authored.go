package ir

import (
	"encoding/json/v2"
	"fmt"

	"github.com/CaliLuke/loom/internal/openapiref"
)

type reusableAdapter[V, R any] struct {
	value     func(R) *V
	metadata  func(*V) []string
	inline    func(*V) R
	reference func(openapiref.Object) R
	set       func(R, openapiref.Object)
}

var (
	headerReferenceAdapter = reusableAdapter[Header, *HeaderRef]{
		value: func(r *HeaderRef) *Header {
			return r.Value
		},
		metadata: func(v *Header) []string {
			return v.ReferenceMetadata
		},
		inline: func(v *Header) *HeaderRef {
			return &HeaderRef{Value: v}
		},
		reference: func(r openapiref.Object) *HeaderRef {
			return &HeaderRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
		set: func(target *HeaderRef, r openapiref.Object) {
			*target = HeaderRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
	}
	bodyReferenceAdapter = reusableAdapter[RequestBody, *RequestBodyRef]{
		value: func(r *RequestBodyRef) *RequestBody {
			return r.Value
		},
		metadata: func(v *RequestBody) []string {
			return v.ReferenceMetadata
		},
		inline: func(v *RequestBody) *RequestBodyRef {
			return &RequestBodyRef{Value: v}
		},
		reference: func(r openapiref.Object) *RequestBodyRef {
			return &RequestBodyRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
		set: func(target *RequestBodyRef, r openapiref.Object) {
			*target = RequestBodyRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
	}
	responseReferenceAdapter = reusableAdapter[Response, *ResponseRef]{
		value: func(r *ResponseRef) *Response {
			return r.Value
		},
		metadata: func(v *Response) []string {
			return v.ReferenceMetadata
		},
		inline: func(v *Response) *ResponseRef {
			return &ResponseRef{Value: v}
		},
		reference: func(r openapiref.Object) *ResponseRef {
			return &ResponseRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
		set: func(target *ResponseRef, r openapiref.Object) {
			*target = ResponseRef{Ref: r.Ref, Description: r.Description, Summary: r.Summary}
		},
	}
)

// authoredReusableComponents allocates canonical identities before inferred
// components. All three families share edge admission and conflict detection.
func authoredReusableComponents(paths map[string]*PathItem) (map[string]*HeaderRef, map[string]*RequestBodyRef, map[string]*ResponseRef, error) {
	headersUses := collectHeaderUsages(paths)
	headers := make([]*HeaderRef, len(headersUses))
	for index, use := range headersUses {
		headers[index] = use.ref
	}
	h, err := hoistAuthoredReferences(headers, "headers", headerReferenceAdapter)
	if err != nil {
		return nil, nil, nil, err
	}
	bodiesUses := collectRequestBodyUsages(paths)
	bodies := make([]*RequestBodyRef, len(bodiesUses))
	for index, use := range bodiesUses {
		bodies[index] = use.ref
	}
	b, err := hoistAuthoredReferences(bodies, "requestBodies", bodyReferenceAdapter)
	if err != nil {
		return nil, nil, nil, err
	}
	responsesUses := collectResponseUsages(paths)
	responses := make([]*ResponseRef, len(responsesUses))
	for index, use := range responsesUses {
		responses[index] = use.ref
	}
	r, err := hoistAuthoredReferences(responses, "responses", responseReferenceAdapter)
	return h, b, r, err
}

func hoistAuthoredReferences[V, R any](refs []R, kind string, adapter reusableAdapter[V, R]) (map[string]R, error) {
	components := make(map[string]R)
	hashes := make(map[string]string)
	install := func(name string, reference R) error {
		hash, err := hashReusableValue(reference)
		if err != nil {
			return err
		}
		if previous, exists := hashes[name]; exists && previous != hash {
			return fmt.Errorf("conflicting authored %s component %q", kind, name)
		}
		hashes[name], components[name] = hash, reference
		return nil
	}
	for _, ref := range refs {
		value := adapter.value(ref)
		metadata := adapter.metadata(value)
		if len(metadata) == 0 {
			continue
		}
		if len(metadata) != 1 {
			return nil, fmt.Errorf("%s reference metadata requires one edge list", kind)
		}
		var chain []openapiref.Object
		if err := json.Unmarshal([]byte(metadata[0]), &chain, json.RejectUnknownMembers(true)); err != nil {
			return nil, fmt.Errorf("invalid %s reference metadata: %w", kind, err)
		}
		if len(chain) == 0 {
			return nil, fmt.Errorf("empty %s reference chain", kind)
		}
		names := make([]string, len(chain))
		seen := make(map[string]bool)
		for i, edge := range chain {
			name, err := openapiref.Name(edge.Ref, "#/components/"+kind+"/")
			if err != nil {
				return nil, fmt.Errorf("invalid %s reference: %w", kind, err)
			}
			if seen[name] {
				return nil, fmt.Errorf("cyclic %s reference %q", kind, name)
			}
			seen[name], names[i] = true, name
		}
		if err := install(names[len(names)-1], adapter.inline(value)); err != nil {
			return nil, err
		}
		for i := len(chain) - 1; i > 0; i-- {
			if err := install(names[i-1], adapter.reference(chain[i])); err != nil {
				return nil, err
			}
		}
		adapter.set(ref, chain[0])
	}
	return components, nil
}

func reservedReusableNames[R any](components map[string]R) map[string]bool {
	names := make(map[string]bool, len(components))
	for name := range components {
		names[name] = true
	}
	return names
}

func mergeReusableComponents[R any](authored, inferred map[string]R) map[string]R {
	if len(authored) == 0 {
		return inferred
	}
	for name, value := range inferred {
		authored[name] = value
	}
	return authored
}
