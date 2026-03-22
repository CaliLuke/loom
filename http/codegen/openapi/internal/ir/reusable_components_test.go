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

	require.Contains(t, doc.Components.Responses, "SharedShapeStatus200Response")
	require.Equal(t, "#/components/responses/SharedShapeStatus200Response", doc.Paths["/one"].Operations["GET"].Responses["200"].Ref)
	require.Equal(t, "#/components/responses/SharedShapeStatus200Response", doc.Paths["/two"].Operations["GET"].Responses["200"].Ref)
}

func TestComponentizeRequestBodiesUsesPublicSchemaNameWhenAvailable(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.one",
						RequestBody: &RequestBodyRef{
							Value: &RequestBody{
								Required: true,
								Content: map[string]*MediaType{
									"application/json": {
										Schema: &Schema{Ref: "#/components/schemas/Payload"},
									},
								},
							},
						},
					},
				},
			},
			"/two": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.two",
						RequestBody: &RequestBodyRef{
							Value: &RequestBody{
								Required: true,
								Content: map[string]*MediaType{
									"application/json": {
										Schema: &Schema{Ref: "#/components/schemas/Payload"},
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
				"Payload": {Type: "object"},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.RequestBodies, "PayloadRequestBody")
	require.Equal(t, "#/components/requestBodies/PayloadRequestBody", doc.Paths["/one"].Operations["POST"].RequestBody.Ref)
	require.Equal(t, "#/components/requestBodies/PayloadRequestBody", doc.Paths["/two"].Operations["POST"].RequestBody.Ref)
}

func TestComponentizeRequestBodiesUsesExplicitComponentNameWhenAvailable(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.one",
						RequestBody: &RequestBodyRef{
							Value: &RequestBody{
								ComponentName: "SearchFiltersRequest",
								Required:      true,
								Content: map[string]*MediaType{
									"application/json": {
										Schema: &Schema{Ref: "#/components/schemas/Payload"},
									},
								},
							},
						},
					},
				},
			},
			"/two": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.two",
						RequestBody: &RequestBodyRef{
							Value: &RequestBody{
								ComponentName: "SearchFiltersRequest",
								Required:      true,
								Content: map[string]*MediaType{
									"application/json": {
										Schema: &Schema{Ref: "#/components/schemas/Payload"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.RequestBodies, "SearchFiltersRequest")
	require.Equal(t, "#/components/requestBodies/SearchFiltersRequest", doc.Paths["/one"].Operations["POST"].RequestBody.Ref)
	require.Equal(t, "#/components/requestBodies/SearchFiltersRequest", doc.Paths["/two"].Operations["POST"].RequestBody.Ref)
}

func TestComponentizeParametersUsesExplicitComponentNameWhenAvailable(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one/{widgetID}": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.one",
						Parameters: []*ParameterRef{
							{Value: &Parameter{Name: "widgetID", In: "path", Required: true, ComponentName: "WidgetIDParam", Schema: &Schema{Type: "string"}}},
						},
					},
				},
			},
			"/two/{widgetID}": {
				Operations: map[string]*Operation{
					"GET": {
						OperationID: "svc.two",
						Parameters: []*ParameterRef{
							{Value: &Parameter{Name: "widgetID", In: "path", Required: true, ComponentName: "WidgetIDParam", Schema: &Schema{Type: "string"}}},
						},
					},
				},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Parameters, "WidgetIDParam")
	require.Equal(t, "#/components/parameters/WidgetIDParam", doc.Paths["/one/{widgetID}"].Operations["GET"].Parameters[0].Ref)
	require.Equal(t, "#/components/parameters/WidgetIDParam", doc.Paths["/two/{widgetID}"].Operations["GET"].Parameters[0].Ref)
}

func TestComponentizeResponsesUsesPublicSchemaNameWhenAvailable(t *testing.T) {
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
											Schema: &Schema{Ref: "#/components/schemas/Result"},
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
											Schema: &Schema{Ref: "#/components/schemas/Result"},
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
				"Result": {Type: "object"},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Responses, "ResultStatus200Response")
	require.Equal(t, "#/components/responses/ResultStatus200Response", doc.Paths["/one"].Operations["GET"].Responses["200"].Ref)
	require.Equal(t, "#/components/responses/ResultStatus200Response", doc.Paths["/two"].Operations["GET"].Responses["200"].Ref)
}

func TestComponentizeResponsesIncludesLinksInReuseHash(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.one",
						Responses: map[string]*ResponseRef{
							"202": {
								Value: &Response{
									Description: "Accepted response.",
									Content: map[string]*MediaType{
										"application/json": {
											Schema: &Schema{Ref: "#/components/schemas/Result"},
										},
									},
									Links: map[string]*ResponseLinkRef{
										"self": {
											Value: &ResponseLink{
												OperationID: "svc.show",
												Parameters: map[string]any{
													"id": "$response.body#/id",
												},
											},
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
					"POST": {
						OperationID: "svc.two",
						Responses: map[string]*ResponseRef{
							"202": {
								Value: &Response{
									Description: "Accepted response.",
									Content: map[string]*MediaType{
										"application/json": {
											Schema: &Schema{Ref: "#/components/schemas/Result"},
										},
									},
									Links: map[string]*ResponseLinkRef{
										"self": {
											Value: &ResponseLink{
												OperationID: "svc.show",
												Parameters: map[string]any{
													"id": "$response.body#/id",
												},
											},
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
				"Result": {Type: "object"},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Responses, "ResultStatus202Response")
	require.Equal(t, "#/components/responses/ResultStatus202Response", doc.Paths["/one"].Operations["POST"].Responses["202"].Ref)
	require.Equal(t, "#/components/responses/ResultStatus202Response", doc.Paths["/two"].Operations["POST"].Responses["202"].Ref)
}

func TestComponentizeResponsesUsesGenericNoContentName(t *testing.T) {
	doc := &Document{
		Paths: map[string]*PathItem{
			"/one": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.one",
						Responses: map[string]*ResponseRef{
							"204": {Value: &Response{Description: "No Content response."}},
						},
					},
				},
			},
			"/two": {
				Operations: map[string]*Operation{
					"POST": {
						OperationID: "svc.two",
						Responses: map[string]*ResponseRef{
							"204": {Value: &Response{Description: "No Content response."}},
						},
					},
				},
			},
		},
	}

	componentizeDocument(doc)

	require.Contains(t, doc.Components.Responses, "NoContentResponse")
	require.Equal(t, "#/components/responses/NoContentResponse", doc.Paths["/one"].Operations["POST"].Responses["204"].Ref)
	require.Equal(t, "#/components/responses/NoContentResponse", doc.Paths["/two"].Operations["POST"].Responses["204"].Ref)
}
