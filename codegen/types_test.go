package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
)

func TestGoTypeDef(t *testing.T) {
	var (
		simpleArray = &expr.AttributeExpr{
			Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Boolean}}}
		simpleMap = &expr.AttributeExpr{
			Type: &expr.Map{
				KeyType:  &expr.AttributeExpr{Type: expr.Int},
				ElemType: &expr.AttributeExpr{Type: expr.String},
			}}
		requiredObj = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "IntField", Attribute: &expr.AttributeExpr{Type: expr.Int}},
				{Name: "StringField", Attribute: &expr.AttributeExpr{Type: expr.String}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"IntField", "StringField"}}}
		defaultObj = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "IntField", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 1}},
				{Name: "StringField", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "foo"}},
			}}
		ut                          = &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}, TypeName: "UserType"}
		rt                          = &expr.ResultTypeExpr{UserTypeExpr: &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}, TypeName: "ResultType"}, Identifier: "application/vnd.loom.example", Views: nil}
		userType                    = &expr.AttributeExpr{Type: ut}
		resultType                  = &expr.AttributeExpr{Type: rt}
		stringMetaType              = expr.MetaExpr{"struct:field:type": []string{"string"}}
		jsonWithImportMetaType      = expr.MetaExpr{"struct:field:type": []string{"jsontext.Value", "encoding/json/jsontext"}}
		jsonWithRenameMetaType      = expr.MetaExpr{"struct:field:type": []string{"jtext.Value", "encoding/json/jsontext", "jason"}}
		structPkgPathMetaType       = expr.MetaExpr{"struct:pkg:path": []string{"types"}}
		nestedStructPkgPathMetaType = expr.MetaExpr{"struct:pkg:path": []string{"nested/pkg"}}
		utPkgPathMeta               = &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean, Meta: structPkgPathMetaType}, TypeName: "UserType"}
		nestedUTPkgPathMeta         = &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean, Meta: nestedStructPkgPathMetaType}, TypeName: "NestedUserType"}
		unionType                   = &expr.Union{TypeName: "StringOrInt", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}, {Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}}}}
		optionalUnionObj            = &expr.AttributeExpr{Type: &expr.Object{{Name: "choice", Attribute: &expr.AttributeExpr{Type: unionType}}}}
		requiredUnionObj            = &expr.AttributeExpr{Type: &expr.Object{{Name: "choice", Attribute: &expr.AttributeExpr{Type: unionType}}}, Validation: &expr.ValidationExpr{Required: []string{"choice"}}}
		nullableObject              = &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Nullable: true}
		nullableObjectArray         = &expr.AttributeExpr{Type: &expr.Array{ElemType: nullableObject}}
		nullableUnionArray          = &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: unionType, Nullable: true}}}
		nullableObjectMap           = &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: nullableObject}}

		mixedObj = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "IntField", Attribute: &expr.AttributeExpr{Type: expr.Int}},
				{Name: "ArrayField", Attribute: simpleArray},
				{Name: "MapField", Attribute: simpleMap},
				{Name: "UserTypeField", Attribute: userType},
				{Name: "MetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithImportMetaType}},
				{Name: "QualifiedMetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithRenameMetaType}},
				{Name: "StructPkgPath", Attribute: &expr.AttributeExpr{Type: utPkgPathMeta}},
				{Name: "NestedStructPkgPath", Attribute: &expr.AttributeExpr{Type: nestedUTPkgPathMeta}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"IntField", "ArrayField", "MapField", "UserTypeField", "MetaTypeField", "QualifiedMetaTypeField"}}}
		mixedObjWithStructPkgPath = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "IntField", Attribute: &expr.AttributeExpr{Type: expr.Int}},
				{Name: "ArrayField", Attribute: simpleArray},
				{Name: "MapField", Attribute: simpleMap},
				{Name: "UserTypeField", Attribute: userType},
				{Name: "MetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithImportMetaType}},
				{Name: "QualifiedMetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithRenameMetaType}},
				{Name: "StructPkgPath", Attribute: &expr.AttributeExpr{Type: utPkgPathMeta}},
				{Name: "NestedStructPkgPath", Attribute: &expr.AttributeExpr{Type: nestedUTPkgPathMeta}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"IntField", "ArrayField", "MapField", "UserTypeField", "MetaTypeField", "QualifiedMetaTypeField"}},
			Meta:       structPkgPathMetaType,
		}
		mixedObjWithNestedStructPkgPath = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "IntField", Attribute: &expr.AttributeExpr{Type: expr.Int}},
				{Name: "ArrayField", Attribute: simpleArray},
				{Name: "MapField", Attribute: simpleMap},
				{Name: "UserTypeField", Attribute: userType},
				{Name: "MetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithImportMetaType}},
				{Name: "QualifiedMetaTypeField", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: jsonWithRenameMetaType}},
				{Name: "StructPkgPath", Attribute: &expr.AttributeExpr{Type: utPkgPathMeta}},
				{Name: "NestedStructPkgPath", Attribute: &expr.AttributeExpr{Type: nestedUTPkgPathMeta}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"IntField", "ArrayField", "MapField", "UserTypeField", "MetaTypeField", "QualifiedMetaTypeField"}},
			Meta:       nestedStructPkgPathMetaType,
		}
	)
	cases := map[string]struct {
		att        *expr.AttributeExpr
		pointer    bool
		usedefault bool
		expected   string
	}{
		"BooleanKind": {&expr.AttributeExpr{Type: expr.Boolean}, false, true, "bool"},
		"IntKind":     {&expr.AttributeExpr{Type: expr.Int}, false, true, "int"},
		"Int32Kind":   {&expr.AttributeExpr{Type: expr.Int32}, false, true, "int32"},
		"Int64Kind":   {&expr.AttributeExpr{Type: expr.Int64}, false, true, "int64"},
		"UIntKind":    {&expr.AttributeExpr{Type: expr.UInt}, false, true, "uint"},
		"UInt32Kind":  {&expr.AttributeExpr{Type: expr.UInt32}, false, true, "uint32"},
		"UInt64Kind":  {&expr.AttributeExpr{Type: expr.UInt64}, false, true, "uint64"},
		"Float32Kind": {&expr.AttributeExpr{Type: expr.Float32}, false, true, "float32"},
		"Float64Kind": {&expr.AttributeExpr{Type: expr.Float64}, false, true, "float64"},
		"StringKind":  {&expr.AttributeExpr{Type: expr.String}, false, true, "string"},
		"BytesKind":   {&expr.AttributeExpr{Type: expr.Bytes}, false, true, "[]byte"},
		"AnyKind":     {&expr.AttributeExpr{Type: expr.Any}, false, true, "loom.JSONValue"},

		"Array":               {simpleArray, false, true, "[]bool"},
		"Map":                 {simpleMap, false, true, "map[int]string"},
		"NullableObjectArray": {nullableObjectArray, false, true, "[]loom.Nullable[struct {\n\tValue *string `json:\"value,omitempty\"`\n}]"},
		"NullableUnionArray":  {nullableUnionArray, false, true, "[]loom.Nullable[StringOrInt]"},
		"NullableObjectMap":   {nullableObjectMap, false, true, "map[string]loom.Nullable[struct {\n\tValue *string `json:\"value,omitempty\"`\n}]"},
		"UserTypeExpr":        {userType, false, true, "UserType"},
		"ResultTypeExpr":      {resultType, false, true, "ResultType"},

		"Object":          {requiredObj, false, true, "struct {\n\tIntField int `json:\"IntField\"`\n\tStringField string `json:\"StringField\"`\n}"},
		"ObjDefault":      {defaultObj, false, true, "struct {\n\tIntField int `json:\"IntField,omitempty\"`\n\tStringField string `json:\"StringField,omitempty\"`\n}"},
		"ObjDefaultNoDef": {defaultObj, false, false, "struct {\n\tIntField *int `json:\"IntField,omitempty\"`\n\tStringField *string `json:\"StringField,omitempty\"`\n}"},
		"OptionalUnion":   {optionalUnionObj, false, true, "struct {\n\tChoice *StringOrInt `json:\"choice,omitempty\"`\n}"},
		"RequiredUnion":   {requiredUnionObj, false, true, "struct {\n\tChoice StringOrInt `json:\"choice\"`\n}"},
		"NullableFields": {&expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "required", Attribute: &expr.AttributeExpr{Type: expr.String, Nullable: true}},
				{Name: "optional", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}, Nullable: true}},
				{Name: "anything", Attribute: &expr.AttributeExpr{Type: expr.Any}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"required"}},
		}, false, true, "struct {\n\tRequired loom.Nullable[string] `json:\"required\"`\n\tOptional loom.Nullable[[]int] `json:\"optional,omitzero\"`\n\tAnything loom.JSONValue `json:\"anything,omitzero\"`\n}"},
		"ObjMixed":                               {mixedObj, false, true, "struct {\n\tIntField int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField UserType `json:\"UserTypeField\"`\n\tMetaTypeField jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *types.UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *pkg.NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},
		"ObjMixedPointer":                        {mixedObj, true, true, "struct {\n\tIntField *int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField *UserType `json:\"UserTypeField\"`\n\tMetaTypeField *jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField *jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *types.UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *pkg.NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},
		"ObjMixedWithStructPkgPath":              {mixedObjWithStructPkgPath, false, true, "struct {\n\tIntField int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField UserType `json:\"UserTypeField\"`\n\tMetaTypeField jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *pkg.NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},
		"ObjMixedWithStructPkgPathPointer":       {mixedObjWithStructPkgPath, true, true, "struct {\n\tIntField *int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField *UserType `json:\"UserTypeField\"`\n\tMetaTypeField *jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField *jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *pkg.NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},
		"ObjMixedWithNestedStructPkgPath":        {mixedObjWithNestedStructPkgPath, false, true, "struct {\n\tIntField int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField UserType `json:\"UserTypeField\"`\n\tMetaTypeField jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *types.UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},
		"ObjMixedWithNestedStructPkgPathPointer": {mixedObjWithNestedStructPkgPath, true, true, "struct {\n\tIntField *int `json:\"IntField\"`\n\tArrayField []bool `json:\"ArrayField\"`\n\tMapField map[int]string `json:\"MapField\"`\n\tUserTypeField *UserType `json:\"UserTypeField\"`\n\tMetaTypeField *jsontext.Value `json:\"MetaTypeField\"`\n\tQualifiedMetaTypeField *jtext.Value `json:\"QualifiedMetaTypeField\"`\n\tStructPkgPath *types.UserType `json:\"StructPkgPath,omitempty\"`\n\tNestedStructPkgPath *NestedUserType `json:\"NestedStructPkgPath,omitempty\"`\n}"},

		"MetaTypeSameAsDesign":                      {&expr.AttributeExpr{Type: expr.String, Meta: stringMetaType}, false, true, "string"},
		"MetaTypeOverrideDesign":                    {&expr.AttributeExpr{Type: expr.String, Meta: jsonWithImportMetaType}, false, true, "jsontext.Value"},
		"MetaTypeOverrideDesignWithQualifiedImport": {&expr.AttributeExpr{Type: expr.String, Meta: jsonWithRenameMetaType}, false, true, "jtext.Value"},
	}

	for k, tc := range cases {
		scope := NewNameScope()
		actual := scope.GoTypeDef(tc.att, tc.pointer, tc.usedefault)
		if actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeTagsWithNameNormalizesPresenceOverride(t *testing.T) {
	field := &expr.AttributeExpr{
		Type:     expr.String,
		Nullable: true,
		Meta:     expr.MetaExpr{"struct:tag:json": []string{"wire_name,omitempty"}},
	}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}

	if got, want := AttributeTagsWithName(parent, "value", field), " `json:\"wire_name,omitzero\"`"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGoValueTypeDefStripsInheritedNamedPresence(t *testing.T) {
	root := &expr.AttributeExpr{Type: expr.String, Nullable: true}
	named := &expr.UserTypeExpr{TypeName: "NullableText", AttributeExpr: root}
	attribute := &expr.AttributeExpr{Type: named}
	scope := NewNameScope()

	if got, want := scope.GoValueTypeDef(attribute, false, true), "NullableText"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := scope.GoTypeDef(attribute, false, true), "loom.Nullable[NullableText]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGoNativeTypeName(t *testing.T) {
	cases := map[string]struct {
		dataType expr.DataType
		expected string
	}{
		"BooleanKind": {expr.Boolean, "bool"},
		"IntKind":     {expr.Int, "int"},
		"Int32Kind":   {expr.Int32, "int32"},
		"Int64Kind":   {expr.Int64, "int64"},
		"UIntKind":    {expr.UInt, "uint"},
		"UInt32Kind":  {expr.UInt32, "uint32"},
		"UInt64Kind":  {expr.UInt64, "uint64"},
		"Float32Kind": {expr.Float32, "float32"},
		"Float64Kind": {expr.Float64, "float64"},
		"StringKind":  {expr.String, "string"},
		"BytesKind":   {expr.Bytes, "[]byte"},
		"AnyKind":     {expr.Any, "any"},
	}

	for k, tc := range cases {
		actual := GoNativeTypeName(tc.dataType)
		if actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeTagsWithName_JSONName(t *testing.T) {
	parent := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "RequiredField", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "OptionalField", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"RequiredField"}},
	}
	required := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"struct:tag:json:name": []string{"required_field"}},
	}
	optional := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"struct:tag:json:name": []string{"optional_field"}},
	}
	if got, want := AttributeTagsWithName(parent, "RequiredField", required), " `json:\"required_field\"`"; got != want {
		t.Fatalf("required: got %q, want %q", got, want)
	}
	if got, want := AttributeTagsWithName(parent, "OptionalField", optional), " `json:\"optional_field,omitempty\"`"; got != want {
		t.Fatalf("optional: got %q, want %q", got, want)
	}
}

