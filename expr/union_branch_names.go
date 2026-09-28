package expr

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/eval"
)

type (
	unionBranchGroup struct {
		base  string
		path  string
		types []*UserTypeExpr
	}

	unionBranchRoot struct {
		path      string
		attribute *AttributeExpr
	}

	unionBranchNames struct {
		groups     map[uint64]*unionBranchGroup
		anchors    map[UserType]string
		authored   map[string]bool
		attributes map[*AttributeExpr]bool
		types      map[DataType]bool
	}
)

// NewUnionBranch constructs an automatically named branch type. Its private
// identity keeps distinct definitions separate during DSL copies. Prepare
// assigns stable names before transport preparation; copies of one definition
// receive the same name. Callers must retain the returned type in the design.
func (r *RootExpr) NewUnionBranch(name string, attribute *AttributeExpr) *UserTypeExpr {
	r.unionBranchSequence++
	branch := &UserTypeExpr{
		AttributeExpr: attribute,
		TypeName:      name,
		unionBranchID: r.unionBranchSequence,
	}
	branch.unionBranchName = branch.Name()
	return branch
}

// Prepare gives automatically promoted union branches stable, distinct names
// before identity-based transport preparation and validation run.
func (r *RootExpr) Prepare() {
	if r.unionBranchesPrepared == r.unionBranchSequence {
		return
	}
	names := &unionBranchNames{
		groups:     make(map[uint64]*unionBranchGroup),
		anchors:    make(map[UserType]string),
		authored:   make(map[string]bool),
		attributes: make(map[*AttributeExpr]bool),
		types:      make(map[DataType]bool),
	}
	roots := r.unionBranchRoots(names.anchors)
	slices.SortFunc(roots, func(a, b unionBranchRoot) int {
		return cmp.Compare(a.path, b.path)
	})
	for _, root := range roots {
		names.walkAttribute(root.attribute, root.path)
	}
	if eval.Context.Errors != nil {
		return
	}
	names.assign()
	r.unionBranchesPrepared = r.unionBranchSequence
}

func unionBranchPath(path, kind, name string) string {
	return path + "/" + kind + "/" + strconv.Quote(name)
}

func (n *unionBranchNames) walkAttribute(attribute *AttributeExpr, path string) {
	if attribute == nil || n.attributes[attribute] {
		return
	}
	n.attributes[attribute] = true
	if err := attribute.resolveTypeRef(); err != nil {
		eval.ReportError(err.Error())
		return
	}
	n.walkType(attribute.Type, path)
	for i, base := range attribute.Bases {
		n.walkType(base, unionBranchPath(path, "base", strconv.Itoa(i)))
	}
	for i, reference := range attribute.References {
		n.walkType(reference, unionBranchPath(path, "reference", strconv.Itoa(i)))
	}
}

func (n *unionBranchNames) walkType(typ DataType, path string) {
	if typ == nil || n.types[typ] {
		return
	}
	n.types[typ] = true
	switch actual := typ.(type) {
	case UserType:
		if anchor, ok := n.anchors[actual]; ok {
			path = anchor
		}
		if branch, ok := actual.(*UserTypeExpr); ok && branch.unionBranchID != 0 {
			group := n.groups[branch.unionBranchID]
			if group == nil {
				group = &unionBranchGroup{base: branch.unionBranchName, path: path}
				n.groups[branch.unionBranchID] = group
			}
			group.path = min(group.path, path)
			group.types = append(group.types, branch)
		} else {
			n.authored[actual.Name()] = true
		}
		n.walkAttribute(actual.Attribute(), path)
	case *Object:
		n.walkFields(*actual, path, "field")
	case *Union:
		n.walkFields(actual.Values, path, "branch")
	case *Array:
		n.walkAttribute(actual.ElemType, path+"/items")
	case *Map:
		n.walkAttribute(actual.KeyType, path+"/keys")
		n.walkAttribute(actual.ElemType, path+"/values")
	}
}

func (n *unionBranchNames) walkFields(fields []*NamedAttributeExpr, path, kind string) {
	ordered := slices.Clone(fields)
	slices.SortFunc(ordered, func(a, b *NamedAttributeExpr) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, field := range ordered {
		n.walkAttribute(field.Attribute, unionBranchPath(path, kind, field.Name))
	}
}

func (n *unionBranchNames) assign() {
	groups := make([]*unionBranchGroup, 0, len(n.groups))
	reserved := make(map[string]bool, len(n.groups)+len(n.authored))
	for name := range n.authored {
		reserved[name] = true
	}
	for _, group := range n.groups {
		groups = append(groups, group)
		reserved[group.base] = true
	}
	slices.SortFunc(groups, func(a, b *unionBranchGroup) int {
		if result := cmp.Compare(a.base, b.base); result != 0 {
			return result
		}
		return cmp.Compare(a.path, b.path)
	})
	used := n.authored
	for _, group := range groups {
		name := group.base
		if used[name] {
			for suffix := 2; ; suffix++ {
				candidate := group.base + strconv.Itoa(suffix)
				if !reserved[candidate] && !used[candidate] {
					name = candidate
					break
				}
			}
		}
		used[name] = true
		for _, branch := range group.types {
			if branch.Name() != name {
				branch.Rename(name)
			}
		}
	}
}
