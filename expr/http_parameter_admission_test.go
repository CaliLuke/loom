package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestParameterSerializationAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, location, shape string
		metadata              map[string]string
		rejected              string
	}{
		{"typed object JSON body", "body", "typed", nil, ""},
		{"typed object path", "path", "typed", nil, "no supported parameter codec"},
		{"typed object query", "query", "typed", map[string]string{"style": "deepObject"}, "no supported parameter codec"},
		{"typed object header", "header", "typed", nil, "no supported parameter codec"},
		{"typed object cookie", "cookie", "typed", nil, "no supported parameter codec"},
		{"typed object form", "form", "typed", nil, "do not support objects with AdditionalAttributes"},
		{"path scalar", "path", "scalar", map[string]string{"style": "matrix"}, ""},
		{"path list", "path", "list", map[string]string{"style": "label", "explode": "true"}, ""},
		{"path object", "path", "object", map[string]string{"explode": "true"}, ""},
		{"path map", "path", "map", map[string]string{"style": "matrix", "explode": "true"}, ""},
		{"query object", "query", "object", map[string]string{"style": "deepObject"}, ""},
		{"query map", "query", "map", map[string]string{"style": "form"}, ""},
		{"query map JSON", "query", "map", nil, ""},
		{"header object", "header", "object", map[string]string{"explode": "true"}, ""},
		{"header map", "header", "map", nil, ""},
		{"cookie list", "cookie", "list", map[string]string{"explode": "false"}, ""},
		{"cookie object", "cookie", "object", nil, ""},
		{"cookie map", "cookie", "map", nil, ""},
		{"form object", "form", "object", map[string]string{"style": "deepObject"}, ""},
		{"form ignores content", "form", "object", map[string]string{"contentType": "application/json", "explode": "false"}, ""},
		{"form reserved false", "form", "object", map[string]string{"allowReserved": "false"}, ""},
		{"unknown style", "query", "scalar", map[string]string{"style": "unknown"}, "style unknown"},
		{"wrong location", "query", "scalar", map[string]string{"style": "matrix"}, "style matrix"},
		{"invalid explode", "path", "scalar", map[string]string{"explode": "maybe"}, "explode"},
		{"invalid reserved", "query", "scalar", map[string]string{"allowReserved": "maybe"}, "allowReserved"},
		{"body metadata", "body", "scalar", map[string]string{"style": "form"}, "body field"},
		{"nested map style", "query", "nested", map[string]string{"style": "form"}, "nested values"},
		{"deep list", "query", "list", map[string]string{"style": "deepObject"}, "deepObject"},
		{"space exploded", "query", "list", map[string]string{"style": "spaceDelimited", "explode": "true"}, "explode: true"},
		{"cookie D1", "cookie", "list", map[string]string{"style": "form", "explode": "true"}, "Appendix D.1"},
		{"map reserved alone", "query", "map", map[string]string{"allowReserved": "false"}, "content-based"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				typ := parameterAdmissionType(tc.shape)
				Service("parameters", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("value", typ, func() {
								if tc.location == "body" {
									for key, value := range tc.metadata {
										Meta("openapi:"+key, value)
									}
								}
							})
						})
						HTTP(func() {
							metadata := func() {
								for key, value := range tc.metadata {
									Meta("openapi:"+key, value)
								}
							}
							if tc.location == "path" {
								GET("/{value}")
							} else {
								POST("/")
							}
							switch tc.location {
							case "path", "query":
								Param("value", metadata)
							case "header":
								Header("value", metadata)
							case "cookie":
								Cookie("value", metadata)
							case "form":
								FormRequest()
								Body(func() {
									Attribute("value")
									for key, value := range tc.metadata {
										Meta("openapi:encoding:value:"+key, value)
									}
								})
							}
						})
					})
				})
			}
			if tc.rejected == "" {
				expr.RunDSL(t, design)
			} else {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, tc.rejected)
			}
		})
	}
	for _, location := range []string{"query", "cookie", "form"} {
		t.Run(location+" namespace collision", func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				record := Type("Record", func() {
					Attribute("other", String)
				})
				Service("namespaces", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("value", record)
							Attribute("other", String)
						})
						HTTP(func() {
							POST("/")
							switch location {
							case "query":
								Param("value")
								Param("other")
							case "cookie":
								Cookie("value")
								Cookie("other")
							case "form":
								FormRequest()
								Body(func() {
									Attribute("value")
									Attribute("other")
									Meta("openapi:encoding:value:style", "form")
								})
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, "overlapping parameter namespaces")
		})
	}
}

func parameterAdmissionType(shape string) expr.DataType {
	switch shape {
	case "list":
		return ArrayOf(String)
	case "map":
		return MapOf(String, String)
	case "nested":
		return MapOf(String, ArrayOf(String))
	case "typed":
		return Type("TypedRecord", func() {
			Attribute("min", Int)
			AdditionalAttributes(Int)
		})
	case "object":
		return Type("Record", func() {
			Attribute("min", Int)
			Attribute("max", Int)
		})
	default:
		return String
	}
}

func TestParameterCredentialNamespaces(t *testing.T) {
	for _, session := range []bool{false, true} {
		name := "query API key"
		if session {
			name = "session cookie"
		}
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				key := APIKeySecurity("key")
				record := Type("Record", func() {
					Attribute("sid", String)
				})
				var authentication any
				if session {
					authentication = SessionAuth("session", func() {
						CookieTransport(key, "", func() {
							CookieName("sid")
						})
					})
				}
				Service("credential", func() {
					Method("read", func() {
						if session {
							SessionSecurity(authentication)
						} else {
							Security(key)
						}
						Payload(func() {
							Attribute("value", record)
							if !session {
								APIKey("key", "token", String)
								Required("token")
							}
						})
						HTTP(func() {
							POST("/")
							if session {
								Cookie("value")
							} else {
								Param("value")
								Param("token:sid")
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, "overlapping parameter namespaces")
			require.ErrorContains(t, err, "value")
		})
	}
}
