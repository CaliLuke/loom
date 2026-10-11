package ir

import (
	"encoding/json/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzerProjectsPrimitiveTypesAndFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		type_      expr.DataType
		schemaType string
		format     string
	}{
		{name: "boolean", type_: expr.Boolean, schemaType: "boolean"},
		{name: "int", type_: expr.Int, schemaType: "integer", format: "int64"},
		{name: "int32", type_: expr.Int32, schemaType: "integer", format: "int32"},
		{name: "int64", type_: expr.Int64, schemaType: "integer", format: "int64"},
		{name: "uint", type_: expr.UInt, schemaType: "integer", format: "uint64"},
		{name: "uint32", type_: expr.UInt32, schemaType: "integer", format: "uint32"},
		{name: "uint64", type_: expr.UInt64, schemaType: "integer", format: "uint64"},
		{name: "float32", type_: expr.Float32, schemaType: "number", format: "float"},
		{name: "float64", type_: expr.Float64, schemaType: "number", format: "double"},
		{name: "string", type_: expr.String, schemaType: "string"},
		{name: "bytes", type_: expr.Bytes, schemaType: "string", format: "binary"},
		{name: "any", type_: expr.Any},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			schema := NewAnalyzer(expr.NewRandom(test.name), false).AnalyzeSchema(&expr.AttributeExpr{Type: test.type_})
			require.Equal(t, test.schemaType, schema.Type)
			require.Equal(t, test.format, schema.Format)
		})
	}
}

func TestAnalyzerAssignsUnsignedIntegerFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		type_  expr.DataType
		format string
	}{
		{name: "int", type_: expr.Int, format: "int64"},
		{name: "int32", type_: expr.Int32, format: "int32"},
		{name: "int64", type_: expr.Int64, format: "int64"},
		{name: "uint", type_: expr.UInt, format: "uint64"},
		{name: "uint32", type_: expr.UInt32, format: "uint32"},
		{name: "uint64", type_: expr.UInt64, format: "uint64"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			minimum, maximum := float64(2), float64(17)
			schema := NewAnalyzer(expr.NewRandom(test.name), false).AnalyzeSchema(&expr.AttributeExpr{
				Type: test.type_,
				Validation: &expr.ValidationExpr{
					Minimum: &minimum,
					Maximum: &maximum,
				},
			})

			require.Equal(t, "integer", schema.Type)
			require.Equal(t, test.format, schema.Format)
			require.Equal(t, &minimum, schema.Minimum)
			require.Equal(t, &maximum, schema.Maximum)
		})
	}
}

func TestAnalyzerPreservesUnsignedFormatsAndInheritedBoundsAcrossAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		type_  expr.DataType
		format string
	}{
		{name: "uint32", type_: expr.UInt32, format: "uint32"},
		{name: "uint64", type_: expr.UInt64, format: "uint64"},
		{name: "uint", type_: expr.UInt, format: "uint64"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			minimum, maximum := float64(2), float64(100)
			base := &expr.UserTypeExpr{
				TypeName: "UnsignedBase",
				UID:      "unsigned-base-" + test.name,
				AttributeExpr: &expr.AttributeExpr{
					Type:       test.type_,
					Meta:       expr.MetaExpr{"openapi:typename:canonical": {"true"}},
					Validation: &expr.ValidationExpr{Minimum: &minimum},
				},
			}
			alias := &expr.UserTypeExpr{
				TypeName: "UnsignedAlias",
				UID:      "unsigned-alias-" + test.name,
				AttributeExpr: &expr.AttributeExpr{
					Type:       base,
					Meta:       expr.MetaExpr{"openapi:typename:canonical": {"true"}},
					Validation: &expr.ValidationExpr{Maximum: &maximum},
				},
			}

			analyzer := NewAnalyzer(expr.NewRandom(test.name), false)
			schema := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: alias})
			require.Equal(t, "#/components/schemas/UnsignedAlias", schema.Ref)

			baseSchema := analyzer.Components()["UnsignedBase"]
			require.Equal(t, "integer", baseSchema.Type)
			require.Equal(t, test.format, baseSchema.Format)
			require.Equal(t, &minimum, baseSchema.Minimum)

			aliasSchema := analyzer.Components()["UnsignedAlias"]
			require.Equal(t, "#/components/schemas/UnsignedBase", aliasSchema.Ref)
			require.Equal(t, &maximum, aliasSchema.Maximum)
		})
	}
}

func TestJSONV2DecodesUnsignedIntegerBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decode   func([]byte) (uint64, error)
		valid    []string
		invalid  []string
		native64 bool
	}{
		{
			name: "uint32",
			decode: func(wire []byte) (uint64, error) {
				var value uint32
				err := json.Unmarshal(wire, &value)
				return uint64(value), err
			},
			valid:   []string{"0", "2147483647", "2147483648", "4294967295"},
			invalid: []string{"-1", "4294967296"},
		},
		{
			name: "uint64",
			decode: func(wire []byte) (uint64, error) {
				var value uint64
				err := json.Unmarshal(wire, &value)
				return value, err
			},
			valid:   []string{"0", "9223372036854775807", "9223372036854775808", "18446744073709551615"},
			invalid: []string{"-1", "18446744073709551616"},
		},
		{
			name: "native uint",
			decode: func(wire []byte) (uint64, error) {
				var value uint
				err := json.Unmarshal(wire, &value)
				return uint64(value), err
			},
			valid:    []string{"0", "9223372036854775807", "9223372036854775808", "18446744073709551615"},
			invalid:  []string{"-1", "18446744073709551616"},
			native64: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.native64 && strconv.IntSize != 64 {
				t.Skip("native uint's 64-bit boundary witness requires a 64-bit Go target")
			}

			for _, wire := range test.valid {
				value, err := test.decode([]byte(wire))
				require.NoError(t, err, "decode %s", wire)
				want, err := strconv.ParseUint(wire, 10, 64)
				require.NoError(t, err)
				require.Equal(t, want, value)
			}

			for _, wire := range test.invalid {
				_, err := test.decode([]byte(wire))
				require.Error(t, err, "decode %s", wire)
			}
		})
	}
}
