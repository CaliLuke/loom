package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCFieldNumberAdmission(t *testing.T) {
	for _, tag := range []string{"0", "-3", "abc", "1.5", "19000", "19500", "19999", "536870912", "18446744073709551615", "18446744073709551616"} {
		t.Run(tag, func(t *testing.T) {
			fields := &Object{rpcNumberField("value", tag)}
			err := validateRPCTags(fields, grpcEndpointForTagValidationTest())
			require.NotEmpty(t, err.Errors, "invalid field number must fail at expression admission")
			require.Contains(t, err.Error(), "value")
			require.Contains(t, err.Error(), tag)
		})
	}
}

func TestRPCFieldNumberAdmissionNormalizesIdentity(t *testing.T) {
	fields := &Object{rpcNumberField("first", "01"), rpcNumberField("second", "1")}
	err := validateRPCTags(fields, grpcEndpointForTagValidationTest())
	require.NotEmpty(t, err.Errors)
	require.Contains(t, err.Error(), "already exists")
	require.Contains(t, err.Error(), "first")
	require.Contains(t, err.Error(), "second")
}

func TestRPCUnionFieldNumberAdmission(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		tags []string
	}{
		{"reserved_crossing", "18999", []string{"", ""}},
		{"maximum_crossing", "536870911", []string{"", ""}},
		{"wrapped_derived_slot", "18446744073709551615", []string{"3", "4", ""}},
		{"explicit_reserved", "1", []string{"19500", "2"}},
		{"normalized_explicit_collision", "1", []string{"01", "1"}},
		{"normalized_sibling_collision", "01", []string{"", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := &Object{rpcNumberUnion(tc.base, tc.tags)}
			if tc.name == "normalized_sibling_collision" {
				*fields = append(*fields, rpcNumberField("other", "02"))
			}
			err := validateRPCTags(fields, grpcEndpointForTagValidationTest())
			require.NotEmpty(t, err.Errors)
			require.Contains(t, err.Error(), "choice")
		})
	}
}

func TestRPCFieldNumberAdmissionControls(t *testing.T) {
	for _, tag := range []string{"1", "01", "18999", "20000", "536870911"} {
		t.Run(tag, func(t *testing.T) {
			fields := &Object{rpcNumberField("value", tag)}
			require.Empty(t, validateRPCTags(fields, grpcEndpointForTagValidationTest()).Errors)
		})
	}
	for _, tc := range []struct {
		name string
		base string
		tags []string
	}{
		{"consecutive", "1", []string{"", ""}},
		{"override_reserved_slot", "18999", []string{"", "20000"}},
		{"unused_invalid_seed", "abc", []string{"1", "2"}},
		{"unused_reserved_seed", "19999", []string{"1", ""}},
		{"unused_zero_seed", "0", []string{"3", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Empty(t, validateRPCTags(&Object{rpcNumberUnion(tc.base, tc.tags)}, grpcEndpointForTagValidationTest()).Errors)
		})
	}
}

func rpcNumberField(name, tag string) *NamedAttributeExpr {
	return &NamedAttributeExpr{Name: name, Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"rpc:tag": {tag}}}}
}

func rpcNumberUnion(base string, tags []string) *NamedAttributeExpr {
	branches := make([]*NamedAttributeExpr, len(tags))
	for i, tag := range tags {
		branch := rpcNumberField([]string{"first", "second", "third"}[i], tag)
		if tag == "" {
			branch.Attribute.Meta = nil
		}
		branches[i] = branch
	}
	return &NamedAttributeExpr{Name: "choice", Attribute: &AttributeExpr{
		Type: &Union{TypeName: "Choice", Values: branches},
		Meta: MetaExpr{"rpc:tag": {base}},
	}}
}
