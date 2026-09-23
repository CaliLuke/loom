package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// fieldHasher computes the field-aware hash of a data type. It builds the
// graph of composite types reachable from the hashed type, where each node has
// a label holding its local shape and an ordered list of child nodes. It then
// partitions the nodes into classes of structurally equivalent types by
// iterative refinement and serializes the quotient graph reachable from the
// root. Two types get the same hash exactly when their (possibly infinite)
// unfoldings are the same, and the work is polynomial in the graph size even
// when types are mutually recursive.
type fieldHasher struct {
	// ignoreNames omits user type names from the labels.
	ignoreNames bool
	// ignoreTags omits "struct:field:" metadata from the labels.
	ignoreTags bool
	// index maps each composite type to its node index.
	index map[DataType]int
	// labels holds the local shape of each node.
	labels []string
	// children holds the ordered child node indices of each node.
	children [][]int
	// steps counts the node and edge visits done by the refinement, so it
	// measures the hashing work.
	steps int
}

var (
	arrayPrefix              = "_a_"
	attributePrefix          = "-"
	attributeTypePrefix      = "/"
	mapElemPrefix            = ":"
	mapPrefix                = "_m_"
	unionTypePrefix          = "_u_"
	unionAttributePrefix     = "_*_"
	unionAttributeTypePrefix = "_|_"
	objectPrefix             = "_o_"
	tagPrefix                = "+"
	userTypePrefix           = "_t_"
	nullablePrefix           = "?"
)

// Hash returns a hash value for the given data type. The hash of a primitive
// type is its name. When ignoreFields is true, two types have the same hash
// if:
//   - both types have the same kind
//   - array types have elements whose types have the same hash
//   - map types have keys and elements whose types have the same hash
//   - union types have the same name and variants whose types have the same hash
//   - user types have the same name
//   - object attributes have the same "struct:field:xxx" tags if ignoreTags is false
//
// When ignoreFields is false, two types have the same hash if their unfoldings
// are identical: user types are compared by their attribute types instead of
// only by name, and their names are compared too unless ignoreNames is true.
// Recursive types are handled exactly, so a type that references itself never
// has the same hash as a finite type. Hash results are deterministic.
func Hash(dt DataType, ignoreFields, ignoreNames, ignoreTags bool) string {
	if isPrimitiveKind(dt.Kind()) {
		return dt.Name()
	}
	if ignoreFields {
		return *hash(dt, ignoreTags, make(map[*Object]*string))
	}
	return newFieldHasher(ignoreNames, ignoreTags).hash(dt)
}

func newFieldHasher(ignoreNames, ignoreTags bool) *fieldHasher {
	return &fieldHasher{
		ignoreNames: ignoreNames,
		ignoreTags:  ignoreTags,
		index:       make(map[DataType]int),
	}
}

func hash(dt DataType, ignoreTags bool, seen map[*Object]*string) *string {
	if isPrimitiveKind(dt.Kind()) {
		n := dt.Name()
		return &n
	}
	switch dt.Kind() {
	case ArrayKind:
		return hashArray(dt.(*Array), ignoreTags, seen)
	case MapKind:
		return hashMap(dt.(*Map), ignoreTags, seen)
	case UnionKind:
		return hashUnion(dt.(*Union), ignoreTags, seen)
	case UserTypeKind, ResultTypeKind:
		h := userTypePrefix + dt.Name()
		return &h
	case ObjectKind:
		return hashObject(dt.(*Object), ignoreTags, seen)
	default:
		panic(fmt.Sprintf("invalid type for hashing: %T", dt))
	}
}

func hashArray(a *Array, ignoreTags bool, seen map[*Object]*string) *string {
	h := arrayPrefix + hashAttribute(a.ElemType, ignoreTags, seen)
	return &h
}

func hashMap(m *Map, ignoreTags bool, seen map[*Object]*string) *string {
	h := mapPrefix + hashAttribute(m.KeyType, ignoreTags, seen) +
		mapElemPrefix + hashAttribute(m.ElemType, ignoreTags, seen)
	return &h
}

func hashUnion(u *Union, ignoreTags bool, seen map[*Object]*string) *string {
	h := unionTypePrefix + u.TypeName
	if u.Untagged {
		h += ":untagged"
	}
	for _, nat := range sortedUnionValues(u) {
		h += unionAttributePrefix + nat.Name + unionAttributeTypePrefix + hashAttribute(nat.Attribute, ignoreTags, seen)
	}
	return &h
}

func hashObject(o *Object, ignoreTags bool, seen map[*Object]*string) *string {
	if s, ok := seen[o]; ok {
		return s
	}
	h := objectPrefix
	ph := &h
	seen[o] = ph
	for _, a := range sorted(o) {
		*ph += attributePrefix + a.Name +
			attributeTypePrefix + hashAttribute(a.Attribute, ignoreTags, seen)
		if !ignoreTags {
			for _, k := range sortedStructFieldMetaKeys(a.Attribute.Meta) {
				*ph += fmt.Sprintf("%s%s%s", tagPrefix, k, a.Attribute.Meta[k])
			}
		}
	}
	return ph
}

func hashAttribute(attribute *AttributeExpr, ignoreTags bool, seen map[*Object]*string) string {
	prefix := ""
	if IsNullable(attribute) {
		prefix = nullablePrefix
	}
	return prefix + *hash(attribute.Type, ignoreTags, seen)
}

