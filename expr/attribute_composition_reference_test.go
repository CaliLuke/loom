package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompositionBasesRetainDeclarationIdentity(t *testing.T) {
	base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: &Object{}}}
	attribute := &AttributeExpr{Type: &Object{}, compositionBases: []DataType{base}}
	copied := DupAtt(attribute)
	for _, occurrence := range []*AttributeExpr{attribute, copied} {
		bases := occurrence.CompositionBases()
		require.Equal(t, []DataType{base}, bases)
		require.Same(t, base, bases[0])
		bases[0] = String
		require.Equal(t, []DataType{base}, occurrence.CompositionBases())
	}
	require.Empty(t, (*AttributeExpr)(nil).CompositionBases())
}
