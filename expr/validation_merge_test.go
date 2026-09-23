package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationExprMergeKeepsTighterBounds(t *testing.T) {
	f := func(v float64) *float64 {
		return &v
	}
	i := func(v int) *int {
		return &v
	}
	cases := []struct {
		name  string
		v     ValidationExpr
		other ValidationExpr
		want  ValidationExpr
	}{
		{"exclusive minimum other tighter", ValidationExpr{ExclusiveMinimum: f(1)}, ValidationExpr{ExclusiveMinimum: f(5)}, ValidationExpr{ExclusiveMinimum: f(5)}},
		{"exclusive minimum receiver tighter", ValidationExpr{ExclusiveMinimum: f(5)}, ValidationExpr{ExclusiveMinimum: f(1)}, ValidationExpr{ExclusiveMinimum: f(5)}},
		{"exclusive minimum receiver nil", ValidationExpr{}, ValidationExpr{ExclusiveMinimum: f(1)}, ValidationExpr{ExclusiveMinimum: f(1)}},
		{"exclusive minimum other nil", ValidationExpr{ExclusiveMinimum: f(1)}, ValidationExpr{}, ValidationExpr{ExclusiveMinimum: f(1)}},
		{"minimum other tighter", ValidationExpr{Minimum: f(1)}, ValidationExpr{Minimum: f(5)}, ValidationExpr{Minimum: f(5)}},
		{"minimum receiver tighter", ValidationExpr{Minimum: f(5)}, ValidationExpr{Minimum: f(1)}, ValidationExpr{Minimum: f(5)}},
		{"minimum receiver nil", ValidationExpr{}, ValidationExpr{Minimum: f(1)}, ValidationExpr{Minimum: f(1)}},
		{"minimum other nil", ValidationExpr{Minimum: f(1)}, ValidationExpr{}, ValidationExpr{Minimum: f(1)}},
		{"exclusive maximum other tighter", ValidationExpr{ExclusiveMaximum: f(9)}, ValidationExpr{ExclusiveMaximum: f(5)}, ValidationExpr{ExclusiveMaximum: f(5)}},
		{"exclusive maximum receiver tighter", ValidationExpr{ExclusiveMaximum: f(5)}, ValidationExpr{ExclusiveMaximum: f(9)}, ValidationExpr{ExclusiveMaximum: f(5)}},
		{"exclusive maximum receiver nil", ValidationExpr{}, ValidationExpr{ExclusiveMaximum: f(9)}, ValidationExpr{ExclusiveMaximum: f(9)}},
		{"exclusive maximum other nil", ValidationExpr{ExclusiveMaximum: f(9)}, ValidationExpr{}, ValidationExpr{ExclusiveMaximum: f(9)}},
		{"maximum other tighter", ValidationExpr{Maximum: f(9)}, ValidationExpr{Maximum: f(5)}, ValidationExpr{Maximum: f(5)}},
		{"maximum receiver tighter", ValidationExpr{Maximum: f(5)}, ValidationExpr{Maximum: f(9)}, ValidationExpr{Maximum: f(5)}},
		{"maximum receiver nil", ValidationExpr{}, ValidationExpr{Maximum: f(9)}, ValidationExpr{Maximum: f(9)}},
		{"maximum other nil", ValidationExpr{Maximum: f(9)}, ValidationExpr{}, ValidationExpr{Maximum: f(9)}},
		{"min length other tighter", ValidationExpr{MinLength: i(1)}, ValidationExpr{MinLength: i(5)}, ValidationExpr{MinLength: i(5)}},
		{"min length receiver tighter", ValidationExpr{MinLength: i(5)}, ValidationExpr{MinLength: i(1)}, ValidationExpr{MinLength: i(5)}},
		{"min length receiver nil", ValidationExpr{}, ValidationExpr{MinLength: i(1)}, ValidationExpr{MinLength: i(1)}},
		{"min length other nil", ValidationExpr{MinLength: i(1)}, ValidationExpr{}, ValidationExpr{MinLength: i(1)}},
		{"max length other tighter", ValidationExpr{MaxLength: i(9)}, ValidationExpr{MaxLength: i(5)}, ValidationExpr{MaxLength: i(5)}},
		{"max length receiver tighter", ValidationExpr{MaxLength: i(5)}, ValidationExpr{MaxLength: i(9)}, ValidationExpr{MaxLength: i(5)}},
		{"max length receiver nil", ValidationExpr{}, ValidationExpr{MaxLength: i(9)}, ValidationExpr{MaxLength: i(9)}},
		{"max length other nil", ValidationExpr{MaxLength: i(9)}, ValidationExpr{}, ValidationExpr{MaxLength: i(9)}},
		{"both nil", ValidationExpr{}, ValidationExpr{}, ValidationExpr{}},
		{
			"mixed inclusive and exclusive bounds are all kept",
			ValidationExpr{Minimum: f(2), Maximum: f(8)},
			ValidationExpr{ExclusiveMinimum: f(3), ExclusiveMaximum: f(7)},
			ValidationExpr{Minimum: f(2), Maximum: f(8), ExclusiveMinimum: f(3), ExclusiveMaximum: f(7)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.v
			other := c.other
			got.Merge(&other)
			require.Equal(t, c.want, got)
			require.Equal(t, c.other, other, "Merge must not mutate its argument")
		})
	}
}

