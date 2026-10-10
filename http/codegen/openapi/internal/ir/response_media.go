package ir

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/textproto"
	"reflect"
	"slices"
	"strings"
)

type responseAlternative struct {
	name     string
	response *Response
}

// mergeResponseAlternatives projects runtime alternatives onto OpenAPI's one
// response per status. Disjoint media remain independent; shared media and
// headers retain schema alternatives. Conflicting metadata is rejected.
// No source response is mutated.
func mergeResponseAlternatives(alternatives []responseAlternative) (*Response, error) {
	if len(alternatives) == 1 {
		return alternatives[0].response, nil
	}
	slices.SortFunc(alternatives, func(a, b responseAlternative) int {
		return strings.Compare(a.name, b.name)
	})
	result := &Response{Content: make(map[string]*MediaType), Headers: make(map[string]*HeaderRef)}
	var descriptions, summaries []string
	headerDescriptions := make(map[string][]string)
	for index, alternative := range alternatives {
		response := alternative.response
		descriptions = append(descriptions, response.Description)
		summaries = append(summaries, response.Summary)
		if index > 0 && result.OmitDescription != response.OmitDescription {
			return nil, fmt.Errorf("conflicting response description omission for %q", alternative.name)
		}
		result.OmitDescription = response.OmitDescription
		if result.ComponentName != "" && response.ComponentName != "" && result.ComponentName != response.ComponentName {
			return nil, fmt.Errorf("conflicting response component names %q and %q", result.ComponentName, response.ComponentName)
		}
		if response.ComponentName != "" {
			result.ComponentName = response.ComponentName
		}
		var err error
		result.Links, err = mergeResponseMap(result.Links, response.Links, "link")
		if err != nil {
			return nil, err
		}
		result.Extensions, err = mergeResponseMap(result.Extensions, response.Extensions, "extension")
		if err != nil {
			return nil, err
		}
		if err := mergeResponseContent(result.Content, alternative); err != nil {
			return nil, err
		}
		if err := mergeResponseHeaders(result.Headers, alternative, index, headerDescriptions); err != nil {
			return nil, err
		}
	}
	result.Description = joinedResponseText(descriptions)
	result.Summary = joinedResponseText(summaries)
	for name, descriptions := range headerDescriptions {
		result.Headers[name].Value.Description = joinedResponseText(descriptions)
	}
	return result, nil
}

func mergeResponseMap[T any](target, source map[string]T, kind string) (map[string]T, error) {
	if len(source) == 0 {
		return target, nil
	}
	if target == nil {
		target = make(map[string]T)
	}
	for _, name := range slices.Sorted(maps.Keys(source)) {
		value := source[name]
		if previous, ok := target[name]; ok && !reflect.DeepEqual(previous, value) {
			return nil, fmt.Errorf("conflicting response %s %q", kind, name)
		}
		target[name] = value
	}
	return target, nil
}

func joinedResponseText(values []string) string {
	values = slices.Clone(values)
	slices.Sort(values)
	values = slices.Compact(values)
	values = slices.DeleteFunc(values, func(value string) bool {
		return value == ""
	})
	return strings.Join(values, "\n\n")
}

func responseExamples(name string, example any, examples map[string]*ExampleRef) map[string]*ExampleRef {
	result := make(map[string]*ExampleRef)
	if example != nil {
		result[name] = &ExampleRef{Value: &Example{Value: example}}
	}
	for _, key := range slices.Sorted(maps.Keys(examples)) {
		result[name+"/"+key] = examples[key]
	}
	return result
}

func addResponseExamples(target, source map[string]*ExampleRef) {
	for _, name := range slices.Sorted(maps.Keys(source)) {
		key := name
		for suffix := 2; target[key] != nil; suffix++ {
			key = fmt.Sprintf("%s.%d", name, suffix)
		}
		target[key] = source[name]
	}
}

