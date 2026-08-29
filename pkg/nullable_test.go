package loom

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullableJSONPresence(t *testing.T) {
	type document struct {
		Value Nullable[string] `json:"value"`
	}

	var missing document
	require.NoError(t, json.Unmarshal([]byte(`{}`), &missing))
	require.False(t, missing.Value.Present())

	var null document
	require.NoError(t, json.Unmarshal([]byte(`{"value":null}`), &null))
	require.True(t, null.Value.Present())
	require.True(t, null.Value.IsNull())

	var concrete document
	require.NoError(t, json.Unmarshal([]byte(`{"value":"loom"}`), &concrete))
	value, ok := concrete.Value.Value()
	require.True(t, ok)
	require.Equal(t, "loom", value)

	encoded, err := json.Marshal(document{Value: NullValue[string]()})
	require.NoError(t, err)
	require.JSONEq(t, `{"value":null}`, string(encoded))

	type optionalDocument struct {
		Value Nullable[string] `json:"value,omitzero"`
	}
	encoded, err = json.Marshal(optionalDocument{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(encoded))
	encoded, err = json.Marshal(optionalDocument{Value: NullValue[string]()})
	require.NoError(t, err)
	require.JSONEq(t, `{"value":null}`, string(encoded))

	type anything any
	type optionalNamedAnyDocument struct {
		Value Nullable[anything] `json:"value,omitzero"`
	}
	encoded, err = json.Marshal(optionalNamedAnyDocument{Value: NullValue[anything]()})
	require.NoError(t, err)
	require.JSONEq(t, `{"value":null}`, string(encoded))
}

func TestNullableAbsentMarshalFails(t *testing.T) {
	_, err := json.Marshal(Nullable[string]{})
	require.ErrorContains(t, err, "cannot marshal absent Nullable")
}

func TestNullableConcreteNilMarshalFails(t *testing.T) {
	_, err := json.Marshal(NullableValue[any](nil))
	require.ErrorIs(t, err, ErrConcreteNullNullable)
}

func TestNullableConcreteRoundTrips(t *testing.T) {
	t.Run("scalar zero", func(t *testing.T) {
		assertNullableRoundTrip(t, 0)
	})
	t.Run("scalar non-zero", func(t *testing.T) {
		assertNullableRoundTrip(t, 42)
	})
	t.Run("empty slice", func(t *testing.T) {
		assertNullableRoundTrip(t, []string{})
	})
	t.Run("non-empty slice", func(t *testing.T) {
		assertNullableRoundTrip(t, []string{"loom"})
	})
	t.Run("empty map", func(t *testing.T) {
		assertNullableRoundTrip(t, map[string]int{})
	})
	t.Run("non-empty map", func(t *testing.T) {
		assertNullableRoundTrip(t, map[string]int{"loom": 1})
	})
}

func TestNullableObjectCollectionsRoundTrip(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	type document struct {
		Items  []Nullable[item]          `json:"items"`
		Values map[string]Nullable[item] `json:"values"`
	}
	want := document{
		Items: []Nullable[item]{NullableValue(item{Name: "loom"}), NullValue[item]()},
		Values: map[string]Nullable[item]{
			"concrete": NullableValue(item{Name: "framework"}),
			"null":     NullValue[item](),
		},
	}

	encoded, err := json.Marshal(want)
	require.NoError(t, err)
	var actual document
	require.NoError(t, json.Unmarshal(encoded, &actual))
	require.Equal(t, want, actual)
}

func assertNullableRoundTrip[T any](t *testing.T, value T) {
	t.Helper()

	encoded, err := json.Marshal(NullableValue(value))
	require.NoError(t, err)

	var decoded Nullable[T]
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	actual, ok := decoded.Value()
	require.True(t, ok)
	require.Equal(t, value, actual)
}
