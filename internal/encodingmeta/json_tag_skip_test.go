package encodingmeta

import (
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONAuthoredSkip(t *testing.T) {
	for _, tc := range []struct {
		name string
		meta map[string][]string
	}{
		{"full", map[string][]string{"struct:tag:json": {"-"}}},
		{"name only", map[string][]string{"struct:tag:json:name": {"-"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tag, present := JSONTag(tc.meta, "secret", JSONTagPolicy{Optional: true})
			require.True(t, present)
			typ := reflect.StructOf([]reflect.StructField{{Name: "Secret", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:"` + tag + `"`)}})
			value := reflect.New(typ).Elem()
			value.Field(0).SetString("private")
			wire, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			require.JSONEq(t, `{}`, string(wire))
		})
	}
}
