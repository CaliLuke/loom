package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestObjectValidationNamesSuffixedKeysByAttribute checks that the object
// validation checks the JSON name of the service field of a key declared with
// an element name suffix, such as "a:x", which is the attribute name "a": the
// HTTP endpoint validation checks the element name, which only HTTP and
// JSON-RPC bodies use.
func TestObjectValidationNamesSuffixedKeysByAttribute(t *testing.T) {
	cases := []struct {
		name   string
		fields []string
		want   string
	}{
		{name: "distinct attributes", fields: []string{"a:x", "b:x"}},
		{name: "invalid element name", fields: []string{"a:x,y"}},
		{name: "same attribute", fields: []string{"a:x", "a:y"}, want: `field a:y duplicates JSON field name "a" from field a:x`},
		{name: "attribute of another key", fields: []string{"a:x", "a"}, want: `field a duplicates JSON field name "a" from field a:x`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			object := make(Object, 0, len(c.fields))
			for _, name := range c.fields {
				object = append(object, &NamedAttributeExpr{Name: name, Attribute: &AttributeExpr{Type: String}})
			}
			attribute := &AttributeExpr{Type: &object}
			verr := attribute.validateObjectChildren("", attribute, &object)
			if c.want == "" {
				assert.Empty(t, verr.Errors)
				return
			}
			assert.ErrorContains(t, verr, c.want)
		})
	}
}

// TestUntaggedBranchValidationNamesSuffixedKeysByAttribute checks that the
// validation of the fields of an untagged union branch checks the JSON names
// of the service fields, which are the attribute names of the keys declared
// with an element name suffix.
func TestUntaggedBranchValidationNamesSuffixedKeysByAttribute(t *testing.T) {
	cases := []struct {
		name   string
		fields []string
		want   string
	}{
		{name: "distinct attributes", fields: []string{"a:x", "b:x"}},
		{name: "empty element name", fields: []string{"a:"}},
		{name: "same attribute", fields: []string{"a:x", "a:y"}, want: `untagged OneOf branch "Branch" has duplicate JSON field name "a"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			object := make(Object, 0, len(c.fields))
			for _, name := range c.fields {
				object = append(object, &NamedAttributeExpr{Name: name, Attribute: &AttributeExpr{Type: String}})
			}
			branch := &UserTypeExpr{TypeName: "Branch", AttributeExpr: &AttributeExpr{Type: &object}}
			union := &AttributeExpr{Type: &Union{
				TypeName: "U",
				Untagged: true,
				Values:   []*NamedAttributeExpr{{Name: "Branch", Attribute: &AttributeExpr{Type: branch}}},
			}}
			verr := union.validateChildTypes("", union)
			if c.want == "" {
				assert.Empty(t, verr.Errors)
				return
			}
			assert.ErrorContains(t, verr, c.want)
		})
	}
}
