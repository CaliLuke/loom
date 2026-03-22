package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComponentizeResponsesUsesStandardErrorNames(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.one",
						Responses: map[string]*ResponseRef{
							"401": {Value: &Response{Description: "Unauthorized response."}},
						},
					},
				},
			},
			"/two": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.two",
						Responses: map[string]*ResponseRef{
							"401": {Value: &Response{Description: "Unauthorized response."}},
						},
					},
				},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Responses, "UnauthorizedError")
	require.Equal(t, "#/components/responses/UnauthorizedError", doc.Paths["/one"].Operations["GET"].Responses["401"].Ref)
	require.Equal(t, "#/components/responses/UnauthorizedError", doc.Paths["/two"].Operations["GET"].Responses["401"].Ref)
}

func TestComponentizeResponsesReusesEquivalentPayloadResponsesAcrossSchemaAliases(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.one",
						Responses: map[string]*ResponseRef{
							"200": {
								Value: &Response{
									Description: "OK response.",
									Content: map[string]*MediaType{
										"application/json": {
											Schema: &Schema{Ref: "#/components/schemas/SharedShape"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/two": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.two",
						Responses: map[string]*ResponseRef{
							"200": {
								Value: &Response{
									Description: "OK response.",
									Content: map[string]*MediaType{
										"application/json": {
											Schema: &Schema{Ref: "#/components/schemas/SharedShape_2"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Components: &Components{
			Schemas: map[string]*Schema{
				"SharedShape": {
					Type: "object",
					Properties: map[string]*Schema{
						"id": {Type: "string"},
					},
					Required: []string{"id"},
				},
				"SharedShape_2": {
					Type: "object",
					Properties: map[string]*Schema{
						"id": {Type: "string"},
					},
					Required: []string{"id"},
				},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Responses, "SvcOneStatus200Response")
	require.Equal(t, "#/components/responses/SvcOneStatus200Response", doc.Paths["/one"].Operations["GET"].Responses["200"].Ref)
	require.Equal(t, "#/components/responses/SvcOneStatus200Response", doc.Paths["/two"].Operations["GET"].Responses["200"].Ref)
}
