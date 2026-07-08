package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestFieldAddsTagToUnionBranch(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Envelope", func() {
			OneOf("choice", func() {
				Field(1, "email", String)
				Field(2, "sms", String)
			})
		})
	})

	envelope := root.UserType("Envelope")
	choice := envelope.Attribute().Find("choice")
	if choice == nil {
		t.Fatalf("missing choice attribute")
	}
	union := expr.AsUnion(choice.Type)
	if union == nil || len(union.Values) != 2 {
		t.Fatalf("expected two union branches, got %#v", union)
	}
	if tag, ok := union.Values[0].Attribute.Meta.Last("rpc:tag"); !ok || tag != "1" {
		t.Fatalf("expected email branch tag 1, got %q", tag)
	}
	if tag, ok := union.Values[1].Attribute.Meta.Last("rpc:tag"); !ok || tag != "2" {
		t.Fatalf("expected sms branch tag 2, got %q", tag)
	}
}
