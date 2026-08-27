package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRecursiveValidationCode(t *testing.T) {
	root := RunDSL(t, testdata.ValidationTypesDSL)
	var (
		scope = NewNameScope()

		integerT  = root.UserType("Integer")
		stringT   = root.UserType("String")
		floatT    = root.UserType("Float")
		aliasT    = root.UserType("AliasType")
		userT     = root.UserType("UserType")
		arrayUT   = root.UserType("ArrayUserType")
		arrayT    = root.UserType("Array")
		arrayReqT = root.UserType("ArrayRequired")
		mapT      = root.UserType("Map")
		unionT    = root.UserType("Union")
		rtT       = root.UserType("Result")
		rtcolT    = root.UserType("Collection")
		colT      = root.UserType("TypeWithCollection")
		deepT     = root.UserType("Deep")
	)
	cases := []struct {
		Name       string
		Type       expr.UserType
		Required   bool
		Pointer    bool
		UseDefault bool
	}{
		{"integer-required", integerT, true, false, false},
		{"integer-pointer", integerT, false, true, false},
		{"integer-use-default", integerT, false, false, true},
		{"float-required", floatT, true, false, false},
		{"float-pointer", floatT, false, true, false},
		{"float-use-default", floatT, false, false, true},
		{"string-required", stringT, true, false, false},
		{"string-pointer", stringT, false, true, false},
		{"string-use-default", stringT, false, false, true},
		{"alias-type", aliasT, true, false, false},
		{"user-type-required", userT, true, false, false},
		{"user-type-pointer", userT, false, true, false},
		{"user-type-default", userT, false, false, true},
		{"user-type-array-required", arrayUT, true, true, false},
		{"array-required", arrayT, true, false, false},
		{"array-pointer", arrayT, false, true, false},
		{"array-use-default", arrayT, false, false, true},
		{"array-required-non-nullable-elems", arrayReqT, true, false, false},
		{"map-required", mapT, true, false, false},
		{"map-pointer", mapT, false, true, false},
		{"map-use-default", mapT, false, false, true},
		{"union", unionT, true, false, false},
		{"result-type-pointer", rtT, false, true, false},
		{"collection-required", rtcolT, true, false, false},
		{"collection-pointer", rtcolT, false, true, false},
		{"type-with-collection-pointer", colT, false, true, false},
		{"type-with-embedded-type", deepT, false, true, false},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := NewAttributeContext(c.Pointer, false, c.UseDefault, "", scope)
			code := ValidationCode(&expr.AttributeExpr{Type: c.Type}, nil, ctx, c.Required, expr.IsAlias(c.Type), false, "target")
			code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")
			testutil.AssertGo(t, "testdata/golden/validation_"+c.Name+".go.golden", code)
		})
	}
	// Special case of unions with views
	t.Run("union-with-view", func(t *testing.T) {
		ctx := NewAttributeContext(false, false, false, "", scope)
		code := ValidationCode(&expr.AttributeExpr{Type: unionT}, nil, ctx, true, false, true, "target")
		code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")
		testutil.AssertGo(t, "testdata/golden/validation_union-with-view.go.golden", code)
	})

	// Test case for OneOf with format validation in views (Issue #3747)
	t.Run("union-with-format-validation", func(t *testing.T) {
		// Test with pointer context (typical for views) to ensure union values are not treated as pointers
		ctx := NewAttributeContext(true, false, true, "", scope)
		root := RunDSL(t, testdata.UnionWithFormatValidationDSL)
		oneofT := root.UserType("OneOfWithFormat")
		code := ValidationCode(&expr.AttributeExpr{Type: oneofT}, nil, ctx, true, false, true, "target")
		code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")
		testutil.AssertGo(t, "testdata/golden/validation_union-with-format-validation.go.golden", code)
	})
}

