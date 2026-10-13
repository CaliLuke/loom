package expr

type (
	grpcPresencePair struct {
		local  *AttributeExpr
		source *AttributeExpr
	}
	grpcPresencePosition struct {
		grpcPresencePair
		objectField bool
	}
)

// unsupportedGRPCMappingPresence pairs authored transport positions with their
// source-owned types. It visits all local descendants even when finalization
// replaces their containing type, so authored presence is never silently lost.
func unsupportedGRPCMappingPresence(local, source *AttributeExpr, metadata, objectField, root bool, seen map[grpcPresencePair]bool) bool {
	if local == nil || local.Type == nil || local.Type == Empty {
		return false
	}
	pair := grpcPresencePair{local, source}
	if seen[pair] {
		return false
	}
	seen[pair] = true
	defer delete(seen, pair)
	effective := *local
	if source != nil && source.Type != nil && source.Type != Empty &&
		!(root && IsObject(local.Type) && !IsObject(source.Type)) {
		effective.Type = source.Type
		effective.Nullable = IsNullable(local) || IsNullable(source)
	}
	if metadata && AllowsNull(&effective) || !metadata && unsupportedGRPCPresenceOccurrence(&effective, objectField) {
		return true
	}
	if named, ok := local.Type.(UserType); ok {
		return unsupportedGRPCMappingPresence(named.Attribute(), source, metadata, objectField, root, seen)
	}
	for _, child := range grpcPresenceChildren(local, source, root && !metadata) {
		if unsupportedGRPCMappingPresence(child.local, child.source, metadata, child.objectField, false, seen) {
			return true
		}
	}
	return false
}

func grpcPresenceChildren(local, source *AttributeExpr, root bool) []grpcPresencePosition {
	var sourceType DataType
	if source != nil {
		sourceType = source.Type
	}
	if array := AsArray(local.Type); array != nil {
		var element *AttributeExpr
		if sourceArray := AsArray(sourceType); sourceArray != nil {
			element = sourceArray.ElemType
		}
		return []grpcPresencePosition{{grpcPresencePair: grpcPresencePair{array.ElemType, element}}}
	}
	if mapping := AsMap(local.Type); mapping != nil {
		var key, element *AttributeExpr
		if sourceMap := AsMap(sourceType); sourceMap != nil {
			key, element = sourceMap.KeyType, sourceMap.ElemType
		}
		return []grpcPresencePosition{
			{grpcPresencePair: grpcPresencePair{mapping.KeyType, key}},
			{grpcPresencePair: grpcPresencePair{mapping.ElemType, element}},
		}
	}
	if fields := AsObject(local.Type); fields != nil {
		return grpcPresenceObjectChildren(fields, source, root)
	}
	if union := AsUnion(local.Type); union != nil {
		return grpcPresenceUnionChildren(union, AsUnion(sourceType))
	}
	return nil
}

func grpcPresenceObjectChildren(fields *Object, source *AttributeExpr, root bool) []grpcPresencePosition {
	children := make([]grpcPresencePosition, 0, len(*fields))
	for _, field := range *fields {
		var attribute *AttributeExpr
		if source != nil {
			if IsObject(source.Type) {
				_, attribute = source.FindAttribute(field.Name)
			} else if root {
				// Primitive/array/map message roots have one authored wrapper
				// field whose type comes from the complete service occurrence.
				attribute = source
			}
		}
		children = append(children, grpcPresencePosition{
			grpcPresencePair: grpcPresencePair{field.Attribute, attribute}, objectField: true,
		})
	}
	return children
}

func grpcPresenceUnionChildren(local, source *Union) []grpcPresencePosition {
	children := make([]grpcPresencePosition, 0, len(local.Values))
	for _, branch := range local.Values {
		var attribute *AttributeExpr
		if source != nil {
			for _, candidate := range source.Values {
				if candidate.Name == branch.Name {
					attribute = candidate.Attribute
					break
				}
			}
		}
		children = append(children, grpcPresencePosition{grpcPresencePair: grpcPresencePair{branch.Attribute, attribute}})
	}
	return children
}
