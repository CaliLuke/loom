package ir

import (
	"encoding/json/v2"
	"testing"

	"github.com/CaliLuke/loom/internal/openapiref"
	"github.com/stretchr/testify/require"
)

func TestAuthoredReusableNamesAreReserved(t *testing.T) {
	metadata := func(kind, name string) []string {
		wire, err := json.Marshal([]openapiref.Object{{Ref: "#/components/" + kind + "/" + name}}, json.Deterministic(true))
		require.NoError(t, err)
		return []string{string(wire)}
	}
	headerName := headerComponentBase("X-Trace")
	authored := &Operation{
		OperationID: "authored",
		RequestBody: &RequestBodyRef{Value: &RequestBody{ReferenceMetadata: metadata("requestBodies", "Collision"), Description: "original"}},
		Responses: map[string]*ResponseRef{"200": {Value: &Response{
			ReferenceMetadata: metadata("responses", "Collision"), Description: "original",
			Headers: map[string]*HeaderRef{"X-Trace": {Value: &Header{ReferenceMetadata: metadata("headers", headerName), Description: "original", Schema: &Schema{Type: "string"}}}},
		}}},
	}
	paths := map[string]*PathItem{"/a": {Operations: map[string]*Operation{"POST": authored}}}
	for _, path := range []string{"/b", "/c"} {
		paths[path] = &PathItem{Operations: map[string]*Operation{"POST": {
			OperationID: path,
			RequestBody: &RequestBodyRef{Value: &RequestBody{ComponentName: "Collision", Description: "inferred"}},
			Responses: map[string]*ResponseRef{"200": {Value: &Response{
				ComponentName: "Collision", Description: "inferred",
				Headers: map[string]*HeaderRef{"X-Trace": {Value: &Header{Description: "inferred", Schema: &Schema{Type: "integer"}}}},
			}}},
		}}}
	}
	doc := &Document{Paths: paths}
	require.NoError(t, componentizeDocument(doc))
	require.Equal(t, "original", doc.Components.Headers[headerName].Value.Description)
	require.Equal(t, "original", doc.Components.RequestBodies["Collision"].Value.Description)
	require.Equal(t, "original", doc.Components.Responses["Collision"].Value.Description)
	require.Len(t, doc.Components.Headers, 2)
	require.Len(t, doc.Components.RequestBodies, 2)
	require.Len(t, doc.Components.Responses, 2)
	require.NotEqual(t, authored.RequestBody.Ref, paths["/b"].Operations["POST"].RequestBody.Ref)
	require.NotEqual(t, authored.Responses["200"].Ref, paths["/b"].Operations["POST"].Responses["200"].Ref)
}

func TestAuthoredReusableReferenceAdmission(t *testing.T) {
	for _, value := range []string{"", `[]`, `[{"$ref":"#/components/headers/Wrong"}]`, `[{"$ref":"#/components/requestBodies/A"},{"$ref":"#/components/requestBodies/A"}]`, `[{"$ref":"#/components/requestBodies/A","unknown":true}]`} {
		doc := &Document{Paths: map[string]*PathItem{"/x": {Operations: map[string]*Operation{"POST": {
			RequestBody: &RequestBodyRef{Value: &RequestBody{ReferenceMetadata: []string{value}}},
		}}}}}
		require.Error(t, componentizeDocument(doc), value)
	}
}

func TestAuthoredReusableConflictingDefinitions(t *testing.T) {
	doc := &Document{Paths: map[string]*PathItem{}}
	for _, description := range []string{"first", "second"} {
		doc.Paths["/"+description] = &PathItem{Operations: map[string]*Operation{"POST": {
			RequestBody: &RequestBodyRef{Value: &RequestBody{
				ReferenceMetadata: []string{`[{"$ref":"#/components/requestBodies/Shared"}]`},
				Description:       description,
			}},
		}}}
	}
	require.ErrorContains(t, componentizeDocument(doc), `conflicting authored requestBodies component "Shared"`)
}