func TestMergedValidationWalksFullUserTypeChain(t *testing.T) {
	root := RunDSL(t, func() {
		base := dsl.Type("Base", dsl.String, func() {
			dsl.MinLength(2)
		})
		mid := dsl.Type("Mid", base)
		dsl.Type("ChainHolder", func() {
			dsl.Attribute("one", base)
			dsl.Attribute("two", mid)
		})
	})
	baseValidation := root.UserType("Base").Attribute().Validation
	holder := expr.AsObject(root.UserType("ChainHolder").Attribute().Type)
	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, false, "", scope)

	// Two-level chain (Base -> Mid -> attribute) must retain the inner MinLength.
	twoField := holder.Attribute("two")
	twoMerged := mergedValidation(twoField)
	require.NotNil(t, twoMerged, "multi-level chain must produce merged validation")
	require.NotNil(t, twoMerged.MinLength, "multi-level chain must retain inner MinLength")
	require.Equal(t, 2, *twoMerged.MinLength)
	twoCode := validationCode(twoField, ctx, true, false, "target", "target")
	require.Contains(t, twoCode, "InvalidLengthError", "multi-level chain must render inner MinLength validation")

	// Single-level chain (Base -> attribute) still validates.
	oneField := holder.Attribute("one")
	oneMerged := mergedValidation(oneField)
	require.NotNil(t, oneMerged, "single-level chain must produce merged validation")
	require.NotNil(t, oneMerged.MinLength, "single-level chain must retain MinLength")
	require.Equal(t, 2, *oneMerged.MinLength)

	// Read-only property: shared expr validation state must not be mutated.
	require.Nil(t, twoField.Validation, "field attribute validation must remain untouched")
	require.NotSame(t, baseValidation, twoMerged, "merged validation must be a copy")
	require.NotNil(t, baseValidation.MinLength)
	require.Equal(t, 2, *baseValidation.MinLength)
}

func TestValidationCodeDoesNotMutateSharedUserTypeValidation(t *testing.T) {
	root := RunDSL(t, func() {
		alias := dsl.Type("Status", dsl.String, func() {
			dsl.MinLength(2)
		})
		dsl.Type("Holder", func() {
			dsl.Attribute("status", alias, func() {
				dsl.Enum("ready")
			})
		})
	})
	aliasValidation := root.UserType("Status").Attribute().Validation
	field := expr.AsObject(root.UserType("Holder").Attribute().Type).Attribute("status")
	fieldValidation := field.Validation
	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, false, "", scope)

	_ = ValidationCode(field, nil, ctx, true, false, false, "target")

	require.Same(t, fieldValidation, field.Validation)
	require.Nil(t, fieldValidation.MinLength)
	require.NotNil(t, aliasValidation.MinLength)
	require.Equal(t, 2, *aliasValidation.MinLength)
}

func TestNullableValidationUsesSemanticPresence(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:       expr.String,
		Nullable:   true,
		Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
	}
	ctx := NewAttributeContext(false, false, false, "", NewNameScope())
	code := ValidationCode(attribute, nil, ctx, true, false, false, "body.value")
	require.Contains(t, code, "if !body.value.Present()")
	require.Contains(t, code, "if actual, ok := body.value.Value(); ok")
	require.Contains(t, code, "ValidatePattern")
}

func TestValidationCodeEmitsOneRequiredFieldError(t *testing.T) {
	tests := []struct {
		name      string
		nullable  bool
		required  bool
		pointer   bool
		wantCount int
	}{
		{name: "required nullable pointer", nullable: true, required: true, pointer: true, wantCount: 1},
		{name: "required nullable value", nullable: true, required: true, wantCount: 1},
		{name: "required non-nullable pointer", required: true, pointer: true, wantCount: 1},
		{name: "optional nullable pointer", nullable: true, pointer: true, wantCount: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := &expr.AttributeExpr{
				Type:       expr.String,
				Nullable:   test.nullable,
				Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
			}
			validation := &expr.ValidationExpr{}
			if test.required {
				validation.Required = []string{"value"}
			}
			attribute := &expr.AttributeExpr{
				Type: &expr.Object{{
					Name:      "value",
					Attribute: field,
				}},
				Validation: validation,
			}
			ctx := NewAttributeContext(test.pointer, false, false, "", NewNameScope())
			ctx.JSONPresence = true

			code := ValidationCode(attribute, nil, ctx, true, false, false, "body")

			require.Equal(t, test.wantCount, strings.Count(code, `loom.MissingFieldError("value", "body")`), code)
			require.Contains(t, code, "ValidatePattern")
		})
	}
}

