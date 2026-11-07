package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
)

type nativeTypeSymbols struct {
	union   unionnames.PackageSymbols
	aliases map[string]string
	names   []string
}

// serviceTypeFamilies inventories the validators actually emitted for branches
// of untagged unions. Nominal allocation remains lazy in NameScope, so references
// and declarations claim each complete family at their first use.
func serviceTypeFamilies(root *expr.RootExpr, service *expr.ServiceExpr) func(expr.DataType, string) []string {
	validated := make(map[string]bool)
	attributes := append(serviceAttributes(root, service), forcedTypeAttributes(root, service)...)
	for _, attribute := range attributes {
		err := codegen.Walk(attribute, func(att *expr.AttributeExpr) error {
			union, ok := att.Type.(*expr.Union)
			if !ok || !union.Untagged {
				return nil
			}
			return codegen.Walk(att, func(branch *expr.AttributeExpr) error {
				if typ, named := branch.Type.(expr.UserType); named && expr.AsUnion(typ) == nil && (expr.IsObject(typ) || expr.IsArray(typ) || expr.IsMap(typ)) {
					validated[typ.Hash()] = true
				}
				return nil
			})
		})
		if err != nil {
			panic(codegen.NewError(nil, attribute, err))
		}
	}
	return func(typ expr.DataType, name string) []string {
		var validators []string
		if validated[typ.Hash()] {
			validators = append(validators, "Validate"+name)
		}
		return nativeTypeDeclarations(typ, name, validators).names
	}
}

// viewTypeFamilies includes each projected declaration's validators. View
// names are finalized by projection before NameScope claims their identities.
func viewTypeFamilies(typ expr.DataType, name string) []string {
	return nativeTypeDeclarations(typ, name, viewValidatorNames(typ, name)).names
}

func viewValidatorNames(typ expr.DataType, name string) []string {
	var names []string
	if result, ok := typ.(*expr.ResultTypeExpr); ok {
		suffixes := viewHelperSuffixes(result)
		for _, view := range result.Views {
			names = append(names, "Validate"+name+suffixes[view.Name])
		}
	} else if _, named := typ.(expr.UserType); named {
		names = append(names, "Validate"+name)
	}
	return names
}

// nativeTypeDeclarations owns native-only aliases alongside the common union
// declarations. Reserve all available natural aliases before suffixing collisions
// so a displaced alias cannot steal another branch's natural name.
func nativeTypeDeclarations(typ expr.DataType, name string, validators []string) nativeTypeSymbols {
	symbols := nativeTypeSymbols{names: append([]string(nil), validators...)}
	union := expr.AsUnion(typ)
	if union == nil {
		return symbols
	}
	symbols.union = unionnames.PackageDeclarations(union, name)
	symbols.names = append(symbols.names, symbols.union.Names...)
	symbols.aliases = make(map[string]string)
	local := codegen.NewNameScope()
	local.Unique(name)
	for _, companion := range symbols.names {
		local.Unique(companion)
	}
	members := unionnames.Resolve(union)
	var deferred []*expr.NamedAttributeExpr
	for _, branch := range sortedNamedAttributes(union.Values) {
		if _, named := branch.Attribute.Type.(expr.UserType); named {
			continue
		}
		if primitive, ok := primitiveAliasGoType(branch.Attribute.Type); !ok || primitive != "loom.JSONValue" {
			continue
		}
		preferred := name + members[branch.Name].Method
		if local.PeekUnique(preferred) != preferred {
			deferred = append(deferred, branch)
			continue
		}
		symbols.aliases[branch.Name] = local.Unique(preferred)
	}
	for _, branch := range deferred {
		symbols.aliases[branch.Name] = local.Unique(name + members[branch.Name].Method)
	}
	for _, branch := range sortedNamedAttributes(union.Values) {
		if alias := symbols.aliases[branch.Name]; alias != "" {
			symbols.names = append(symbols.names, alias)
		}
	}
	return symbols
}
