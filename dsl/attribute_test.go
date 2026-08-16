package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestTitleSetsSchemaTitles(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Item", func() {
			Title("Item Resource")
			Attribute("updated_at", String, func() {
				Title("Last Modified At")
			})
		})
	})

	item := root.UserType("Item")
	if item.Attribute().Title != "Item Resource" {
		t.Errorf("got type title %q, expected %q", item.Attribute().Title, "Item Resource")
	}
	updatedAt := item.Attribute().Find("updated_at")
	if updatedAt == nil {
		t.Fatal("missing updated_at attribute")
	}
	if updatedAt.Title != "Last Modified At" {
		t.Errorf("got attribute title %q, expected %q", updatedAt.Title, "Last Modified At")
	}
}

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
