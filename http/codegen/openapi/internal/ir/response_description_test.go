package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseDescriptionsDoNotSplitComponents(t *testing.T) {
	for _, status := range []string{"200", "400", "503", "599", "default"} {
		t.Run(status, func(t *testing.T) {
			left := responseSchemaValue(&Schema{Type: "string"})
			right := responseSchemaValue(&Schema{Type: "string"})
			left.Value.Description = "first: First failure."
			right.Value.Description = "second: Second failure.\n\nthird: Third failure."
			original := *left.Value
			paths := responseDescriptionPaths(left, right, status, status)
			components := componentizeResponses(paths, nil)
			require.Len(t, components, 1)
			require.Equal(t, left.Ref, right.Ref)
			require.Equal(t, original.Description, *left.Description)
			require.Equal(t, "second: Second failure.\n\nthird: Third failure.", *right.Description)
			component := components[strings.TrimPrefix(left.Ref, ResponseComponentRefPrefix)].Value
			require.Equal(t, responseComponentDescription(status), component.Description)
			require.Equal(t, original.Content, component.Content)
			require.Nil(t, left.Value)
			// Already referenced responses are not componentized again.
			require.Empty(t, componentizeResponses(paths, nil))
		})
	}
}

func TestResponseDescriptionSharingPreservesOtherDifferences(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Response)
	}{
		{"schema", func(r *Response) {
			r.Content["application/json"].Schema.Type = "integer"
		}},
		{"media", func(r *Response) {
			r.Content["text/plain"] = r.Content["application/json"]
			delete(r.Content, "application/json")
		}},
		{"header", func(r *Response) {
			r.Headers = map[string]*HeaderRef{"X-Test": {Value: &Header{Description: "extra"}}}
		}},
		{"link", func(r *Response) {
			r.Links = map[string]*ResponseLinkRef{"next": {Value: &ResponseLink{OperationID: "next"}}}
		}},
		{"example", func(r *Response) {
			r.Content["application/json"].Example = "example"
		}},
		{"extension", func(r *Response) {
			r.Extensions = map[string]any{"x-test": true}
		}},
		{"summary", func(r *Response) {
			r.Summary = "different"
		}},
		{"omitted-description", func(r *Response) {
			r.OmitDescription = true
		}},
		{"explicit-name", func(r *Response) {
			r.ComponentName = "Other"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			left := responseSchemaValue(&Schema{Type: "string"})
			right := responseSchemaValue(&Schema{Type: "string"})
			right.Value.Description = "different description"
			tc.change(right.Value)
			components := componentizeResponses(responseDescriptionPaths(left, right, "400", "400"), nil)
			require.Len(t, components, 2)
			require.NotEqual(t, left.Ref, right.Ref)
		})
	}
}

func TestResponseDescriptionsKeepStatusesSeparate(t *testing.T) {
	left := responseSchemaValue(&Schema{Type: "string"})
	right := responseSchemaValue(&Schema{Type: "string"})
	components := componentizeResponses(responseDescriptionPaths(left, right, "400", "503"), nil)
	require.Len(t, components, 2)
	require.NotEqual(t, left.Ref, right.Ref)
}

func TestResponseDescriptionChangesKeepAutomaticName(t *testing.T) {
	build := func(description string) (string, *Response) {
		left := responseSchemaValue(&Schema{Ref: "#/components/schemas/Error"})
		right := responseSchemaValue(&Schema{Ref: "#/components/schemas/Error"})
		left.Value.ComponentName, right.Value.ComponentName = "", ""
		left.Value.Description, right.Value.Description = description, "other: Other error."
		components := componentizeResponses(responseDescriptionPaths(left, right, "503", "503"), nil)
		require.Len(t, components, 1)
		return left.Ref, components["ServiceUnavailableError"].Value
	}
	before, first := build("initial: Initial error.")
	after, second := build("added: Added error.\n\ninitial: Initial error.")
	require.Equal(t, "#/components/responses/ServiceUnavailableError", before)
	require.Equal(t, before, after)
	require.Equal(t, first, second)
}

func TestResponseDescriptionEmptyAndSingleUse(t *testing.T) {
	left := responseSchemaValue(&Schema{Type: "string"})
	right := responseSchemaValue(&Schema{Type: "string"})
	left.Value.Description = ""
	right.Value.Description = "OK response."
	componentizeResponses(responsePaths(left, right, false), nil)
	require.NotNil(t, left.Description)
	require.Empty(t, *left.Description)
	require.Nil(t, right.Description)

	single := responseSchemaValue(&Schema{Type: "string"})
	single.Value.ComponentName = ""
	paths := responsePaths(single, nil, false)
	require.Empty(t, componentizeResponses(paths, nil))
	require.NotNil(t, single.Value, "unnamed single-use responses stay inline")
}

func TestResponseDescriptionOmissionPreservesFallback(t *testing.T) {
	left := responseSchemaValue(&Schema{Type: "string"})
	right := responseSchemaValue(&Schema{Type: "string"})
	left.Value.OmitDescription, right.Value.OmitDescription = true, true
	left.Value.Description, right.Value.Description = "first fallback", "second fallback"
	original := left.Value
	components := componentizeResponses(responsePaths(left, right, false), nil)
	require.Len(t, components, 2, "3.1 fallback descriptions remain part of omitted-description shapes")
	require.Nil(t, left.Description)
	require.Nil(t, right.Description)
	require.Equal(t, "first fallback", original.Description, "componentization cannot mutate the producer")
	require.Equal(t, original, components[strings.TrimPrefix(left.Ref, ResponseComponentRefPrefix)].Value)
}

func responseDescriptionPaths(left, right *ResponseRef, leftStatus, rightStatus string) map[string]*PathItem {
	return map[string]*PathItem{
		"/left":  {Operations: map[string]*Operation{"GET": {Responses: map[string]*ResponseRef{leftStatus: left}}}},
		"/right": {Operations: map[string]*Operation{"GET": {Responses: map[string]*ResponseRef{rightStatus: right}}}},
	}
}