func TestRequiredDefaultJSONPresenceValidationUsesOptionalPresence(t *testing.T) {
	root := RunDSL(t, func() {
		dsl.Type("Sort", func() {
			dsl.Attribute("field", dsl.String, func() {
				dsl.Default("display_id")
				dsl.Enum("display_id", "created_at")
			})
			dsl.Required("field")
		})
	})
	attribute := root.UserType("Sort").Attribute()
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	ctx.JSONPresence = true

	code := validationCode(attribute, ctx, true, false, "body", "body")

	require.Contains(t, code, "if !body.Field.Present()")
	require.NotContains(t, code, "body.Field == nil")
}

func TestOptionalObjectValidationPreservesNestedJSONPresence(t *testing.T) {
	minLength := 1
	minimum := 0.0
	details := &expr.UserTypeExpr{
		TypeName: "Details",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{
			Name: "note",
			Attribute: &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{MinLength: &minLength},
			},
		}}},
	}
	scores := &expr.UserTypeExpr{
		TypeName: "Scores",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name: "code",
					Attribute: &expr.AttributeExpr{
						Type:       expr.String,
						Validation: &expr.ValidationExpr{MinLength: &minLength},
					},
				},
				{
					Name: "confidence",
					Attribute: &expr.AttributeExpr{
						Type:       expr.Float64,
						Validation: &expr.ValidationExpr{Minimum: &minimum},
					},
				},
				{Name: "details", Attribute: &expr.AttributeExpr{Type: details}},
				{
					Name: "labels",
					Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{
						Type:       expr.String,
						Validation: &expr.ValidationExpr{MinLength: &minLength},
					}}},
				},
				{
					Name: "required_code",
					Attribute: &expr.AttributeExpr{
						Type:       expr.String,
						Validation: &expr.ValidationExpr{MinLength: &minLength},
					},
				},
				{Name: "required_details", Attribute: &expr.AttributeExpr{Type: details}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"required_code", "required_details"}},
		},
	}
	attribute := &expr.AttributeExpr{Type: &expr.Object{{
		Name:      "scores",
		Attribute: &expr.AttributeExpr{Type: scores},
	}}}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	ctx.JSONPresence = true

	code := ValidationCode(attribute, nil, ctx, true, false, false, "body")

	require.Contains(t, code, "body.Scores.Value()")
	require.Contains(t, code, "actual.Code.Value()")
	require.Contains(t, code, "actual.Confidence.Value()")
	require.Contains(t, code, "actual.Details.Value()")
	require.Contains(t, code, "actual.Labels.Value()")
	require.Contains(t, code, "actual.RequiredCode != nil")
	require.Contains(t, code, "*actual.RequiredCode")
	require.Contains(t, code, "actual.RequiredDetails != nil")
	require.NotContains(t, code, "actual.Code != nil")
	require.NotContains(t, code, "range actual.Labels")
}

