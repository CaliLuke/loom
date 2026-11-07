package unionnames

import "github.com/CaliLuke/loom/expr"

type (
	// PackageSymbols identifies the package declarations accompanying one union.
	// Its primary type is named by the caller's owning NameScope.
	PackageSymbols struct {
		// KindName is the named type of the union discriminator constants.
		KindName string
		// Branches maps authored branch names to their package declarations.
		Branches map[string]BranchSymbols
		// Names lists every companion declaration, excluding the primary union type.
		// Order follows the authored branches; names use the resolved member identity.
		Names []string
	}

	// BranchSymbols identifies a branch's declarations in its enclosing package.
	BranchSymbols struct {
		// KindConst identifies this branch in the union's discriminator type.
		KindConst string
		// Constructor builds a union value with this branch selected.
		Constructor string
	}
)

// PackageDeclarations derives a union's complete common package declaration
// inventory from its allocated primary name. Allocation, emitters and value
// construction consume this same policy, including sanitized member names.
// Owner-specific declarations, such as native Any aliases, are added by the
// domain that emits them before it allocates the complete family.
func PackageDeclarations(union *expr.Union, primary string) PackageSymbols {
	symbols := PackageSymbols{
		KindName: primary + "Kind",
		Branches: make(map[string]BranchSymbols, len(union.Values)),
	}
	symbols.Names = append(symbols.Names, symbols.KindName)
	members := Resolve(union)
	for _, branch := range union.Values {
		method := members[branch.Name].Method
		names := BranchDeclarations(primary, method)
		symbols.Branches[branch.Name] = names
		symbols.Names = append(symbols.Names, names.KindConst, names.Constructor)
	}
	return symbols
}

// BranchDeclarations derives a branch's package identifiers from an allocated
// union name and the accessor suffix returned by Resolve. Emitters that already
// hold resolved member data use this same policy as the full family inventory.
func BranchDeclarations(primary, method string) BranchSymbols {
	return BranchSymbols{
		KindConst:   primary + "Kind" + method,
		Constructor: "New" + primary + method,
	}
}
