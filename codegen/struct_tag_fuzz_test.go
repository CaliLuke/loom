package codegen

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/expr"
)

// FuzzAttributeTagsWithName checks that struct tags rendered from design
// metadata and field names always compile and, for tag keys that design
// validation accepts, round-trip through reflect.StructTag and encoding/json/v2.
func FuzzAttributeTagsWithName(f *testing.F) {
	seeds := []struct {
		key, value, field string
	}{
		{"xml", "name,attr", "name"},
		{"json", "SSN,omitempty", "ssn"},
		{"json:name", "custom_name", "field"},
		{"form", "user_id", "user_id"},
		{"validate", "required,min=1", "age"},
		{"db", `quoted "value"`, "q"},
		{"db", "back`tick", "b"},
		{"db", `back\slash`, "s"},
		{"db", "new\nline", "n"},
		{"db", "日本語", "ü"},
		{"db", "invalid \xff utf8", "x"},
		{"db", "value", `field"with"quotes`},
		{"db", "value", "field`with`ticks"},
		{"bad key", "v", "f"},
		{"", "v", "f"},
		{"db", "v", "-"},
	}
	for _, s := range seeds {
		f.Add(s.key, s.value, s.field)
	}
	f.Fuzz(func(t *testing.T, key, value, field string) {
		att := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:" + key: {value}}}
		parent := &expr.AttributeExpr{Type: &expr.Object{{Name: field, Attribute: att}}}
		rendered := AttributeTagsWithName(parent, field, att)
		tag, ok := parseRenderedStructTag(t, rendered)
		if !ok {
			return
		}
		want := field
		if key == "json:name" && value != "" {
			want = value
		}
		gotJSON, foundJSON := tag.Lookup("json")
		if key != "json" && want == "" {
			// An empty field name and no json:name leave nothing to tag.
			if foundJSON {
				t.Errorf("tag %s: unexpected json tag %q for an empty field name", rendered, gotJSON)
			}
			return
		}
		// The remaining properties only hold for designs that pass validation.
		if verr := parent.Validate("", nil); len(verr.Errors) > 0 {
			return
		}
		if key != "json" && key != "json:name" {
			if got, found := tag.Lookup(key); !found || got != value {
				t.Errorf("tag %s: Lookup(%q) = %q, %v; want %q", rendered, key, got, found, value)
			}
		}
		if key == "json" {
			return
		}
		if !foundJSON {
			t.Errorf("tag %s: missing json tag, want name %q", rendered, want)
			return
		}
		if name := strings.Split(gotJSON, ",")[0]; name != strings.Split(want, ",")[0] {
			t.Errorf("tag %s: json name = %q, want %q", rendered, name, want)
		}
		// "-" omits the field from JSON, so there is no member to round-trip.
		if want != "-" {
			requireJSONFieldRoundTrip(t, tag, strings.Split(want, ",")[0])
		}
	})
}
