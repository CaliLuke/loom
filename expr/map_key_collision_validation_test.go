package expr

import (
	"context"
	"math"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/CaliLuke/loom/internal/examplevalue"
	"github.com/stretchr/testify/require"
)

type collisionNamedString string

type collisionCodecKey struct {
	Value string
	Calls *int
}

type collisionPointerKey struct {
	Next any
}

type collisionPlainObject struct {
	Value string
}

type collisionPlainNestedObject struct {
	Items map[any]string
}

type collisionCodecMap map[any]string

type collisionCodecObject struct {
	Value string
}

type collisionCodecString string

var collisionCompatibilityCodecCalls int

func (key collisionCodecKey) MarshalText() ([]byte, error) {
	*key.Calls++
	return []byte(key.Value), nil
}

func (collisionCodecMap) MarshalText() ([]byte, error) {
	collisionCompatibilityCodecCalls++
	return []byte("map"), nil
}

func (collisionCodecObject) MarshalText() ([]byte, error) {
	collisionCompatibilityCodecCalls++
	return []byte("object"), nil
}

func (collisionCodecString) MarshalText() ([]byte, error) {
	collisionCompatibilityCodecCalls++
	return []byte("string"), nil
}

func collisionMap(key, element DataType) *AttributeExpr {
	return &AttributeExpr{Type: &Map{
		KeyType:  &AttributeExpr{Type: key},
		ElemType: &AttributeExpr{Type: element},
	}}
}

func TestValidateExamplesRejectsDeclaredMapKeyCollisions(t *testing.T) {
	collision := map[any]string{1: "integer", "1": "string"}
	maxOne := 1
	withLength := collisionMap(Any, String)
	withLength.Validation = &ValidationExpr{MaxLength: &maxOne}
	withKeyEnum := collisionMap(Any, String)
	withKeyEnum.Type.(*Map).KeyType.Validation = &ValidationExpr{Values: []any{true}}
	nested := &AttributeExpr{Type: &Object{{Name: "items", Attribute: collisionMap(Any, String)}}}
	missingRequired := &AttributeExpr{
		Type: &Object{
			{Name: "items", Attribute: collisionMap(Any, String)},
			{Name: "required", Attribute: &AttributeExpr{Type: String}},
		},
		Validation: &ValidationExpr{Required: []string{"required"}},
	}

	tests := []struct {
		name      string
		attribute *AttributeExpr
		value     any
		path      string
		member    string
	}{
		{"root", collisionMap(Any, String), collision, "the example root", "1"},
		{"nested object", nested, map[string]any{"items": collision}, "items", "1"},
		{"missing required sibling", missingRequired, map[string]any{"items": collision}, "items", "1"},
		{"unrelated length", withLength, collision, "the example root", "1"},
		{"unrelated key enum", withKeyEnum, collision, "the example root", "1"},
		{"declared float32", collisionMap(Float32, String), map[any]string{
			float64(1.23456789): "wide",
			float32(1.23456789): "narrow",
		}, "the example root", "1.2345679"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.attribute.UserExamples = []*ExampleExpr{{Value: test.value}}
			errors := test.attribute.validateExamples("attribute - ", test.attribute)
			require.Len(t, errors.Errors, 1)
			require.Contains(t, errors.Error(), `example map keys collide as JSON object member "`+test.member+`"`)
			require.Contains(t, errors.Error(), test.path)
		})
	}
}

