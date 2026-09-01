package loom

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionalJSONPresence(t *testing.T) {
	type document struct {
		Value Optional[[]string] `json:"value,omitzero"`
	}

	var missing document
	require.NoError(t, json.Unmarshal([]byte(`{}`), &missing))
	require.False(t, missing.Value.Present())

	var null document
	require.ErrorContains(t, json.Unmarshal([]byte(`{"value":null}`), &null), "does not allow null")

	var empty document
	require.NoError(t, json.Unmarshal([]byte(`{"value":[]}`), &empty))
	value, ok := empty.Value.Value()
	require.True(t, ok)
	require.Empty(t, value)
	require.NotNil(t, value)

	encoded, err := json.Marshal(empty)
	require.NoError(t, err)
	require.JSONEq(t, `{"value":[]}`, string(encoded))

	encoded, err = json.Marshal(document{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(encoded))
}

func TestOptionalAbsentMarshalFails(t *testing.T) {
	_, err := json.Marshal(Optional[string]{})
	require.ErrorContains(t, err, "cannot marshal absent Optional")
}

func TestOptionalValue(t *testing.T) {
	optional := OptionalValue(0)
	require.True(t, optional.Present())
	value, ok := optional.Value()
	require.True(t, ok)
	require.Zero(t, value)
}

func TestOptionalConcreteNilMarshalFails(t *testing.T) {
	_, err := json.Marshal(OptionalValue[any](nil))
	require.ErrorIs(t, err, ErrNullOptional)
}

func TestOptionalMarshalJSONPreservesDeterministicMapOrdering(t *testing.T) {
	value := OptionalValue(map[string]string{
		"zulu":  "last",
		"alpha": "first",
	})

	for range 20 {
		encoded, err := json.Marshal(value, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, `{"alpha":"first","zulu":"last"}`, string(encoded))
	}
}

func TestJSONPresenceRoundTripsZeroEmptyAndNonEmptyValues(t *testing.T) {
	type child struct {
		Name string `json:"name"`
	}
	type document struct {
		Count  Optional[int]            `json:"count,omitzero"`
		Tags   Optional[[]string]       `json:"tags,omitzero"`
		Labels Optional[map[string]int] `json:"labels,omitzero"`
		Child  Optional[child]          `json:"child,omitzero"`
		Clear  Nullable[string]         `json:"clear,omitzero"`
	}

	var value document
	require.NoError(t, json.Unmarshal([]byte(`{
		"count": 0,
		"tags": [],
		"labels": {},
		"child": {"name": ""},
		"clear": null
	}`), &value))
	count, ok := value.Count.Value()
	require.True(t, ok)
	require.Zero(t, count)
	tags, ok := value.Tags.Value()
	require.True(t, ok)
	require.NotNil(t, tags)
	require.Empty(t, tags)
	labels, ok := value.Labels.Value()
	require.True(t, ok)
	require.NotNil(t, labels)
	require.Empty(t, labels)
	childValue, ok := value.Child.Value()
	require.True(t, ok)
	require.Empty(t, childValue.Name)
	require.True(t, value.Clear.IsNull())

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, `{"count":0,"tags":[],"labels":{},"child":{"name":""},"clear":null}`, string(encoded))

	value.Tags = OptionalValue([]string{"loom"})
	value.Labels = OptionalValue(map[string]int{"version": 1})
	encoded, err = json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, `{"count":0,"tags":["loom"],"labels":{"version":1},"child":{"name":""},"clear":null}`, string(encoded))

	encoded, err = json.Marshal(document{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(encoded))
}
