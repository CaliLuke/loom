package codegen

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

type nameFamilies struct {
	members    map[string][]string
	typePolicy func(expr.DataType, string) []string
}

// SetTypeFamilies installs the owning package's companion declaration policy.
// Local user types and unions allocate their complete family on the first type
// reference, including types synthesized during recursive analysis. Qualified
// external declarations retain their external ownership and are not allocated
// by this policy. NewNameScopeLike does not inherit it.
//
// Install the policy once, before resolving local type references. The callback
// must be deterministic and must not allocate names or resolve type references
// in this scope. It returns identifiers derived from the candidate primary name,
// excluding that primary. NameScope owns collision handling and cached identity.
func (s *NameScope) SetTypeFamilies(companions func(expr.DataType, string) []string) {
	if companions == nil || s.families.typePolicy != nil {
		panic("type declaration family policy must be configured exactly once")
	}
	s.families.typePolicy = companions
}

// SetTypeNames installs a read-only resolver for declarations allocated by their
// physical package inventory, including forced declarations in this scope's own
// package. It returns false for locally allocated or opaque imported Go types.
// Resolved names are used consistently for
// qualified and unqualified references and are not allocated in this scope.
// NewNameScopeLike and WithoutPackageNames preserve this resolver. Configure it
// before any type reference; the resolver must not mutate a naming scope.
func (s *NameScope) SetTypeNames(location *Location, resolve func(expr.DataType, *Location) (string, bool)) {
	if resolve == nil || s.typeNames != nil {
		panic("destination type names must be configured exactly once")
	}
	s.typeNames = resolve
	s.typeLocation = location
}

// WithTypeNameLocation returns a scope sharing all naming state while resolving
// bare declarations in location. Named user types retain their explicit location.
// This is used while rendering a relocated definition containing inline unions;
// import aliases do not identify a physical declaration namespace.
func (s *NameScope) WithTypeNameLocation(location *Location) *NameScope {
	if location == nil {
		return s
	}
	copy := *s
	copy.typeLocation = location
	return &copy
}

// HashedUniqueFamily allocates a primary declaration and its companion package
// declarations atomically. companions returns the identifiers derived from each
// candidate primary name, excluding that primary. The first collision-free
// family wins; every later use of key retains that allocation.
//
// Callers must register families before rendering references. An already cached
// primary may acquire a family only while its companions are unoccupied. A late
// collision or a changed companion inventory panics rather than changing names
// already embedded in references. Methods and fields need separate scopes.
func (s *NameScope) HashedUniqueFamily(key Hasher, name string, companions func(string) []string) string {
	hash := key.Hash()
	if allocated, exists := s.names[hash]; exists {
		family := declarationFamily(allocated, companions)
		if previous, registered := s.families.members[hash]; registered {
			if !slices.Equal(previous, family) {
				panic(fmt.Errorf("declaration family for %q changed after allocation", allocated))
			}
			return allocated
		}
		for _, member := range family {
			if member != allocated && s.counts[member] != 0 {
				panic(fmt.Errorf("declaration family for %q registered after %q was occupied", allocated, member))
			}
		}
		for _, member := range family {
			if member != allocated {
				s.counts[member] = 1
			}
		}
		s.families.members[hash] = family
		return allocated
	}
	preferred := name
	for suffix := 2; ; suffix++ {
		family := declarationFamily(name, companions)
		available := true
		for _, member := range family {
			if s.counts[member] != 0 {
				available = false
				break
			}
		}
		if available {
			for _, member := range family {
				s.counts[member] = 1
			}
			s.names[hash] = name
			s.families.members[hash] = family
			return name
		}
		name = preferred + strconv.Itoa(suffix)
	}
}

func declarationFamily(primary string, companions func(string) []string) []string {
	family := append([]string{primary}, companions(primary)...)
	slices.Sort(family)
	for index, name := range family {
		if name == "" || index > 0 && name == family[index-1] {
			panic(fmt.Errorf("invalid declaration family for %q: duplicate or empty identifier %q", primary, name))
		}
	}
	return family
}

// goNominalValueTypeName resolves the physical owner before allocating local names.
func (s *NameScope) goNominalValueTypeName(actual expr.DataType, pkg string) string {
	if expr.IsDefaultErrorResult(actual) {
		return "loom.ServiceError"
	}
	// Qualified type references (pkg.Type) do not compete in the local
	// identifier namespace.
	//
	// When generating qualified references, we must not blindly apply local
	// scoping (suffixing) to the referenced type name, otherwise we can emit
	// identifiers that do not exist in the referenced package (e.g., pkg.Foo2).
	//
	// However, when the scope already assigned a unique name to the referenced
	// type (i.e., the type is defined in this scope and got suffixed due to a
	// collision), qualified references must use that assigned name to stay
	// consistent across packages. This is critical for transport packages that
	// refer to types defined in the service package (e.g., grpc referencing a
	// payload type defined as Request2).
	if s.typeNames != nil {
		location := UserTypeLocation(actual)
		if location == nil {
			location = s.typeLocation
		}
		if name, ok := s.typeNames(actual, location); ok {
			if pkg != "" {
				return pkg + "." + name
			}
			return name
		}
	}
	base := Goify(actual.Name(), true)
	if pkg == "" {
		var name string
		if s.families.typePolicy != nil && UserTypeLocation(actual) == nil {
			name = s.HashedUniqueFamily(actual, base, func(name string) []string {
				return s.families.typePolicy(actual, name)
			})
		} else {
			name = s.HashedUnique(actual, base, "")
		}
		// Keep reservations that protect generated helpers, but declare
		// relocated types by the canonical name used by qualified references.
		if UserTypeLocation(actual) != nil {
			return base
		}
		return name
	}
	if UserTypeLocation(actual) == nil {
		if n, ok := s.names[actual.Hash()]; ok {
			return pkg + "." + n
		}
	}
	return pkg + "." + base
}
