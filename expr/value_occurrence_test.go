package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueOccurrenceSeparatesDeclarationAndOccurrence(t *testing.T) {
	minimum := 2
	blob := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}
	plain := &AttributeExpr{Type: blob}
	optional := &AttributeExpr{
		Type: blob, Nullable: true, DefaultValue: []byte("default"),
		Meta:         MetaExpr{"source": {"nullable"}},
		Validation:   &ValidationExpr{MinLength: &minimum},
		UserExamples: []*ExampleExpr{{Value: []byte("example")}},
	}
	root := &AttributeExpr{
		Type:       &Object{{Name: "plain", Attribute: plain}, {Name: "optional:wire", Attribute: optional}},
		Validation: &ValidationExpr{Required: []string{"plain"}},
	}
	context := NewValueContext()
	first, err := context.NewOccurrence(root)
	require.NoError(t, err)
	second, err := context.NewOccurrence(root)
	require.NoError(t, err)
	require.False(t, first.ID() == second.ID())
	members := first.node.declaration.members
	require.NotSame(t, members[0].node, members[1].node)
	require.Same(t, members[0].node.declaration, members[1].node.declaration)
	require.NotEqual(t, members[0].id, members[1].id)
	require.True(t, valueMemberRequired(first.node, members[0]))
	require.False(t, valueMemberRequired(first.node, members[1]))
	require.Equal(t, "wire", members[1].wire)
	require.True(t, members[1].node.attribute.Nullable)
	minimum = 99
	optional.Meta["source"][0] = "changed"
	optional.DefaultValue.([]byte)[0] = 'X'
	optional.UserExamples[0].Value.([]byte)[0] = 'X'
	require.Equal(t, 2, *members[1].node.attribute.Validation.MinLength)
	require.Equal(t, "nullable", members[1].node.attribute.Meta["source"][0])
	require.Equal(t, []byte("default"), members[1].node.defaultValue.raw)
	require.Equal(t, []byte("example"), members[1].node.examples[0].source.raw)
	require.Same(t, optional.UserExamples[0], members[1].node.examples[0].origin)
}

func TestValueOccurrenceRecursiveDeclaration(t *testing.T) {
	list := &UserTypeExpr{TypeName: "List", AttributeExpr: &AttributeExpr{}}
	list.Type = &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: String}},
		{Name: "next", Attribute: &AttributeExpr{Type: list, Nullable: true}},
	}
	root, err := NewValueContext().NewOccurrence(&AttributeExpr{Type: list})
	require.NoError(t, err)
	next := root.node.declaration.alias.declaration.members[1].node
	require.Same(t, root.node.declaration, next.declaration)
	require.True(t, next.attribute.Nullable)
	require.False(t, root.node.attribute.Nullable)
}

func TestValueOccurrenceMalformedGraphAndDeferredSource(t *testing.T) {
	context := NewValueContext()
	alias := &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{}}
	alias.Type = alias
	for _, attribute := range []*AttributeExpr{nil, {}, {Type: alias}, {Type: &Array{}}} {
		occurrence, err := context.NewOccurrence(attribute)
		require.Error(t, err)
		require.Equal(t, ValueIdentity{}, occurrence.ID())
	}
	cyclic := map[string]any{}
	cyclic["cycle"] = cyclic
	valid, err := context.NewOccurrence(&AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{{Value: cyclic}}})
	require.NoError(t, err)
	require.Error(t, valid.node.examples[0].source.err)
}

func TestValueOccurrenceMemberSpellings(t *testing.T) {
	root, err := NewValueContext().NewOccurrence(&AttributeExpr{Type: &Object{
		{Name: "authored:element", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"wire,omitempty"}}}},
		{Name: "first", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"-"}}}},
		{Name: "second", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"-"}}}},
	}})
	require.NoError(t, err)
	members := root.node.declaration.members
	require.Equal(t, "authored:element", members[0].name)
	require.Equal(t, "wire", members[0].wire)
	require.Equal(t, "-", members[1].wire)
	require.Equal(t, "-", members[2].wire)
}