func TestOptionalUnionInlineObjectValidationUsesNativePresence(t *testing.T) {
	minLength := 1
	details := &expr.UserTypeExpr{
		TypeName: "Details",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "code",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{MinLength: &minLength},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"code"}},
		},
	}
	union := &expr.Union{
		TypeName: "Choice",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "object",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{
							Name: "optional_code",
							Attribute: &expr.AttributeExpr{
								Type:       expr.String,
								Validation: &expr.ValidationExpr{MinLength: &minLength},
							},
						},
						{
							Name: "required_code",
							Attribute: &expr.AttributeExpr{
								Type:       expr.String,
								Validation: &expr.ValidationExpr{MinLength: &minLength},
							},
						},
					},
					Validation: &expr.ValidationExpr{Required: []string{"required_code"}},
				},
			},
			{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{
				Name: "objects",
				Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{
					Type: &expr.Object{{
						Name: "array_code",
						Attribute: &expr.AttributeExpr{
							Type:       expr.String,
							Validation: &expr.ValidationExpr{MinLength: &minLength},
						},
					}},
				}}},
			},
			{
				Name: "nullable_items",
				Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{
					Type:     details,
					Nullable: true,
				}}},
			},
		},
	}
	attribute := &expr.AttributeExpr{Type: &expr.Object{{
		Name:      "choice",
		Attribute: &expr.AttributeExpr{Type: union},
	}}}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	ctx.JSONPresence = true
	ctx.JSONPresenceTypes = map[string]bool{details.ID(): true}
	ctx.PresencePointerTypes = map[string]bool{details.ID(): true}

	code := ValidationCode(attribute, nil, ctx, true, false, false, "body")

	require.Contains(t, code, "body.Choice.Value()")
	require.Contains(t, code, "actual.OptionalCode != nil")
	require.Contains(t, code, "*actual.OptionalCode")
	require.Contains(t, code, "utf8.RuneCountInString(actual.RequiredCode)")
	require.NotContains(t, code, "actual.OptionalCode.Value()")
	require.NotContains(t, code, "actual.RequiredCode != nil")
	require.NotContains(t, code, "*actual.RequiredCode")
	require.Contains(t, code, "e.ArrayCode != nil")
	require.Contains(t, code, "*e.ArrayCode")
	require.NotContains(t, code, "e.ArrayCode.Value()")
	require.Contains(t, code, "e.Value()")
	require.Contains(t, code, "actual.Code != nil")
	require.Contains(t, code, "*actual.Code")
}

func TestNullableObjectValidationPreservesNestedPointerPresence(t *testing.T) {
	minLength := 1
	details := &expr.UserTypeExpr{
		TypeName: "Details",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "code",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{MinLength: &minLength},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"code"}},
		},
	}
	attribute := &expr.AttributeExpr{Type: details, Nullable: true}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	ctx.JSONPresence = true

	code := ValidationCode(attribute, details, ctx, true, false, false, "body.details")

	require.Contains(t, code, "body.details.Value()")
	require.Contains(t, code, "actual.Code != nil")
	require.Contains(t, code, "*actual.Code")
}

func TestOptionalMapInlineObjectValidationPreservesPointerLayout(t *testing.T) {
	minLength := 1
	entry := &expr.AttributeExpr{
		Type: &expr.Object{{
			Name: "code",
			Attribute: &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{MinLength: &minLength},
			},
		}},
		Validation: &expr.ValidationExpr{Required: []string{"code"}},
	}
	entries := &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: expr.String},
		ElemType: entry,
	}
	attribute := &expr.AttributeExpr{Type: &expr.Object{{
		Name:      "entries",
		Attribute: &expr.AttributeExpr{Type: entries},
	}}}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	ctx.JSONPresence = true

	code := ValidationCode(attribute, nil, ctx, true, false, false, "body")

	require.Contains(t, code, "body.Entries.Value()")
	require.Contains(t, code, "if v != nil")
	require.Contains(t, code, "v.Code != nil")
	require.Contains(t, code, "*v.Code")
}

func TestNullableRecursiveValidationTerminates(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node"}
	node.AttributeExpr = &expr.AttributeExpr{
		Type: &expr.Object{{
			Name:      "child",
			Attribute: &expr.AttributeExpr{Type: node, Nullable: true},
		}},
		Validation: &expr.ValidationExpr{},
	}
	attribute := &expr.AttributeExpr{Type: node, Nullable: true}

	code := ValidationCode(attribute, node, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "node")
	require.Contains(t, code, ".Present()")
	require.Less(t, len(code), 5000)
}

