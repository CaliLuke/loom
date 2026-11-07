package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

type collectionTargetSpec struct {
	attribute        *expr.AttributeExpr
	ref, declaration string
	values           []string
}

func TestCollectionTransformsPreserveTargetPresence(t *testing.T) {
	var generated strings.Builder
	generated.WriteString("package carriers\nimport (\"reflect\";\"testing\";loom \"github.com/CaliLuke/loom/pkg\")\nfunc ptr[T any](v T)*T{return &v}\n")
	index := 0
	for _, mapping := range []bool{false, true} {
		for _, shape := range []string{"scalar", "object", "union", "nested", "alias", "nullable"} {
			for _, sourceJSON := range []bool{false, true} {
				for _, targetJSON := range []bool{false, true} {
					generated.WriteString(collectionTargetCase(t, index, shape, mapping, sourceJSON, targetJSON, false))
					index++
				}
			}
		}
		generated.WriteString(collectionTargetCase(t, index, "alias", mapping, false, false, true))
		index++
	}
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/carriers\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "carriers_test.go"), []byte(generated.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}

func collectionTargetCase(t *testing.T, index int, shape string, mapping, sourceJSON, targetJSON, namedOverride bool) string {
	t.Helper()
	source := collectionTargetElement(shape, fmt.Sprintf("Source%d", index), sourceJSON)
	target := collectionTargetElement(shape, fmt.Sprintf("Target%d", index), targetJSON || namedOverride)
	sourceCtx := NewAttributeContext(sourceJSON, false, !sourceJSON, "", NewNameScope())
	targetCtx := NewAttributeContext(targetJSON, false, !targetJSON, "", NewNameScope())
	sourceCtx.JSONPresence, sourceCtx.CollectionElementPresence = sourceJSON, sourceJSON
	targetCtx.JSONPresence, targetCtx.CollectionElementPresence = targetJSON, targetJSON
	if namedOverride {
		targetCtx.JSONPresenceTypes = map[string]bool{target.attribute.Type.(expr.UserType).Hash(): true}
	}
	root := func(element *expr.AttributeExpr) *expr.AttributeExpr {
		if mapping {
			return &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: element}}
		}
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
	}
	sourceRoot, targetRoot := root(source.attribute), root(target.attribute)
	forward, err := GoTransform(sourceRoot, targetRoot, "source", "target", sourceCtx, targetCtx, fmt.Sprintf("forward%d", index), true)
	require.NoError(t, err)
	backward, err := GoTransform(targetRoot, sourceRoot, "source", "target", targetCtx, sourceCtx, fmt.Sprintf("backward%d", index), true)
	require.NoError(t, err)
	sourceElementRef, targetElementRef := source.ref, target.ref
	if sourceJSON && !expr.AllowsNull(source.attribute) {
		sourceElementRef = "loom.Optional[" + sourceElementRef + "]"
	}
	if targetJSON && !expr.AllowsNull(target.attribute) {
		targetElementRef = "loom.Optional[" + targetElementRef + "]"
	}
	prefix := "[]"
	if mapping {
		prefix = "map[string]"
	}
	sourceRef, targetRef := prefix+sourceElementRef, prefix+targetElementRef
	var generated strings.Builder
	generated.WriteString(source.declaration + target.declaration)
	fmt.Fprintf(&generated, "func forward%d(source %s) %s {\n%s\nreturn target\n}\n", index, sourceRef, targetRef, forward.Code)
	fmt.Fprintf(&generated, "func backward%d(source %s) %s {\n%s\nreturn target\n}\n", index, targetRef, sourceRef, backward.Code)
	for _, plan := range []TransformPlan{forward, backward} {
		for _, helper := range plan.Helpers {
			require.False(t, helper.ErrorAware)
			fmt.Fprintf(&generated, "func %s(v %s) %s {\n%s\nreturn res\n}\n", helper.Name, helper.ParamTypeRef, helper.ResultTypeRef, helper.Code)
		}
	}
	values := make([]string, len(source.values))
	for i, value := range source.values {
		if sourceJSON && !expr.AllowsNull(source.attribute) {
			value = "loom.OptionalValue[" + source.ref + "](" + value + ")"
		}
		if mapping {
			value = fmt.Sprintf("%q:", fmt.Sprint(i)) + value
		}
		values[i] = value
	}
	fmt.Fprintf(&generated, `func TestCase%d(t *testing.T) {
for _,source:=range []%s{nil,{}, {%s}} {
 target:=forward%d(source)
 if (source==nil)!=(target==nil) {t.Fatal("nil presence changed")}
 if len(source)!=len(target) {t.Fatal("member count changed")}
 roundtrip:=backward%d(target)
 if !reflect.DeepEqual(source,roundtrip) {t.Errorf("roundtrip changed: %%#v => %%#v => %%#v",source,target,roundtrip)}
}
}
`, index, sourceRef, strings.Join(values, ","), index, index)
	if sourceJSON && !expr.AllowsNull(source.attribute) {
		member, indexExpression := "{}", "0"
		if mapping {
			member, indexExpression = "\"missing\":{}", "\"missing\""
		}
		fmt.Fprintf(&generated, "func TestAbsentSlot%d(t *testing.T) {source:=%s{%s};target:=forward%d(source);if len(target)!=1 {t.Fatal(\"absent slot was dropped\")}\n", index, sourceRef, member, index)
		if targetJSON {
			fmt.Fprintf(&generated, "if target[%s].Present() {t.Fatal(\"absent slot became supplied\")}\n", indexExpression)
		}
		generated.WriteString("}\n")
	}
	return generated.String()
}

