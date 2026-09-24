package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var GRPCEndpointWithUnionContainingAny = func() {
	var AnyAlias = Type("AnyAlias", func() {
		Attribute("value", Any)
	})

	var U = Type("U", func() {
		OneOf("choice", func() {
			Field(1, "plain_any", Any)
			Field(2, "array_any", ArrayOf(AnyAlias))
			Field(3, "map_any", MapOf(String, AnyAlias))
		})
	})

	Service("Service", func() {
		Method("MethodUnion", func() {
			Payload(U)
			GRPC(func() {})
		})
	})
}

// GRPCEndpointWithConstructorUnionField declares a constructor OneOf passed
// to Field. Its branches take the numbers 2 and 3 after the field number, so
// the next field uses number 4.
var GRPCEndpointWithConstructorUnionField = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "id", String)
				Field(2, "pick", OneOf(Leaf, Other))
				Field(4, "next", String)
			})
			GRPC(func() {})
		})
	})
}

// GRPCEndpointWithConstructorUnionFieldCollision declares a field whose
// number is also the number of the second branch of a constructor OneOf
// passed to Field.
var GRPCEndpointWithConstructorUnionFieldCollision = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(2, "pick", OneOf(Leaf, Other))
				Field(3, "next", String)
			})
			GRPC(func() {})
		})
	})
}

// GRPCEndpointWithUntaggedUnionBranches declares a constructor OneOf passed
// to Attribute and a OneOf block whose branches are defined with Attribute,
// so no union branch has a field number.
var GRPCEndpointWithUntaggedUnionBranches = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "id", String)
				Attribute("pick", OneOf(Leaf, Other))
				OneOf("mode", func() {
					Attribute("fast", String)
					Field(3, "slow", Int)
				})
			})
			GRPC(func() {})
		})
	})
}

// GRPCEndpointWithUnionCollections uses a named union and a constructor
// OneOf as array elements, a named union as map values, and a named union as
// array elements in a nested type. A protocol buffer oneof cannot be repeated
// or used as a map value.
var GRPCEndpointWithUnionCollections = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	var Bag = Type("Bag", func() {
		Field(1, "items", ArrayOf(Choice))
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "choices", ArrayOf(Choice))
				Field(2, "picks", ArrayOf(OneOf(Leaf, Other)))
			})
			Result(func() {
				Field(1, "by_name", MapOf(String, Choice))
				Field(2, "bag", Bag)
			})
			GRPC(func() {})
		})
	})
}

// GRPCEndpointWithNamedUnionField passes a named union to Field. Its
// branches take the numbers 2 and 3 after the field number, so the next field
// uses number 4.
var GRPCEndpointWithNamedUnionField = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "id", String)
				Field(2, "pick", Choice)
				Field(4, "next", String)
			})
			GRPC(func() {})
		})
	})
}