// TestRecursiveValidationWithCycleGuard tests that recursive types are
// properly handled without infinite loops. The recursion guard should prevent
// cycles while still allowing validation of the same type in different contexts.
func TestRecursiveValidationWithCycleGuard(t *testing.T) {
	root := RunDSL(t, testdata.RecursiveValidationDSL)
	scope := NewNameScope()

	cases := []struct {
		Name     string
		TypeName string
		Pointer  bool
	}{
		{"recursive-type-pointer", "RecursiveType", true},
		{"recursive-type-required", "RecursiveType", false},
		{"container-with-recursive-array", "ContainerWithRecursiveArray", true},
		{"container-with-recursive-map", "ContainerWithRecursiveMap", true},
		{"nested-recursive-pointer", "NestedRecursive", true},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := NewAttributeContext(c.Pointer, false, false, "", scope)
			ut := root.UserType(c.TypeName)
			code := ValidationCode(&expr.AttributeExpr{Type: ut}, nil, ctx, !c.Pointer, false, false, "target")
			// Verify code is generated (not empty) and doesn't cause infinite recursion
			if code == "" {
				t.Error("Expected validation code to be generated")
			}
			// Verify the code can be formatted (indicates valid Go code)
			code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")
			if code == "" {
				t.Error("Expected formatted code to be generated")
			}
		})
	}
}

// TestMultipleAliasTypesInSameStruct tests that multiple fields with the same
// alias type can be validated independently. Previously, the recursion guard
// would incorrectly block validation of the second field.
func TestMultipleAliasTypesInSameStruct(t *testing.T) {
	root := RunDSL(t, testdata.ValidationTypesDSL)
	scope := NewNameScope()

	aliasT := root.UserType("AliasType")
	ctx := NewAttributeContext(false, false, false, "", scope)

	code := ValidationCode(&expr.AttributeExpr{Type: aliasT}, nil, ctx, true, false, false, "target")
	code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")

	// Verify both alias fields are validated (required_alias and alias)
	// The code should contain validation for both fields
	if !strings.Contains(code, "required_alias") {
		t.Error("Expected validation code for 'required_alias' field")
	}
	if !strings.Contains(code, "target.alias") {
		t.Error("Expected validation code for 'alias' field")
	}
	// Verify both get pattern validation
	if strings.Count(code, "ValidatePattern") < 2 {
		t.Errorf("Expected at least 2 pattern validations (one per alias field), got %d", strings.Count(code, "ValidatePattern"))
	}
}

// TestAliasTypeInArrayAndMap tests that alias types work correctly when
// nested in arrays and maps, ensuring the recursion guard doesn't interfere.
func TestAliasTypeInArrayAndMap(t *testing.T) {
	root := RunDSL(t, testdata.ValidationTypesDSL)
	scope := NewNameScope()

	var (
		alias = root.UserType("Alias")
	)

	// Create a type with alias in array
	arrayWithAlias := &expr.AttributeExpr{
		Type: &expr.Array{
			ElemType: &expr.AttributeExpr{Type: alias},
		},
	}

	// Create a type with alias in map
	mapWithAlias := &expr.AttributeExpr{
		Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.String},
			ElemType: &expr.AttributeExpr{Type: alias},
		},
	}

	cases := []struct {
		Name   string
		Att    *expr.AttributeExpr
		Target string
	}{
		{"alias-in-array", arrayWithAlias, "target"},
		{"alias-in-map", mapWithAlias, "target"},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := NewAttributeContext(false, false, false, "", scope)
			code := ValidationCode(c.Att, nil, ctx, true, false, false, c.Target)
			code = FormatTestCode(t, "package foo\nfunc Validate() (err error){\n"+code+"}")

			// Verify validation code is generated
			if code == "" {
				t.Error("Expected validation code to be generated")
			}
			// Verify it contains validation for the alias type
			if !strings.Contains(code, "ValidatePattern") && !strings.Contains(code, "InvalidLengthError") {
				t.Error("Expected validation code to contain pattern or length validation for alias type")
			}
		})
	}
}

