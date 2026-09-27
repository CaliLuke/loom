package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestGoFieldNameCollisions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		left     string
		right    string
		override string
		valid    bool
	}{
		{"case", "foo_bar", "fooBar", "", false},
		{"acronym", "res_id", "resId", "", false},
		{"punctuation", "foo-bar", "foo_bar", "", false},
		{"JSON suffix", "foo_bar:a", "fooBar:b", "", false},
		{"override collision", "foo_bar", "different", "fooBar", false},
		{"override resolves", "foo_bar", "fooBar", "OtherFooBar", true},
		{"distinct", "foo_bar", "other", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				Service("fields", func() {
					Method("echo", func() {
						Payload(func() {
							Attribute(tc.left, String)
							Attribute(tc.right, String, func() {
								if tc.override != "" {
									Meta("struct:field:name", tc.override)
								}
							})
						})
					})
				})
			}
			if tc.valid {
				expr.RunDSL(t, design)
			} else {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, "both generate Go field")
				require.ErrorContains(t, err, "struct:field:name")
			}
		})
	}
}

func TestGoFieldNameCollisionShapes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		declare func(func())
	}{
		{"named", func(fields func()) {
			Type("Value", fields)
		}},
		{"result", func(fields func()) {
			Service("fields", func() {
				Method("echo", func() {
					Result(fields)
				})
			})
		}},
		{"streaming payload", func(fields func()) {
			Service("fields", func() {
				Method("echo", func() {
					StreamingPayload(fields)
				})
			})
		}},
		{"nested collection", func(fields func()) {
			Type("Container", ArrayOf(MapOf(String, Type("Entry", fields))))
		}},
		{"recursive", func(fields func()) {
			Type("Node", func() {
				fields()
				Attribute("next", "Node")
			})
		}},
		{"extended", func(fields func()) {
			base := Type("Base", func() {
				Attribute("foo_bar", String)
			})
			Type("Value", func() {
				Extend(base)
				Attribute("fooBar", String)
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				tc.declare(func() {
					Attribute("foo_bar", String)
					Attribute("fooBar", String)
				})
			})
			require.ErrorContains(t, err, "both generate Go field")
		})
	}
}

func TestGoFieldNameInheritance(t *testing.T) {
	for _, extend := range []bool{false, true} {
		for _, override := range []bool{false, true} {
			name := "reference"
			if extend {
				name = "extend"
			}
			if override {
				name += " with override"
			}
			t.Run(name, func(t *testing.T) {
				design := func() {
					base := Type("Base", func() {
						Attribute("foo_bar", String, func() {
							if override {
								Meta("struct:field:name", "OtherFooBar")
							}
						})
					})
					middle := Type("Middle", func() {
						Extend(base)
					})
					Type("Leaf", func() {
						if extend {
							Extend(middle)
						} else {
							Reference(base)
							Attribute("foo_bar")
						}
						Attribute("fooBar", String)
					})
				}
				if override {
					expr.RunDSL(t, design)
				} else {
					require.ErrorContains(t, expr.RunInvalidDSL(t, design), "both generate Go field")
				}
			})
		}
	}
}

func TestGoFieldNameNamedWrapperInheritance(t *testing.T) {
	for _, position := range []string{"payload", "result", "streaming payload"} {
		for _, override := range []bool{false, true} {
			name := position
			if override {
				name += " with override"
			}
			t.Run(name, func(t *testing.T) {
				design := func() {
					left := Type("Left", func() {
						Attribute("foo_bar", String)
					})
					right := Type("Right", func() {
						Attribute("fooBar", String, func() {
							if override {
								Meta("struct:field:name", "OtherFooBar")
							}
						})
					})
					Service("fields", func() {
						Method("echo", func() {
							extend := func() {
								Extend(right)
							}
							switch position {
							case "payload":
								Payload(left, extend)
							case "result":
								Result(left, extend)
							case "streaming payload":
								StreamingPayload(left, extend)
							}
						})
					})
				}
				if override {
					expr.RunDSL(t, design)
				} else {
					require.ErrorContains(t, expr.RunInvalidDSL(t, design), "both generate Go field")
				}
			})
		}
	}
}