func TestMapKeyCollisionUnionComposition(t *testing.T) {
	floatMap := collisionMap(Float32, String)
	anyMap := collisionMap(Any, String)
	value := map[any]string{float64(1.23456789): "wide", float32(1.23456789): "narrow"}
	union := func(other *AttributeExpr) *AttributeExpr {
		return &AttributeExpr{Type: &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "Float", Attribute: floatMap},
			{Name: "Other", Attribute: other},
		}}}
	}

	for _, test := range []struct {
		name    string
		attr    *AttributeExpr
		rejects bool
	}{
		{"compatible collision-free branch", union(anyMap), false},
		{"incompatible branch", union(&AttributeExpr{Type: Boolean}), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.attr.UserExamples = []*ExampleExpr{{Value: value}}
			errors := test.attr.validateExamples("attribute - ", test.attr)
			if test.rejects {
				require.Contains(t, errors.Error(), "map keys collide")
			} else {
				require.Empty(t, errors.Errors)
			}
		})
	}

	selected := union(anyMap)
	selected.UserExamples = []*ExampleExpr{{Value: examplevalue.Union{Branch: 0, Value: value}}}
	require.Contains(t, selected.validateExamples("attribute - ", selected).Error(), "map keys collide")
	selected.UserExamples = []*ExampleExpr{{Value: examplevalue.Union{Branch: 1, Value: value}}}
	require.Empty(t, selected.validateExamples("attribute - ", selected).Errors)
	selected.UserExamples = []*ExampleExpr{{Value: examplevalue.Union{Branch: 9, Value: value}}}
	selectedErrors := selected.validateExamples("attribute - ", selected)
	require.NotContains(t, selectedErrors.Error(), "map keys collide")
	require.Contains(t, selectedErrors.Error(), "incompatible")

	allColliding := union(floatMap)
	context := NewValueContext()
	occurrence, err := context.newCollisionOccurrence(allColliding)
	require.NoError(t, err)
	result := context.mapKeyCollisionResult(occurrence, context.SupplyValue(ValueInput{Raw: value}))
	require.Equal(t, valueCollisionCompatible, result.compatibility)
	require.Len(t, result.evidence.collisions, 2)
	require.Equal(t, []string{"Float"}, result.evidence.collisions[0].branchNames)
	require.Equal(t, []string{"Other"}, result.evidence.collisions[1].branchNames)
	require.NotEqual(t, result.evidence.collisions[0].branches, result.evidence.collisions[1].branches)

	zeroCompatible := &AttributeExpr{Type: &Union{TypeName: "None", Untagged: true,
		Values: []*NamedAttributeExpr{
			{Name: "String", Attribute: &AttributeExpr{Type: String}},
			{Name: "Boolean", Attribute: &AttributeExpr{Type: Boolean}},
		}}}
	zeroCompatible.UserExamples = []*ExampleExpr{{Value: value}}
	zeroErrors := zeroCompatible.validateExamples("attribute - ", zeroCompatible)
	require.NotContains(t, zeroErrors.Error(), "map keys collide")
	require.Contains(t, zeroErrors.Error(), "incompatible")
}

func TestMapKeyCollisionUnknownUnionEligibilityBlocksRejection(t *testing.T) {
	attribute := &AttributeExpr{Type: &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Map", Attribute: collisionMap(Float32, String)},
		{Name: "Boolean", Attribute: &AttributeExpr{Type: Boolean}},
	}}}
	value := collisionCodecMap{float64(1.23456789): "wide", float32(1.23456789): "narrow"}
	collisionCompatibilityCodecCalls = 0
	attribute.UserExamples = []*ExampleExpr{{Value: value}}
	require.Empty(t, attribute.validateExamples("attribute - ", attribute).Errors)
	require.Zero(t, collisionCompatibilityCodecCalls)
}

func TestMapKeyCollisionSurvivesCyclicPointerKeyAndOpaqueKey(t *testing.T) {
	cycle := &collisionPointerKey{}
	cycle.Next = cycle
	calls := 0
	opaque := collisionCodecKey{Value: "opaque", Calls: &calls}
	attribute := collisionMap(Any, String)
	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{
		1:      "integer",
		"1":    "string",
		cycle:  "cycle",
		opaque: "opaque",
	}}}

	errors := attribute.validateExamples("attribute - ", attribute)
	require.Contains(t, errors.Error(), `map keys collide as JSON object member "1"`)
	require.Zero(t, calls)

	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{opaque: "opaque"}}}
	require.Empty(t, attribute.validateExamples("attribute - ", attribute).Errors)
	require.Zero(t, calls)
}

func TestMapKeyCollisionTreatsCodecScalarKeyAsOpaque(t *testing.T) {
	collisionCompatibilityCodecCalls = 0
	attribute := collisionMap(Any, String)
	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{
		collisionCodecString("1"): "codec",
		"1":                       "plain",
	}}}

	report := attribute.validateExamples("attribute - ", attribute)
	require.Empty(t, report.Errors)
	require.Zero(t, collisionCompatibilityCodecCalls)

	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{
		collisionCodecString("1"): "codec",
		"1":                       "plain",
		2:                         "integer",
		"2":                       "string",
	}}}
	report = attribute.validateExamples("attribute - ", attribute)
	require.Contains(t, report.Error(), `map keys collide as JSON object member "2"`)
	require.NotContains(t, report.Error(), `map keys collide as JSON object member "1"`)
	require.Zero(t, collisionCompatibilityCodecCalls)
}

