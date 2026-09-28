package testdata

import . "github.com/CaliLuke/loom/dsl"

// UnionResponseBodyDSL selects union fields as success and custom error bodies,
// covering field presence, named unions, mapped names, and viewed results.
var UnionResponseBodyDSL = func() {
	API("unionresponse", func() {})
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
		attributes := func() {
			key := "choice"
			if c.mapped {
				key = "choice:wire_choice"
			}
			choice := OneOf(leaf, other)
			if c.named {
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
		Service(c.name, func() {
			Method("show", func() {
				Result(result)
				Error("bad", envelope)
				HTTP(func() {
					GET("/" + c.name)
					Response(StatusOK, func() {
						Body("choice")
					})
					Response("bad", StatusBadRequest, func() {
						Body("choice")
					})
				})
			})
		})
	}
}
