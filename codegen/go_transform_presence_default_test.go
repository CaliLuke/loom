package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestPresenceDefaultMaterialization(t *testing.T) {
	type specimen struct {
		name, ref, zero, fallback string
		datatype                  expr.DataType
		value                     any
	}
	named := func(name string, datatype expr.DataType) expr.DataType {
		return &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: datatype}}
	}
	specimens := []specimen{
		{"int", "int", "0", "7", expr.Int, 7},
		{"named", "Count", "Count(0)", "Count(7)", named("Count", expr.Int), 7},
		{"bytes", "Data", "Data{}", "Data{7}", named("Data", expr.Bytes), []byte{7}},
		{"array", "Counts", "Counts{}", "Counts{7}", named("Counts", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}), []int{7}},
		{"map", "Lookup", "Lookup{}", "Lookup{\"k\":7}", named("Lookup", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}), map[string]int{"k": 7}},
		{"object", "Record", "Record{}", "Record{}", named("Record", &expr.Object{}), map[string]any{}},
	}
	var code strings.Builder
	code.WriteString("package bytesdefaults\nimport(\"testing\";\"reflect\";loom \"github.com/CaliLuke/loom/pkg\")\ntype Count int\ntype Data []byte\ntype Counts []int\ntype Lookup map[string]int\ntype Record struct{}\n")
	index := 0
	for _, spec := range specimens {
		for _, pair := range [][2]string{{"native", "optional"}, {"optional", "optional"}, {"optional", "native"}, {"nullable", "nullable"}} {
			for _, policy := range []string{"apply", "disabled", "pointer"} {
				sourceField := &expr.AttributeExpr{Type: spec.datatype, Nullable: pair[0] == "nullable"}
				targetField := expr.DupAtt(sourceField)
				targetField.DefaultValue = spec.value
				sourceParent := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: sourceField}}})
				targetParent := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: targetField}}})
				sc := NewAttributeContext(true, false, false, "", NewNameScope())
				sc.JSONPresence = pair[0] == "optional"
				tc := NewAttributeContext(policy == "pointer", false, policy != "disabled", "", NewNameScope())
				tc.JSONPresence = pair[1] == "optional"
				fieldRef := func(parent *expr.MappedAttributeExpr, field *expr.AttributeExpr, context *AttributeContext) string {
					switch context.FieldPresence(parent, "value", field) {
					case OptionalPresence:
						return "loom.Optional[" + spec.ref + "]"
					case NullablePresence:
						return "loom.Nullable[" + spec.ref + "]"
					}
					if nativeObjectFieldPointer(parent, "value", field, context) {
						return "*" + spec.ref
					}
					return spec.ref
				}
				sr, tr := fieldRef(sourceParent, sourceField, sc), fieldRef(targetParent, targetField, tc)
				source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: fmt.Sprintf("Source%d", index), AttributeExpr: sourceParent.AttributeExpr}}
				target := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: fmt.Sprintf("Target%d", index), AttributeExpr: targetParent.AttributeExpr}}
				plan, err := GoTransform(source, target, "source", "target", sc, tc, fmt.Sprintf("case%d", index), true)
				require.NoError(t, err)
				fmt.Fprintf(&code, "type Source%d struct{Value %s}\ntype Target%d struct{Value %s}\nfunc convert%d(source *Source%d)*Target%d{%s;return target}\n", index, sr, index, tr, index, index, index, plan.Code)
				for _, helper := range plan.Helpers {
					require.False(t, helper.ErrorAware)
					fmt.Fprintf(&code, "func %s(v %s)%s{%s;return res}\n", helper.Name, helper.ParamTypeRef, helper.ResultTypeRef, helper.Code)
				}
				supplied := spec.zero
				switch pair[0] {
				case "optional":
					supplied = "loom.OptionalValue(" + supplied + ")"
				case "nullable":
					supplied = "loom.NullableValue(" + supplied + ")"
				default:
					if strings.HasPrefix(sr, "*") {
						supplied = "&zero"
					}
				}
				extract := "value:=result.Value;present:=true"
				switch {
				case pair[1] != "native":
					extract = "value,present:=result.Value.Value()"
				case strings.HasPrefix(tr, "*"):
					extract = "var value " + spec.ref + ";present:=result.Value!=nil;if present {value=*result.Value}"
				case spec.name == "bytes" || spec.name == "array" || spec.name == "map":
					extract = "value:=result.Value;present:=value!=nil"
				}
				absentExpected := "var expected " + spec.ref
				if policy == "apply" {
					absentExpected = "expected:=" + spec.fallback
				}
				absentPresence := "false"
				if policy == "apply" || pair[1] == "native" && !strings.HasPrefix(tr, "*") && (spec.name == "int" || spec.name == "named") {
					absentPresence = "true"
				}
				fmt.Fprintf(&code, `func TestCase%d(t *testing.T){
 t.Run(%q,func(t *testing.T){result:=convert%d(&Source%d{});%s;%s;if present!=%s||!reflect.DeepEqual(value,expected){t.Fatalf("absent: %%#v present=%%t",value,present)}})
 t.Run("supplied",func(t *testing.T){zero:=%s;result:=convert%d(&Source%d{Value:%s});%s;if !present||!reflect.DeepEqual(value,zero){t.Fatalf("supplied: %%#v present=%%t",value,present)}})
`, index, spec.name+"/"+pair[0]+"/"+pair[1]+"/"+policy, index, index, extract, absentExpected, absentPresence, spec.zero, index, index, supplied, extract)
				if pair[0] == "nullable" {
					fmt.Fprintf(&code, "result:=convert%d(&Source%d{Value:loom.NullValue[%s]()});if !result.Value.IsNull(){t.Fatal(\"explicit null changed\")}\n", index, index, spec.ref)
				}
				code.WriteString("}\n")
				index++
			}
		}
	}
	runBytesDefaultModule(t, code.String())
}