func collectionTargetElement(shape, prefix string, jsonPresence bool) collectionTargetSpec {
	switch shape {
	case "scalar":
		return collectionTargetSpec{attribute: &expr.AttributeExpr{Type: expr.Int}, ref: "int", values: []string{"0", "7"}}
	case "object":
		object := &expr.AttributeExpr{Type: &expr.Object{
			{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			{Name: "optional", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}, Validation: &expr.ValidationExpr{Required: []string{"value"}}}
		typ := &expr.UserTypeExpr{TypeName: prefix + "Object", AttributeExpr: object}
		fieldType, optionalType, zero, seven, optional := "int", "*string", "0", "7", "ptr(\"hello\")"
		if jsonPresence {
			fieldType, optionalType, zero, seven, optional = "*int", "loom.Optional[string]", "ptr(0)", "ptr(7)", "loom.OptionalValue(\"hello\")"
		}
		return collectionTargetSpec{attribute: &expr.AttributeExpr{Type: typ}, ref: "*" + typ.Name(), declaration: fmt.Sprintf("type %s struct {Value %s;Optional %s}\n", typ.Name(), fieldType, optionalType), values: []string{"nil", "&" + typ.Name() + "{Value:" + zero + "}", "&" + typ.Name() + "{Value:" + seven + ",Optional:" + optional + "}"}}
	case "union":
		typ := &expr.UserTypeExpr{TypeName: prefix + "Choice", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{TypeName: prefix + "Choice", Values: []*expr.NamedAttributeExpr{{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}}, {Name: "Number", Attribute: &expr.AttributeExpr{Type: expr.Int}}}}}}
		declaration := fmt.Sprintf(`type %[1]s struct {kind string;text string;number int}
func (u %[1]s) Kind() string {return u.kind}
func (u %[1]s) AsText() (string,bool) {return u.text,u.kind=="Text"}
func (u %[1]s) AsNumber() (int,bool) {return u.number,u.kind=="Number"}
func (u *%[1]s) SetText(v string) {u.kind="Text";u.text=v}
func (u *%[1]s) SetNumber(v int) {u.kind="Number";u.number=v}
`, typ.Name())
		return collectionTargetSpec{attribute: &expr.AttributeExpr{Type: typ}, ref: typ.Name(), declaration: declaration, values: []string{typ.Name() + `{kind:"Text",text:""}`, typ.Name() + `{kind:"Number",number:7}`}}
	default:
		attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}
		ref, values := "[]int", []string{"nil", "[]int{}", "[]int{0,7}"}
		if jsonPresence {
			ref = "[]loom.Optional[int]"
			values = []string{"nil", ref + "{}", ref + "{loom.OptionalValue(0),loom.OptionalValue(7)}"}
		}
		declaration := ""
		if shape == "alias" {
			typ := &expr.UserTypeExpr{TypeName: prefix + "List", AttributeExpr: attribute}
			attribute = &expr.AttributeExpr{Type: typ}
			declaration = "type " + typ.Name() + " " + ref + "\n"
			ref = typ.Name()
			values[1] = ref + "{}"
			values[2] = ref + strings.TrimPrefix(values[2], strings.Split(values[2], "{")[0])
		}
		if shape == "nullable" {
			attribute.Nullable = true
			values = []string{"loom.Nullable[" + ref + "]{}", "loom.NullValue[" + ref + "]()", "loom.NullableValue[" + ref + "](" + values[1] + ")", "loom.NullableValue[" + ref + "](" + values[2] + ")"}
			ref = "loom.Nullable[" + ref + "]"
		}
		return collectionTargetSpec{attribute: attribute, ref: ref, declaration: declaration, values: values}
	}
}

func TestRequiredCollectionTargetLayout(t *testing.T) {
	array := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}
	sourceList := &expr.UserTypeExpr{TypeName: "SourceList", AttributeExpr: expr.DupAtt(array)}
	targetList := &expr.UserTypeExpr{TypeName: "TargetList", AttributeExpr: expr.DupAtt(array)}
	object := func(list expr.DataType) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: &expr.AttributeExpr{Type: list}}}, Validation: &expr.ValidationExpr{Required: []string{"values"}}}
	}
	sourceType := &expr.UserTypeExpr{TypeName: "Source", AttributeExpr: object(sourceList)}
	targetType := &expr.UserTypeExpr{TypeName: "Target", AttributeExpr: object(targetList)}
	sourceContext := NewAttributeContext(false, false, true, "", NewNameScope())
	targetContext := NewAttributeContext(false, false, true, "", NewNameScope())
	targetContext.JSONPresenceTypes = map[string]bool{targetList.Hash(): true}
	plan, err := GoTransform(&expr.AttributeExpr{Type: sourceType}, &expr.AttributeExpr{Type: targetType}, "source", "target", sourceContext, targetContext, "", true)
	require.NoError(t, err)
	require.Empty(t, plan.Helpers)
	source := fmt.Sprintf(`package bytesdefaults
import ("testing";loom "github.com/CaliLuke/loom/pkg")
type SourceList []int
type TargetList []loom.Optional[int]
type Source struct { Values SourceList }
type Target struct { Values TargetList }
func convert(source *Source) *Target {
%s
return target
}
func TestRequiredArray(t *testing.T) {
 for _,input:=range []SourceList{nil,{}, {0,7}} {
  got:=convert(&Source{Values:input})
  if got.Values==nil || len(got.Values)!=len(input) {t.Fatal("required collection materialization changed")}
  for i,slot:=range got.Values {value,ok:=slot.Value();if !ok || value!=input[i] {t.Fatal("target element carrier changed")}}
 }
}
`, plan.Code)
	runBytesDefaultModule(t, source)
}
