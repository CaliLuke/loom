package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/testutil"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	ctestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/expr"
)

type protoBufTransformCase struct {
	Name    string
	Source  expr.DataType
	Target  expr.DataType
	ToProto bool
	Ctx     *codegen.AttributeContext
}

type protoBufTransformPair struct {
	ProtoName     string
	ServiceName   string
	ProtoSource   expr.DataType
	ProtoTarget   expr.DataType
	ServiceSource expr.DataType
	ServiceTarget expr.DataType
	ServiceCtx    *codegen.AttributeContext
	ProtoCtx      *codegen.AttributeContext
}

func TestProtoBufTransform(t *testing.T) {
	root := codegen.RunDSL(t, ctestdata.TestTypesDSL)
	var (
		sd = &ServiceData{Name: "Service", Scope: codegen.NewNameScope()}

		// types to test
		primitive = expr.Int

		simple     = root.UserType("Simple")
		required   = root.UserType("Required")
		defaultT   = root.UserType("Default")
		customtype = root.UserType("CustomTypes")

		simpleMap  = root.UserType("SimpleMap")
		nestedMap  = root.UserType("NestedMap")
		arrayMap   = root.UserType("ArrayMap")
		defaultMap = root.UserType("DefaultMap")

		simpleArray  = root.UserType("SimpleArray")
		nestedArray  = root.UserType("NestedArray")
		mapArray     = root.UserType("MapArray")
		typeArray    = root.UserType("TypeArray")
		defaultArray = root.UserType("DefaultArray")

		recursive   = root.UserType("Recursive")
		composite   = root.UserType("Composite")
		customField = root.UserType("CompositeWithCustomField")
		optional    = root.UserType("Optional")
		defaults    = root.UserType("WithDefaults")

		resultType = root.UserType("ResultType")
		rtCol      = root.UserType("ResultTypeCollection")

		simpleOneOf    = root.UserType("SimpleOneOf")
		embeddedOneOf  = root.UserType("EmbeddedOneOf")
		recursiveOneOf = root.UserType("RecursiveOneOf")

		pkgOverride = root.UserType("CompositePkgOverride")

		// attribute contexts used in test cases
		svcCtx = serviceTypeContext("proto", sd.Scope)
		ptrCtx = pointerContext("proto", sd.Scope)
		pbCtx  = protoBufTypeContext("proto", sd.Scope, true)
	)

	// gRPC does not support any
	obj := expr.AsObject(defaults)
	for _, nat := range *obj {
		if nat.Name == "any" {
			nat.Attribute.Type = expr.String
		}
		if nat.Name == "required_any" {
			nat.Attribute.Type = expr.String
		}
	}

	pairs := []protoBufTransformPair{
		{ProtoName: "primitive-to-primitive", ServiceName: "primitive-to-primitive", ProtoSource: primitive, ProtoTarget: primitive, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "simple-to-simple", ServiceName: "simple-to-simple", ProtoSource: simple, ProtoTarget: simple, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "simple-to-required", ServiceName: "simple-to-required", ProtoSource: simple, ProtoTarget: required, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "required-to-simple", ServiceName: "required-to-simple", ProtoSource: required, ProtoTarget: simple, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "simple-to-default", ServiceName: "simple-to-default", ProtoSource: simple, ProtoTarget: defaultT, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "default-to-simple", ServiceName: "default-to-simple", ProtoSource: defaultT, ProtoTarget: simple, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{
			ProtoName:     "required-ptr-to-simple",
			ServiceName:   "simple-to-required-ptr",
			ProtoSource:   required,
			ProtoTarget:   simple,
			ServiceSource: simple,
			ServiceTarget: required,
			ServiceCtx:    ptrCtx,
			ProtoCtx:      ptrCtx,
		},
		{
			ProtoName:     "simple-to-customtype",
			ServiceName:   "simple-to-customtype",
			ProtoSource:   customtype,
			ProtoTarget:   simple,
			ServiceSource: simple,
			ServiceTarget: customtype,
			ServiceCtx:    svcCtx,
			ProtoCtx:      svcCtx,
		},
		{ProtoName: "customtype-to-customtype", ServiceName: "customtype-to-customtype", ProtoSource: customtype, ProtoTarget: customtype, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "map-to-map", ServiceName: "map-to-map", ProtoSource: simpleMap, ProtoTarget: simpleMap, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "nested-map-to-nested-map", ServiceName: "nested-map-to-nested-map", ProtoSource: nestedMap, ProtoTarget: nestedMap, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "array-map-to-array-map", ServiceName: "array-map-to-array-map", ProtoSource: arrayMap, ProtoTarget: arrayMap, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "default-map-to-default-map", ServiceName: "default-map-to-default-map", ProtoSource: defaultMap, ProtoTarget: defaultMap, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "array-to-array", ServiceName: "array-to-array", ProtoSource: simpleArray, ProtoTarget: simpleArray, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "nested-array-to-nested-array", ServiceName: "nested-array-to-nested-array", ProtoSource: nestedArray, ProtoTarget: nestedArray, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "type-array-to-type-array", ServiceName: "type-array-to-type-array", ProtoSource: typeArray, ProtoTarget: typeArray, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "map-array-to-map-array", ServiceName: "map-array-to-map-array", ProtoSource: mapArray, ProtoTarget: mapArray, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "default-array-to-default-array", ServiceName: "default-array-to-default-array", ProtoSource: defaultArray, ProtoTarget: defaultArray, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "recursive-to-recursive", ServiceName: "recursive-to-recursive", ProtoSource: recursive, ProtoTarget: recursive, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "composite-to-custom-field", ServiceName: "composite-to-custom-field", ProtoSource: composite, ProtoTarget: customField, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "custom-field-to-composite", ServiceName: "custom-field-to-composite", ProtoSource: customField, ProtoTarget: composite, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "result-type-to-result-type", ServiceName: "result-type-to-result-type", ProtoSource: resultType, ProtoTarget: resultType, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "result-type-collection-to-result-type-collection", ServiceName: "result-type-collection-to-result-type-collection", ProtoSource: rtCol, ProtoTarget: rtCol, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "optional-to-optional", ServiceName: "optional-to-optional", ProtoSource: optional, ProtoTarget: optional, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "defaults-to-defaults", ServiceName: "defaults-to-defaults", ProtoSource: defaults, ProtoTarget: defaults, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "oneof-to-oneof", ServiceName: "oneof-to-oneof", ProtoSource: simpleOneOf, ProtoTarget: simpleOneOf, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "embedded-oneof-to-embedded-oneof", ServiceName: "embedded-oneof-to-embedded-oneof", ProtoSource: embeddedOneOf, ProtoTarget: embeddedOneOf, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "recursive-oneof-to-recursive-oneof", ServiceName: "recursive-oneof-to-recursive-oneof", ProtoSource: recursiveOneOf, ProtoTarget: recursiveOneOf, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
		{ProtoName: "pkg-override-to-pkg-override", ServiceName: "pkg-override-to-pkg-override", ProtoSource: pkgOverride, ProtoTarget: pkgOverride, ServiceCtx: svcCtx, ProtoCtx: svcCtx},
	}
	tc := map[string][]protoBufTransformCase{
		"to-protobuf-type": buildProtoBufTransformCases(true, pairs),
		"to-service-type":  buildProtoBufTransformCases(false, pairs),
	}
	for name, cases := range tc {
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.Name, func(t *testing.T) {
					source := &expr.AttributeExpr{Type: c.Source}
					target := &expr.AttributeExpr{Type: c.Target}
					srcCtx := c.Ctx
					tgtCtx := c.Ctx
					if c.ToProto {
						target = makeProtoBufMessage(expr.DupAtt(target), target.Type.Name(), sd)
						tgtCtx = pbCtx
					} else {
						source = makeProtoBufMessage(expr.DupAtt(source), source.Type.Name(), sd)
						srcCtx = pbCtx
					}
					code, _, err := protoBufTransform(source, target, "source", "target", srcCtx, tgtCtx, c.ToProto, true)
					require.NoError(t, err)
					code = codegen.FormatTestCode(t, "package foo\nfunc transform(){\n"+code+"}")
					testutil.AssertGo(t, "testdata/golden/protobuf_type_encode_"+name+"_"+c.Name+".go.golden", code)
				})
			}
		})
	}
}

