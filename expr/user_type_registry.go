package expr

import "sort"

type (
	// UserTypeRegistry stores named temporary user types for synthesized
	// expression trees and returns them in deterministic order.
	UserTypeRegistry struct {
		types map[string]*UserTypeExpr
	}
)

// NewUserTypeRegistry creates an empty registry for temporary named user types.
func NewUserTypeRegistry() *UserTypeRegistry {
	return &UserTypeRegistry{types: make(map[string]*UserTypeExpr)}
}

// GetOrCreate returns the named user type, creating it with builder on the
// first call only.
func (r *UserTypeRegistry) GetOrCreate(name string, builder func() *AttributeExpr) *UserTypeExpr {
	if t, ok := r.types[name]; ok {
		return t
	}
	t := &UserTypeExpr{
		TypeName:      name,
		AttributeExpr: builder(),
	}
	r.types[name] = t
	return t
}

// Attribute returns an attribute that references the named user type.
func (r *UserTypeRegistry) Attribute(name string, builder func() *AttributeExpr) *AttributeExpr {
	return &AttributeExpr{Type: r.GetOrCreate(name, builder)}
}

// Collect returns all registered user types sorted by name.
func (r *UserTypeRegistry) Collect() []UserType {
	keys := make([]string, 0, len(r.types))
	for k := range r.types {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	types := make([]UserType, 0, len(keys))
	for _, key := range keys {
		types = append(types, r.types[key])
	}
	return types
}
