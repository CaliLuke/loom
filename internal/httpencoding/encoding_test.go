package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerializationContract(t *testing.T) {
	yes, no := true, false
	for _, test := range []struct {
		name     string
		location Location
		shape    Shape
		source   Serialization
		mode     Mode
		accepted bool
	}{
		{"query scalar", NamedQuery, Scalar, Serialization{}, Text, true},
		{"query repeated", NamedQuery, Sequence, Serialization{Style: "form", Explode: &yes}, Repeated, true},
		{"query comma", NamedQuery, Sequence, Serialization{Style: "form", Explode: &no}, Repeated, false},
		{"query JSON map", NamedQuery, Map, Serialization{MediaType: "application/json"}, JSON, true},
		{"query implicit map", NamedQuery, Map, Serialization{}, JSON, false},
		{"query brackets", NamedQuery, Map, Serialization{Style: "deepObject", Explode: &yes}, JSON, false},
		{"query fixed object", NamedQuery, Object, Serialization{MediaType: "application/json"}, Unsupported, false},
		{"query JSON scalar", NamedQuery, Scalar, Serialization{MediaType: "application/json"}, Text, false},
		{"query content style", NamedQuery, Scalar, Serialization{MediaType: "application/json", Style: "form"}, Text, false},
		{"form object default", FormField, Object, Serialization{}, JSON, true},
		{"form map JSON", FormField, Map, Serialization{MediaType: "application/json"}, JSON, true},
		{"form flattened", FormField, Object, Serialization{Style: "form", Explode: &yes}, JSON, false},
		{"form style overrides media", FormField, Map, Serialization{MediaType: "application/json", Style: "form"}, JSON, false},
		{"form reserved explicit", FormField, Object, Serialization{AllowReserved: &no}, JSON, false},
		{"query reserved permitted", NamedQuery, Scalar, Serialization{AllowReserved: &yes}, Text, true},
		{"query escaping explicit", NamedQuery, Scalar, Serialization{AllowReserved: &no}, Text, true},
		{"root query", WholeQuery, Map, Serialization{}, Root, false},
		{"root form", FormRoot, Object, Serialization{}, Root, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.mode, Select(test.location, test.shape))
			err := Match(test.location, test.shape, test.source)
			if test.accepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