func TestValidationExprMergeIsConstraintIntersection(t *testing.T) {
	// A deterministic grid of optional bounds; -1 encodes "unset".
	bounds := []int{-1, 0, 2, 3, 5, 7}
	opt := func(b int) *float64 {
		if b < 0 {
			return nil
		}
		v := float64(b)
		return &v
	}
	optInt := func(b int) *int {
		if b < 0 {
			return nil
		}
		return &b
	}
	build := func(seed int) *ValidationExpr {
		n := len(bounds)
		return &ValidationExpr{
			ExclusiveMinimum: opt(bounds[seed%n]),
			Minimum:          opt(bounds[(seed/n)%n]),
			ExclusiveMaximum: opt(bounds[(seed*7+3)%n]),
			Maximum:          opt(bounds[(seed*5+1)%n]),
			MinLength:        optInt(bounds[(seed*3+2)%n]),
			MaxLength:        optInt(bounds[(seed*11+4)%n]),
		}
	}
	acceptsNumber := func(v *ValidationExpr, x float64) bool {
		return (v.ExclusiveMinimum == nil || x > *v.ExclusiveMinimum) &&
			(v.Minimum == nil || x >= *v.Minimum) &&
			(v.ExclusiveMaximum == nil || x < *v.ExclusiveMaximum) &&
			(v.Maximum == nil || x <= *v.Maximum)
	}
	acceptsLength := func(v *ValidationExpr, l int) bool {
		return (v.MinLength == nil || l >= *v.MinLength) &&
			(v.MaxLength == nil || l <= *v.MaxLength)
	}
	for a := range 36 {
		for b := range 36 {
			left, right := build(a), build(b*13+5)
			merged := left.Dup()
			merged.Merge(right)
			for x := -1; x <= 9; x++ {
				ctx := fmt.Sprintf("a=%s b=%s x=%d", describeBounds(left), describeBounds(right), x)
				if got, want := acceptsNumber(merged, float64(x)), acceptsNumber(left, float64(x)) && acceptsNumber(right, float64(x)); got != want {
					t.Errorf("numeric acceptance mismatch: got %v want %v (%s)", got, want, ctx)
				}
				if x < 0 {
					continue
				}
				if got, want := acceptsLength(merged, x), acceptsLength(left, x) && acceptsLength(right, x); got != want {
					t.Errorf("length acceptance mismatch: got %v want %v (%s)", got, want, ctx)
				}
			}
		}
	}
}

func describeBounds(v *ValidationExpr) string {
	f := func(p *float64) string {
		if p == nil {
			return "-"
		}
		return fmt.Sprint(*p)
	}
	i := func(p *int) string {
		if p == nil {
			return "-"
		}
		return fmt.Sprint(*p)
	}
	return fmt.Sprintf("{xmin:%s min:%s xmax:%s max:%s minlen:%s maxlen:%s}",
		f(v.ExclusiveMinimum), f(v.Minimum), f(v.ExclusiveMaximum), f(v.Maximum), i(v.MinLength), i(v.MaxLength))
}
