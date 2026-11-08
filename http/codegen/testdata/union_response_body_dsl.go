package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// UnionResponseBodyDSL selects union fields as success and custom error bodies,
// covering field presence, named unions, mapped names, and viewed results.
var UnionResponseBodyDSL = func() {
	API("unionresponse", func() {})
	unionResponseBodyDSL(false)
}

// UnionResponseStatusDSL selects a union at known and extension statuses.
// It omits API metadata so transport specimens can compose it with other services.
var UnionResponseStatusDSL = func() {
	unionResponseBodyDSL(true)
}

func unionResponseBodyDSL(statuses bool) {
	leaf := Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	other := Type("Other", func() {
		Attribute("count", Int)
		Required("count")
	})
	named := Type("NamedChoice", OneOf(leaf, other))
	for _, c := range []struct {
		name     string
		required bool
		nullable bool
		named    bool
		mapped   bool
		viewed   bool
	}{
		{name: "required", required: true},
		{name: "optional"},
		{name: "required_nullable", required: true, nullable: true},
		{name: "optional_nullable", nullable: true},
		{name: "required_named", required: true, named: true},
		{name: "optional_named", named: true},
		{name: "required_named_nullable", required: true, nullable: true, named: true},
		{name: "optional_named_nullable", nullable: true, named: true},
		{name: "required_mapped", required: true, mapped: true},
		{name: "optional_mapped", mapped: true},
		{name: "required_viewed", required: true, viewed: true},
		{name: "optional_viewed", viewed: true},
	} {
		if statuses && c.name != "required_named" {
			continue
		}
		attributes := func() {
			if statuses {
				Attribute("kind", String)
			}
			key := "choice"
			if c.mapped {
				key = "choice:wire_choice"
			}
			choice := OneOf(leaf, other)
			if c.named && !statuses {
				choice = named
			}
			Attribute(key, choice, func() {
				if c.nullable {
					Nullable()
				}
			})
			if c.required {
				Required(key)
			}
		}
		envelope := Type(c.name+"_envelope", attributes)
		var result any = envelope
		if c.viewed {
			result = ResultType("application/vnd."+c.name, func() {
				attributes()
				View("default", func() {
					Attribute("choice")
				})
			})
		}
		unionResponseService(c.name, result, envelope, statuses)
	}
}

func unionResponseService(name string, result any, envelope expr.UserType, statuses bool) {
	Service(name, func() {
		Method("show", func() {
			Result(result)
			Error("bad", envelope)
			HTTP(func() {
				GET("/" + name)
				Response(StatusOK, func() {
					if statuses {
						Tag("kind", "known")
					}
					Body("choice")
				})
				if statuses {
					Response(299, func() {
						Body("choice")
					})
				}
				Response("bad", StatusBadRequest, func() {
					Body("choice")
				})
			})
		})
	})
}