func TestCollectionDefaultPresenceStorage(t *testing.T) {
	scalar := &expr.AttributeExpr{Type: expr.Int}
	array := func(element *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
	}
	record := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}}
	cases := []struct {
		name      string
		attribute *expr.AttributeExpr
		raw       any
		expected  string
	}{
		{"array", array(scalar), []int{7}, "[]loom.Optional[int]{loom.OptionalValue(7)}"},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: scalar}}, map[string]int{"k": 7}, "map[string]loom.Optional[int]{\"k\":loom.OptionalValue(7)}"},
		{"nested", array(array(scalar)), [][]int{{7}}, "[]loom.Optional[[]loom.Optional[int]]{loom.OptionalValue([]loom.Optional[int]{loom.OptionalValue(7)})}"},
		{"objects", array(record), []any{map[string]any{}}, "[]loom.Optional[*Record]{loom.OptionalValue(&Record{})}"},
		{"named", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "JSONCounts", AttributeExpr: array(scalar)}}, []int{7}, "JSONCounts{loom.OptionalValue(7)}"},
	}
	var code strings.Builder
	code.WriteString("package bytesdefaults\nimport(\"testing\";\"reflect\";loom \"github.com/CaliLuke/loom/pkg\")\ntype Record struct{}\ntype JSONCounts []loom.Optional[int]\n")
	for _, tc := range cases {
		context := NewAttributeContext(false, false, true, "", NewNameScope())
		context.JSONPresence = true
		context.CollectionElementPresence = true
		if tc.name == "named" {
			context.JSONPresence = false
			context.CollectionElementPresence = false
			context.JSONPresenceTypes = map[string]bool{tc.attribute.Type.(expr.UserType).Hash(): true}
		}
		plan, err := context.DefaultValue(tc.attribute, tc.raw)
		require.NoError(t, err)
		require.False(t, plan.ErrorAware)
		fmt.Fprintf(&code, "func Test_%s(t *testing.T){got:=%s;expected:=%s;if !reflect.DeepEqual(got,expected){t.Fatalf(\"unexpected default: %%#v\",got)}}\n", tc.name, plan.Expression, tc.expected)
	}
	runBytesDefaultModule(t, code.String())
}
