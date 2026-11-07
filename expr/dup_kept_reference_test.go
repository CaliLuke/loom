package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDupKeepingReportsOccurrenceCorrespondence(t *testing.T) {
	target := &UserTypeExpr{TypeName: "Pending", AttributeExpr: &AttributeExpr{Type: &Object{}}}
	source := &AttributeExpr{Type: target, Description: "source"}
	object := &Object{{Name: "left", Attribute: source}, {Name: "right", Attribute: source}}
	copied, kept := DupKeeping(object, []UserType{target})
	require.Len(t, kept, 2)
	for index, name := range []string{"left", "right"} {
		require.Same(t, source, kept[index].Source)
		require.Same(t, AsObject(copied).Attribute(name), kept[index].Copy)
		require.NotSame(t, source, kept[index].Copy)
		require.Same(t, target, kept[index].Copy.Type)
	}
	require.NotSame(t, kept[0].Copy, kept[1].Copy)
	kept[0].Copy.Description = "local"
	require.Equal(t, "source", source.Description)
	require.Equal(t, "source", kept[1].Copy.Description)
}