func TestJSONFieldNameUsesGeneratedTagPrecedence(t *testing.T) {
	tests := map[string]struct {
		attribute *expr.AttributeExpr
		want      string
	}{
		"default": {
			attribute: &expr.AttributeExpr{Type: expr.String},
			want:      "AuthoredName",
		},
		"name metadata": {
			attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json:name": {"wire_name"}}},
			want:      "wire_name",
		},
		"complete tag wins": {
			attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{
				"struct:tag:json":      {"exact,omitempty"},
				"struct:tag:json:name": {"ignored"},
			}},
			want: "exact",
		},
		"split complete tag": {
			attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{
				"struct:tag:json": {"wire_name", "omitempty"},
			}},
			want: "wire_name",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := JSONFieldName("AuthoredName", test.attribute); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestAttributeTagsWithName_DefaultJSONName(t *testing.T) {
	parent := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "required_field", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "optional_field", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"required_field"}},
	}
	required := &expr.AttributeExpr{Type: expr.String}
	optional := &expr.AttributeExpr{Type: expr.String}
	if got, want := AttributeTagsWithName(parent, "required_field", required), " `json:\"required_field\"`"; got != want {
		t.Fatalf("required: got %q, want %q", got, want)
	}
	if got, want := AttributeTagsWithName(parent, "optional_field", optional), " `json:\"optional_field,omitempty\"`"; got != want {
		t.Fatalf("optional: got %q, want %q", got, want)
	}
}

