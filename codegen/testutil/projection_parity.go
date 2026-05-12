package testutil

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/expr"
)

// ProjectionParityDiff describes a projection drift found while comparing
// canonical and generated projection attributes.
type ProjectionParityDiff struct {
	Path string
	Want string
	Got  string
}

// ProjectionParityDiffs compares expected and actual projection shapes.
func ProjectionParityDiffs(expected, actual *expr.AttributeExpr) []ProjectionParityDiff {
	var diffs []ProjectionParityDiff
	compareProjectionField(&diffs, "", expected, actual)
	return diffs
}

// ProjectionViewParityDiffs compares a result type view with a generated
// projected attribute. Extra fields in actual are ignored because generated
// projected types intentionally carry the union of all view fields while
// view-specific constructors and validators select the active subset.
func ProjectionViewParityDiffs(result *expr.ResultTypeExpr, view string, actual *expr.AttributeExpr) []ProjectionParityDiff {
	projected, err := expr.Project(result, view)
	if err != nil {
		return []ProjectionParityDiff{{
			Path: result.TypeName,
			Want: fmt.Sprintf("projectable view %q", view),
			Got:  err.Error(),
		}}
	}
	var diffs []ProjectionParityDiff
	compareProjectionFieldAllowingExtras(&diffs, "", projected.Attribute(), actual)
	return diffs
}

// AssertProjectionParity fails the test if the projection shape drifts.
func AssertProjectionParity(t testing.TB, expected, actual *expr.AttributeExpr) {
	t.Helper()
	if diffs := ProjectionParityDiffs(expected, actual); len(diffs) > 0 {
		t.Fatalf("projection parity drift:\n%s", FormatProjectionParityDiffs(diffs))
	}
}

// AssertProjectionViewParity fails the test if actual does not contain every
// field required by the result type view.
func AssertProjectionViewParity(t testing.TB, result *expr.ResultTypeExpr, view string, actual *expr.AttributeExpr) {
	t.Helper()
	if diffs := ProjectionViewParityDiffs(result, view, actual); len(diffs) > 0 {
		t.Fatalf("projection view parity drift:\n%s", FormatProjectionParityDiffs(diffs))
	}
}

// FormatProjectionParityDiffs formats parity diffs with stable ordering.
func FormatProjectionParityDiffs(diffs []ProjectionParityDiff) string {
	lines := make([]string, len(diffs))
	for i, diff := range diffs {
		path := diff.Path
		if path == "" {
			path = "<root>"
		}
		lines[i] = fmt.Sprintf("- %s: want %s, got %s", path, diff.Want, diff.Got)
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

func compareProjectionField(diffs *[]ProjectionParityDiff, path string, expected, actual *expr.AttributeExpr) {
	compareProjectionFieldWithMode(diffs, path, expected, actual, false)
}

func compareProjectionFieldAllowingExtras(diffs *[]ProjectionParityDiff, path string, expected, actual *expr.AttributeExpr) {
	compareProjectionFieldWithMode(diffs, path, expected, actual, true)
}

func compareProjectionFieldWithMode(diffs *[]ProjectionParityDiff, path string, expected, actual *expr.AttributeExpr, allowExtras bool) {
	if expected == nil {
		if actual != nil && !allowExtras {
			*diffs = append(*diffs, ProjectionParityDiff{Path: path, Want: "no field", Got: "field present"})
		}
		return
	}
	if actual == nil {
		*diffs = append(*diffs, ProjectionParityDiff{Path: path, Want: typeKind(expected.Type), Got: "missing field"})
		return
	}
	expectedObj := expr.AsObject(expected.Type)
	actualObj := expr.AsObject(actual.Type)
	if expectedObj != nil || actualObj != nil {
		compareProjectionObjects(diffs, path, expected, actual, expectedObj, actualObj, allowExtras)
		return
	}
	expectedArray := expr.AsArray(expected.Type)
	actualArray := expr.AsArray(actual.Type)
	if expectedArray != nil || actualArray != nil {
		if expectedArray == nil || actualArray == nil {
			*diffs = append(*diffs, ProjectionParityDiff{Path: path, Want: typeKind(expected.Type), Got: typeKind(actual.Type)})
			return
		}
		compareProjectionFieldWithMode(diffs, appendProjectionPath(path, "[]"), expectedArray.ElemType, actualArray.ElemType, allowExtras)
		return
	}
	expectedMap := expr.AsMap(expected.Type)
	actualMap := expr.AsMap(actual.Type)
	if expectedMap != nil || actualMap != nil {
		if expectedMap == nil || actualMap == nil {
			*diffs = append(*diffs, ProjectionParityDiff{Path: path, Want: typeKind(expected.Type), Got: typeKind(actual.Type)})
			return
		}
		compareProjectionFieldWithMode(diffs, appendProjectionPath(path, "{}"), expectedMap.ElemType, actualMap.ElemType, allowExtras)
	}
}

func compareProjectionObjects(
	diffs *[]ProjectionParityDiff,
	path string,
	expected, actual *expr.AttributeExpr,
	expectedObj, actualObj *expr.Object,
	allowExtras bool,
) {
	if expectedObj == nil || actualObj == nil {
		*diffs = append(*diffs, ProjectionParityDiff{Path: path, Want: typeKind(expected.Type), Got: typeKind(actual.Type)})
		return
	}
	for _, expectedAttr := range *expectedObj {
		fieldPath := appendProjectionPath(path, expectedAttr.Name)
		actualAttr := actualObj.Attribute(expectedAttr.Name)
		if actualAttr == nil {
			*diffs = append(*diffs, ProjectionParityDiff{Path: fieldPath, Want: "field present", Got: "missing field"})
			continue
		}
		if expected.IsRequired(expectedAttr.Name) != actual.IsRequired(expectedAttr.Name) {
			*diffs = append(*diffs, ProjectionParityDiff{
				Path: fieldPath,
				Want: requiredLabel(expected.IsRequired(expectedAttr.Name)),
				Got:  requiredLabel(actual.IsRequired(expectedAttr.Name)),
			})
		}
		compareProjectionFieldWithMode(diffs, fieldPath, expectedAttr.Attribute, actualAttr, allowExtras)
	}
	if allowExtras {
		return
	}
	for _, actualAttr := range *actualObj {
		if expectedObj.Attribute(actualAttr.Name) == nil {
			*diffs = append(*diffs, ProjectionParityDiff{Path: appendProjectionPath(path, actualAttr.Name), Want: "no field", Got: "field present"})
		}
	}
}

func appendProjectionPath(base, elem string) string {
	if base == "" {
		return elem
	}
	if elem == "[]" || elem == "{}" {
		return base + elem
	}
	return base + "." + elem
}

func requiredLabel(required bool) string {
	if required {
		return "required field"
	}
	return "optional field"
}

func typeKind(dt expr.DataType) string {
	if dt == nil {
		return "<nil>"
	}
	if expr.AsObject(dt) != nil {
		return "object"
	}
	if expr.AsArray(dt) != nil {
		return "array"
	}
	if expr.AsMap(dt) != nil {
		return "map"
	}
	return fmt.Sprintf("kind %d", dt.Kind())
}
