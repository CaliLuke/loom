package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestBytesDefaultPresenceTransform keeps absent, explicit null, supplied empty,
// and nonempty states separate when native or named bytes have defaults.
func TestBytesDefaultPresenceTransform(t *testing.T) {
	var source strings.Builder
	source.WriteString(bytesPresencePrelude)
	types := bytesDefaultTypes()[:3]
	label := &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	types = append(types, bytesDefaultType{name: "named_string", datatype: label, defaultValue: "hi"})
	for _, typ := range types {
		for _, nullable := range []bool{false, true} {
			for mask := range 8 {
				name := fmt.Sprintf("%s_%t_%d", typ.name, nullable, mask)
				targetDefaults, targetPointer, sourcePointer := mask&1 != 0, mask&2 != 0, mask&4 != 0
				field := &expr.AttributeExpr{Type: typ.datatype, DefaultValue: typ.defaultValue, Nullable: nullable}
				parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: field}}}
				scope := NewNameScope()
				src := NewAttributeContext(sourcePointer, false, false, "", scope)
				src.JSONPresence = !nullable
				dst := NewAttributeContext(targetPointer, false, targetDefaults, "", scope)
				code, helpers, err := GoTransform(parent, parent, "source", "target", src, dst, "", true)
				require.NoError(t, err, name)
				require.Empty(t, helpers, name)
				wrapper := "Optional"
				if nullable {
					wrapper = "Nullable"
				}
				valueRef := scope.GoValueTypeRef(field)
				fmt.Fprintf(&source, "type source_%s struct { Data loom.%s[%s] }\n", name, wrapper, valueRef)
				fmt.Fprintf(&source, "func convert_%s(source *source_%s) any {\n%s\nreturn target\n}\n", name, name, code)
				fmt.Fprintf(&source, "func Test_%s(t *testing.T) {checkPresence(t,new(source_%s),%t,%t,%t,func(v any) any {return convert_%s(v.(*source_%s))})}\n", name, name, nullable, typ.bytes, targetDefaults && !targetPointer, name, name)
			}
		}
	}
	runBytesDefaultModule(t, source.String())
}

const bytesPresencePrelude = `package bytesdefaults

import (
 "reflect"
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)

type Blob []byte
type BlobAlias Blob
type Label string

func checkPresence(t *testing.T, source any, nullable, bytes, applyDefault bool, convert func(any) any) {
 for _, state:=range []string{"absent", "null", "empty", "nonempty"} {
  if state=="null" && !nullable { continue }
  field:=reflect.ValueOf(source).Elem().FieldByName("Data")
  field.SetZero()
  if state=="null" {
   field.Addr().MethodByName("SetNull").Call(nil)
  } else if state!="absent" {
   text:=""
   if state=="nonempty" {text="ok"}
   value:=reflect.ValueOf(text)
   if bytes {value=reflect.ValueOf([]byte(text))}
   set:=field.Addr().MethodByName("SetValue")
   set.Call([]reflect.Value{value.Convert(set.Type().In(0))})
  }
  result:=reflect.ValueOf(convert(source)).Elem().FieldByName("Data")
  if nullable {
   isNull:=result.MethodByName("IsNull").Call(nil)[0].Bool()
   if isNull!=(state=="null") {t.Errorf("state=%s null=%t",state,isNull)}
   if state=="null" {continue}
   values:=result.MethodByName("Value").Call(nil)
   present:=values[1].Bool()
   if state=="absent" && !applyDefault {
    if present {t.Error("absent nullable became present without target defaults")}
    continue
   }
   if !present {t.Errorf("state=%s became absent",state);continue}
   result=values[0]
  }
  nilValue:=false
  if result.Kind()==reflect.Pointer {
   nilValue=result.IsNil()
   if !nilValue {result=result.Elem()}
  }
  got:=""
  if !nilValue {
   if bytes {nilValue=result.IsNil();got=string(result.Bytes())} else {got=result.String()}
  }
  if state=="absent" && !applyDefault {
   if !nilValue {t.Error("absent native field defaulted without target defaults")}
   continue
  }
  want:=""
  if state=="absent" {want="hi"}
  if state=="nonempty" {want="ok"}
  if nilValue || got!=want {t.Errorf("state=%s got=%q nil=%t want=%q",state,got,nilValue,want)}
 }
}
`
