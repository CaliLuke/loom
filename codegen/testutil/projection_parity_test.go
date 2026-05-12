package testutil_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectionParityDiffs(t *testing.T) {
	t.Run("exact parity", func(t *testing.T) {
		diffs := testutil.ProjectionParityDiffs(
			objectAttr(required("id"), field("id", expr.String), field("name", expr.String)),
			objectAttr(required("id"), field("id", expr.String), field("name", expr.String)),
		)
		require.Empty(t, diffs)
	})

	t.Run("missing field names path", func(t *testing.T) {
		diffs := testutil.ProjectionParityDiffs(
			objectAttr(nil, field("id", expr.String), field("profile", objectAttr(nil, field("name", expr.String)).Type)),
			objectAttr(nil, field("id", expr.String), field("profile", (&expr.Object{}))),
		)
		got := testutil.FormatProjectionParityDiffs(diffs)
		require.Contains(t, got, "profile.name")
		require.Contains(t, got, "missing field")
	})

	t.Run("extra field names path", func(t *testing.T) {
		diffs := testutil.ProjectionParityDiffs(
			objectAttr(nil, field("id", expr.String)),
			objectAttr(nil, field("id", expr.String), field("debug", expr.String)),
		)
		got := testutil.FormatProjectionParityDiffs(diffs)
		require.Contains(t, got, "debug")
		require.Contains(t, got, "field present")
	})

	t.Run("required presence mismatch names path", func(t *testing.T) {
		diffs := testutil.ProjectionParityDiffs(
			objectAttr(required("id"), field("id", expr.String)),
			objectAttr(nil, field("id", expr.String)),
		)
		got := testutil.FormatProjectionParityDiffs(diffs)
		require.Contains(t, got, "id")
		require.Contains(t, got, "required field")
		require.Contains(t, got, "optional field")
	})
}

func TestProjectionViewParityDiffs(t *testing.T) {
	nested := resultType("Nested", "application/vnd.nested",
		objectAttr(required("visible"), field("visible", expr.String), field("hidden", expr.String)),
		view("default", field("visible", expr.String), field("hidden", expr.String)),
		view("tiny", field("visible", expr.String)),
	)
	root := resultType("Root", "application/vnd.root",
		objectAttr(required("child"), field("child", nested)),
		view("default", field("child", nested, "tiny")),
	)

	actual := objectAttr(required("child"),
		field("child", resultType("NestedView", "application/vnd.nested",
			objectAttr(required("visible"), field("visible", expr.String)),
			view("default", field("visible", expr.String)),
		)),
	)

	diffs := testutil.ProjectionViewParityDiffs(root, "default", actual)
	require.Empty(t, diffs)

	missingNested := objectAttr(required("child"), field("child", objectAttr(nil).Type))
	got := testutil.FormatProjectionParityDiffs(testutil.ProjectionViewParityDiffs(root, "default", missingNested))
	require.True(t, strings.Contains(got, "child.visible"), got)
}

func objectAttr(validation *expr.ValidationExpr, attrs ...*expr.NamedAttributeExpr) *expr.AttributeExpr {
	obj := expr.Object(attrs)
	return &expr.AttributeExpr{Type: &obj, Validation: validation}
}

func field(name string, typ expr.DataType, viewName ...string) *expr.NamedAttributeExpr {
	attr := &expr.AttributeExpr{Type: typ}
	if len(viewName) > 0 {
		attr.Meta = expr.MetaExpr{expr.ViewMetaKey: []string{viewName[0]}}
	}
	return &expr.NamedAttributeExpr{Name: name, Attribute: attr}
}

func required(names ...string) *expr.ValidationExpr {
	return &expr.ValidationExpr{Required: names}
}

func view(name string, attrs ...*expr.NamedAttributeExpr) *expr.ViewExpr {
	obj := expr.Object(attrs)
	return &expr.ViewExpr{Name: name, AttributeExpr: &expr.AttributeExpr{Type: &obj}}
}

func resultType(name, id string, attr *expr.AttributeExpr, views ...*expr.ViewExpr) *expr.ResultTypeExpr {
	return &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{
			AttributeExpr: attr,
			TypeName:      name,
			UID:           id,
		},
		Identifier: id,
		Views:      views,
	}
}