func TestMapKeyCollisionRecursiveObjectMapTerminates(t *testing.T) {
	const child = "LOOM456_RECURSIVE_OBJECT_COLLISION_HELPER"
	if os.Getenv(child) == "1" {
		recursive := &UserTypeExpr{TypeName: "RecursiveCollision", AttributeExpr: &AttributeExpr{}}
		recursive.Type = &Object{
			{Name: "items", Attribute: collisionMap(Any, String)},
			{Name: "next", Attribute: &AttributeExpr{Type: recursive, Nullable: true}},
		}
		value := map[string]any{
			"items": map[any]string{1: "integer", "1": "string"},
		}
		value["next"] = value
		attribute := &AttributeExpr{
			Type:         recursive,
			UserExamples: []*ExampleExpr{{Value: value}},
		}

		report := attribute.validateExamples("attribute - ", attribute)
		require.Contains(t, report.Error(), `map keys collide as JSON object member "1"`)
		require.Contains(t, report.Error(), "items")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMapKeyCollisionRecursiveObjectMapTerminates$", "-test.count=1")
	command.Env = append(os.Environ(), child+"=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NoError(t, ctx.Err(), string(output))
}

func TestMapKeyCollisionSurvivesPointerInterfaceKeyCycle(t *testing.T) {
	var cycle any
	pointer := &cycle
	cycle = pointer
	attribute := collisionMap(Float32, String)
	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{
		float64(1.23456789): "wide",
		float32(1.23456789): "narrow",
		pointer:             "cycle",
	}}}

	errors := attribute.validateExamples("attribute - ", attribute)
	require.Contains(t, errors.Error(), `map keys collide as JSON object member "1.2345679"`)
}

func TestMapKeyCollisionEvidenceMayTraverseIncompatiblePointer(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{{
		Name: "items", Attribute: collisionMap(Any, String),
	}}}
	value := &collisionPlainNestedObject{Items: map[any]string{1: "integer", "1": "string"}}
	context := NewValueContext()
	occurrence, err := context.newCollisionOccurrence(attribute)
	require.NoError(t, err)
	result := context.mapKeyCollisionResult(occurrence, context.SupplyValue(ValueInput{Raw: value}))
	require.Equal(t, valueCollisionIncompatible, result.compatibility)
	require.Len(t, result.evidence.collisions, 1)
	require.Equal(t, []string{"items"}, result.evidence.collisions[0].path)
}

func TestMapKeyCollisionDiagnosticsComposeWithExistingDesignErrors(t *testing.T) {
	invalidDefault := collisionMap(Any, String)
	invalidDefault.DefaultValue = true
	invalidDefault.UserExamples = []*ExampleExpr{{Value: map[any]string{1: "integer", "1": "string"}}}

	validated = make(map[*AttributeExpr]bool)
	errors := invalidDefault.Validate("attribute", invalidDefault)
	require.Contains(t, errors.Error(), "default")
	require.Contains(t, errors.Error(), "map keys collide")

	result := &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			TypeName: "CollisionResult",
			AttributeExpr: &AttributeExpr{Type: &Object{{
				Name: "items", Attribute: collisionMap(Any, String),
			}}},
		},
		Identifier: "application/vnd.collision",
		Views: []*ViewExpr{{
			Name: "valid",
			AttributeExpr: &AttributeExpr{Type: &Object{{
				Name: "items", Attribute: collisionMap(Any, String),
			}}},
		}},
	}
	attribute := &AttributeExpr{
		Type: result,
		Meta: MetaExpr{ViewMetaKey: []string{"missing"}},
		UserExamples: []*ExampleExpr{{Value: map[string]any{
			"items": map[any]string{1: "integer", "1": "string"},
		}}},
	}
	validated = make(map[*AttributeExpr]bool)
	errors = attribute.Validate("attribute", attribute)
	require.Contains(t, errors.Error(), `does not define view "missing"`)
	require.Contains(t, errors.Error(), "map keys collide")
}

