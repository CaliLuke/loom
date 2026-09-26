package codegen

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestResultPayloadBodyLayouts checks that the body types of a result type
// used as a method result and as the element of an array payload keep the
// layout that each body type has when the service uses the result type only
// as a payload or only as a result. The request and response body types of
// the result type share its identifier, so a layout keyed by identifier gave
// the client request body types the layout of the response body types.
func TestResultPayloadBodyLayouts(t *testing.T) {
	combined := resultPayloadBodyTypeDefs(t, true, true)
	for _, isolated := range []map[string]string{
		resultPayloadBodyTypeDefs(t, true, false),
		resultPayloadBodyTypeDefs(t, false, true),
	} {
		for key, def := range isolated {
			got, ok := combined[key]
			if !assert.True(t, ok, "no %s type", key) {
				continue
			}
			assert.Equal(t, def, got, "%s", key)
		}
	}
}

// unionNameSuffix matches the number that the name scope appends to the
// second union type of a package, which only the combined design has.
var unionNameSuffix = regexp.MustCompile(`\bChoice\d+\b`)

// resultPayloadBodyTypeDefs returns the definitions of the body types of the
// service of resultPayloadDSL, keyed by side and type name, with the union
// names normalized.
func resultPayloadBodyTypeDefs(t *testing.T, payload, result bool) map[string]string {
	t.Helper()
	root := RunHTTPDSL(t, func() { resultPayloadDSL(payload, result) })
	sd := CreateHTTPServices(root).Get("svc")
	defs := make(map[string]string)
	add := func(side string, data *TypeData) {
		if data != nil {
			defs[side+" "+data.Name] = unionNameSuffix.ReplaceAllString(data.Def, "Choice")
		}
	}
	for _, data := range sd.ServerBodyAttributeTypes {
		add("server", data)
	}
	for _, data := range sd.ClientBodyAttributeTypes {
		add("client", data)
	}
	for _, endpoint := range sd.Endpoints {
		if endpoint.Payload.Request != nil {
			add("server", endpoint.Payload.Request.ServerBody)
			add("client", endpoint.Payload.Request.ClientBody)
		}
		if endpoint.Result == nil {
			continue
		}
		for _, response := range endpoint.Result.Responses {
			for _, data := range response.ServerBody {
				add("server", data)
			}
			add("client", response.ClientBody)
		}
	}
	return defs
}

// resultPayloadDSL defines a service whose methods take an array of a result
// type that holds a named union as payload, return the result type or a
// collection of it, or both.
func resultPayloadDSL(payload, result bool) {
	leaf, other := unionBodyBranchTypes()
	choice := Type("Choice", OneOf(leaf, other))
	rt := ResultType("application/vnd.rt", "RT", func() {
		Attribute("id", String)
		Attribute("c", choice)
		Attribute("d", choice)
		Required("d")
	})
	method := func(name string, res expr.DataType) {
		Method(name, func() {
			if payload {
				Payload(ArrayOf(rt))
			}
			if result {
				Result(res)
			}
			HTTP(func() {
				POST("/" + name)
			})
		})
	}
	Service("svc", func() {
		method("put", rt)
		method("puts", CollectionOf(rt))
	})
}
