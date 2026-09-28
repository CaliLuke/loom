package ir

import (
	"encoding/base64"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOpenAPIByteValuesMatchJSONEncoding(t *testing.T) {
	allBytes := make([]byte, 256)
	for index := range allBytes {
		allBytes[index] = byte(index)
	}
	for _, bytes := range [][]byte{{}, []byte("archive"), {0, 255}, allBytes} {
		encoded := base64.StdEncoding.EncodeToString(bytes)
		for _, tc := range []struct {
			name     string
			datatype expr.DataType
			value    any
			expected any
		}{
			{"bytes", expr.Bytes, bytes, encoded},
			{"text-authored-bytes", expr.Bytes, string(bytes), encoded},
			{"named-bytes", &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}, bytes, encoded},
			{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, [][]byte{bytes}, []any{encoded}},
			{"object", &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Bytes}}}, map[string]any{"data": bytes}, map[string]any{"data": encoded}},
			{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, map[string][]byte{"data": bytes}, map[string]any{"data": encoded}},
		} {
			t.Run(tc.name+"/"+encoded, func(t *testing.T) {
				attribute := &expr.AttributeExpr{Type: tc.datatype, DefaultValue: tc.value, Validation: &expr.ValidationExpr{Values: []any{tc.value}}}
				analyzer := NewAnalyzer(expr.NewRandom("byte-values"), false, WithExampleValue(OpenAPIExampleValue))
				var schema *Schema
				require.NotPanics(t, func() {
					schema = analyzer.AnalyzeSchema(attribute, true)
				})
				require.Equal(t, []any{tc.expected}, schema.Enum)
				require.Equal(t, tc.expected, schema.DefaultValue)
				value, ok := OpenAPIExampleValue(attribute, tc.value)
				require.True(t, ok)
				require.Equal(t, tc.expected, value)
				actualJSON, err := json.Marshal(value, json.Deterministic(true))
				require.NoError(t, err)
				expectedJSON, err := json.Marshal(tc.expected, json.Deterministic(true))
				require.NoError(t, err)
				require.Equal(t, expectedJSON, actualJSON)
			})
		}
	}
}

func TestRawBinaryMediaExamplesPreserveSerializedBytes(t *testing.T) {
	for _, datatype := range []expr.DataType{
		expr.Bytes,
		&expr.UserTypeExpr{TypeName: "RawBlob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}},
	} {
		for _, tc := range []struct {
			name  string
			bytes []byte
			valid bool
		}{
			{"empty", []byte{}, true},
			{"ascii", []byte("archive"), true},
			{"utf8", []byte("µ"), true},
			{"binary", []byte{0, 255}, false},
		} {
			t.Run(datatype.Name()+"/"+tc.name, func(t *testing.T) {
				encoded := base64.StdEncoding.EncodeToString(tc.bytes)
				schema := &Schema{Example: encoded, Enum: []any{encoded}}
				media := &MediaType{
					Schema:  schema,
					Example: encoded,
					Examples: map[string]*ExampleRef{
						"literal":    {Value: &Example{Value: encoded}},
						"data":       {Value: &Example{DataValue: encoded}},
						"serialized": {Value: &Example{Value: encoded, SerializedValue: "authored serialized value"}},
					},
				}
				serializeBinaryMediaExamples(media, &expr.AttributeExpr{Type: datatype})
				if tc.valid {
					require.Equal(t, string(tc.bytes), media.Example)
					require.Equal(t, string(tc.bytes), media.Examples["literal"].Value.Value)
				} else {
					require.Nil(t, media.Example)
					require.NotContains(t, media.Examples, "literal")
				}
				require.Same(t, schema, media.Schema)
				require.Equal(t, encoded, schema.Example)
				require.Equal(t, []any{encoded}, schema.Enum)
				require.Equal(t, encoded, media.Examples["data"].Value.DataValue)
				require.Contains(t, media.Examples, "serialized")
				require.Equal(t, "authored serialized value", media.Examples["serialized"].Value.SerializedValue)
			})
		}
	}
}

func TestOpenAPIByteEnumNilMatchesEmptyJSONBytes(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:       expr.Bytes,
		Validation: &expr.ValidationExpr{Values: []any{[]byte(nil)}},
	}
	schema := NewAnalyzer(expr.NewRandom("nil-byte-enum"), false).AnalyzeSchema(attribute)
	require.Equal(t, []any{""}, schema.Enum)
}

func TestOpenAPIEnumExamplesUseDeclaredValueCoercions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
		value    any
	}{
		{"float32", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float32}}, []float64{1.23456789}},
		{"struct-bytes", &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Bytes}}}}}, []struct {
			Data string `json:"data"`
		}{{Data: "\x00\xff"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: tc.datatype, DefaultValue: tc.value, Validation: &expr.ValidationExpr{Values: []any{tc.value}}}
			analyzer := NewAnalyzer(expr.NewRandom("enum-examples"), false, WithExampleValue(OpenAPIExampleValue))
			schema := analyzer.AnalyzeSchema(attribute)
			require.Len(t, schema.Enum, 1)
			encodedEnum, err := json.Marshal(schema.Enum[0], json.Deterministic(true))
			require.NoError(t, err)
			encodedExample, err := json.Marshal(schema.Example, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, string(encodedEnum), string(encodedExample))
			encodedDefault, err := json.Marshal(schema.DefaultValue, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, string(encodedEnum), string(encodedDefault))
		})
	}
}

func TestOpenAPIExamplesWithPointerAndNestedCollectionFields(t *testing.T) {
	bytes := []byte{0, 255}
	array := []*[]byte{&bytes}
	mapping := map[string]*[]byte{"blob": &bytes}
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
		value    any
		expected any
	}{
		{"bytes-pointer", expr.Bytes, &bytes, "AP8="},
		{"nil-bytes-pointer", expr.Bytes, (*[]byte)(nil), nil},
		{"array-pointer", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, &array, []any{"AP8="}},
		{"map-pointer", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, &mapping, map[string]any{"blob": "AP8="}},
		{"unexpected-bytes", expr.Bytes, 42, 42},
		{"unexpected-array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, 42, 42},
		{"unexpected-map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, 42, 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := struct {
				Data any `json:"data"`
			}{Data: tc.value}
			root := codegen.RunDSL(t, func() {
				dsl.Service("example", func() {
					dsl.Method("send", func() {
						dsl.Payload(func() {
							dsl.Attribute("data", tc.datatype)
							dsl.Example(value)
						})
						dsl.HTTP(func() {
							dsl.POST("/")
						})
					})
				})
			})
			require.NotPanics(t, func() {
				_, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(OpenAPIExampleValue))
				require.NoError(t, err)
			})
			attribute := root.Service("example").Method("send").Payload
			projected, ok := OpenAPIExampleValue(attribute, value)
			require.True(t, ok)
			require.Equal(t, map[string]any{"data": tc.expected}, projected)
		})
	}
}
