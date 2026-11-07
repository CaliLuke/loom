package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceContentApplication(t *testing.T) {
	scalar := BranchShape{Shape: Scalar}
	object := BranchShape{Shape: Object}
	array := BranchShape{Shape: Sequence, Items: &scalar}
	objects := BranchShape{Shape: UnsupportedShape, Items: &object}
	nested := BranchShape{Shape: UnsupportedShape, Items: &array}
	yes, no := true, false
	for _, tc := range []struct {
		name        string
		shape       BranchShape
		source      Serialization
		application Application
		style       Style
	}{
		{"native scalar", scalar, Serialization{}, Value, Form},
		{"native array", array, Serialization{}, Items, Form},
		{"object items", objects, Serialization{}, Items, JSON},
		{"nested array items", nested, Serialization{}, Items, JSON},
		{"JSON string items", array, Serialization{MediaType: "application/json"}, Items, JSON},
		{"text string items", array, Serialization{MediaType: "text/plain"}, Items, Form},
		{"style owns array", array, Serialization{MediaType: "application/json", Style: "form", Explode: &no}, Value, Form},
		{"false flag owns array", array, Serialization{MediaType: "application/json", AllowReserved: &no}, Value, Form},
		{"typeless JSON array union", BranchShape{Shape: Union, Branches: []BranchShape{array, objects}}, Serialization{MediaType: "application/json"}, Value, JSON},
		{"source object union", BranchShape{SourceType: "object", Shape: Union, Branches: []BranchShape{object, object}}, Serialization{}, Value, JSON},
		{"source scalar union", BranchShape{SourceType: "string", Shape: Union, Branches: []BranchShape{scalar, scalar}}, Serialization{}, Value, Form},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, _, err := Resolve(FormField, "field", tc.shape, tc.source)
			require.NoError(t, err)
			require.Equal(t, tc.application, codec.Application)
			require.Equal(t, tc.style, codec.Style)
		})
	}
	for _, flags := range []Serialization{
		{MediaType: "application/json", Style: "form"},
		{MediaType: "application/json", Explode: &yes},
		{MediaType: "application/json", AllowReserved: &no},
	} {
		codec, err := ResolveMultipart("field", scalar, flags)
		require.NoError(t, err)
		require.Equal(t, "text/plain", codec.ContentType)
		require.Equal(t, TextPart, codec.Kind)
		require.NotNil(t, codec.StyleCodec)
	}
	for _, shape := range []BranchShape{
		{SourceType: "absent", Shape: Scalar},
		{SourceType: "absent", Shape: Sequence, Items: &scalar},
		{SourceType: "absent", Shape: Object},
	} {
		_, _, err := Resolve(FormField, "field", shape, Serialization{})
		require.ErrorContains(t, err, "application/octet-stream")
	}
}

func TestSourceArrayUnionContentCodecs(t *testing.T) {
	scalar := BranchShape{Shape: Scalar}
	object := BranchShape{Shape: Object}
	value := BranchShape{SourceType: "array", Shape: Union, Branches: []BranchShape{
		{Shape: Sequence, Items: &scalar},
		{Shape: UnsupportedShape, Items: &object},
	}}
	codec, keys, err := Resolve(FormField, "field", value, Serialization{})
	require.NoError(t, err)
	require.Equal(t, Items, codec.Application)
	require.Equal(t, Content, codec.Style)
	require.Equal(t, KeySet{Literals: []string{"field"}}, keys)
	require.Len(t, codec.Branches, 2)
	require.Equal(t, Form, codec.Branches[0].Codec.Style)
	require.Equal(t, JSON, codec.Branches[1].Codec.Style)
	for _, branch := range codec.Branches {
		require.Equal(t, Items, branch.Codec.Application)
		require.Equal(t, keys, branch.Keys)
	}
	multipart, err := ResolveMultipart("field", value, Serialization{})
	require.NoError(t, err)
	require.Equal(t, BranchPart, multipart.Kind)
	require.Equal(t, Items, multipart.Application)
	require.Empty(t, multipart.ContentType, "different branch labels cannot become one asserted label")
	require.Equal(t, keys, multipart.Keys)
	require.Len(t, multipart.Branches, 2)
	require.Equal(t, "text/plain", multipart.Branches[0].ContentType)
	require.Equal(t, "application/json", multipart.Branches[1].ContentType)
	require.Equal(t, "array", value.SourceType)
	require.Empty(t, value.Branches[0].SourceType, "resolution must not write inherited source facts into alternatives")
}

func TestMultipartAppliedBinaryAndKeys(t *testing.T) {
	bytes := BranchShape{Shape: Scalar, Binary: true}
	array := BranchShape{Shape: Sequence, Items: &bytes}
	codec, err := ResolveMultipart("files", array, Serialization{MediaType: "image/png"})
	require.NoError(t, err)
	require.Equal(t, Items, codec.Application)
	require.Equal(t, BinaryPart, codec.Kind)
	require.Equal(t, "image/png", codec.ContentType)
	require.Equal(t, KeySet{Literals: []string{"files"}}, codec.Keys)
	object := BranchShape{Shape: Object, Properties: []PropertyFact{{WireName: "x", Kind: String}, {WireName: "y", Kind: Integer}}}
	codec, err = ResolveMultipart("filter", object, Serialization{Style: "form", MediaType: "application/json"})
	require.NoError(t, err)
	require.Equal(t, Value, codec.Application)
	require.Equal(t, TextPart, codec.Kind)
	require.Equal(t, "text/plain", codec.ContentType)
	require.Equal(t, KeySet{Literals: []string{"x", "y"}}, codec.Keys)
	require.NotNil(t, codec.StyleCodec)
	require.Equal(t, Form, codec.StyleCodec.Style)
	require.True(t, codec.StyleCodec.Explode)
	require.Error(t, CheckNamespaces(FormField, []Binding{{Name: "filter", Keys: codec.Keys}, {Name: "x", Keys: KeySet{Literals: []string{"x"}}}}))
}

func TestAnyJSONItemAndNullableOuterArray(t *testing.T) {
	any := BranchShape{Shape: AnyValue, Nullable: true}
	array := BranchShape{Shape: UnsupportedShape, Items: &any}
	for _, source := range []Serialization{{}, {MediaType: "application/json"}} {
		form, _, formErr := Resolve(FormField, "field", array, source)
		multipart, multipartErr := ResolveMultipart("field", array, source)
		if source.MediaType == "" {
			require.ErrorContains(t, formErr, "application/octet-stream")
			require.ErrorContains(t, multipartErr, "application/octet-stream")
		} else {
			require.NoError(t, formErr)
			require.NoError(t, multipartErr)
			require.Equal(t, Items, form.Application)
			require.Equal(t, JSON, form.Style)
			require.Equal(t, Items, multipart.Application)
			require.Equal(t, JSONPart, multipart.Kind)
		}
	}
	array.Nullable = true
	_, _, err := Resolve(FormField, "field", array, Serialization{MediaType: "application/json"})
	require.ErrorContains(t, err, "nullable array")
	_, err = ResolveMultipart("field", array, Serialization{MediaType: "application/json"})
	require.ErrorContains(t, err, "nullable array")
}
