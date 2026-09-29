package expr

import (
	"fmt"
	"reflect"
)

type (
	// ValueOccurrence identifies an immutable, finalized semantic occurrence in
	// one ValueContext. Its zero value is invalid. Named declaration reuse does
	// not merge the validation, presence or sources of different occurrences.
	ValueOccurrence struct {
		context *valueContextIdentity
		graph   *valueOccurrenceGraph
		node    *valueOccurrenceNode
	}

	valueContextIdentity struct {
		marker byte
	}

	valueOccurrenceGraph struct {
		root         *valueOccurrenceNode
		nodes        []*valueOccurrenceNode
		declarations []*valueDeclarationNode
		next         uint64
	}

	valueOccurrenceNode struct {
		id           uint64
		origin       *AttributeExpr
		attribute    *AttributeExpr
		declaration  *valueDeclarationNode
		examples     []valueOccurrenceExample
		defaultValue *valueSourceSnapshot
		enumValues   []valueSourceSnapshot
	}

	valueOccurrenceExample struct {
		origin  *ExampleExpr
		example *ExampleExpr
		source  valueSourceSnapshot
	}

	valueDeclarationNode struct {
		id                  uint64
		kind                Kind
		typ                 DataType
		members             []valueOccurrenceMember
		branches            []valueOccurrenceBranch
		element             *valueOccurrenceNode
		key                 *valueOccurrenceNode
		alias               *valueOccurrenceNode
		nonNullableElements bool
		untagged            bool
		typeKey             string
		valueKey            string
		rank                uint64
	}

	valueOccurrenceMember struct {
		id   uint64
		name string
		wire string
		node *valueOccurrenceNode
	}

	valueOccurrenceBranch struct {
		id   uint64
		name string
		tag  string
		node *valueOccurrenceNode
	}

	valueOccurrenceBuilder struct {
		graph *valueOccurrenceGraph
		types map[DataType]*valueDeclarationNode
	}
)

// NewOccurrence captures an effective finalized attribute and returns a new
// semantic root even when another root uses the same named type. It preserves
// finite recursive declarations. Malformed declaration graphs fail construction;
// invalid authored values remain deferred source failures until selected.
func (c *ValueContext) NewOccurrence(finalized *AttributeExpr) (ValueOccurrence, error) {
	if c == nil || c.identity == nil {
		return ValueOccurrence{}, fmt.Errorf("value occurrence requires a value context")
	}
	b := &valueOccurrenceBuilder{graph: &valueOccurrenceGraph{}, types: make(map[DataType]*valueDeclarationNode)}
	root, err := b.occurrence(finalized)
	if err != nil {
		return ValueOccurrence{}, err
	}
	b.graph.root = root
	if err := b.checkRanks(); err != nil {
		return ValueOccurrence{}, err
	}
	return ValueOccurrence{context: c.identity, graph: b.graph, node: root}, nil
}

// ID returns a generation-local identity for this occurrence. It is not a
// public type name, content hash, or stable synthesis seed.
func (o ValueOccurrence) ID() ValueIdentity {
	if o.context == nil || o.graph == nil || o.node == nil {
		return ValueIdentity{}
	}
	return ValueIdentity{context: o.context, graph: o.graph, index: o.node.id}
}

func valueMemberRequired(owner *valueOccurrenceNode, member valueOccurrenceMember) bool {
	return owner != nil && owner.attribute.IsRequired(member.name)
}

func (b *valueOccurrenceBuilder) identity() uint64 {
	b.graph.next++
	return b.graph.next
}

