package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type familyTestKey string

func (k familyTestKey) Hash() string {
	return string(k)
}

func TestNameScopeHashedUniqueFamily(t *testing.T) {
	companions := func(name string) []string {
		return []string{"New" + name, name + "Kind"}
	}
	for _, test := range []struct {
		name     string
		occupied []string
		expected string
	}{
		{"ordinary", nil, "Choice"},
		{"primary", []string{"Choice"}, "Choice2"},
		{"constructor", []string{"NewChoice"}, "Choice2"},
		{"occupied suffix", []string{"NewChoice", "Choice2", "Choice3Kind"}, "Choice4"},
	} {
		t.Run(test.name, func(t *testing.T) {
			scope := NewNameScope()
			for _, name := range test.occupied {
				scope.Unique(name)
			}
			key := familyTestKey("choice")
			name := scope.HashedUniqueFamily(key, "Choice", companions)
			require.Equal(t, test.expected, name)
			require.Equal(t, name, scope.HashedUnique(key, "ignored"))
			require.Equal(t, name, scope.HashedUniqueFamily(key, "ignored", companions))
			require.Equal(t, name, scope.WithoutPackageNames().HashedUniqueFamily(key, "ignored", companions))
			for _, declared := range append([]string{name}, companions(name)...) {
				require.NotEqual(t, declared, scope.PeekUnique(declared))
			}
			require.Equal(t, "Choice", NewNameScopeLike(scope).HashedUniqueFamily(key, "Choice", companions))
		})
	}
}

func TestNameScopeFamilyRejectsLateOrChangedInventory(t *testing.T) {
	t.Run("late collision is atomic", func(t *testing.T) {
		scope := NewNameScope()
		key := familyTestKey("choice")
		require.Equal(t, "Choice", scope.HashedUnique(key, "Choice"))
		scope.Unique("NewChoice")
		require.Panics(t, func() {
			scope.HashedUniqueFamily(key, "Choice", func(name string) []string {
				return []string{"Free", "New" + name}
			})
		})
		require.Equal(t, "Choice", scope.HashedUnique(key, "ignored"))
		require.Equal(t, "Free", scope.PeekUnique("Free"))
	})
	t.Run("unoccupied cached name", func(t *testing.T) {
		scope := NewNameScope()
		key := familyTestKey("choice")
		scope.HashedUnique(key, "Choice")
		require.Equal(t, "Choice", scope.HashedUniqueFamily(key, "ignored", func(name string) []string {
			return []string{"New" + name}
		}))
		require.Panics(t, func() {
			scope.HashedUniqueFamily(key, "Choice", func(name string) []string {
				return []string{name + "Kind"}
			})
		})
	})
	t.Run("invalid inventory", func(t *testing.T) {
		scope := NewNameScope()
		require.Panics(t, func() {
			scope.HashedUniqueFamily(familyTestKey("choice"), "Choice", func(name string) []string {
				return []string{name}
			})
		})
		require.Equal(t, "Choice", scope.PeekUnique("Choice"))
	})
}

func TestNameScopeTypeFamilyFirstReference(t *testing.T) {
	typ := &expr.UserTypeExpr{TypeName: "Choice", UID: "choice", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	scope := NewNameScope()
	scope.Unique("NewChoice")
	alias := scope.WithoutPackageNames()
	scope.SetTypeFamilies(func(_ expr.DataType, primary string) []string {
		return []string{"New" + primary}
	})
	attribute := &expr.AttributeExpr{Type: typ}
	require.Equal(t, "Choice2", alias.GoTypeName(attribute))
	require.Equal(t, "Choice2", scope.GoTypeName(attribute))
	require.Equal(t, "service.Choice2", scope.GoFullTypeName(attribute, "service"))
	require.Equal(t, "Choice2", scope.WithoutPackageNames().GoTypeName(attribute))
	require.Equal(t, "Choice", NewNameScopeLike(scope).GoTypeName(attribute))
	require.Panics(t, func() {
		scope.SetTypeFamilies(func(_ expr.DataType, primary string) []string {
			return nil
		})
	})
}
