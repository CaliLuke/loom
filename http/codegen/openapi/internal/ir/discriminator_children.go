package ir

import (
	"maps"
	"slices"

	"github.com/CaliLuke/loom/expr"
)

// recordDiscriminatorChildren derives ParentHint publication dependencies from
// declaration identity. A declaration whose schema references a hinted parent
// declaration through allOf is an implicit member of that parent's
// discriminator, so publishing the parent must also publish the child even
// when no operation uses it. Flat extensions, grandchildren and occurrence
// overlays are not direct allOf children of the parent declaration. The named
// root analysis that admits the hint records it, so a parent whose own schema
// is wrapped by composition still counts. The direct children of each hinted
// root are also kept as declarations so that every projected representation
// of the parent publishes them in that representation.
func (a *Analyzer) recordDiscriminatorChildren(types []expr.UserType, resultTypes []*expr.ResultTypeExpr) {
	declarations := make([]expr.UserType, 0, len(types)+len(resultTypes))
	declarations = append(declarations, types...)
	for _, result := range resultTypes {
		declarations = append(declarations, result)
	}
	for _, child := range declarations {
		a.declarationTypes[child.ID()] = child
	}
	for _, child := range declarations {
		childName, ok := a.declarationComponentName(child)
		if !ok {
			continue
		}
		for _, base := range child.Attribute().CompositionReferenceBases() {
			parent, ok := base.(expr.UserType)
			if !ok {
				continue
			}
			parentName, ok := a.declarationComponentName(parent)
			if !ok {
				continue
			}
			if _, hinted := a.discriminatorHints.parentRoots[a.schemas[parentName]]; !hinted {
				continue
			}
			a.addDiscriminatorChild(parentName, childName)
			if !slices.Contains(a.discriminatorFamilies[parentName], child) {
				a.discriminatorFamilies[parentName] = append(a.discriminatorFamilies[parentName], child)
			}
		}
	}
}

// addDiscriminatorChild records that publishing parent publishes child.
func (a *Analyzer) addDiscriminatorChild(parent, child string) {
	if slices.Contains(a.discriminatorChildren[parent], child) {
		return
	}
	if a.discriminatorChildren == nil {
		a.discriminatorChildren = make(map[string][]string)
	}
	a.discriminatorChildren[parent] = append(a.discriminatorChildren[parent], child)
}

// declarationComponentName returns the local component name analyzed for the
// declaration root of typ.
func (a *Analyzer) declarationComponentName(typ expr.UserType) (string, bool) {
	root := a.declarationRoots[typ.ID()]
	if root == nil {
		return "", false
	}
	name, ok := schemaComponentName(root.Ref)
	if !ok || a.schemas[name] == nil {
		return "", false
	}
	return name, true
}

// renameDiscriminatorChildren carries recorded dependencies to the component
// names assigned by representation allocation and drops discarded components.
func renameDiscriminatorChildren(children map[string][]string, names map[string]string) map[string][]string {
	var renamed map[string][]string
	for _, parent := range slices.Sorted(maps.Keys(children)) {
		assigned, ok := names[parent]
		if !ok {
			continue
		}
		for _, local := range children[parent] {
			child, ok := names[local]
			if !ok || slices.Contains(renamed[assigned], child) {
				continue
			}
			if renamed == nil {
				renamed = make(map[string][]string)
			}
			renamed[assigned] = append(renamed[assigned], child)
		}
	}
	return renamed
}
