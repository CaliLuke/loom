package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartContentTypeRequiresConcreteMedia(t *testing.T) {
	for _, tc := range []struct {
		media string
		valid bool
	}{
		{"image/png", true},
		{`image/png; profile="*"`, true},
		{"image/*", false},
		{"*/*", false},
		{"image", false},
	} {
		t.Run(tc.media, func(t *testing.T) {
			_, err := ResolveMultipart("field", BranchShape{Shape: Scalar, Binary: true}, Serialization{MediaType: tc.media})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "concrete")
			}
		})
	}
}

func TestMultipartSerialization(t *testing.T) {
	no, yes := false, true
	for _, tc := range []struct {
		name     string
		shape    Shape
		source   Serialization
		accepted bool
	}{
		{"plain scalar", Scalar, Serialization{}, true},
		{"scalar form", Scalar, Serialization{Style: "form", Explode: &no}, true},
		{"plain list", Sequence, Serialization{}, true},
		{"repeated list", Sequence, Serialization{Style: "form", Explode: &yes}, true},
		{"comma list", Sequence, Serialization{Style: "form", Explode: &no}, true},
		{"plain object", Object, Serialization{}, true},
		{"plain map", ScalarMap, Serialization{}, true},
		{"flattened object", Object, Serialization{Style: "form"}, true},
		{"flag object", Object, Serialization{AllowReserved: &no}, true},
		{"nested JSON", Nested, Serialization{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveMultipart("field", BranchShape{Shape: tc.shape}, tc.source)
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestResolveMultipartFieldCodecs(t *testing.T) {
	no, yes := false, true
	scalarUnion := BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: Scalar}}}
	mixedUnion := BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: Object}}}
	listUnion := BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: Sequence}}}
	for _, tc := range []struct {
		name   string
		value  BranchShape
		binary bool
		source Serialization
		want   MultipartCodec
		err    string
	}{
		{"scalar default", BranchShape{Shape: Scalar}, false, Serialization{}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"repeated default", BranchShape{Shape: Sequence}, false, Serialization{}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"object default", BranchShape{Shape: Object}, false, Serialization{}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"map default", BranchShape{Shape: ScalarMap}, false, Serialization{}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"nested default", BranchShape{Shape: Nested}, false, Serialization{}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"binary default", BranchShape{Shape: Scalar}, true, Serialization{}, MultipartCodec{ContentType: "application/octet-stream", Kind: BinaryPart}, ""},
		{"binary JSON label stays raw", BranchShape{Shape: Scalar}, true, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: BinaryPart}, ""},
		{"binary image label", BranchShape{Shape: Scalar}, true, Serialization{MediaType: "image/png"}, MultipartCodec{ContentType: "image/png", Kind: BinaryPart}, ""},
		{"binary style", BranchShape{Shape: Scalar}, true, Serialization{Style: "form", Explode: &no}, MultipartCodec{ContentType: "application/octet-stream", Kind: BinaryPart}, ""},
		{"binary nullable", BranchShape{Shape: Scalar, Nullable: true}, true, Serialization{MediaType: "application/json"}, MultipartCodec{}, "non-nullable"},
		{"JSON scalar", BranchShape{Shape: Scalar}, false, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"JSON list", BranchShape{Shape: Sequence}, false, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"JSON nullable", BranchShape{Shape: Scalar, Nullable: true}, false, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"JSON nullable union", BranchShape{Shape: Union, Nullable: true, Branches: mixedUnion.Branches}, false, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"JSON suffix and parameters", mixedUnion, false, Serialization{MediaType: "application/example+json; charset=utf-8"}, MultipartCodec{ContentType: "application/example+json; charset=utf-8", Kind: JSONPart}, ""},
		{"captured object or array source default", mixedUnion, false, Serialization{MediaType: "application/json"}, MultipartCodec{ContentType: "application/json", Kind: JSONPart}, ""},
		{"typeless union", scalarUnion, false, Serialization{}, MultipartCodec{}, "declare contentType or style"},
		{"text union", scalarUnion, false, Serialization{MediaType: "text/plain"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"text nested union", BranchShape{Shape: Union, Branches: []BranchShape{scalarUnion, {Shape: Scalar}}}, false, Serialization{MediaType: "text/plain"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"text mixed union", mixedUnion, false, Serialization{MediaType: "text/plain"}, MultipartCodec{}, "scalar"},
		{"text list union", listUnion, false, Serialization{MediaType: "text/plain"}, MultipartCodec{}, "scalar"},
		{"text nullable branch", BranchShape{Shape: Union, Branches: []BranchShape{{Shape: Scalar, Nullable: true}, {Shape: Scalar}}}, false, Serialization{MediaType: "text/plain"}, MultipartCodec{}, "nullable"},
		{"text list override", BranchShape{Shape: Sequence}, false, Serialization{MediaType: "text/plain"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"text HTML label", BranchShape{Shape: Scalar}, false, Serialization{MediaType: "text/html"}, MultipartCodec{ContentType: "text/html", Kind: TextPart}, ""},
		{"plain nullable scalar", BranchShape{Shape: Scalar, Nullable: true}, false, Serialization{}, MultipartCodec{}, "nullable"},
		{"plain nullable list", BranchShape{Shape: Sequence, Nullable: true}, false, Serialization{}, MultipartCodec{}, "nullable"},
		{"styled union", scalarUnion, false, Serialization{Style: "form"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"styled list union", listUnion, false, Serialization{Style: "form", Explode: &yes}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"styled comma union", listUnion, false, Serialization{Style: "form", Explode: &no}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"styled object union", mixedUnion, false, Serialization{Style: "form"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"style overrides JSON", scalarUnion, false, Serialization{MediaType: "application/json", Style: "form"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"explicit false overrides JSON", scalarUnion, false, Serialization{MediaType: "application/json", AllowReserved: &no}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"style flattens object despite JSON", BranchShape{Shape: Object}, false, Serialization{MediaType: "application/json", Style: "form"}, MultipartCodec{ContentType: "text/plain", Kind: TextPart}, ""},
		{"unsupported media", scalarUnion, false, Serialization{MediaType: "image/png"}, MultipartCodec{}, "no equivalent"},
		{"malformed media", scalarUnion, false, Serialization{MediaType: "application/json; bad"}, MultipartCodec{}, "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.value.Binary = tc.binary
			got, err := ResolveMultipart("field", tc.value, tc.source)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			if serializationFlags(tc.source) {
				require.NotNil(t, got.StyleCodec)
				require.Equal(t, Value, got.Application)
			} else if tc.value.Shape == Sequence {
				tc.want.Application = Items
			}
			require.Equal(t, tc.want.ContentType, got.ContentType)
			require.Equal(t, tc.want.Kind, got.Kind)
			require.Equal(t, tc.want.Application, got.Application)
		})
	}
}
