package service

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// viewedHelperKey separates physical helper roles from nominal identities.
	// Selection is a logical view, occurrence layout, or parent conversion identity.
	viewedHelperKey struct {
		role      string
		source    string
		target    string
		selection string
	}

	viewConversionSymbols struct {
		projections     map[string]string
		reconstructions map[string]string
	}

	viewWrapperSymbols struct {
		toViewed string
		toResult string
	}
)

func (k viewedHelperKey) Hash() string {
	return fmt.Sprintf("_service_view_helper_%q", [4]string{k.role, k.source, k.target, k.selection})
}

// allocateViewConversions names every emitted conversion in the service package.
// Projected names may be allocated late; helpers therefore have their own cached
// identities and never require renaming an already referenced nominal type.
func allocateViewConversions(att, projected *expr.AttributeExpr, scope, viewScope *codegen.NameScope) viewConversionSymbols {
	result := att.Type.(*expr.ResultTypeExpr)
	sourceName := scope.GoValueTypeName(att)
	targetName := viewScope.GoValueTypeName(projected)
	suffixes := viewHelperSuffixes(result)
	names := viewConversionSymbols{
		projections:     make(map[string]string, len(result.Views)),
		reconstructions: make(map[string]string, len(result.Views)),
	}
	for _, view := range result.Views {
		key := viewedHelperKey{role: "projection", source: att.Type.Hash(), target: projected.Type.Hash(), selection: view.Name}
		names.projections[view.Name] = scope.HashedUnique(key, "Project"+sourceName+suffixes[view.Name])
		key.role = "reconstruction"
		names.reconstructions[view.Name] = scope.HashedUnique(key, "New"+sourceName+"From"+targetName+suffixes[view.Name])
	}
	return names
}

// allocateViewedWrappers separates occurrence wrappers from canonical conversion
// helpers. Explicit view selection does not change a wrapper's physical identity.
func allocateViewedWrappers(att, envelope *expr.AttributeExpr, scope *codegen.NameScope, name string) viewWrapperSymbols {
	key := viewedHelperKey{
		role:      "view-wrapper",
		source:    att.Type.Hash(),
		target:    envelope.Type.Hash(),
		selection: fmt.Sprint(expr.IsNullable(att)),
	}
	toViewed := scope.HashedUnique(key, "NewViewed"+name)
	key.role = "result-wrapper"
	return viewWrapperSymbols{toViewed: toViewed, toResult: scope.HashedUnique(key, "New"+name)}
}
