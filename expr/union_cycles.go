package expr

// hasInlineUnionCycle reports a cycle through union branches and collections.
// Object fields stop this traversal: those shapes have named transform helpers,
// while union branches and collections are expanded inline by the generators.
func hasInlineUnionCycle(union *Union) bool {
	const (
		active = 1
		done   = 2
	)
	states := make(map[DataType]int)
	var visit func(DataType) bool
	visit = func(current DataType) bool {
		if states[current] == active {
			return true
		}
		if states[current] == done {
			return false
		}
		states[current] = active
		var children []DataType
		switch actual := current.(type) {
		case UserType:
			children = []DataType{actual.Attribute().Type}
		case *Union:
			for _, branch := range actual.Values {
				children = append(children, branch.Attribute.Type)
			}
		case *Array:
			children = []DataType{actual.ElemType.Type}
		case *Map:
			children = []DataType{actual.KeyType.Type, actual.ElemType.Type}
		}
		for _, child := range children {
			if visit(child) {
				return true
			}
		}
		states[current] = done
		return false
	}
	return visit(union)
}