func sortedStructFieldMetaKeys(meta MetaExpr) []string {
	keys := make([]string, 0, len(meta))
	for k := range meta {
		if strings.HasPrefix(k, "struct:field:") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func sorted(o *Object) Object {
	if o == nil {
		return nil
	}
	s := make([]*NamedAttributeExpr, len(*o))
	copy(s, *o)
	sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	return Object(s)
}

// writeHashToken writes s to b with a length prefix so that concatenated
// tokens can never be confused with one another.
func writeHashToken(b *strings.Builder, s string) {
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteByte(':')
	b.WriteString(s)
}

// internHashClass returns the class number of key in classes, adding key with
// the next number when it is new.
func internHashClass(classes map[string]int, key string) int {
	if c, ok := classes[key]; ok {
		return c
	}
	c := len(classes)
	classes[key] = c
	return c
}

// hash returns the canonical encoding of the quotient graph reachable from
// the composite type dt.
func (h *fieldHasher) hash(dt DataType) string {
	root := h.node(dt)
	classes := h.refine()

	// Serialize the quotient graph in breadth-first order, numbering each
	// class when first reached. Every member of a class has the same label
	// and the same child classes, so any member represents the class.
	var out strings.Builder
	number := map[int]int{classes[root]: 0}
	queue := []int{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		out.WriteByte(';')
		writeHashToken(&out, h.labels[node])
		for _, child := range h.children[node] {
			n, ok := number[classes[child]]
			if !ok {
				n = len(number)
				number[classes[child]] = n
				queue = append(queue, child)
			}
			out.WriteByte('@')
			out.WriteString(strconv.Itoa(n))
		}
	}
	return out.String()
}

// refine partitions the nodes into classes of equivalent types and returns the
// class of each node. It starts from the classes of equal labels and splits
// classes whose members have children in different classes until no class
// splits. Each round keeps or splits classes, so there are at most as many
// rounds as nodes.
func (h *fieldHasher) refine() []int {
	byLabel := make(map[string]int)
	classes := make([]int, len(h.labels))
	for i, label := range h.labels {
		classes[i] = internHashClass(byLabel, label)
	}
	count := len(byLabel)
	for {
		bySignature := make(map[string]int)
		next := make([]int, len(classes))
		var sig strings.Builder
		for i, class := range classes {
			h.steps += 1 + len(h.children[i])
			sig.Reset()
			sig.WriteString(strconv.Itoa(class))
			for _, child := range h.children[i] {
				sig.WriteByte(',')
				sig.WriteString(strconv.Itoa(classes[child]))
			}
			next[i] = internHashClass(bySignature, sig.String())
		}
		classes = next
		if len(bySignature) == count {
			return classes
		}
		count = len(bySignature)
	}
}

// writeType writes a reference to dt into label: the name of a primitive type
// or a child slot bound to the node of a composite type.
func (h *fieldHasher) writeType(label *strings.Builder, children *[]int, dt DataType) {
	if isPrimitiveKind(dt.Kind()) {
		writeHashToken(label, dt.Name())
		return
	}
	label.WriteByte('#')
	*children = append(*children, h.node(dt))
}

// writeAttribute writes a reference to the type of att, preceded by its
// nullability, into label.
func (h *fieldHasher) writeAttribute(label *strings.Builder, children *[]int, att *AttributeExpr) {
	if IsNullable(att) {
		writeHashToken(label, nullablePrefix)
	}
	h.writeType(label, children, att.Type)
}

// writeTags writes the "struct:field:" metadata of meta into label unless tags
// are ignored.
func (h *fieldHasher) writeTags(label *strings.Builder, meta MetaExpr) {
	if h.ignoreTags {
		return
	}
	for _, k := range sortedStructFieldMetaKeys(meta) {
		writeHashToken(label, tagPrefix)
		writeHashToken(label, k)
		writeHashToken(label, strconv.Itoa(len(meta[k])))
		for _, v := range meta[k] {
			writeHashToken(label, v)
		}
	}
}

// node returns the index of the node of the composite type dt, adding the node
// and the nodes it reaches when dt is new. The index is assigned before the
// children are visited so that recursive references terminate.
func (h *fieldHasher) node(dt DataType) int {
	if i, ok := h.index[dt]; ok {
		return i
	}
	i := len(h.labels)
	h.index[dt] = i
	h.labels = append(h.labels, "")
	h.children = append(h.children, nil)
	var label strings.Builder
	var children []int
	switch actual := dt.(type) {
	case *Array:
		writeHashToken(&label, arrayPrefix)
		h.writeAttribute(&label, &children, actual.ElemType)
	case *Map:
		writeHashToken(&label, mapPrefix)
		h.writeAttribute(&label, &children, actual.KeyType)
		writeHashToken(&label, mapElemPrefix)
		h.writeAttribute(&label, &children, actual.ElemType)
	case *Union:
		writeHashToken(&label, unionTypePrefix)
		writeHashToken(&label, actual.TypeName)
		writeHashToken(&label, strconv.FormatBool(actual.Untagged))
		for _, nat := range sortedUnionValues(actual) {
			writeHashToken(&label, unionAttributePrefix)
			writeHashToken(&label, nat.Name)
			h.writeAttribute(&label, &children, nat.Attribute)
		}
	case *Object:
		writeHashToken(&label, objectPrefix)
		for _, nat := range sorted(actual) {
			writeHashToken(&label, attributePrefix)
			writeHashToken(&label, nat.Name)
			h.writeAttribute(&label, &children, nat.Attribute)
			h.writeTags(&label, nat.Attribute.Meta)
		}
	case UserType:
		writeHashToken(&label, userTypePrefix)
		if !h.ignoreNames {
			writeHashToken(&label, actual.Name())
		}
		att := actual.Attribute()
		h.writeAttribute(&label, &children, att)
		h.writeTags(&label, att.Meta)
	default:
		panic(fmt.Sprintf("invalid type for hashing: %T", dt))
	}
	h.labels[i] = label.String()
	h.children[i] = children
	return i
}
