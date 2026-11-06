package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestObjectOpennessMetadata checks the authored policy states and rejects
// values outside the documented true/false contract.
func TestObjectOpennessMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		meta    []string
		want    expr.ObjectOpenness
		failure string
	}{
		{name: "unset", want: expr.ObjectOpennessUnset},
		{name: "explicit open", meta: []string{"true"}, want: expr.ObjectOpen},
		{name: "closed", meta: []string{"false"}, want: expr.ObjectClosed},
		{name: "last value wins", meta: []string{"true", "false"}, want: expr.ObjectClosed},
		{name: "invalid", meta: []string{"no"}, failure: `openapi:additionalProperties must be "true" or "false"`},
		{name: "no value", meta: []string{}, failure: `openapi:additionalProperties must be "true" or "false", got no value`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var record expr.UserType
			design := func() {
				record = Type("Record", func() {
					Attribute("id", String)
					if test.meta != nil {
						Meta("openapi:additionalProperties", test.meta...)
					}
				})
				Service("records", func() {
					Method("create", func() {
						Payload(record)
					})
				})
			}
			if test.failure != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), test.failure)
				return
			}
			expr.RunDSL(t, design)
			require.Equal(t, test.want, record.Attribute().ObjectOpenness())
			require.Equal(t, test.want != expr.ObjectClosed, record.Attribute().ObjectOpenness().AdmitsUndeclared())
		})
	}
}

// TestExtendClosedBasePerSource checks #1062: a closed base constrains every
// wire key it does not declare, so composition cannot add members over it.
func TestExtendClosedBasePerSource(t *testing.T) {
	closedBase := func() expr.UserType {
		return Type("Base", func() {
			Attribute("id", String)
			Meta("openapi:additionalProperties", "false")
		})
	}
	for _, test := range []struct {
		name    string
		design  func() expr.UserType
		failure string
		want    expr.ObjectOpenness
	}{
		{"derived adds a member", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Attribute("note", String)
			})
		}, `does not declare "note"`, 0},
		{"member declared before Extend", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Attribute("note", String)
				Extend(base)
			})
		}, `does not declare "note"`, 0},
		{"peer base adds a member", func() expr.UserType {
			base := closedBase()
			peer := Type("Peer", func() {
				Attribute("note", String)
			})
			return Type("Derived", func() {
				Extend(base)
				Extend(peer)
			})
		}, `does not declare "note"`, 0},
		{"chained derived adds a member", func() expr.UserType {
			base := closedBase()
			middle := Type("Middle", func() {
				Extend(base)
			})
			return Type("Derived", func() {
				Extend(middle)
				Attribute("note", String)
			})
		}, `does not declare "note"`, 0},
		{"closed peers declare different members", func() expr.UserType {
			base := closedBase()
			peer := Type("Peer", func() {
				Attribute("note", String)
				Meta("openapi:additionalProperties", "false")
			})
			return Type("Derived", func() {
				Extend(base)
				Extend(peer)
			})
		}, `does not declare`, 0},
		{"alias of a closed base adds a member", func() expr.UserType {
			alias := Type("Alias", closedBase())
			return Type("Derived", func() {
				Extend(alias)
				Attribute("note", String)
			})
		}, `does not declare "note"`, 0},
		{"chained alias of a closed base adds a member", func() expr.UserType {
			inner := Type("Inner", closedBase())
			alias := Type("Alias", inner)
			return Type("Derived", func() {
				Extend(alias)
				Attribute("note", String)
			})
		}, `does not declare "note"`, 0},
		{"reopening alias used only through Extend", func() expr.UserType {
			alias := Type("Alias", closedBase(), func() {
				Meta("openapi:additionalProperties", "true")
			})
			return Type("Derived", func() {
				Extend(alias)
				Attribute("note", String)
			})
		}, `does not declare "note"`, 0},
		{"policy on a non-object", func() expr.UserType {
			return usedType(Type("Holder", func() {
				Attribute("labels", MapOf(String, String), func() {
					Meta("openapi:additionalProperties", "false")
				})
			}))
		}, "openapi:additionalProperties applies only to objects", 0},
		{"policy on an array of a closed type", func() expr.UserType {
			base := closedBase()
			return usedType(Type("Holder", func() {
				Attribute("items", ArrayOf(base), func() {
					Meta("openapi:additionalProperties", "true")
				})
			}))
		}, "openapi:additionalProperties applies only to objects", 0},
		{"alias reopens a closed base", func() expr.UserType {
			return usedType(Type("Alias", closedBase(), func() {
				Meta("openapi:additionalProperties", "true")
			}))
		}, `cannot reopen closed type "Base"`, 0},
		{"occurrence reopens a closed base", func() expr.UserType {
			base := closedBase()
			return usedType(Type("Holder", func() {
				Attribute("child", base, func() {
					Meta("openapi:additionalProperties", "true")
				})
			}))
		}, `cannot reopen closed type "Base"`, 0},
		{"reopening stays rejected", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Meta("openapi:additionalProperties", "true")
			})
		}, "cannot reopen inherited closed object", 0},
		{"key without a value over a closed base", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Meta("openapi:additionalProperties")
			})
		}, "cannot reopen inherited closed object", 0},
		{"repeating closure", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Meta("openapi:additionalProperties", "false")
			})
		}, "", expr.ObjectClosed},
		{"inherits closure without new members", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
			})
		}, "", expr.ObjectClosed},
		{"alias inherits closure without new members", func() expr.UserType {
			alias := Type("Alias", closedBase())
			return Type("Derived", func() {
				Extend(alias)
			})
		}, "", expr.ObjectClosed},
		{"refines a declared member", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Attribute("id", String, func() {
					MinLength(1)
				})
			})
		}, "", expr.ObjectClosed},
		{"member outside the wire", func() expr.UserType {
			base := closedBase()
			return Type("Derived", func() {
				Extend(base)
				Attribute("internal", String, func() {
					Meta("struct:tag:json", "-")
				})
			})
		}, "", expr.ObjectClosed},
		{"closed peers declare the same members", func() expr.UserType {
			base := closedBase()
			peer := Type("Peer", func() {
				Attribute("id", String)
				Meta("openapi:additionalProperties", "false")
			})
			return Type("Derived", func() {
				Extend(base)
				Extend(peer)
			})
		}, "", expr.ObjectClosed},
		{"migration: open base and closed derived", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
			})
			return Type("Derived", func() {
				Extend(base)
				Attribute("note", String)
				Meta("openapi:additionalProperties", "false")
			})
		}, "", expr.ObjectClosed},
	} {
		t.Run(test.name, func(t *testing.T) {
			var derived expr.UserType
			design := func() {
				derived = test.design()
			}
			if test.failure != "" {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, test.failure)
				return
			}
			expr.RunDSL(t, design)
			require.Equal(t, test.want, derived.Attribute().ObjectOpenness())
		})
	}
}

// usedType references typ from a method so validation visits it.
func usedType(typ expr.UserType) expr.UserType {
	Service("records", func() {
		Method("create", func() {
			Payload(typ)
		})
	})
	return typ
}