func TestAttributeTagsWithName_PresenceUsesOmitZero(t *testing.T) {
	parent := &expr.AttributeExpr{Type: &expr.Object{}}
	nullable := &expr.AttributeExpr{Type: expr.String, Nullable: true}
	anything := &expr.AttributeExpr{Type: expr.Any}
	if got, want := AttributeTagsWithName(parent, "nullable", nullable), " `json:\"nullable,omitzero\"`"; got != want {
		t.Fatalf("nullable: got %q, want %q", got, want)
	}
	if got, want := AttributeTagsWithName(parent, "anything", anything), " `json:\"anything,omitzero\"`"; got != want {
		t.Fatalf("anything: got %q, want %q", got, want)
	}
}

func TestIsExplicitPresenceTypeRecognizesNullableMetaType(t *testing.T) {
	nullable := &expr.AttributeExpr{
		Type: expr.Any,
		Meta: expr.MetaExpr{
			"struct:field:type": []string{"loom.Nullable[any]", "github.com/CaliLuke/loom/pkg", "loom"},
		},
	}
	custom := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"struct:field:type": []string{"jsontext.Value", "encoding/json/jsontext"}},
	}

	if !IsExplicitPresenceType(nullable) {
		t.Error("loom.Nullable meta type was not recognized as an explicit presence type")
	}
	if IsExplicitPresenceType(custom) {
		t.Error("ordinary custom meta type was recognized as an explicit presence type")
	}
}

