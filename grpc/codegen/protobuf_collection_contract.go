package codegen

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

type (
	collectionMessageContract struct {
		name     string
		contents collectionValidationContract
	}

	// collectionValidationContract captures owned occurrence facts without
	// retaining mutable expression graphs. Named edges keep their declaration
	// identity and occurrence facts; their intrinsic children belong to that
	// declaration, which may still be under recursive normalization.
	collectionValidationContract struct {
		kind             expr.Kind
		declaration      string
		validation       *expr.ValidationExpr
		clauses          []expr.EffectiveValidationClause
		defaultValue     any
		hasDefault       bool
		nullable         bool
		nonNullableElems bool
		members          []string
		children         []collectionValidationContract
	}
)

func collectionContract(att *expr.AttributeExpr) collectionValidationContract {
	constraints, err := expr.EffectiveConstraintsFor(att)
	if err != nil {
		panic(fmt.Sprintf("invalid anonymous protobuf collection constraints: %v", err))
	}
	validation := constraints.Validation()
	contract := collectionValidationContract{
		kind:       att.Type.Kind(),
		validation: validation.Lowered(),
		clauses:    validation.Clauses(),
		nullable:   expr.IsNullable(att),
	}
	for i := range contract.clauses {
		contract.clauses[i].Provenance = expr.EffectiveConstraintSource{}
	}
	contract.defaultValue, contract.hasDefault = constraints.Default()
	switch dt := att.Type.(type) {
	case expr.UserType:
		contract.declaration = dt.ID()
	case *expr.Array:
		contract.nonNullableElems = dt.NonNullableElems
		contract.children = []collectionValidationContract{collectionContract(dt.ElemType)}
	case *expr.Map:
		contract.children = []collectionValidationContract{collectionContract(dt.KeyType), collectionContract(dt.ElemType)}
	case *expr.Object:
		for _, member := range *dt {
			contract.members = append(contract.members, member.Name)
			contract.children = append(contract.children, collectionContract(member.Attribute))
		}
	case *expr.Union:
		for _, member := range dt.Values {
			contract.members = append(contract.members, member.Name)
			contract.children = append(contract.children, collectionContract(member.Attribute))
		}
	}
	return contract
}

func (sd *ServiceData) collectionMessageName(att *expr.AttributeExpr, name string) string {
	shape := strconv.Quote(name) + ":" + protoAttributeShape(wrapperAttribute(att, true))
	contract := collectionContract(att)
	for _, previous := range sd.collectionMessages[shape] {
		if reflect.DeepEqual(previous.contents, contract) {
			return previous.name
		}
	}
	if sd.collectionMessages == nil {
		sd.collectionMessages = make(map[string][]collectionMessageContract)
	}
	owner := "collection:" + shape + ":contract:" + strconv.Itoa(len(sd.collectionMessages[shape]))
	name = sd.anonymousMessageName(messageScope{name: name, path: owner})
	sd.collectionMessages[shape] = append(sd.collectionMessages[shape], collectionMessageContract{name: name, contents: contract})
	return name
}