// mergeResponseSchemas uses anyOf because designed error shapes may overlap.
// Flatten only the empty wrapper this function creates, and order exact schema
// encodings so declaration order cannot choose the public alternative order.
func mergeResponseSchemas(left, right *Schema) (*Schema, error) {
	if left == nil || right == nil {
		return nil, nil
	}
	if reflect.DeepEqual(left, right) {
		return left, nil
	}
	unique := make(map[string]*Schema)
	for _, schema := range []*Schema{left, right} {
		branches := []*Schema{schema}
		if len(schema.AnyOf) > 0 && reflect.DeepEqual(*schema, Schema{AnyOf: schema.AnyOf}) {
			branches = schema.AnyOf
		}
		for _, branch := range branches {
			encoded, err := json.Marshal(branch, json.Deterministic(true))
			if err != nil {
				return nil, fmt.Errorf("encode response schema alternative: %w", err)
			}
			unique[string(encoded)] = branch
		}
	}
	branches := make([]*Schema, 0, len(unique))
	for _, key := range slices.Sorted(maps.Keys(unique)) {
		branches = append(branches, unique[key])
	}
	return &Schema{AnyOf: branches}, nil
}

func mergeResponseContent(target map[string]*MediaType, alternative responseAlternative) error {
	var err error
	for _, mediaType := range slices.Sorted(maps.Keys(alternative.response.Content)) {
		media := alternative.response.Content[mediaType]
		previous := target[mediaType]
		if previous == nil {
			copy := *media
			copy.Example = nil
			copy.Examples = responseExamples(alternative.name, media.Example, media.Examples)
			target[mediaType] = &copy
			continue
		}
		left, right := *previous, *media
		left.Schema, right.Schema = nil, nil
		left.Example, left.Examples = nil, nil
		right.Example, right.Examples = nil, nil
		if !reflect.DeepEqual(left, right) {
			return fmt.Errorf("conflicting response media type %q for %q", mediaType, alternative.name)
		}
		previous.Schema, err = mergeResponseSchemas(previous.Schema, media.Schema)
		if err != nil {
			return err
		}
		addResponseExamples(previous.Examples, responseExamples(alternative.name, media.Example, media.Examples))
	}
	return nil
}

func mergeResponseHeaders(target map[string]*HeaderRef, alternative responseAlternative, index int, headerDescriptions map[string][]string) error {
	var err error
	currentHeaders := make(map[string]bool)
	for _, name := range slices.Sorted(maps.Keys(alternative.response.Headers)) {
		key := textproto.CanonicalMIMEHeaderKey(name)
		currentHeaders[key] = true
		header := alternative.response.Headers[name]
		if header.Value != nil {
			headerDescriptions[key] = append(headerDescriptions[key], header.Value.Description)
		}
		previous := target[key]
		if previous == nil {
			copy := *header
			if header.Value != nil {
				value := *header.Value
				value.Required = index == 0 && value.Required
				value.Example = nil
				value.Examples = responseExamples(alternative.name, header.Value.Example, header.Value.Examples)
				copy.Value = &value
			}
			target[key] = &copy
			continue
		}
		if previous.Ref != header.Ref || previous.Value == nil || header.Value == nil {
			if !reflect.DeepEqual(previous, header) {
				return fmt.Errorf("conflicting response header %q", key)
			}
			continue
		}
		left, right := *previous.Value, *header.Value
		left.Required, right.Required = false, false
		left.Description, right.Description = "", ""
		left.Schema, right.Schema = nil, nil
		left.Example, left.Examples = nil, nil
		right.Example, right.Examples = nil, nil
		if !reflect.DeepEqual(left, right) {
			return fmt.Errorf("conflicting response header %q", key)
		}
		previous.Value.Schema, err = mergeResponseSchemas(previous.Value.Schema, header.Value.Schema)
		if err != nil {
			return err
		}
		previous.Value.Required = previous.Value.Required && header.Value.Required
		addResponseExamples(previous.Value.Examples, responseExamples(alternative.name, header.Value.Example, header.Value.Examples))
	}
	for name, header := range target {
		if !currentHeaders[name] && header.Value != nil {
			header.Value.Required = false
		}
	}
	return nil
}
