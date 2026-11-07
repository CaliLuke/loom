package service

import (
	"fmt"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

type (
	// destinationTypeIdentity is one declaration in a physical generated package.
	// Import aliases and implicit-versus-explicit location spelling are not identity.
	destinationTypeIdentity struct {
		path string
		hash string
	}

	destinationTypeNames struct {
		packages map[string][]destinationTypeName
		names    map[destinationTypeIdentity]string
	}

	destinationTypeName struct {
		typ        expr.DataType
		name       string
		companions []string
	}
)

func destinationTypeFamily(policies []func(expr.DataType, string) []string) func(expr.DataType, string) []string {
	return func(typ expr.DataType, name string) []string {
		seen := make(map[string]bool)
		var result []string
		for _, policy := range policies {
			for _, companion := range policy(typ, name) {
				if !seen[companion] {
					seen[companion] = true
					result = append(result, companion)
				}
			}
		}
		sort.Strings(result)
		return result
	}
}

// configureDestinationTypes joins each service analysis to the generated
// destination owner. Import aliases and service analysis order never allocate
// another name for the same declaration. Types placed in this service's own
// package reserve their complete families before local nominal references.
func (d *ServicesData) configureDestinationTypes(service *expr.ServiceExpr, scope *codegen.NameScope) {
	if d.base != nil {
		d.destinationTypes = d.base.generatedDestinationTypes()
	} else {
		d.destinationTypes = d.generatedDestinationTypes()
	}
	owner := d.destinationTypes
	for _, record := range owner.packages[naming.ServiceDir(service.Name)] {
		name := scope.HashedUniqueFamily(record.typ, record.name, func(string) []string {
			return record.companions
		})
		if name != record.name {
			panic(fmt.Errorf("destination declaration %q conflicts with service package reservations", record.name))
		}
	}
	scope.SetTypeNames(&codegen.Location{RelImportPath: naming.ServiceDir(service.Name)}, owner.lookup)
}

func (d *ServicesData) generatedDestinationTypes() *destinationTypeNames {
	if d.destinationTypes != nil {
		return d.destinationTypes
	}
	owner := &destinationTypeNames{packages: make(map[string][]destinationTypeName), names: make(map[destinationTypeIdentity]string)}
	d.destinationTypes = owner
	var policies []func(expr.DataType, string) []string
	scopes := make(map[string]*codegen.NameScope)
	types := make(map[string]map[string]expr.DataType)
	for _, service := range d.Root.Services {
		policies = append(policies, serviceTypeFamilies(d.Root, service))
		// The same package-role inventory is used when its service is analyzed.
		scope, _, _, _ := newServiceScopes(d.Root, service, nil)
		d.serviceSymbols(service, scope)
		scopes[naming.ServiceDir(service.Name)] = scope
		attributes := append(serviceAttributes(d.Root, service), forcedTypeAttributes(d.Root, service)...)
		for _, method := range service.Methods {
			for _, requirement := range authorizationRequirements(method.Authorization) {
				attributes = append(attributes, requirement.Input)
			}
		}
		collector := destinationTypeCollector{types: types, seen: make(map[destinationVisit]bool)}
		for _, attribute := range attributes {
			collector.walk(attribute, nil, false)
		}
	}
	family := destinationTypeFamily(policies)
	for path, byHash := range types {
		owner.allocate(path, byHash, scopes[path], family)
	}
	return owner
}

func (d *destinationTypeNames) lookup(typ expr.DataType, location *codegen.Location) (string, bool) {
	if location == nil {
		return "", false
	}
	name, found := d.names[destinationTypeIdentity{path: location.RelImportPath, hash: typ.Hash()}]
	return name, found
}

func (d *destinationTypeNames) allocate(path string, byHash map[string]expr.DataType, scope *codegen.NameScope, family func(expr.DataType, string) []string) {
	if scope == nil {
		scope = codegen.NewNameScope()
	}
	ordered := make([]expr.DataType, 0, len(byHash))
	for _, typ := range byHash {
		ordered = append(ordered, typ)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := expr.AsUnion(ordered[i]) != nil, expr.AsUnion(ordered[j]) != nil
		if left != right {
			return !left
		}
		return ordered[i].Hash() < ordered[j].Hash()
	})
	for _, typ := range ordered {
		name := scope.HashedUniqueFamily(typ, codegen.Goify(typ.Name(), true), func(candidate string) []string {
			return family(typ, candidate)
		})
		d.names[destinationTypeIdentity{path: path, hash: typ.Hash()}] = name
		d.packages[path] = append(d.packages[path], destinationTypeName{typ: typ, name: name, companions: family(typ, name)})
	}
}