func buildProtoBufTransformCases(toProto bool, pairs []protoBufTransformPair) []protoBufTransformCase {
	cases := make([]protoBufTransformCase, 0, len(pairs))
	for _, pair := range pairs {
		tc := protoBufTransformCase{
			Source:  pair.ProtoSource,
			Target:  pair.ProtoTarget,
			ToProto: toProto,
			Ctx:     pair.ProtoCtx,
			Name:    pair.ProtoName,
		}
		if !toProto {
			tc.Name = pair.ServiceName
			if pair.ServiceSource != nil {
				tc.Source = pair.ServiceSource
			}
			if pair.ServiceTarget != nil {
				tc.Target = pair.ServiceTarget
			}
			tc.Ctx = pair.ServiceCtx
		}
		cases = append(cases, tc)
	}
	return cases
}

func TestProtoBufTransformAnyType(t *testing.T) {
	var (
		sd     = &ServiceData{Name: "Service", Scope: codegen.NewNameScope()}
		svcCtx = codegen.NewAttributeContext(false, false, true, "", sd.Scope)
		pbCtx  = protoBufTypeContext("", sd.Scope, false)
	)

	cases := []struct {
		Name    string
		ToProto bool
		NewVar  bool
		Ctx     *codegen.AttributeContext
	}{
		{"any-to-proto-new-var", true, true, svcCtx},
		{"proto-to-any-new-var", false, true, svcCtx},
		{"any-to-proto-assign-var", true, false, svcCtx},
		{"proto-to-any-assign-var", false, false, svcCtx},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: expr.Any}
			target := &expr.AttributeExpr{Type: expr.Any}
			srcCtx := c.Ctx
			tgtCtx := c.Ctx
			if c.ToProto {
				tgtCtx = pbCtx
			} else {
				srcCtx = pbCtx
			}
			code, _, err := protoBufTransform(source, target, "source", "target", srcCtx, tgtCtx, c.ToProto, c.NewVar)
			require.NoError(t, err)
			t.Logf("Generated code: %s", code)

			// Check if transformation contains Any type conversion logic
			if c.ToProto {
				require.Contains(t, code, "func() *structpb.Value", "To proto conversion should generate Value type conversion function")
				require.Contains(t, code, "loomgrpc.NewProtoValue(source)")
				require.Contains(t, code, "*transformErr = err")
				require.NotContains(t, code, "panic(", "To proto conversion should not panic on values unsupported by structpb")
			} else {
				require.Contains(t, code, "func() any", "From proto conversion should generate any type conversion function")
			}

			// Check the assignment operator used based on newVar parameter
			if c.NewVar {
				require.Contains(t, code, "target :=", "New variable should use := operator")
			} else {
				require.Contains(t, code, "target =", "Assignment should use = operator")
				require.NotContains(t, code, "target :=", "Assignment should not use := operator")
			}
		})
	}
}