func TestGoify(t *testing.T) {
	cases := map[string]struct {
		str        string
		firstUpper bool
		expected   string
	}{
		"empty":             {"", false, ""},
		"first upper false": {"blue_id", false, "blueID"},
		"first upper false normal identifier all lower":     {"blue", false, "blue"},
		"first upper false and UUID":                        {"blue_uuid", false, "blueUUID"},
		"first upper true":                                  {"blue_id", true, "BlueID"},
		"first upper true and UUID":                         {"blue_uuid", true, "BlueUUID"},
		"first upper true normal identifier all lower":      {"blue", true, "Blue"},
		"first upper false normal identifier":               {"Blue", false, "blue"},
		"first upper true normal identifier":                {"Blue", true, "Blue"},
		"invalid identifier":                                {"Blue%50", true, "Blue50"},
		"numeric identifier":                                {"0", true, "Val0"},
		"numeric unexported identifier":                     {"0", false, "val0"},
		"invalid identifier firstupper false":               {"Blue%50", false, "blue50"},
		"only UUID and firstupper false":                    {"UUID", false, "uuid"},
		"consecutives invalid identifiers firstupper false": {"[[fields___type]]", false, "fieldsType"},
		"consecutives invalid identifiers":                  {"[[fields___type]]", true, "FieldsType"},
		"invalid identifiers":                               {"[[", false, "val"},
		"middle upper firstupper false":                     {"MiddleUpper", false, "middleUpper"},
		"middle upper":                                      {"MiddleUpper", true, "MiddleUpper"},
	}

	for k, tc := range cases {
		actual := Goify(tc.str, tc.firstUpper)

		if actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}
