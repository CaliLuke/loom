package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// protoGoReservedNames returns the method reservations of protoc-gen-go.
func protoGoReservedNames() map[string]bool {
	return map[string]bool{
		"Reset": true, "String": true, "ProtoMessage": true, "Marshal": true,
		"Unmarshal": true, "ExtensionRangeArray": true, "ExtensionMap": true,
		"Descriptor": true,
	}
}

// reserveProtoGoName follows protogen's declaration-order conflict resolution.
// Its historical oneof rule also clears a prior reservation for Get+name.
func reserveProtoGoName(name string, getter bool, used map[string]bool) string {
	for used[name] || getter && used["Get"+name] {
		name += "_"
	}
	used[name] = true
	used["Get"+name] = getter
	return name
}

// protoMapEntryGoName returns the Go name of protoc's implicit map-entry
// message. The parser removes all underscores before GoCamelCase is applied.
func protoMapEntryGoName(field string) string {
	var b strings.Builder
	upper := true
	for i := range len(field) {
		c := field[i]
		if c == '_' {
			upper = true
			continue
		}
		if upper && isASCIILower(c) {
			c -= 'a' - 'A'
		}
		b.WriteByte(c)
		upper = false
	}
	return protoGoName(b.String() + "Entry")
}

// allocateGoNames follows the emitted descriptor order, including synthetic
// oneofs for optional primitives. It reports whether protoc's resulting Go
// message has unique selectors; historical oneof getter collisions require
// different wire oneof names before the next attempt.
func (n *protoMessageNames) allocateGoNames(att *expr.AttributeExpr) bool {
	obj := expr.AsObject(att.Type)
	n.goNames = make(map[string]string)
	n.goWrappers = make(map[string]string)
	used := protoGoReservedNames()
	nested := make(map[string]bool)
	for _, nat := range *obj {
		wire := n.field(nat.Name)
		if expr.IsMap(nat.Attribute.Type) {
			nested[protoMapEntryGoName(wire)] = true
		}
		if union := expr.AsUnion(nat.Attribute.Type); union != nil {
			for i, branch := range n.oneofFields(nat.Name) {
				n.goNames[branch] = reserveProtoGoName(protoGoName(branch), true, used)
				if i == 0 {
					n.goNames[wire] = reserveProtoGoName(protoGoName(wire), false, used)
				}
			}
			continue
		}
		n.goNames[wire] = reserveProtoGoName(protoGoName(wire), true, used)
		if protoBufOptionalField(nat) != "" {
			// Loom wire names start with a letter, so the parser's leading
			// underscore cannot collide with another field or explicit oneof.
			reserveProtoGoName(protoGoName("_"+wire), false, used)
		}
	}
	for _, branches := range n.branches {
		for _, branch := range branches {
			name := n.goNames[branch]
			for nested[name] {
				name += "_"
			}
			n.goWrappers[branch] = name
		}
	}
	return n.goSelectorsUnique(obj)
}

// goSelectorsUnique checks the actual message selectors, including oneof
// getters that protogen's allocation omits. Branch fields live in wrappers;
// only their getters are selectors of the containing message.
func (n *protoMessageNames) goSelectorsUnique(obj *expr.Object) bool {
	used := protoGoReservedNames()
	// The plugin emits ProtoReflect but omits it from its reservations.
	used["ProtoReflect"] = true
	unique := true
	claim := func(name string) {
		if used[name] {
			unique = false
		}
		used[name] = true
	}
	for _, nat := range *obj {
		name := n.goField(nat.Name)
		claim(name)
		claim("Get" + name)
		for _, branch := range n.goBranches(nat.Name) {
			claim("Get" + branch)
		}
	}
	return unique
}

// repairGoSelectors changes the wire names that can collide with selectors
// omitted from protogen's reservations. The wire allocator's taken and claim
// functions preserve its reservations while these names are replaced.
func (n *protoMessageNames) repairGoSelectors(obj *expr.Object, taken func(string) bool, claim func(string, string)) {
	// protoc omits oneof getters and ProtoReflect from its reservations.
	// Repair wire names only when its resulting Go selectors collide.
	for _, nat := range *obj {
		if !expr.IsUnion(nat.Attribute.Type) {
			if n.goField(nat.Name) == "ProtoReflect" {
				field := n.field(nat.Name) + "_field"
				for taken(field) {
					field += "_field"
				}
				claim(field, fmt.Sprintf("attribute %q", nat.Name))
				n.fields[protoNameKey(nat.Name)] = field
			}
			continue
		}
		oneof := n.field(nat.Name) + "_oneof"
		for taken(oneof) {
			oneof += "_oneof"
		}
		claim(oneof, fmt.Sprintf("oneof of attribute %q", nat.Name))
		n.fields[protoNameKey(nat.Name)] = oneof
	}
}