func (b *valueOccurrenceBuilder) occurrence(source *AttributeExpr) (*valueOccurrenceNode, error) {
	if source == nil || source.Type == nil {
		return nil, fmt.Errorf("value occurrence has no finalized type")
	}
	attribute := copyValueOccurrenceAttribute(source)
	node := &valueOccurrenceNode{id: b.identity(), origin: source, attribute: attribute}
	b.graph.nodes = append(b.graph.nodes, node)
	for _, example := range source.UserExamples {
		if example == nil {
			return nil, fmt.Errorf("value occurrence contains a nil example descriptor")
		}
		copy := *example
		copy.Meta = copyValueMeta(example.Meta)
		origin := valueExampleOrigin(example)
		snapshot := snapshotValueSource(origin.Value)
		copy.Value = snapshot.raw
		copy.ExplicitNull = origin.ExplicitNull
		node.examples = append(node.examples, valueOccurrenceExample{origin: origin, example: &copy, source: snapshot})
		attribute.UserExamples = append(attribute.UserExamples, &copy)
	}
	if source.DefaultValue != nil {
		snapshot := snapshotValueSource(source.DefaultValue)
		node.defaultValue = &snapshot
		attribute.DefaultValue = snapshot.raw
	}
	if source.Validation != nil {
		for _, value := range source.Validation.Values {
			snapshot := snapshotValueSource(value)
			node.enumValues = append(node.enumValues, snapshot)
			attribute.Validation.Values = append(attribute.Validation.Values, snapshot.raw)
		}
	}
	declaration, err := b.declaration(source.Type)
	if err != nil {
		return nil, err
	}
	node.declaration = declaration
	attribute.Type = declaration.typ
	for _, entry := range []struct {
		sources []DataType
		target  *[]DataType
	}{
		{source.References, &attribute.References},
		{source.Bases, &attribute.Bases},
	} {
		for _, typ := range entry.sources {
			decl, err := b.declaration(typ)
			if err != nil {
				return nil, err
			}
			*entry.target = append(*entry.target, decl.typ)
		}
	}
	return node, nil
}

func (b *valueOccurrenceBuilder) declaration(source DataType) (*valueDeclarationNode, error) {
	if source == nil || !reflect.TypeOf(source).Comparable() {
		return nil, fmt.Errorf("value declaration has no supported identity")
	}
	if prior, ok := b.types[source]; ok {
		return prior, nil
	}
	decl := &valueDeclarationNode{id: b.identity()}
	b.types[source] = decl
	b.graph.declarations = append(b.graph.declarations, decl)
	switch actual := source.(type) {
	case Primitive:
		decl.kind, decl.typ = actual.Kind(), actual
	case *Object:
		return b.object(decl, actual)
	case *Array:
		return b.array(decl, actual)
	case *Map:
		return b.mapping(decl, actual)
	case *Union:
		return b.union(decl, actual)
	case UserType:
		return b.named(decl, actual)
	default:
		return nil, fmt.Errorf("unsupported value declaration %T", source)
	}
	return decl, nil
}

func (b *valueOccurrenceBuilder) object(decl *valueDeclarationNode, actual *Object) (*valueDeclarationNode, error) {
	if actual == nil {
		return nil, fmt.Errorf("value declaration contains a nil object")
	}
	object := &Object{}
	decl.kind, decl.typ = ObjectKind, object
	names := make(map[string]bool)
	for _, field := range *actual {
		if field == nil {
			return nil, fmt.Errorf("value declaration contains a nil member")
		}
		name, wire := field.Name, JSONFieldName(ElementName(field.Name), field.Attribute)
		if names[name] {
			return nil, fmt.Errorf("value declaration contains duplicate member %q", field.Name)
		}
		names[name] = true
		child, err := b.occurrence(field.Attribute)
		if err != nil {
			return nil, fmt.Errorf("member %q: %w", field.Name, err)
		}
		decl.members = append(decl.members, valueOccurrenceMember{id: b.identity(), name: name, wire: wire, node: child})
		*object = append(*object, &NamedAttributeExpr{Name: field.Name, Attribute: child.attribute})
	}
	return decl, nil
}

func (b *valueOccurrenceBuilder) array(decl *valueDeclarationNode, actual *Array) (*valueDeclarationNode, error) {
	if actual == nil {
		return nil, fmt.Errorf("value declaration contains a nil array")
	}
	array := &Array{NonNullableElems: actual.NonNullableElems}
	decl.kind, decl.typ = ArrayKind, array
	decl.nonNullableElements = actual.NonNullableElems
	child, err := b.occurrence(actual.ElemType)
	if err != nil {
		return nil, fmt.Errorf("array element: %w", err)
	}
	decl.element, array.ElemType = child, child.attribute
	return decl, nil
}

