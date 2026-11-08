package openapiimport

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// Weak ParentHint children exist only in the parent's normalized body. A parent
// first retained through a raw source reference must still reach them.
func TestDiscriminatorSourceRetainedWeakChild(t *testing.T) {
	requireSourceRetention(t, "weak child", "P", `    P: {type: object, properties: {kind: {type: string}}, required: [kind], discriminator: {propertyName: kind}}
    C: {allOf: [{$ref: '#/components/schemas/P'}, {type: object, properties: {name: {type: string}}}]}
`, []string{"A", "B", "C", "P", "U"})
}

// Synthetic seam, not public input: a hand-built raw reference retains
// ParentHint H, whose weak child K is reachable only through H's normalized body.
func TestComponentPrunerSourceRetainedWeakChild(t *testing.T) {
	hint := &Discriminator{Parent: true, validated: true}
	components := Components{Schemas: []NamedSchema{
		{Name: "H", Schema: &Schema{Type: "object", Discriminator: hint}},
		{Name: "K", Schema: &Schema{Type: "object"}},
		{Name: "Unused", Schema: &Schema{Type: "object"}},
	}}
	bundle := &bundledDescription{references: map[string]string{
		"#/paths/~1raw/get/responses/200/content/application~1json/schema/properties/h": "#/components/schemas/H",
	}}
	for _, normalizedAfter := range []bool{false, true} {
		pruner := newComponentPruner(components)
		pruner.weakChildren = map[*Discriminator][]string{hint: {"K"}}
		pruner.visitSourcePrefix(bundle, "#/paths/~1raw/get", make(map[string]bool))
		if normalizedAfter {
			pruner.visitSchema(&Schema{Ref: componentDeclaration("H")})
		}
		label := "normalizedAfter=" + strconv.FormatBool(normalizedAfter)
		require.Equal(t, map[string]struct{}{"H": {}, "K": {}}, pruner.closure.schemas, label)
		for _, named := range components.Schemas[:2] {
			require.Contains(t, pruner.schemaOccurrences, named.Schema, "%s body walked, %s", named.Name, label)
		}
	}
}