func TestProtoBufTransformAnyObjectPreservesPresence(t *testing.T) {
	object := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Any}}},
		Validation: &expr.ValidationExpr{Required: []string{"data"}},
	}
	scope := codegen.NewNameScope()
	svcCtx := codegen.NewAttributeContext(false, false, true, "", scope)
	pbCtx := protoBufTypeContext("", scope, false)

	toProto, _, err := protoBufTransform(object, object, "source", "target", svcCtx, pbCtx, true, true)
	require.NoError(t, err)
	require.Contains(t, toProto, "source.Data.IsNull()")
	require.Contains(t, toProto, "source.Data.Value()")
	require.Contains(t, toProto, "structpb.NewNullValue()")
	require.Contains(t, toProto, "loomgrpc.NewProtoValue(actual)")
	require.NotContains(t, toProto, "loomgrpc.NewProtoValue(source.Data)")

	fromProto, _, err := protoBufTransform(object, object, "source", "target", pbCtx, svcCtx, false, true)
	require.NoError(t, err)
	require.Contains(t, fromProto, "source.Data.GetKind().(*structpb.Value_NullValue)")
	require.Contains(t, fromProto, "target.Data.SetNull()")
	require.Contains(t, fromProto, "target.Data.SetValue(source.Data.AsInterface())")
}

