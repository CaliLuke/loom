package transportir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNormalizationUsesSourceIdentity(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			label := &expr.UserTypeExpr{TypeName: "Alias", UID: "alias", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
			count := &expr.UserTypeExpr{TypeName: "Alias", UID: "alias", AttributeExpr: &expr.AttributeExpr{Type: expr.Int}}
			left := &expr.UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: label}}}}}
			right := &expr.UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: count}}}}}
			expr.AsObject(left).Set("next", &expr.AttributeExpr{Type: left})
			expr.AsObject(right).Set("next", &expr.AttributeExpr{Type: right})
			fields := &expr.Object{
				{Name: "one", Attribute: &expr.AttributeExpr{Type: left}},
				{Name: "two", Attribute: &expr.AttributeExpr{Type: right}},
				{Name: "again", Attribute: &expr.AttributeExpr{Type: left}},
			}
			if reverse {
				(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
			}
			got := normalizeHTTPAttribute(&expr.AttributeExpr{Type: fields})
			require.Equal(t, expr.String, got.Find("one").Find("value").Type)
			require.Equal(t, expr.Int, got.Find("two").Find("value").Type)
			require.Same(t, got.Find("one").Type, got.Find("again").Type)
			for _, member := range []string{"one", "two"} {
				require.Same(t, got.Find(member).Type, got.Find(member).Find("next").Type)
			}
			require.Same(t, label, left.Attribute().Find("value").Type)
			require.Same(t, count, right.Attribute().Find("value").Type)
		})
	}
}
