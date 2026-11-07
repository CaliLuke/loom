package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateCopyMetadataRemovalIsolation(t *testing.T) {
	source := &AttributeExpr{Type: String, Meta: MetaExpr{"keep": {"source"}}}
	source.DeleteMeta("old")
	copied := DupAtt(source)
	copied.DeleteMeta("keep")
	assert.NotContains(t, source.removedMeta, "keep")
	assert.Equal(t, []string{"source"}, source.Meta["keep"])
	assert.True(t, copied.removedMeta["old"])
	assert.True(t, copied.removedMeta["keep"])

	base := &AttributeExpr{Type: String, Meta: MetaExpr{"keep": {"inherited"}}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		want      []string
	}{
		{name: "source still inherits", attribute: source, want: []string{"inherited"}},
		{name: "copy retains removal", attribute: copied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inherited := make(MetaExpr)
			owners := make(map[string]*AttributeExpr)
			assert.True(t, composeInheritedMetadata(tc.attribute, base, inherited, owners, "keep", "value"))
			assert.Equal(t, tc.want, inherited["keep"])
		})
	}
}

func TestTemplatePublicationKeepsAuthorshipAndEndsCapture(t *testing.T) {
	minimum, err := NewNumericBound(0)
	require.NoError(t, err)
	maximum, err := NewNumericBound(10)
	require.NoError(t, err)
	original := &AttributeExpr{Type: Int, DefaultValue: 7, Validation: &ValidationExpr{Minimum: &minimum, Maximum: &maximum}}
	source := &AttributeExpr{Type: &Object{{Name: "value", Attribute: original}}}
	body := NewBodyTemplate(source)
	selected, ok := body.TemplateAttribute("value")
	require.True(t, ok)
	selected.SetDefault(7)
	localMinimum, err := NewNumericBound(1)
	require.NoError(t, err)
	selected.SetMinimum(localMinimum)
	AsObject(body.Type).Set("value", selected)
	copied := DupAtt(body)
	copiedValue := AsObject(copied.Type).Attribute("value")
	copyMinimum, err := NewNumericBound(2)
	require.NoError(t, err)
	copiedValue.SetMinimum(copyMinimum)
	body.EndTemplateCapture()
	body.EndTemplateCapture()
	original.SetDefault(9)
	frameworkMinimum, err := NewNumericBound(8)
	require.NoError(t, err)
	selected.SetMinimum(frameworkMinimum)
	composer := &designComposer{seen: make(map[*AttributeExpr]bool), active: make(map[*AttributeExpr]bool)}
	composer.walk(body, "body")
	composer.walk(copied, "copied")
	for _, tc := range []struct {
		attribute *AttributeExpr
		minimum   NumericBound
	}{
		{selected, localMinimum}, {copiedValue, copyMinimum},
	} {
		assert.Equal(t, 7, tc.attribute.DefaultValue, "same-valued Default was still an authored choice")
		assert.Equal(t, tc.minimum, *tc.attribute.Validation.Minimum)
		assert.Equal(t, maximum, *tc.attribute.Validation.Maximum)
		assert.Nil(t, tc.attribute.template)
	}
	assert.Equal(t, 9, original.DefaultValue)
	assert.Equal(t, minimum, *original.Validation.Minimum)
}