func TestObjectSourceDuplicateStringSpellingsRejected(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{{Name: "items", Attribute: &AttributeExpr{Type: String}}}}
	value := map[any]any{collisionNamedString("items"): "named", "items": "plain"}
	for attempt := range 100 {
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		source := context.SupplyValue(ValueInput{Raw: value})
		require.Equal(t, ValueInvalid, context.Resolve(occurrence, source, ValueRoleExample).Outcome(), "attempt %d", attempt)
		require.Equal(t, ValueInvalid, context.ResolveDeclaredShape(occurrence, source).Outcome(), "attempt %d", attempt)
	}
}

func TestMapKeyCollisionDiagnosticOrderingIsStructural(t *testing.T) {
	mapAttribute := func() *AttributeExpr { return collisionMap(Any, String) }
	attribute := &AttributeExpr{Type: &Object{
		{Name: "z", Attribute: mapAttribute()},
		{Name: "a", Attribute: &AttributeExpr{Type: &Object{{Name: "b.c", Attribute: mapAttribute()}}}},
	}}
	attribute.UserExamples = []*ExampleExpr{{Value: map[string]any{
		"z": map[any]string{"a": "one", collisionNamedString("a"): "two"},
		"a": map[string]any{"b.c": map[any]string{"z": "one", collisionNamedString("z"): "two"}},
	}}}

	errors := attribute.validateExamples("attribute - ", attribute)
	require.Len(t, errors.Errors, 1)
	require.Contains(t, errors.Error(), `member "z" at a.b.c`)
}

func TestMapKeyCollisionDiagnosticIsPermutationInvariant(t *testing.T) {
	value := map[any]string{
		1:          "integer",
		"1":        "string",
		true:       "boolean",
		"true":     "boolean string",
		math.NaN(): "first non-reflexive key",
		math.Float64frombits(0x7ff8_0000_0000_0002): "second non-reflexive key",
	}
	require.Len(t, valueSortedMapEntries(reflect.ValueOf(value)), 6)
	attribute := collisionMap(Any, String)
	attribute.UserExamples = []*ExampleExpr{{Value: value}}
	for attempt := range 100 {
		errors := attribute.validateExamples("attribute - ", attribute)
		require.Len(t, errors.Errors, 1, "attempt %d", attempt)
		require.Contains(t, errors.Error(), `map keys collide as JSON object member "1"`, "attempt %d", attempt)
	}
}

func TestCollisionCompatibilityPreservesFiniteExampleCompatibility(t *testing.T) {
	object := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}
	stringValue := "value"
	arrayValue := []int{1, 2}
	mapValue := map[any]string{1: "one"}
	objectValue := collisionPlainObject{Value: "ok"}
	var typedNilObject *collisionPlainObject
	var typedNilMap map[any]string
	var typedNilSlice []int
	namedString := &UserTypeExpr{
		TypeName:      "NullableName",
		AttributeExpr: &AttributeExpr{Type: String},
	}
	declarations := []struct {
		name      string
		attribute *AttributeExpr
	}{
		{"any", &AttributeExpr{Type: Any}},
		{"boolean", &AttributeExpr{Type: Boolean}},
		{"float32", &AttributeExpr{Type: Float32}},
		{"string", &AttributeExpr{Type: String}},
		{"nullable string", &AttributeExpr{Type: String, Nullable: true}},
		{"outer nullable alias", &AttributeExpr{Type: namedString, Nullable: true}},
		{"object", object},
		{"array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}},
		{"map", collisionMap(Any, String)},
		{"union", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
			{Name: "String", Attribute: &AttributeExpr{Type: String}},
			{Name: "Boolean", Attribute: &AttributeExpr{Type: Boolean}},
		}}}},
	}
	values := []struct {
		name  string
		value any
	}{
		{"raw nil", nil},
		{"typed nil pointer", typedNilObject},
		{"typed nil map", typedNilMap},
		{"typed nil slice", typedNilSlice},
		{"string", "value"},
		{"named string", collisionNamedString("value")},
		{"integer", 1},
		{"float32", float32(1.25)},
		{"float64", float64(1.25)},
		{"boolean", true},
		{"array", []int{1, 2}},
		{"array child mismatch", []any{"ok", 1}},
		{"map", map[any]string{1: "one"}},
		{"map child mismatch", map[any]any{1: true}},
		{"object map", map[string]any{"value": "ok"}},
		{"object struct", objectValue},
		{"pointer scalar", &stringValue},
		{"pointer array", &arrayValue},
		{"pointer map", &mapValue},
		{"pointer object", &objectValue},
		{"codec map", collisionCodecMap{1: "one"}},
		{"codec object", collisionCodecObject{Value: "ok"}},
		{"codec named string", collisionCodecString("value")},
	}
	collisionCompatibilityCodecCalls = 0
	for _, declaration := range declarations {
		t.Run(declaration.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.newCollisionOccurrence(declaration.attribute)
			require.NoError(t, err)
			for _, value := range values {
				t.Run(value.name, func(t *testing.T) {
					want := legacyExampleValueCompatible(declaration.attribute, value.value)
					result := context.mapKeyCollisionResult(occurrence,
						context.SupplyValue(ValueInput{Raw: value.value}))
					require.NotEqual(t, valueCollisionUnknown, result.compatibility)
					require.Equal(t, want, result.compatibility == valueCollisionCompatible)
				})
			}
		})
	}
	require.Zero(t, collisionCompatibilityCodecCalls)
}