func TestProtoBufTransformNamedAnyObjectPreservesPresence(t *testing.T) {
	anything := &expr.UserTypeExpr{
		TypeName:      "Anything",
		AttributeExpr: &expr.AttributeExpr{Type: expr.Any},
	}
	object := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: anything}}},
		Validation: &expr.ValidationExpr{Required: []string{"data"}},
	}
	scope := codegen.NewNameScope()
	svcCtx := codegen.NewAttributeContext(false, false, true, "", scope)
	pbCtx := protoBufTypeContext("", scope, false)

	toProto, _, err := protoBufTransform(object, object, "source", "target", svcCtx, pbCtx, true, true)
	require.NoError(t, err)
	require.Contains(t, toProto, "source.Data.IsNull()")
	require.Contains(t, toProto, "source.Data.Value()")
	require.Contains(t, toProto, "loomgrpc.NewProtoValue(actual)")

	fromProto, _, err := protoBufTransform(object, object, "source", "target", pbCtx, svcCtx, false, true)
	require.NoError(t, err)
	require.Contains(t, fromProto, "target.Data.SetNull()")
	require.Contains(t, fromProto, "target.Data.SetValue(Anything(source.Data.AsInterface()))")
}

func TestProtoBufTransformSeams(t *testing.T) {
	root := codegen.RunDSL(t, ctestdata.TestTypesDSL)
	sd := &ServiceData{Name: "Service", Scope: codegen.NewNameScope()}
	svcCtx := serviceTypeContext("proto", sd.Scope)
	ptrCtx := pointerContext("proto", sd.Scope)
	pbCtx := protoBufTypeContext("proto", sd.Scope, true)

	cases := []struct {
		name     string
		source   expr.DataType
		target   expr.DataType
		toProto  bool
		newVar   bool
		ctx      *codegen.AttributeContext
		contains string
	}{
		{
			name:     "wrapped scalar field to proto",
			source:   expr.Int,
			target:   expr.Int,
			toProto:  true,
			newVar:   true,
			ctx:      svcCtx,
			contains: ".Field",
		},
		{
			name:     "optional pointer field to proto checks nil",
			source:   root.UserType("Optional"),
			target:   root.UserType("Optional"),
			toProto:  true,
			newVar:   true,
			ctx:      ptrCtx,
			contains: "!= nil",
		},
		{
			name:     "alias conversion to service uses cast",
			source:   root.UserType("Simple"),
			target:   root.UserType("CustomTypes"),
			toProto:  false,
			newVar:   true,
			ctx:      svcCtx,
			contains: "CustomTypes",
		},
		{
			name:     "union to proto uses concrete oneof conversion",
			source:   root.UserType("SimpleOneOf"),
			target:   root.UserType("SimpleOneOf"),
			toProto:  true,
			newVar:   true,
			ctx:      svcCtx,
			contains: "case",
		},
		{
			name:     "assignment path uses equals",
			source:   expr.String,
			target:   expr.String,
			toProto:  false,
			newVar:   false,
			ctx:      svcCtx,
			contains: "target =",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: tc.source}
			target := &expr.AttributeExpr{Type: tc.target}
			srcCtx := tc.ctx
			tgtCtx := tc.ctx
			if tc.toProto {
				target = makeProtoBufMessage(expr.DupAtt(target), target.Type.Name(), sd)
				tgtCtx = pbCtx
			} else {
				source = makeProtoBufMessage(expr.DupAtt(source), source.Type.Name(), sd)
				srcCtx = pbCtx
			}

			code, _, err := protoBufTransform(source, target, "source", "target", srcCtx, tgtCtx, tc.toProto, tc.newVar)
			require.NoError(t, err)
			require.Contains(t, code, tc.contains)
		})
	}
}

func pointerContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(true, false, true, pkg, scope)
}
