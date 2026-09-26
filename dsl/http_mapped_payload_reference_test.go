package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestTransportNamesSuffixedPayloadAttributes checks that Header, Param,
// Cookie, Body and a route wildcard select a payload attribute declared with
// an element name suffix, such as "tok:t", by its attribute name. The mapped
// attribute inherits the type, validations and requiredness of the payload
// attribute, and its transport element is named by the mapping, not by the
// payload suffix, which names only the field of a body.
func TestTransportNamesSuffixedPayloadAttributes(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("svc", func() {
			Method("m", func() {
				Payload(func() {
					Attribute("id:i", Int)
					Attribute("tok:t", String, func() {
						MinLength(2)
					})
					Attribute("ver:v", String)
					Attribute("q:qq", String)
					Attribute("sid:s", String)
					Attribute("name:nm", String)
					Required("id:i", "tok:t", "sid:s")
				})
				HTTP(func() {
					POST("/{id}")
					Header("tok")
					Header("ver:X-Version")
					Param("q")
					Cookie("sid")
				})
			})
		})
	})
	e := root.API.HTTP.Service("svc").Endpoint("m")

	assert.Equal(t, "tok", e.Headers.ElemName("tok"))
	assert.Equal(t, "X-Version", e.Headers.ElemName("ver"))
	tok := expr.AsObject(e.Headers.Type).Attribute("tok")
	require.NotNil(t, tok)
	assert.Equal(t, expr.String, tok.Type)
	require.NotNil(t, tok.Validation)
	require.NotNil(t, tok.Validation.MinLength)
	assert.Equal(t, 2, *tok.Validation.MinLength)
	assert.True(t, e.Headers.IsRequired("tok"))
	assert.False(t, e.Headers.IsRequired("ver"))

	path := e.PathParams()
	assert.Equal(t, []string{"id"}, mappedNames(path))
	assert.Equal(t, expr.Int, expr.AsObject(path.Type).Attribute("id").Type)
	assert.True(t, path.IsRequired("id"))
	query := e.QueryParams()
	assert.Equal(t, []string{"q"}, mappedNames(query))
	assert.Equal(t, "q", query.ElemName("q"))

	assert.Equal(t, "sid", e.Cookies.ElemName("sid"))
	assert.True(t, e.Cookies.IsRequired("sid"))

	body := expr.NewMappedAttributeExpr(e.Body)
	assert.Equal(t, []string{"name"}, mappedNames(body))
	assert.Equal(t, "nm", body.ElemName("name"))
}

// TestBodyNamesSuffixedAttributes checks that Body selects a payload, result
// or error attribute declared with an element name suffix by its attribute
// name, both with an attribute name argument and in a Body function, and that
// response and error headers and cookies do the same.
func TestBodyNamesSuffixedAttributes(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Fault = Type("Fault", func() {
			Attribute("code:c", String)
			Attribute("msg:m", String)
			Required("code:c")
		})
		Service("svc", func() {
			Method("create", func() {
				Payload(func() {
					Attribute("id:i", Int)
					Attribute("body:b", MapOf(String, Int))
					Required("body:b")
				})
				Result(func() {
					Attribute("loc:l", String)
					Attribute("sid:s", String)
					Attribute("data:d", ArrayOf(String))
					Required("loc:l", "data:d")
				})
				Error("bad", Fault)
				HTTP(func() {
					POST("/{id}")
					Body("body")
					Response(StatusOK, func() {
						Header("loc:Location")
						Cookie("sid")
						Body("data")
					})
					Response("bad", StatusBadRequest, func() {
						Header("code:X-Code")
					})
				})
			})
			Method("update", func() {
				Payload(func() {
					Attribute("name:nm", String, func() {
						MinLength(2)
					})
					Attribute("age:ag", Int)
					Required("name:nm")
				})
				Result(func() {
					Attribute("name:nm", String)
					Attribute("age:ag", Int)
					Required("name:nm")
				})
				HTTP(func() {
					PUT("/")
					Body(func() {
						Attribute("name")
						Attribute("age:a")
						Required("name")
					})
					Response(StatusOK, func() {
						Body(func() {
							Attribute("name")
							Attribute("age:a")
						})
					})
				})
			})
		})
	})
	svc := root.API.HTTP.Service("svc")

	create := svc.Endpoint("create")
	assert.Equal(t, expr.AsMap(create.Body.Type) != nil, true, "the body is the map attribute")
	response := create.Responses[0]
	assert.True(t, expr.IsArray(response.Body.Type), "the response body is the array attribute")
	assert.Equal(t, "Location", response.Headers.ElemName("loc"))
	assert.True(t, response.Headers.IsRequired("loc"))
	assert.Equal(t, expr.String, expr.AsObject(response.Headers.Type).Attribute("loc").Type)
	require.Len(t, response.Cookies, 1)
	assert.Equal(t, "sid", response.Cookies[0].AttributeName())
	errHeaders := create.HTTPErrors[0].Response.Headers
	assert.Equal(t, "X-Code", errHeaders.ElemName("code"))
	assert.True(t, errHeaders.IsRequired("code"))

	update := svc.Endpoint("update")
	body := expr.NewMappedAttributeExpr(update.Body)
	assert.Equal(t, []string{"name", "age"}, mappedNames(body))
	assert.Equal(t, "name", body.ElemName("name"))
	assert.Equal(t, "a", body.ElemName("age"))
	assert.True(t, body.IsRequired("name"))
	name := expr.AsObject(body.Type).Attribute("name")
	assert.Equal(t, expr.String, name.Type)
	require.NotNil(t, name.Validation)
	assert.Equal(t, 2, *name.Validation.MinLength)
	assert.Equal(t, expr.Int, expr.AsObject(body.Type).Attribute("age").Type)
	resp := expr.NewMappedAttributeExpr(update.Responses[0].Body)
	assert.True(t, resp.IsRequired("name"))
	assert.Equal(t, expr.Int, expr.AsObject(resp.Type).Attribute("age").Type)
}
