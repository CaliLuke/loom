package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAuthorizationInvalidDeclarations(t *testing.T) {
	cases := []struct {
		name   string
		design func()
		want   string
	}{
		{"empty name", func() {
			Authorization("", Empty)
		}, "name"},
		{"nil input", func() {
			Authorization("edit", nil)
		}, "input"},
		{"scalar input", func() {
			Authorization("edit", String)
		}, "named object"},
		{"duplicate name", func() {
			Authorization("edit", Empty)
			Authorization("edit", Empty)
		}, "already defined"},
		{"API strict", func() {
			API("strict", func() {
				StrictAuthorization()
			})
			Service("documents", func() {
				Method("forgotten", func() {
				})
			})
		}, "authorization classification"},
		{"wrong scope", func() {
			StrictAuthorization()
		}, "invalid use"},
		{"binding outside requirement", func() {
			Bind("id", "id")
		}, "invalid use"},
		{"case outside selection", func() {
			AuthorizationCase("read", func() {
			})
		}, "invalid use"},
		{"nil requirement", func() {
			Service("documents", func() {
				Method("update", func() {
					Authorize(nil)
				})
			})
		}, "registered requirement"},
		{"missing selector", func() {
			invalidAuthorizationSelection("missing", "read", "write")
		}, "missing"},
		{"unbounded selector", func() {
			invalidAuthorizationSelection("id", "read", "write")
		}, "enum or union"},
		{"duplicate case", func() {
			invalidAuthorizationSelection("action", "read", "read")
		}, "duplicate"},
		{"unknown case", func() {
			invalidAuthorizationSelection("action", "read", "remove")
		}, "unknown"},
		{"empty cases", func() {
			invalidAuthorizationSelection("action")
		}, "missing authorization case"},
		{"non-case declaration", func() {
			Service("documents", func() {
				Method("update", func() {
					AuthorizeBy("action", func() {
						NoAccessCheck("exempt")
					})
				})
			})
		}, "exemption"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.design)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func invalidAuthorizationSelection(selector string, cases ...string) {
	Service("documents", func() {
		Method("update", func() {
			Payload(func() {
				Attribute("id", String)
				Attribute("action", String, func() {
					Enum("read", "write")
				})
			})
			AuthorizeBy(selector, func() {
				for _, value := range cases {
					AuthorizationCase(value, func() {
						NoAccessCheck("Test exemption")
					})
				}
			})
		})
	})
}

func TestAuthorizationContextOnlyAndOptIn(t *testing.T) {
	root := expr.RunDSL(t, func() {
		access := Authorization("account.create", Empty)
		Service("accounts", func() {
			Method("create", func() {
				Authorize(access)
			})
			Method("legacy", func() {
			})
		})
	})
	require.NotNil(t, root.Services[0].Methods[0].Authorization)
	require.Nil(t, root.Services[0].Methods[1].Authorization)
}

func TestAuthorizationBindingRepresentations(t *testing.T) {
	cases := []struct {
		name   string
		fields func() (func(), func())
	}{
		{"custom field", func() (func(), func()) {
			return func() {
					Attribute("value", String, func() {
						Meta("struct:field:type", "time.Time", "time")
					})
				}, func() {
					Attribute("value", String)
				}
		}},
		{"array named identity", func() (func(), func()) {
			a := Type("ItemA", func() {
				Attribute("id", String)
			})
			b := Type("ItemB", func() {
				Attribute("id", String)
			})
			return func() {
					Attribute("value", ArrayOf(a))
				}, func() {
					Attribute("value", ArrayOf(b))
				}
		}},
		{"map named identity", func() (func(), func()) {
			a := Type("ItemA", func() {
				Attribute("id", String)
			})
			b := Type("ItemB", func() {
				Attribute("id", String)
			})
			return func() {
					Attribute("value", MapOf(String, a))
				}, func() {
					Attribute("value", MapOf(String, b))
				}
		}},
		{"anonymous presence", func() (func(), func()) {
			return func() {
					Attribute("value", func() {
						Attribute("id", String)
						Required("id")
					})
				}, func() {
					Attribute("value", func() {
						Attribute("id", String)
					})
				}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				input, source := tc.fields()
				ref := Type("Input", input)
				access := Authorization("edit", ref)
				Service("documents", func() {
					Method("update", func() {
						Payload(source)
						Authorize(access, func() {
							Bind("value", "value")
						})
					})
				})
			})
			require.ErrorContains(t, err, "incompatible")
		})
	}
}
