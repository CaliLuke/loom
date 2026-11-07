package expr

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/eval"
)

type (
	unionClosureFinalizer struct {
		root *RootExpr
	}

	unionClosureVariants struct {
		seen     map[*AttributeExpr]bool
		reserved map[string]bool
		named    map[copyTypeKey][]unionClosureVariant
		unions   map[*Union]*Union
	}

	unionClosureVariant struct {
		source UserType
		target UserType
	}
)

func (*unionClosureFinalizer) EvalName() string {
	return "union occurrence closure"
}

func (f *unionClosureFinalizer) Finalize() {
	roots := f.root.unionBranchRoots(make(map[UserType]string))
	slices.SortFunc(roots, func(a, b unionBranchRoot) int {
		return cmp.Compare(a.path, b.path)
	})
	variants := &unionClosureVariants{
		seen: make(map[*AttributeExpr]bool), reserved: make(map[string]bool),
		named: make(map[copyTypeKey][]unionClosureVariant), unions: make(map[*Union]*Union),
	}
	for _, root := range roots {
		variants.walk(root.attribute, false)
	}
	clear(variants.seen)
	for _, root := range roots {
		variants.walk(root.attribute, true)
	}
}

// walk reserves authored names before introducing private declarations. Both
// passes visit occurrences by pointer: copied named types may share an ID.
func (v *unionClosureVariants) walk(attribute *AttributeExpr, close bool) {
	if attribute == nil || v.seen[attribute] {
		return
	}
	v.seen[attribute] = true
	switch actual := attribute.Type.(type) {
	case UserType:
		if !close {
			v.reserved[actual.Name()] = true
			v.reserved[actual.ID()] = true
		}
		v.walk(actual.Attribute(), close)
		if result, ok := actual.(*ResultTypeExpr); ok {
			for _, view := range result.Views {
				v.walk(view.AttributeExpr, close)
			}
		}
	case *Union:
		if !close {
			v.reserved[actual.Name()] = true
		}
		v.fields(actual.Values, close)
	case *Object:
		v.fields(*actual, close)
	case *Array:
		v.walk(actual.ElemType, close)
	case *Map:
		v.walk(actual.KeyType, close)
		v.walk(actual.ElemType, close)
	}
	v.walk(attribute.AdditionalAttributes, close)
	if value, present := attribute.Meta.Last(unionClosureMeta); close && present && value == "false" {
		attribute.Type = v.closedType(attribute.Type)
	}
}

func (v *unionClosureVariants) fields(fields []*NamedAttributeExpr, close bool) {
	ordered := slices.Clone(fields)
	slices.SortFunc(ordered, func(a, b *NamedAttributeExpr) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, field := range ordered {
		v.walk(field.Attribute, close)
	}
}

func (v *unionClosureVariants) closedType(typ DataType) DataType {
	switch actual := typ.(type) {
	case *Union:
		if actual.closed {
			return actual
		}
		if prior := v.unions[actual]; prior != nil {
			return prior
		}
		// Branch occurrences already own their finalized policies. Share those
		// references so closing the enclosing union does not close nested uses.
		copy := *actual
		copy.closed = true
		v.unions[actual] = &copy
		return &copy
	case UserType:
		if actual.Attribute().UnionClosed() {
			return actual
		}
		key := typeCopyKey(actual)
		for _, prior := range v.named[key] {
			if sameUnionClosureContract(actual, prior.source) {
				return prior.target
			}
		}
		attribute := copyValueAttribute(actual.Attribute())
		copy := actual.Dup(attribute)
		name := v.name(actual.Name() + "Closed")
		copy.Rename(name)
		switch named := copy.(type) {
		case *UserTypeExpr:
			named.unionClosureID = name
		case *ResultTypeExpr:
			named.unionClosureID = name
		default:
			eval.ReportError("union occurrence closure requires a framework-owned named type, got %T", actual)
			return typ
		}
		v.named[key] = append(v.named[key], unionClosureVariant{source: actual, target: copy})
		attribute.Type = v.closedType(actual.Attribute().Type)
		return copy
	}
	return typ
}

func (v *unionClosureVariants) name(base string) string {
	name := base
	for suffix := 2; v.reserved[name]; suffix++ {
		name = base + strconv.Itoa(suffix)
	}
	v.reserved[name] = true
	return name
}
