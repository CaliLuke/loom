package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueOccurrenceQueriesPreserveOpaqueIdentity(t *testing.T) {
	list := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{}}
	list.Type = &Object{
		{Name: "items:wire", Attribute: &AttributeExpr{Type: &Map{
			KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: list}}},
		}}},
	}
	context := NewValueContext()
	root, err := context.NewOccurrence(&AttributeExpr{Type: list})
	require.NoError(t, err)
	members := root.Members()
	require.Len(t, members, 1)
	require.Equal(t, "items:wire", members[0].Name)
	require.Equal(t, "wire", members[0].WireName)
	require.NotEqual(t, members[0].ID, members[0].Occurrence.ID(), "member edges and occurrences are distinct")
	require.Equal(t, root.Underlying().ID(), members[0].Occurrence.Element().Element().Underlying().ID(), "recursive references reuse the declaration graph")
	require.NotEqual(t, ValueIdentity{}, members[0].Occurrence.Key().ID())
	members[0].Name = "changed"
	require.Equal(t, "items:wire", root.Members()[0].Name)

	union := &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
		{Name: "first", Attribute: &AttributeExpr{Type: String}},
		{Name: "second", Attribute: &AttributeExpr{Type: String}},
	}}}
	choice, err := context.NewOccurrence(union)
	require.NoError(t, err)
	branches := choice.Branches()
	require.Len(t, branches, 2)
	require.NotEqual(t, branches[0].ID, branches[1].ID, "equal payloads retain distinct branch edges")
	branches[0].Tag = "changed"
	require.Equal(t, "first", choice.Branches()[0].Tag)
	var zero ValueOccurrence
	require.Empty(t, zero.Members())
	require.Empty(t, zero.Branches())
	require.Equal(t, ValueIdentity{}, zero.Underlying().ID())
	require.Equal(t, ValueIdentity{}, zero.Element().ID())
	require.Equal(t, ValueIdentity{}, zero.Key().ID())
}
