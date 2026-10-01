package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueSourcePrecedenceAndSuppression(t *testing.T) {
	local := &ExampleExpr{Value: "local"}
	reference := &UserTypeExpr{TypeName: "Reference", AttributeExpr: &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{Value: "reference"}}}}
	base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{Value: "base"}}}}
	named := &UserTypeExpr{TypeName: "Named", AttributeExpr: &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{Value: "type"}}}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		policy    ExamplePolicy
		state     ExampleState
		want      any
	}{
		{"local last", &AttributeExpr{Type: named, References: []DataType{reference}, Bases: []DataType{base}, UserExamples: []*ExampleExpr{{Value: "old"}, local}}, ExamplePolicy{Reachable: true}, ExampleSelected, "local"},
		{"reference", &AttributeExpr{Type: named, References: []DataType{reference}, Bases: []DataType{base}}, ExamplePolicy{Reachable: true}, ExampleSelected, "reference"},
		{"base", &AttributeExpr{Type: named, Bases: []DataType{base}}, ExamplePolicy{Reachable: true}, ExampleSelected, "base"},
		{"type", &AttributeExpr{Type: named}, ExamplePolicy{Reachable: true}, ExampleSelected, "type"},
		{"authored suppression exception", &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{local}}, ExamplePolicy{Reachable: true, SuppressGenerated: true}, ExampleSelected, "local"},
		{"explicit null", &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{ExplicitNull: true}}}, ExamplePolicy{Reachable: true}, ExampleSelected, nil},
		{"suppressed", &AttributeExpr{Type: String}, ExamplePolicy{Reachable: true, SuppressGenerated: true}, ExampleSuppressed, nil},
		{"excluded", &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{local}}, ExamplePolicy{}, ExampleExcluded, nil},
		{"absent", &AttributeExpr{Type: String}, ExamplePolicy{Reachable: true}, ExampleAbsent, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			selection := context.SelectExample(occurrence, tc.policy)
			require.Equal(t, tc.state, selection.State())
			source, present := selection.Source()
			require.Equal(t, tc.state == ExampleSelected, present)
			if present {
				require.Equal(t, tc.want, source.data.snapshot.raw)
			}
		})
	}
}

func TestValueSourceOwnershipAndDeferredFailure(t *testing.T) {
	context := NewValueContext()
	raw := map[string]any{"value": []byte("hi")}
	example := &ExampleExpr{Value: raw}
	attribute := &AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{example}}
	first, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	second, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	firstSource, present := context.SelectExample(first, ExamplePolicy{Reachable: true}).Source()
	require.True(t, present)
	secondSource, present := context.SelectExample(second, ExamplePolicy{Reachable: true}).Source()
	require.True(t, present)
	require.Equal(t, firstSource.ID(), secondSource.ID())
	require.False(t, first.ID() == second.ID())
	raw["value"].([]byte)[0] = 'x'
	require.Equal(t, []byte("hi"), firstSource.data.snapshot.raw.(map[string]any)["value"])
	cycle := map[string]any{}
	cycle["self"] = cycle
	cyclic, err := context.NewOccurrence(&AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{{Value: cycle}}})
	require.NoError(t, err)
	selection := context.SelectExample(cyclic, ExamplePolicy{Reachable: true})
	require.Equal(t, ExampleSelected, selection.State())
	source, present := selection.Source()
	require.True(t, present)
	require.ErrorContains(t, source.data.snapshot.err, "cyclic value")
}

func TestValueSourceContainsNestedAuthoredOrigin(t *testing.T) {
	child := &AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{Value: "child"}}}
	context := NewValueContext()
	service, err := context.NewOccurrence(&AttributeExpr{Type: &Object{{Name: "body", Attribute: child}}})
	require.NoError(t, err)
	body, err := context.NewOccurrence(child)
	require.NoError(t, err)
	source, present := context.SelectExample(body, ExamplePolicy{Reachable: true}).Source()
	require.True(t, present)
	require.True(t, context.ContainsExampleSource(service, source))
	require.False(t, context.ContainsExampleSource(service, context.SupplyValue(ValueInput{Raw: "child"})))
	require.False(t, NewValueContext().ContainsExampleSource(service, source))
}

func TestValueSourceEntriesPreserveSelectedGroup(t *testing.T) {
	reference := &UserTypeExpr{TypeName: "Reference", AttributeExpr: &AttributeExpr{Type: String,
		UserExamples: []*ExampleExpr{
			{Summary: "first", Description: "one", Meta: MetaExpr{"x": {"a"}}, Value: "a"},
			{Summary: "second", Description: "two", Meta: MetaExpr{"x": {"b"}}, Value: "b"},
		}}}
	attribute := &AttributeExpr{Type: String, References: []DataType{reference}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	entries := selection.Entries()
	require.Len(t, entries, 2)
	require.Equal(t, []string{"first", "second"}, []string{entries[0].Summary(), entries[1].Summary()})
	require.Equal(t, []string{"one", "two"}, []string{entries[0].Description(), entries[1].Description()})
	require.Equal(t, "a", entries[0].Source().data.snapshot.raw)
	require.Equal(t, "b", entries[1].Source().data.snapshot.raw)
	last, present := selection.Source()
	require.True(t, present)
	require.True(t, last.ID() == entries[1].Source().ID())

	entries[0] = ExampleEntry{}
	meta := entries[1].Meta()
	meta["x"][0] = "changed"
	again := selection.Entries()
	require.Equal(t, "first", again[0].Summary())
	require.Equal(t, MetaExpr{"x": {"b"}}, again[1].Meta())
	for _, policy := range []ExamplePolicy{{}, {Reachable: true, SuppressGenerated: true}} {
		empty, emptyErr := context.NewOccurrence(&AttributeExpr{Type: String})
		require.NoError(t, emptyErr)
		require.Empty(t, context.SelectExample(empty, policy).Entries())
	}
}
