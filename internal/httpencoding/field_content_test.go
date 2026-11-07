package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormFieldContentCodecs(t *testing.T) {
	scalarUnion := BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: Scalar}}}
	nestedUnion := BranchShape{Shape: Union, Branches: []BranchShape{scalarUnion, {Shape: Scalar}}}
	no := false
	for _, tc := range []struct {
		name   string
		value  BranchShape
		source Serialization
		style  Style
		err    string
	}{
		{"text scalar", BranchShape{Shape: Scalar}, Serialization{MediaType: "text/plain"}, Form, ""},
		{"text union", scalarUnion, Serialization{MediaType: "text/plain"}, Form, ""},
		{"nested text union", nestedUnion, Serialization{MediaType: "text/plain"}, Form, ""},
		{"text mixed union", BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: Sequence}}}, Serialization{MediaType: "text/plain"}, 0, "scalar"},
		{"text nullable branch", BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar, Nullable: true}, {Shape: Scalar}}}, Serialization{MediaType: "text/plain"}, 0, "nullable"},
		{"text nullable occurrence", BranchShape{Shape: Scalar, Nullable: true}, Serialization{MediaType: "text/plain"}, 0, "nullable"},
		{"text array", BranchShape{Shape: Sequence}, Serialization{MediaType: "text/plain"}, Form, ""},
		{"text object", BranchShape{Shape: Object}, Serialization{MediaType: "text/plain"}, 0, "scalar"},
		{"text empty union", BranchShape{Shape: Union}, Serialization{MediaType: "text/plain"}, 0, "branch"},
		{"JSON scalar", BranchShape{Shape: Scalar}, Serialization{MediaType: "application/json"}, JSON, ""},
		{"JSON nullable", BranchShape{Shape: Scalar, Nullable: true}, Serialization{MediaType: "application/json"}, JSON, ""},
		{"JSON sequence", BranchShape{Shape: Sequence}, Serialization{MediaType: "application/json"}, JSON, ""},
		{"JSON union", nestedUnion, Serialization{MediaType: "application/json"}, JSON, ""},
		{"JSON unsupported", BranchShape{Shape: UnsupportedShape}, Serialization{MediaType: "application/json"}, 0, "media type"},
		{"typeless default", scalarUnion, Serialization{}, 0, "application/octet-stream"},
		{"style overrides text media", BranchShape{Shape: Object}, Serialization{MediaType: "text/plain", Style: "form", Explode: &no}, Form, ""},
		{"explicit false overrides JSON", scalarUnion, Serialization{MediaType: "application/json", AllowReserved: &no}, Form, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, keys, err := Resolve(FormField, "field", tc.value, tc.source)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.style, codec.Style)
			require.NotEmpty(t, keys.Literals)
		})
	}
}

func TestMultipartJSONUnionCodec(t *testing.T) {
	value := BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar, Nullable: true}, {Shape: Object}}}
	codec, err := ResolveMultipart("field", value, Serialization{MediaType: "application/json"})
	require.NoError(t, err)
	require.Equal(t, MultipartCodec{ContentType: "application/json", Kind: JSONPart, Keys: KeySet{Literals: []string{"field"}}}, codec)
}
