package openapiimport

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/CaliLuke/loom/internal/openapiref"
)

type reusableIndex[T any] struct {
	kind      string
	values    map[string]T
	used      map[string]bool
	reference func(T) openapiref.Object
}

type reusableIndexes struct {
	bodies    reusableIndex[RequestBody]
	responses reusableIndex[Response]
	headers   reusableIndex[Header]
}

var reusableComponentName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// resolveReusableDocument creates occurrence copies before admission. Shared
// definitions remain unchanged; reference edges retain their own annotations.
func resolveReusableDocument(document *Document) (*Document, Diagnostics) {
	indexes := indexReusableComponents(document.Components)
	result := *document
	result.Operations = slices.Clone(document.Operations)
	var diagnostics Diagnostics
	fail := func(code, path string, err error) {
		diagnostics = append(diagnostics, Diagnostic{Code: code, Path: path, Message: err.Error()})
	}
	for index := range result.Operations {
		operation := &result.Operations[index]
		path := fmt.Sprintf("#/operations/%d", index)
		if operation.RequestBody != nil {
			body, chain, err := indexes.bodies.resolve(*operation.RequestBody)
			if err != nil {
				fail("request-body-reference", path+"/requestBody", err)
			} else {
				body.Ref, body.referenceChain = "", chain
				operation.RequestBody = &body
			}
		}
		operation.Responses = slices.Clone(operation.Responses)
		for ri := range operation.Responses {
			status := &operation.Responses[ri]
			responsePath := path + "/responses/" + status.Status
			response, chain, err := indexes.responses.resolve(status.Response)
			if err != nil {
				fail("response-reference", responsePath, err)
				continue
			}
			response.Ref, response.referenceChain = "", chain
			response.Headers = slices.Clone(response.Headers)
			for hi := range response.Headers {
				named := &response.Headers[hi]
				header, chain, err := indexes.headers.resolve(named.Header)
				if err != nil {
					fail("header-reference", responsePath+"/headers/"+escapeJSONPointer(named.Name), err)
					continue
				}
				header.Ref, header.referenceChain = "", chain
				named.Header = header
			}
			status.Response = response
		}
	}
	diagnostics = append(diagnostics, indexes.bodies.unused("component-request-body")...)
	diagnostics = append(diagnostics, indexes.responses.unused("component-response")...)
	diagnostics = append(diagnostics, indexes.headers.unused("component-header")...)
	return &result, diagnostics.sorted()
}

func (index reusableIndex[T]) resolve(value T) (T, []openapiref.Object, error) {
	var chain []openapiref.Object
	seen := make(map[string]bool)
	for {
		reference := index.reference(value)
		if reference.Ref == "" {
			return value, chain, nil
		}
		name, err := openapiref.Name(reference.Ref, "#/components/"+index.kind+"/")
		if err != nil {
			return value, nil, fmt.Errorf("%s reference %q %w", index.kind, reference.Ref, err)
		}
		if !reusableComponentName.MatchString(name) {
			return value, nil, fmt.Errorf("invalid %s component name %q", index.kind, name)
		}
		if seen[name] {
			return value, nil, fmt.Errorf("cyclic %s reference %q", index.kind, name)
		}
		seen[name], index.used[name] = true, true
		target, exists := index.values[name]
		if !exists {
			return value, nil, fmt.Errorf("%s reference %q does not resolve", index.kind, reference.Ref)
		}
		reference.Ref = "#/components/" + index.kind + "/" + name
		chain = append(chain, reference)
		value = target
	}
}

func (index reusableIndex[T]) unused(code string) Diagnostics {
	var diagnostics Diagnostics
	for name := range index.values {
		if !index.used[name] {
			diagnostics = append(diagnostics, Diagnostic{Code: code, Path: "#/components/" + index.kind + "/" + escapeJSONPointer(name), Message: "unreferenced reusable components cannot be rendered faithfully"})
		}
	}
	return diagnostics
}

func indexReusableComponents(components Components) reusableIndexes {
	bodies := reusableIndex[RequestBody]{"requestBodies", map[string]RequestBody{}, map[string]bool{}, func(v RequestBody) openapiref.Object {
		return openapiref.Object{Ref: v.Ref, Description: v.ReferenceDescription, Summary: v.ReferenceSummary}
	}}
	responses := reusableIndex[Response]{"responses", map[string]Response{}, map[string]bool{}, func(v Response) openapiref.Object {
		return openapiref.Object{Ref: v.Ref, Description: v.ReferenceDescription, Summary: v.ReferenceSummary}
	}}
	headers := reusableIndex[Header]{"headers", map[string]Header{}, map[string]bool{}, func(v Header) openapiref.Object {
		return openapiref.Object{Ref: v.Ref, Description: v.ReferenceDescription, Summary: v.ReferenceSummary}
	}}
	for _, named := range components.RequestBodies {
		bodies.values[named.Name] = named.RequestBody
	}
	for _, named := range components.Responses {
		responses.values[named.Name] = named.Response
	}
	for _, named := range components.Headers {
		headers.values[named.Name] = named.Header
	}
	return reusableIndexes{bodies: bodies, responses: responses, headers: headers}
}
