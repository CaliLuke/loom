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

type nestedUnionScope struct {
	Attributor
	valueBranch string
}

func TestNestedUnionTransformPreservesSelectedNil(t *testing.T) {
	var generated strings.Builder
	generated.WriteString("package nested\nimport \"testing\"\n")
	index := 0
	for _, sourcePointer := range []bool{false, true} {
		for _, targetPointer := range []bool{false, true} {
			for _, newVar := range []bool{false, true} {
				sourcePrefix, targetPrefix := fmt.Sprintf("Source%d", index), fmt.Sprintf("Target%d", index)
				source, target := nullableTransformUnion("union", sourcePrefix, "ordinary"), nullableTransformUnion("union", targetPrefix, "ordinary")
				sourceCtx := NewAttributeContext(false, false, true, "", NewNameScope())
				targetCtx := NewAttributeContext(false, false, true, "", NewNameScope())
				if !sourcePointer {
					sourceCtx.Scope = nestedUnionScope{Attributor: sourceCtx.Scope, valueBranch: sourcePrefix + "Branch"}
				}
				if !targetPointer {
					targetCtx.Scope = nestedUnionScope{Attributor: targetCtx.Scope, valueBranch: targetPrefix + "Branch"}
				}
				plan, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "convert", newVar)
				require.NoError(t, err)
				require.Empty(t, plan.Helpers)
				sourceRef, targetRef := sourcePrefix+"Branch", targetPrefix+"Branch"
				sourceValue := ""
				targetNilCheck := ""
				if sourcePointer {
					sourceRef = "*" + sourceRef
					sourceValue = "&"
				}
				if targetPointer {
					targetRef = "*" + targetRef
					targetNilCheck = "if actual == nil {t.Fatal(\"populated nested value was lost\")}"
				}
				for _, side := range []struct{ prefix, ref string }{{sourcePrefix, sourceRef}, {targetPrefix, targetRef}} {
					fmt.Fprintf(&generated, `type %[1]sBranch struct { kind string; text string }
func (u %[1]sBranch) Kind() string {return u.kind}
func (u %[1]sBranch) AsText() (string,bool) {return u.text,u.kind=="text"}
func (u *%[1]sBranch) SetText(value string) {u.kind="text";u.text=value}
type %[1]sUnion struct {kind string; branch %[2]s}
func (u %[1]sUnion) Kind() string {return u.kind}
func (u %[1]sUnion) AsBranch() (%[2]s,bool) {return u.branch,u.kind=="branch"}
func (u *%[1]sUnion) SetBranch(value %[2]s) {u.kind="branch";u.branch=value}
`, side.prefix, side.ref)
				}
				fmt.Fprintf(&generated, "func convert%d(source *%sUnion) *%sUnion {\n", index, sourcePrefix, targetPrefix)
				if !newVar {
					fmt.Fprintf(&generated, "var target *%sUnion\n", targetPrefix)
				}
				generated.WriteString(plan.Code + "\nreturn target\n}\n")
				fmt.Fprintf(&generated, `func TestCase%d(t *testing.T) {
for _,text:=range []string{"","hello"} {
 source:=&%sUnion{};source.SetBranch(%s%sBranch{kind:"text",text:text})
 target:=convert%d(source)
 if target==nil || target.Kind()!="branch" {t.Fatal("outer selection lost")}
 actual,ok:=target.AsBranch();if !ok {t.Fatal("selected branch unavailable")}
 %s
 value,ok:=actual.AsText();if !ok || value!=text {t.Fatal("nested value changed")}
}
`, index, sourcePrefix, sourceValue, sourcePrefix, index, targetNilCheck)
				if sourcePointer {
					fmt.Fprintf(&generated, "source:=&%sUnion{};source.SetBranch(nil);target:=convert%d(source)\nif target==nil || target.Kind()!=\"branch\" {t.Fatal(\"selected nil branch was dropped\")}\nactual,ok:=target.AsBranch();if !ok {t.Fatal(\"selected nil unavailable\")}\n", sourcePrefix, index)
					if targetPointer {
						generated.WriteString("if actual!=nil {t.Fatal(\"nil branch was fabricated\")}\n")
					} else {
						generated.WriteString("if actual.Kind()!=\"\" {t.Fatal(\"nil became a concrete nested selection\")}\n")
					}
				}
				fmt.Fprintf(&generated, "emptySource:=&%sUnion{};emptySource.SetBranch(%s%sBranch{});emptyTarget:=convert%d(emptySource)\nif emptyTarget==nil || emptyTarget.Kind()!=\"branch\" {t.Fatal(\"selected empty nested union was dropped\")}\n", sourcePrefix, sourceValue, sourcePrefix, index)
				fmt.Fprintf(&generated, "if convert%d(&%sUnion{}) != nil {t.Fatal(\"absent outer selection became present\")}\n", index, sourcePrefix)
				generated.WriteString("}\n")
				index++
			}
		}
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/nested\n\ngo 1.27.3\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested_test.go"), []byte(generated.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}

func (s nestedUnionScope) Ref(attribute *expr.AttributeExpr, pkg string) string {
	ref := s.Attributor.Ref(attribute, pkg)
	if named, ok := attribute.Type.(expr.UserType); ok && named.Name() == s.valueBranch {
		return strings.TrimPrefix(ref, "*")
	}
	return ref
}
