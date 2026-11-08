package expr

import "slices"

// HTTPBodyProjection records how an HTTP body occurrence projects its authored
// source. The source keeps the authored identity that lowering flattens, such
// as composition-reference bases and discriminator bindings, separately from
// the members the body physically selects.
type HTTPBodyProjection struct {
	// Source is the authored attribute the body carries before flattening:
	// the attribute of the named payload, result or Body declaration, or the
	// inline object occurrence. It is shared and must not be mutated.
	Source *AttributeExpr
	// Excluded lists the authored member names (AttributeName) excluded by
	// the original body occurrence, in removal order. Contributors retain the
	// complete selection even when their own declaration lacks a member: that
	// member may belong to a subsequently published child or sibling.
	Excluded []string
}

// DiscriminatorMember binds property, a literal schema property name of a
// discriminator hint, to the member of source whose JSON wire name it is. It
// reads the authored source before any transport renames its members and
// returns the member's AttributeName.
func DiscriminatorMember(source *AttributeExpr, property string) (string, bool) {
	object := AsObject(source.Type)
	if object == nil {
		return "", false
	}
	for _, member := range *object {
		if JSONFieldName(ElementName(member.Name), member.Attribute) == property {
			return AttributeName(member.Name), true
		}
	}
	return "", false
}

// ProjectHTTPBodyContributor projects the own contract of declaration through
// projection: the declaration as an HTTP body occurrence without the members
// the projection excluded. The result keeps the declaration's own members,
// requiredness, defaults, examples, openness and metadata, records its own
// projection and names the declaration, so that composition and hints that
// the declaration authors follow the same projection.
func ProjectHTTPBodyContributor(declaration UserType, projection *HTTPBodyProjection) *AttributeExpr {
	body := projectHTTPBody(&AttributeExpr{Type: declaration})
	for _, name := range projection.Excluded {
		if body.Type.(*Object).Attribute(name) != nil {
			removeAttribute(body, name)
		}
	}
	// Local deletion must not narrow the context forwarded to publication.
	// Source remains this declaration for its own composition and hint binding.
	body.bodyProjection.Excluded = slices.Clone(projection.Excluded)
	att := body.Attribute()
	CopyObjectOpenness(att, declaration.Attribute())
	for _, key := range []string{"openapi:typename", "openapi:typename:canonical"} {
		if values, ok := declaration.Attribute().Meta[key]; ok {
			att.AddMeta(key, values...)
		}
	}
	return &AttributeExpr{
		Type:         &UserTypeExpr{AttributeExpr: att, TypeName: declaration.Name(), UID: declaration.ID()},
		Validation:   att.Validation,
		UserExamples: att.UserExamples,
	}
}

// HTTPBodyProjection returns the projection recorded on the HTTP body object
// that a refers to, following named body wrappers, or nil. The result is
// detached from the record.
func (a *AttributeExpr) HTTPBodyProjection() *HTTPBodyProjection {
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		if a.bodyProjection != nil {
			return &HTTPBodyProjection{Source: a.bodyProjection.Source, Excluded: slices.Clone(a.bodyProjection.Excluded)}
		}
		named, ok := a.Type.(UserType)
		if !ok {
			return nil
		}
		a = named.Attribute()
	}
	return nil
}

// CompositionReferenceBases returns the composition bases that the schema of a
// references through allOf: its authored composition bases when a asks for
// reference identity with the openapi:allOf:reference metadata.
func (a *AttributeExpr) CompositionReferenceBases() []DataType {
	if value, ok := a.Meta.Last("openapi:allOf:reference"); !ok || value == "false" {
		return nil
	}
	return a.CompositionBases()
}

// projectHTTPBody flattens source into a body object and records the
// projection. Callers then remove the members that their bindings own; the
// removals extend the record.
func projectHTTPBody(source *AttributeExpr) *MappedAttributeExpr {
	body := NewMappedAttributeExpr(source)
	body.Meta = HTTPBodyProjectionMetadata(source)
	RemovePkgPath(body.AttributeExpr)
	inheritBodyReferences(body)
	body.bodyProjection = &HTTPBodyProjection{Source: httpBodyProjectionSource(source)}
	return body
}

// recordHTTPBodyProjection records that the explicit body body projects source
// without removing members. The record belongs to the owned object attribute
// that body refers to.
func recordHTTPBodyProjection(body, source *AttributeExpr) {
	seen := make(map[*AttributeExpr]bool)
	for body != nil && !seen[body] {
		seen[body] = true
		if _, ok := body.Type.(*Object); ok {
			body.bodyProjection = &HTTPBodyProjection{Source: httpBodyProjectionSource(source)}
			return
		}
		named, ok := body.Type.(UserType)
		if !ok {
			return
		}
		body = named.Attribute()
	}
}

// httpBodyProjectionSource returns the authored attribute that owns the
// composition and hints of a body built from source: the attribute of the
// first named declaration, or source itself when it is inline.
func httpBodyProjectionSource(source *AttributeExpr) *AttributeExpr {
	if named, ok := source.Type.(UserType); ok {
		return named.Attribute()
	}
	return source
}