func (b *valueOccurrenceBuilder) mapping(decl *valueDeclarationNode, actual *Map) (*valueDeclarationNode, error) {
	if actual == nil {
		return nil, fmt.Errorf("value declaration contains a nil map")
	}
	mapping := &Map{}
	decl.kind, decl.typ = MapKind, mapping
	key, err := b.occurrence(actual.KeyType)
	if err != nil {
		return nil, fmt.Errorf("map key: %w", err)
	}
	child, err := b.occurrence(actual.ElemType)
	if err != nil {
		return nil, fmt.Errorf("map value: %w", err)
	}
	decl.key, mapping.KeyType = key, key.attribute
	decl.element, mapping.ElemType = child, child.attribute
	return decl, nil
}

func (b *valueOccurrenceBuilder) union(decl *valueDeclarationNode, actual *Union) (*valueDeclarationNode, error) {
	if actual == nil {
		return nil, fmt.Errorf("value declaration contains a nil union")
	}
	union := *actual
	union.Values = nil
	decl.kind, decl.typ = UnionKind, &union
	decl.untagged, decl.typeKey, decl.valueKey = actual.Untagged, actual.GetTypeKey(), actual.GetValueKey()
	tags := make(map[string]bool)
	for _, branch := range actual.Values {
		if branch == nil {
			return nil, fmt.Errorf("value declaration contains a nil branch")
		}
		tag := UnionVariantTag(branch)
		if tags[tag] {
			return nil, fmt.Errorf("value declaration contains duplicate branch tag %q", tag)
		}
		tags[tag] = true
		child, err := b.occurrence(branch.Attribute)
		if err != nil {
			return nil, fmt.Errorf("branch %q: %w", branch.Name, err)
		}
		decl.branches = append(decl.branches, valueOccurrenceBranch{id: b.identity(), name: branch.Name, tag: tag, node: child})
		union.Values = append(union.Values, &NamedAttributeExpr{Name: branch.Name, Attribute: child.attribute})
	}
	return decl, nil
}

func (b *valueOccurrenceBuilder) named(decl *valueDeclarationNode, actual UserType) (*valueDeclarationNode, error) {
	if reflect.ValueOf(actual).Kind() == reflect.Pointer && reflect.ValueOf(actual).IsNil() {
		return nil, fmt.Errorf("value declaration contains a nil named type")
	}
	copy := actual.Dup(nil)
	decl.kind, decl.typ = actual.Kind(), copy
	child, err := b.occurrence(actual.Attribute())
	if err != nil {
		return nil, fmt.Errorf("named declaration %q: %w", actual.ID(), err)
	}
	decl.alias = child
	copy.SetAttribute(child.attribute)
	if result, ok := copy.(*ResultTypeExpr); ok {
		for _, view := range result.Views {
			viewNode, err := b.occurrence(view.AttributeExpr)
			if err != nil {
				return nil, fmt.Errorf("view %q: %w", view.Name, err)
			}
			view.AttributeExpr = viewNode.attribute
			view.Parent = result
		}
	}
	return decl, nil
}

func (b *valueOccurrenceBuilder) checkRanks() error {
	state := make(map[*valueDeclarationNode]uint8)
	var visit func(*valueDeclarationNode) error
	visit = func(decl *valueDeclarationNode) error {
		if state[decl] == 2 {
			return nil
		}
		if state[decl] == 1 {
			return fmt.Errorf("value declaration has a non-consuming alias or union cycle")
		}
		state[decl] = 1
		children := make([]*valueOccurrenceNode, 0, len(decl.branches)+1)
		if decl.alias != nil {
			children = append(children, decl.alias)
		}
		for _, branch := range decl.branches {
			children = append(children, branch.node)
		}
		for _, child := range children {
			if err := visit(child.declaration); err != nil {
				return err
			}
			decl.rank = max(decl.rank, child.declaration.rank+1)
		}
		state[decl] = 2
		return nil
	}
	for _, decl := range b.graph.declarations {
		if err := visit(decl); err != nil {
			return err
		}
	}
	return nil
}
