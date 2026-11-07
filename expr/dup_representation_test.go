package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDupPreservesDistinctRepresentations(t *testing.T) {
	for _, jsonFirst := range []bool{false, true} {
		name := "text first"
		if jsonFirst {
			name = "JSON first"
		}
		t.Run(name, func(t *testing.T) {
			text := &UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &AttributeExpr{
				Type: &Object{{Name: "x", Attribute: &AttributeExpr{Type: String}}},
			}}
			json := &UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &AttributeExpr{
				Type: &Object{{Name: "jsonX", Attribute: &AttributeExpr{Type: String}}},
			}}
			AsObject(text).Set("next", &AttributeExpr{Type: text})
			AsObject(json).Set("next", &AttributeExpr{Type: json})
			fields := &Object{
				{Name: "text", Attribute: &AttributeExpr{Type: text}},
				{Name: "json", Attribute: &AttributeExpr{Type: json}},
				{Name: "textAgain", Attribute: &AttributeExpr{Type: text}},
			}
			if jsonFirst {
				(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
			}
			copied := AsObject(DupAtt(&AttributeExpr{Type: fields}).Type)
			textCopy := copied.Attribute("text").Type.(*UserTypeExpr)
			jsonCopy := copied.Attribute("json").Type.(*UserTypeExpr)
			require.NotSame(t, textCopy, jsonCopy, "distinct source nodes must not merge by nominal ID")
			require.NotSame(t, text, textCopy)
			require.NotSame(t, json, jsonCopy)
			require.NotNil(t, AsObject(textCopy).Attribute("x"))
			require.Nil(t, AsObject(textCopy).Attribute("jsonX"))
			require.NotNil(t, AsObject(jsonCopy).Attribute("jsonX"))
			require.Nil(t, AsObject(jsonCopy).Attribute("x"))
			require.Same(t, textCopy, copied.Attribute("textAgain").Type)
			require.Same(t, textCopy, AsObject(textCopy).Attribute("next").Type)
			require.Same(t, jsonCopy, AsObject(jsonCopy).Attribute("next").Type)
			AsObject(textCopy).Attribute("x").Description = "copied"
			require.Empty(t, AsObject(text).Attribute("x").Description)
		})
	}
}