func legacyExampleValueCompatible(attribute *AttributeExpr, value any) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if value == nil {
		return AllowsNull(attribute)
	}
	if userType, ok := attribute.Type.(UserType); ok {
		return legacyExampleValueCompatible(userType.Attribute(), value)
	}
	switch actual := attribute.Type.(type) {
	case *Array:
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Array && kind != reflect.Slice {
			return false
		}
		items := reflect.ValueOf(value)
		for index := range items.Len() {
			if !legacyExampleValueCompatible(actual.ElemType, items.Index(index).Interface()) {
				return false
			}
		}
		return true
	case *Map:
		if reflect.TypeOf(value).Kind() != reflect.Map {
			return false
		}
		mapping := reflect.ValueOf(value)
		for _, key := range mapping.MapKeys() {
			if !legacyExampleValueCompatible(actual.KeyType, key.Interface()) ||
				!legacyExampleValueCompatible(actual.ElemType, mapping.MapIndex(key).Interface()) {
				return false
			}
		}
		return true
	case *Union:
		for _, variant := range actual.Values {
			if legacyExampleValueCompatible(variant.Attribute, value) {
				return true
			}
		}
		return false
	default:
		return attribute.Type.IsCompatible(value)
	}
}

func TestCollisionCompatibilityUnknownIsDeferred(t *testing.T) {
	if os.Getenv("LOOM456_RECURSIVE_COMPATIBILITY_HELPER") == "1" {
		recursiveMap := &AttributeExpr{}
		recursiveMap.Type = &Map{
			KeyType:  &AttributeExpr{Type: String},
			ElemType: recursiveMap,
		}
		recursiveValue := map[string]any{}
		recursiveValue["self"] = recursiveValue
		recursiveMap.UserExamples = []*ExampleExpr{{Value: recursiveValue}}
		require.Empty(t, recursiveMap.validateExamples("attribute - ", recursiveMap).Errors)

		collisionBranch := collisionMap(Float32, Any)
		cyclicChoice := map[any]any{}
		cyclicChoice[float64(1.23456789)] = cyclicChoice
		cyclicChoice[float32(1.23456789)] = cyclicChoice

		for _, test := range []struct {
			name      string
			keyType   *AttributeExpr
			wantError bool
		}{
			{name: "compatible cycle is unknown", keyType: &AttributeExpr{Type: Any}},
			{name: "incompatible cycle does not block", keyType: &AttributeExpr{Type: String}, wantError: true},
		} {
			t.Run(test.name, func(t *testing.T) {
				recursive := &AttributeExpr{}
				recursive.Type = &Map{KeyType: test.keyType, ElemType: recursive}
				choice := &AttributeExpr{Type: &Union{TypeName: "RecursiveChoice", Untagged: true,
					Values: []*NamedAttributeExpr{
						{Name: "Collision", Attribute: collisionBranch},
						{Name: "Recursive", Attribute: recursive},
					}}}
				choice.UserExamples = []*ExampleExpr{{Value: cyclicChoice}}
				errors := choice.validateExamples("attribute - ", choice).Errors
				if test.wantError {
					require.NotEmpty(t, errors)
					return
				}
				require.Empty(t, errors)
			})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCollisionCompatibilityUnknownIsDeferred$")
	command.Env = append(os.Environ(), "LOOM456_RECURSIVE_COMPATIBILITY_HELPER=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NoError(t, ctx.Err(), string(output))
}

func TestCollisionOccurrenceOmitsSemanticSourcesAndResultViews(t *testing.T) {
	child := collisionMap(Any, String)
	child.DefaultValue = map[any]string{"default": "value"}
	child.Validation = &ValidationExpr{Values: []any{map[any]string{"default": "value"}}}
	child.UserExamples = []*ExampleExpr{{Value: map[any]string{"example": "value"}}}
	result := &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			TypeName: "CollisionResult",
			AttributeExpr: &AttributeExpr{Type: &Object{{
				Name: "items", Attribute: child,
			}}},
		},
		Identifier: "application/vnd.collision",
		Views: []*ViewExpr{{
			Name:          "summary",
			AttributeExpr: &AttributeExpr{Type: &Object{}},
		}},
	}
	root := &AttributeExpr{Type: result}

	context := NewValueContext()
	collisionOccurrence, err := context.newCollisionOccurrence(root)
	require.NoError(t, err)
	copyResult, ok := collisionOccurrence.node.declaration.typ.(*ResultTypeExpr)
	require.True(t, ok)
	require.Equal(t, result.Identifier, copyResult.Identifier)
	require.Equal(t, result.Name(), copyResult.Name())
	require.Empty(t, copyResult.Views)
	copiedChild := collisionOccurrence.node.declaration.alias.declaration.members[0].node.attribute
	require.Nil(t, copiedChild.Validation)
	require.Nil(t, copiedChild.DefaultValue)
	require.Nil(t, copiedChild.UserExamples)

	ordinary, err := context.NewOccurrence(root)
	require.NoError(t, err)
	ordinaryResult := ordinary.node.declaration.typ.(*ResultTypeExpr)
	require.Len(t, ordinaryResult.Views, 1)
	ordinaryChild := ordinary.node.declaration.alias.declaration.members[0].node
	require.NotNil(t, ordinaryChild.attribute.Validation)
	require.NotNil(t, ordinaryChild.defaultValue)
	require.Len(t, ordinaryChild.examples, 1)
}

func TestCollisionOccurrenceOmitsMalformedResultViews(t *testing.T) {
	result := &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			TypeName: "CollisionResult",
			AttributeExpr: &AttributeExpr{Type: &Object{{
				Name: "items", Attribute: collisionMap(Any, String),
			}}},
		},
		Identifier: "application/vnd.collision",
		Views: []*ViewExpr{{
			Name:          "malformed",
			AttributeExpr: &AttributeExpr{},
		}},
	}
	attribute := &AttributeExpr{Type: result}
	context := NewValueContext()
	_, err := context.NewOccurrence(attribute)
	require.ErrorContains(t, err, `view "malformed"`)

	occurrence, err := context.newCollisionOccurrence(attribute)
	require.NoError(t, err)
	raw := map[string]any{"items": map[any]string{1: "integer", "1": "string"}}
	query := context.mapKeyCollisionResult(occurrence, context.SupplyValue(ValueInput{Raw: raw}))
	require.Len(t, query.evidence.collisions, 1)
	require.Equal(t, []string{"items"}, query.evidence.collisions[0].path)
}

func TestCollisionOccurrenceDefersMalformedDeclarationShape(t *testing.T) {
	attribute := collisionMap(Any, nil)
	attribute.UserExamples = []*ExampleExpr{{Value: map[any]string{1: "integer", "1": "string"}}}
	context := NewValueContext()
	_, ordinaryErr := context.NewOccurrence(attribute)
	require.ErrorContains(t, ordinaryErr, "map value")
	_, collisionErr := context.newCollisionOccurrence(attribute)
	require.ErrorContains(t, collisionErr, "map value")

	validated = make(map[*AttributeExpr]bool)
	errors := attribute.Validate("attribute", attribute)
	require.Contains(t, errors.Error(), "attribute type is nil")
	require.NotContains(t, errors.Error(), "map keys collide")
}
