package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckedDefaultOwner(t *testing.T) {
	capture := func(attribute *AttributeExpr) DefaultValueOwner {
		return checkedDefaultOwner(t, attribute)
	}
	for _, authored := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct finalized", true: "Default assignment"}[authored], func(t *testing.T) {
			original := &AttributeExpr{Type: String, DefaultValue: "same"}
			if authored {
				original.SetDefault("same")
			}
			first := capture(original)
			require.True(t, first == capture(original), "independent capture keeps authored owner")
			copied := DupAtt(original)
			require.True(t, first == capture(copied), "controlled copy keeps authored owner")
			separate := &AttributeExpr{Type: String, DefaultValue: "same"}
			require.True(t, first != capture(separate), "equal raw value does not equate declarations")
			copied.SetDefault("same")
			override := capture(copied)
			require.True(t, first != override, "an explicit local assignment is a distinct owner even with equal value")
			require.True(t, override == capture(DupAtt(copied)))
			copied.SetDefault("next")
			require.True(t, override != capture(copied), "new assignment cannot rewrite a captured owner")
			require.True(t, first == capture(original), "copy override never changes original")
		})
	}
}

func TestTemplatePublicationDefaultOwner(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, override := range []string{"", "original", "override"} {
			name := map[bool]string{false: "direct", true: "named"}[named] + "/" + override
			t.Run(name, func(t *testing.T) {
				declaration := &AttributeExpr{Type: String}
				declaration.SetDefault("original")
				original := declaration
				if named {
					original = &AttributeExpr{Type: &UserTypeExpr{TypeName: "DefaultString", UID: "default-string", AttributeExpr: declaration}}
				}
				owner := checkedDefaultOwner(t, original)
				source := &AttributeExpr{Type: &Object{{Name: "value", Attribute: original}}}
				body := NewBodyTemplate(source)
				selected, ok := body.TemplateAttribute("value")
				require.True(t, ok)
				want := owner
				if override != "" {
					selected.SetDefault(override)
					want = checkedDefaultOwner(t, selected)
					require.True(t, owner != want)
				}
				body.Type = &Object{{Name: "value", Attribute: selected}}
				body.EndTemplateCapture()
				composer := &designComposer{seen: make(map[*AttributeExpr]bool), active: make(map[*AttributeExpr]bool)}
				composer.walk(body, "body")
				value := "original"
				if override != "" {
					value = override
				}
				require.Equal(t, value, selected.DefaultValue)
				require.True(t, want == checkedDefaultOwner(t, selected), "publication must transfer the selected default owner with its value")
				require.True(t, want == checkedDefaultOwner(t, DupAtt(selected)), "published copies keep the selected owner")
				require.True(t, owner == checkedDefaultOwner(t, original), "publication leaves the source owner intact")
			})
		}
	}
}

func checkedDefaultOwner(t *testing.T, attribute *AttributeExpr) DefaultValueOwner {
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{TargetOccurrence: occurrence, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
	require.NoError(t, err)
	if !annotations.HasDefault {
		named, ok := attribute.Type.(UserType)
		require.True(t, ok, "default must belong to the captured node or its named declaration")
		return checkedDefaultOwner(t, named.Attribute())
	}
	owner := annotations.Default.DefaultOwner()
	require.True(t, owner != (DefaultValueOwner{}))
	return owner
}