func TestValidationCodeUsesExplicitUnionVariantTagsForSumUnions(t *testing.T) {
	t.Parallel()

	scope := NewNameScope()
	// Pointer context exercises both union branches because object union
	// branches in value context only emit cases when the inner attribute
	// produces non-empty validation; this test focuses on tag selection.
	ctx := NewAttributeContext(true, false, false, "", scope)
	single := &expr.UserTypeExpr{
		TypeName: "SingleAction",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name: "value",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
					},
				},
			},
			Validation: &expr.ValidationExpr{Required: []string{"value"}},
			Meta:       expr.MetaExpr{"oneof:type:tag": []string{"single"}},
		},
	}
	batch := &expr.UserTypeExpr{
		TypeName: "BatchAction",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name: "values",
					Attribute: &expr.AttributeExpr{
						Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
					},
				},
			},
			Validation: &expr.ValidationExpr{Required: []string{"values"}},
			Meta:       expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
		},
	}
	union := &expr.Union{
		TypeName: "Selection",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: single,
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: batch,
				},
			},
		},
	}

	code := ValidationCode(&expr.AttributeExpr{Type: union}, nil, ctx, true, false, false, "target")

	if !strings.Contains(code, `case "single":`) {
		t.Errorf("expected validation code to use explicit tag 'single':\n%s", code)
	}
	if !strings.Contains(code, `case "batch":`) {
		t.Errorf("expected validation code to use explicit tag 'batch':\n%s", code)
	}
	if strings.Contains(code, `case "Single":`) {
		t.Errorf("expected validation code to avoid branch name 'Single':\n%s", code)
	}
	if strings.Contains(code, `case "Batch":`) {
		t.Errorf("expected validation code to avoid branch name 'Batch':\n%s", code)
	}
}

func TestNewValidationRenderDataHandlesAliasPointersAndDefaults(t *testing.T) {
	t.Parallel()

	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, false, "", scope)
	alias := &expr.UserTypeExpr{
		TypeName: "AliasString",
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
		},
	}

	t.Run("optional primitive pointer uses dereference", func(t *testing.T) {
		att := &expr.AttributeExpr{Type: expr.String}
		data := newValidationRenderData(att, ctx, false, false, "target", "target")
		if !data.IsPointer {
			t.Fatal("expected optional primitive to be pointer-like")
		}
		if data.TargetValue != "*target" {
			t.Fatalf("expected dereferenced target value, got %q", data.TargetValue)
		}
	})

	t.Run("alias uses underlying type cast", func(t *testing.T) {
		att := &expr.AttributeExpr{Type: alias}
		data := newValidationRenderData(att, ctx, true, true, "target", "target")
		if data.TargetValue != "string(target)" {
			t.Fatalf("expected alias cast target value, got %q", data.TargetValue)
		}
		if !data.IsString {
			t.Fatal("expected alias over string to be treated as string")
		}
	})
}

func TestValidationCodeEmitsExclusiveMaximumWithExclusiveMinimum(t *testing.T) {
	t.Parallel()

	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, false, "", scope)
	min := 1.0
	max := 100.0
	att := &expr.AttributeExpr{
		Type: expr.Int,
		Validation: &expr.ValidationExpr{
			ExclusiveMinimum: &min,
			ExclusiveMaximum: &max,
		},
	}

	code := validationCode(att, ctx, false, false, "target", "target")
	expected := strings.Join([]string{
		"if target != nil {",
		"\tif *target <= 1 {",
		"\t\terr = loom.MergeErrors(err, loom.InvalidRangeError(\"target\", *target, 1, true))",
		"\t}",
		"}",
		"if target != nil {",
		"\tif *target >= 100 {",
		"\t\terr = loom.MergeErrors(err, loom.InvalidRangeError(\"target\", *target, 100, false))",
		"\t}",
		"}",
	}, "\n")
	if code != expected {
		t.Fatalf("unexpected validation code:\n%s", code)
	}
}

