package dsl_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestExplicitBodyElementNameInheritsPayload checks that an attribute of an
// explicit Body declared with an element name suffix, such as "name:n",
// inherits the payload or result attribute of the same attribute name as an
// attribute declared without the suffix does: its type, description and
// validations, and its requiredness when the body inherits the required
// attributes of a user type or is a response body. The body keeps the
// suffixed keys and names its required attributes after them.
func TestExplicitBodyElementNameInheritsPayload(t *testing.T) {
	cases := map[string]struct {
		userType bool
		response bool
	}{
		"inline payload":   {},
		"user type":        {userType: true},
		"inline result":    {response: true},
		"user type result": {userType: true, response: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			plain := explicitBody(t, explicitBodyDSL(tc.userType, tc.response, "name", "age"), tc.response)
			mapped := explicitBody(t, explicitBodyDSL(tc.userType, tc.response, "name:n", "age:ag"), tc.response)

			assert.Equal(t, []string{"name:n", "age:ag"}, attributeKeys(mapped))
			for _, pair := range [][2]string{{"name", "name:n"}, {"age", "age:ag"}} {
				want, got := bodyChild(t, plain, pair[0]), bodyChild(t, mapped, pair[1])
				assert.Equal(t, want.Type, got.Type, pair[1])
				assert.Equal(t, want.Description, got.Description, pair[1])
				assert.Equal(t, want.Validation, got.Validation, pair[1])
			}
			assert.Equal(t, expr.String, bodyChild(t, mapped, "name:n").Type)
			assert.Equal(t, expr.Int, bodyChild(t, mapped, "age:ag").Type)
			require.NotNil(t, bodyChild(t, mapped, "name:n").Validation)
			assert.Equal(t, 2, *bodyChild(t, mapped, "name:n").Validation.MinLength)

			plainMapped, mappedMapped := expr.NewMappedAttributeExpr(plain), expr.NewMappedAttributeExpr(mapped)
			for _, att := range []string{"name", "age"} {
				assert.Equal(t, plainMapped.IsRequired(att), mappedMapped.IsRequired(att), att)
			}
			assert.Equal(t, tc.userType || tc.response, mappedMapped.IsRequired("name"))
			assert.Equal(t, "n", mappedMapped.ElemName("name"))
			assert.Equal(t, "ag", mappedMapped.ElemName("age"))
			if tc.userType || tc.response {
				assert.Contains(t, bodyRequired(mapped), "name:n", "required names use the body keys")
				assert.NotContains(t, bodyRequired(mapped), "name")
			}
		})
	}
}

// TestExplicitBodyElementNameRequired checks that a Required list of an
// explicit Body that names attributes with their element name suffix is
// compared with the required attributes of the payload by attribute name, and
// that it must name such an attribute with its key, as in a type.
func TestExplicitBodyElementNameRequired(t *testing.T) {
	cases := map[string]struct {
		payloadRequired []string
		bodyRequired    []string
		wantErr         string
	}{
		"suffixed body required":            {payloadRequired: []string{"name"}, bodyRequired: []string{"name:nm"}},
		"attribute name body required":      {payloadRequired: []string{"name"}, bodyRequired: []string{"name"}, wantErr: `required attribute "name" of the HTTP request body is declared as "name:nm"; use Required("name:nm")`},
		"optional payload attribute":        {payloadRequired: []string{"name"}, bodyRequired: []string{"age:ag"}, wantErr: "age:ag"},
		"optional payload attribute, plain": {bodyRequired: []string{"name"}, wantErr: "name"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			design := func() {
				Service("svc", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("name", String)
							Attribute("age", Int)
							if len(tc.payloadRequired) > 0 {
								Required(tc.payloadRequired...)
							}
						})
						HTTP(func() {
							POST("/")
							Body(func() {
								Attribute("name:nm")
								Attribute("age:ag")
								Required(tc.bodyRequired...)
							})
						})
					})
				})
			}
			if tc.wantErr == "" {
				root := expr.RunDSL(t, design)
				body := root.API.HTTP.Service("svc").Endpoint("create").Body
				assert.True(t, expr.NewMappedAttributeExpr(body).IsRequired("name"))
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.Error(t, err)
			want := tc.wantErr
			if !strings.Contains(want, " ") {
				want = "The following HTTP request body attribute is required but the corresponding method payload attribute is not: " + want + "."
			}
			assert.Contains(t, err.Error(), want)
		})
	}
}

// TestExplicitResponseBodyRequiredKey checks that the Required list of an
// explicit response Body must name an attribute declared with an element name
// suffix with its key.
func TestExplicitResponseBodyRequiredKey(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("svc", func() {
			Method("show", func() {
				Result(func() {
					Attribute("name", String)
				})
				HTTP(func() {
					GET("/")
					Response(StatusOK, func() {
						Body(func() {
							Attribute("name:n")
							Required("name")
						})
					})
				})
			})
		})
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required attribute "name" of the HTTP response body is declared as "name:n"; use Required("name:n")`)
}

func explicitBodyDSL(userType, response bool, name, age string) func() {
	return func() {
		attributes := func() {
			Attribute("name", String, "Account name", func() {
				MinLength(2)
			})
			Attribute("age", Int, "Account age")
			Required("name")
		}
		var shape any = attributes
		if userType {
			shape = Type("Account", attributes)
		}
		body := func() {
			Body(func() {
				Attribute(name)
				Attribute(age)
			})
		}
		Service("svc", func() {
			Method("create", func() {
				if response {
					Result(shape)
				} else {
					Payload(shape)
				}
				HTTP(func() {
					POST("/")
					if response {
						Response(StatusOK, body)
					} else {
						body()
					}
				})
			})
		})
	}
}

func explicitBody(t *testing.T, design func(), response bool) *expr.AttributeExpr {
	t.Helper()
	root := expr.RunDSL(t, design)
	endpoint := root.API.HTTP.Service("svc").Endpoint("create")
	if response {
		return endpoint.Responses[0].Body
	}
	return endpoint.Body
}

func bodyChild(t *testing.T, body *expr.AttributeExpr, key string) *expr.AttributeExpr {
	t.Helper()
	child := expr.AsObject(body.Type).Attribute(key)
	require.NotNil(t, child, key)
	return child
}

func bodyRequired(body *expr.AttributeExpr) []string {
	if ut, ok := body.Type.(expr.UserType); ok && ut.Attribute().Validation != nil {
		return ut.Attribute().Validation.Required
	}
	if body.Validation == nil {
		return nil
	}
	return body.Validation.Required
}

func attributeKeys(att *expr.AttributeExpr) []string {
	object := *expr.AsObject(att.Type)
	keys := make([]string, 0, len(object))
	for _, nat := range object {
		keys = append(keys, nat.Name)
	}
	return keys
}
