import ValueContract.NumericExtraction

namespace ValueContract.Candidate.NumericIntervals

/-- Finite positive-integer multiplication expressions model the two sides after
cross-multiplying the normalized factor tuple. Repeated squaring yields a finite
tree for every finite exponent, without requiring this model to evaluate it. -/
inductive Product where
  | leaf : Nat → Product
  | mul : Product → Product → Product

def value : Product → Nat
  | .leaf n => n
  | .mul left right => value left * value right

def floorCap (n capacity : Nat) : Nat :=
  if n ≤ capacity then n else n / capacity * capacity

def ceilCap (n capacity : Nat) : Nat :=
  if n ≤ capacity then n else (n / capacity + 1) * capacity

/-- Abstract outward normalization. The Go prototype uses scaled binary
mantissas; this model records its enclosure and exact-below-capacity obligations,
not an extraction proof for BitLen, big.Int or binary scale arithmetic. -/
def enclosure (capacity : Nat) : Product → Nat × Nat
  | .leaf n => (floorCap n capacity, ceilCap n capacity)
  | .mul left right =>
      let l := enclosure capacity left
      let r := enclosure capacity right
      (floorCap (l.1 * r.1) capacity, ceilCap (l.2 * r.2) capacity)

theorem floorCap_le (n capacity : Nat) : floorCap n capacity ≤ n := by
  unfold floorCap
  split
  · exact Nat.le_refl _
  · exact Nat.div_mul_le_self _ _

theorem le_ceilCap (n capacity : Nat) (positive : 0 < capacity) : n ≤ ceilCap n capacity := by
  unfold ceilCap
  split
  · exact Nat.le_refl _
  · have remainder := Nat.mod_lt n positive
    have decomposition := Nat.mod_add_div n capacity
    simp only [Nat.add_mul, Nat.one_mul]
    rw [Nat.mul_comm capacity (n / capacity)] at decomposition
    omega

/-- Every approximation contains the true integer result. -/
theorem enclosure_contains (expression : Product) (capacity : Nat) (positive : 0 < capacity) :
    (enclosure capacity expression).1 ≤ value expression ∧
    value expression ≤ (enclosure capacity expression).2 := by
  induction expression with
  | leaf n => exact ⟨floorCap_le n capacity, le_ceilCap n capacity positive⟩
  | mul left right il ir =>
      exact ⟨Nat.le_trans (floorCap_le _ _) (Nat.mul_le_mul il.1 ir.1),
        Nat.le_trans (Nat.mul_le_mul il.2 ir.2) (le_ceilCap _ _ positive)⟩

/-- Decision only from separated enclosures is sound; overlap is not equality. -/
theorem separated_enclosures_order (left right : Product) (capacity : Nat)
    (positive : 0 < capacity)
    (separated : (enclosure capacity left).2 < (enclosure capacity right).1) :
    value left < value right := by
  have l := enclosure_contains left capacity positive
  have r := enclosure_contains right capacity positive
  omega

def need : Product → Nat
  | .leaf n => n
  | .mul left right => max (value left * value right) (max (need left) (need right))

/-- A sufficiently large finite capacity makes every intermediate exact. This
is an existence bound, not a promised efficient universal resource bound. -/
theorem enclosure_exact (expression : Product) (capacity : Nat) (enough : need expression ≤ capacity) :
    enclosure capacity expression = (value expression, value expression) := by
  induction expression with
  | leaf n => simp only [need] at enough; simp [enclosure, value, floorCap, ceilCap, enough]
  | mul left right il ir =>
      have product : value left * value right ≤ capacity := Nat.le_trans (Nat.le_max_left _ _) enough
      have children : max (need left) (need right) ≤ capacity := Nat.le_trans (Nat.le_max_right _ _) enough
      have l : need left ≤ capacity := Nat.le_trans (Nat.le_max_left _ _) children
      have r : need right ≤ capacity := Nat.le_trans (Nat.le_max_right _ _) children
      simp [enclosure, il l, ir r, value, floorCap, ceilCap, product]

/-- The doubling-precision schedule reaches a capacity separating any unequal
finite pair. Canonical equality must be handled before adaptive comparison.
No claim is made that the witness round is practically small. -/
theorem refinement_eventually_separates (left right : Product) (different : value left ≠ value right) :
    ∃ round : Nat,
      (enclosure (2 ^ (2 ^ round)) left).2 < (enclosure (2 ^ (2 ^ round)) right).1 ∨
      (enclosure (2 ^ (2 ^ round)) right).2 < (enclosure (2 ^ (2 ^ round)) left).1 := by
  let round := max (need left) (need right)
  have growing : round < 2 ^ (2 ^ round) := Nat.lt_trans Nat.lt_two_pow_self Nat.lt_two_pow_self
  have l : need left ≤ 2 ^ (2 ^ round) := Nat.le_trans (Nat.le_max_left _ _) (Nat.le_of_lt growing)
  have r : need right ≤ 2 ^ (2 ^ round) := Nat.le_trans (Nat.le_max_right _ _) (Nat.le_of_lt growing)
  refine ⟨round, ?_⟩
  rw [enclosure_exact left _ l, enclosure_exact right _ r]
  exact Nat.lt_or_gt_of_ne different

/-- Low precision overlaps even though values differ; later refinement decides.
A comparison that equates rounded points would lose this distinction. -/
theorem rounded_overlap_control :
    enclosure 10 (.leaf 1001) = (1000, 1010) ∧
    enclosure 10 (.leaf 1000) = (1000, 1010) ∧
    (enclosure 2048 (.leaf 1000)).2 < (enclosure 2048 (.leaf 1001)).1 := by
  decide

end ValueContract.Candidate.NumericIntervals
