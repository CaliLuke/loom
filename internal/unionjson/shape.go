package unionjson

import (
	"fmt"
	"strings"

	loom "github.com/CaliLuke/loom/pkg"
)

func renderShape(b *strings.Builder, name string, root *loom.JSONShape) {
	nodes := []*loom.JSONShape{}
	indexes := make(map[*loom.JSONShape]int)
	var visit func(*loom.JSONShape)
	visit = func(node *loom.JSONShape) {
		if node == nil {
			return
		}
		if _, exists := indexes[node]; exists {
			return
		}
		indexes[node] = len(nodes)
		nodes = append(nodes, node)
		visit(node.Child)
		for _, field := range node.Fields {
			visit(field.Shape)
		}
	}
	visit(root)
	fmt.Fprintf(b, "%sNodes := make([]loom.JSONShape, %d)\n", name, len(nodes))
	for index, node := range nodes {
		fmt.Fprintf(b, "%sNodes[%d] = loom.JSONShape{Kind: %q, Nullable: %t, Closed: %t, NonNullableElements: %t,\n", name, index, node.Kind, node.Nullable, node.Closed, node.NonNullableElements)
		if node.Child != nil {
			fmt.Fprintf(b, "Child: &%sNodes[%d],\n", name, indexes[node.Child])
		}
		if len(node.Fields) > 0 {
			b.WriteString("Fields: []loom.JSONShapeField{\n")
			for _, field := range node.Fields {
				fmt.Fprintf(b, "{Name: %q, Required: %t, Shape: &%sNodes[%d]},\n", field.Name, field.Required, name, indexes[field.Shape])
			}
			b.WriteString("},\n")
		}
		renderRules(b, node.Rules)
		b.WriteString("}\n")
	}
	fmt.Fprintf(b, "%s := &%sNodes[0]\n", name, name)
}

func renderRules(b *strings.Builder, r loom.JSONShapeRules) {
	fmt.Fprintf(b, "Rules: loom.JSONShapeRules{Minimum: %q, Maximum: %q, ExclusiveMinimum: %q, ExclusiveMaximum: %q,\n", r.Minimum, r.Maximum, r.ExclusiveMinimum, r.ExclusiveMaximum)
	for _, bound := range []struct {
		name  string
		value *int
	}{{"MinLength", r.MinLength}, {"MaxLength", r.MaxLength}} {
		if bound.value != nil {
			fmt.Fprintf(b, "%s: new(%d),\n", bound.name, *bound.value)
		}
	}
	if len(r.Patterns) > 0 {
		fmt.Fprintf(b, "Patterns: %#v,\n", r.Patterns)
	}
	if len(r.Formats) > 0 {
		b.WriteString("Formats: []loom.Format{\n")
		for _, format := range r.Formats {
			fmt.Fprintf(b, "%q,\n", format)
		}
		b.WriteString("},\n")
	}
	if len(r.Enums) > 0 {
		b.WriteString("Enums: [][]jsontext.Value{\n")
		for _, clause := range r.Enums {
			b.WriteString("{\n")
			for _, value := range clause {
				fmt.Fprintf(b, "jsontext.Value(%q),\n", string(value))
			}
			b.WriteString("},\n")
		}
		b.WriteString("},\n")
	}
	b.WriteString("},\n")
}