// TestUnionValidationPreservesValueContextForRequiredOnlyObjectBranches
// regresses union validation code generation when sum-type object branches are
// validated in value contexts such as HTTP request bodies. Object branches
// should only receive pointer-wrapped validation when the enclosing attribute
// context is itself pointer-based.
func TestUnionValidationPreservesValueContextForRequiredOnlyObjectBranches(t *testing.T) {
	t.Parallel()

	scope := NewNameScope()
	someType := &expr.UserTypeExpr{
		TypeName: "SomeType",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name:      "a",
					Attribute: &expr.AttributeExpr{Type: expr.String},
				},
			},
			Validation: &expr.ValidationExpr{Required: []string{"a"}},
		},
	}
	someOtherType := &expr.UserTypeExpr{
		TypeName: "SomeOtherType",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{
					Name:      "b",
					Attribute: &expr.AttributeExpr{Type: expr.String},
				},
			},
			Validation: &expr.ValidationExpr{Required: []string{"b"}},
		},
	}
	union := &expr.Union{
		TypeName: "UnionUserValidate",
		Values: []*expr.NamedAttributeExpr{
			{Name: "SomeType", Attribute: &expr.AttributeExpr{Type: someType}},
			{Name: "SomeOtherType", Attribute: &expr.AttributeExpr{Type: someOtherType}},
		},
	}
	att := &expr.AttributeExpr{Type: union}

	valueCtx := NewAttributeContext(false, false, false, "", scope)
	valueCode := ValidationCode(att, nil, valueCtx, true, false, false, "target")
	if strings.Contains(valueCode, "ValidateSomeType(actual)") {
		t.Errorf("value context must not pointer-wrap ValidateSomeType call:\n%s", valueCode)
	}
	if strings.Contains(valueCode, "ValidateSomeOtherType(actual)") {
		t.Errorf("value context must not pointer-wrap ValidateSomeOtherType call:\n%s", valueCode)
	}
	if strings.Contains(valueCode, "if actual != nil") {
		t.Errorf("value context must not emit pointer-presence guard:\n%s", valueCode)
	}

	pointerCtx := NewAttributeContext(true, false, false, "", scope)
	pointerCode := ValidationCode(att, nil, pointerCtx, true, false, false, "target")
	if !strings.Contains(pointerCode, "ValidateSomeType(actual)") {
		t.Errorf("pointer context must emit ValidateSomeType call:\n%s", pointerCode)
	}
	if !strings.Contains(pointerCode, "ValidateSomeOtherType(actual)") {
		t.Errorf("pointer context must emit ValidateSomeOtherType call:\n%s", pointerCode)
	}
}

func TestConstantPanicsOnUnknownFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name   string
		Format string
	}{
		{"empty", ""},
		{"unknown-name", "not-a-format"},
		{"case-sensitive", "UUID"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := recoverCodegenPanic(t, func() {
				constant(c.Format, nil)
			})
			require.ErrorContains(t, err, `unknown validation format "`+c.Format+`"`)
		})
	}
}

func TestValidationCodePanicsOnUnknownFormat(t *testing.T) {
	t.Parallel()

	// The DSL rejects unsupported formats at eval time so this branch is only
	// reachable from a hand-built expression tree, e.g. plugins mutating the
	// design after validation.
	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, false, "", scope)
	att := &expr.AttributeExpr{
		Type:       expr.String,
		Validation: &expr.ValidationExpr{Format: "not-a-format"},
	}
	err := recoverCodegenPanic(t, func() {
		ValidationCode(att, nil, ctx, true, false, false, "target")
	})
	require.ErrorContains(t, err, `unknown validation format "not-a-format"`)
	var codegenErr *Error
	require.ErrorAs(t, err, &codegenErr)
	require.Same(t, att, codegenErr.Expr)
}

func recoverCodegenPanic(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		recovered := recover()
		require.NotNil(t, recovered, "expected code generation to panic")
		var ok bool
		err, ok = recovered.(error)
		require.True(t, ok, "panic value is not an error: %v", recovered)
	}()
	fn()
	return nil
}
