package expr

import "fmt"

func selectEffectiveDefault(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) error {
	// Only an authored occurrence default resolves competing inherited choices.
	// Resolve all candidates in this occurrence so collection member identities
	// and scalar normalization use the same effective contract.
	if node.defaultValue == nil && len(node.constraintParents) > 1 {
		var selected *valueOccurrenceNode
		var selectedValue ResolvedValue
		for _, parent := range node.constraintParents {
			layer := effectiveDefaultLayer(parent)
			if layer == nil {
				continue
			}
			result, err := resolveEffectiveDefault(context, graph, node, layer)
			if err != nil {
				return err
			}
			if selected != nil && !resolvedEnumEqual(selectedValue, result.value) {
				return fmt.Errorf("%sconflicting inherited defaults from %q and %q; declare a local Default",
					effectiveConstraintPrefix(node), effectiveLayerName(selected), effectiveLayerName(layer))
			}
			selected, selectedValue = layer, result.value
		}
	}
	layer := effectiveDefaultLayer(node)
	if layer == nil {
		return nil
	}
	if _, err := resolveEffectiveDefault(context, graph, node, layer); err != nil {
		return err
	}
	copy := *layer.defaultValue
	copy.raw = copyValueRaw(copy.raw)
	node.constraints.defaultValue = &copy
	node.constraints.defaultSource = constraintSource(context, graph, layer)
	return nil
}

func effectiveDefaultLayer(node *valueOccurrenceNode) *valueOccurrenceNode {
	for _, layer := range effectiveConstraintLayers(node) {
		if layer.defaultValue != nil {
			return layer
		}
	}
	return nil
}

func resolveEffectiveDefault(context *ValueContext, graph *valueOccurrenceGraph, node, layer *valueOccurrenceNode) (ValueResult, error) {
	if layer.defaultValue.err != nil {
		return ValueResult{}, fmt.Errorf("%sinvalid authored default: %w", effectiveConstraintPrefix(node), layer.defaultValue.err)
	}
	result := resolveConstraintValue(context, graph, node, *layer.defaultValue, nil, false)
	if !constraintValueAdmitted(result) {
		return result, fmt.Errorf("%sdefault value %#v declared by %q violates the effective contract for %q",
			effectiveConstraintPrefix(node), layer.defaultValue.raw, effectiveLayerName(layer), effectiveLayerName(node))
	}
	return result, nil
}
