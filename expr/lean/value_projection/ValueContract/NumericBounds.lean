import ValueContract.ScalarSemantics

namespace ValueContract.Candidate

private theorem denominator_pos (a : Decimal) : 0 < (a.fraction.2 : Int) := by
  unfold Decimal.fraction
  split
  · have h := @Nat.pow_pos 10 a.exponent.natAbs (by decide)
    simp only
    omega
  · simp

private theorem decimal_lt_of_le_of_lt {a b c : Decimal}
    (hab : a.le b = true) (hbc : b.lt c = true) : a.lt c = true := by
  simp only [Decimal.le, Decimal.lt, decide_eq_true_eq] at *
  have h1 := Int.mul_le_mul_of_nonneg_right hab (Int.le_of_lt (denominator_pos c))
  have h2 := Int.mul_lt_mul_of_pos_right hbc (denominator_pos a)
  have h : (a.fraction.1 * (c.fraction.2 : Int)) * (b.fraction.2 : Int) <
      (c.fraction.1 * (a.fraction.2 : Int)) * (b.fraction.2 : Int) := by
    grind only [Int.mul_comm, Int.mul_assoc]
  apply Int.lt_of_not_ge
  intro hn'
  have hh := Int.mul_le_mul_of_nonneg_right hn' (Int.le_of_lt (denominator_pos b))
  exact Int.not_le_of_gt h hh

private theorem decimal_lt_of_lt_of_le {a b c : Decimal}
    (hab : a.lt b = true) (hbc : b.le c = true) : a.lt c = true := by
  simp only [Decimal.le, Decimal.lt, decide_eq_true_eq] at *
  have h1 := Int.mul_lt_mul_of_pos_right hab (denominator_pos c)
  have h2 := Int.mul_le_mul_of_nonneg_right hbc (Int.le_of_lt (denominator_pos a))
  have h : (a.fraction.1 * (c.fraction.2 : Int)) * (b.fraction.2 : Int) <
      (c.fraction.1 * (a.fraction.2 : Int)) * (b.fraction.2 : Int) := by
    grind only [Int.mul_comm, Int.mul_assoc]
  apply Int.lt_of_not_ge
  intro hn'
  have hh := Int.mul_le_mul_of_nonneg_right hn' (Int.le_of_lt (denominator_pos b))
  exact Int.not_le_of_gt h hh

/-- The four independently authored numerical restrictions at one occurrence. -/
structure AuthoredNumericBounds where
  minimum : Option Decimal := none
  exclusiveMinimum : Option Decimal := none
  maximum : Option Decimal := none
  exclusiveMaximum : Option Decimal := none
  deriving BEq, DecidableEq, Repr

/-- Independent authored conjunction; no selected-bound or candidate-lowering call. -/
def AuthoredNumericBounds.Allows (bounds : AuthoredNumericBounds) (value : Decimal) : Prop :=
  (∀ lower ∈ bounds.minimum, lower.le value = true) ∧
  (∀ lower ∈ bounds.exclusiveMinimum, lower.lt value = true) ∧
  (∀ upper ∈ bounds.maximum, value.le upper = true) ∧
  (∀ upper ∈ bounds.exclusiveMaximum, value.lt upper = true)

/-- Current lossy exclusive-overwrite behavior, retained only as a counterexample. -/
def legacyNumericLowering (bounds : AuthoredNumericBounds) : NumericBounds :=
  { minimum := bounds.exclusiveMinimum.or bounds.minimum
    maximum := bounds.exclusiveMaximum.or bounds.maximum
    exclusiveMinimum := bounds.exclusiveMinimum.isSome
    exclusiveMaximum := bounds.exclusiveMaximum.isSome }

/-- Pick the stronger lower restriction; a numerical tie keeps the open endpoint. -/
def effectiveLower : Option Decimal → Option Decimal → Option Decimal × Bool
  | none, exclusive => (exclusive, exclusive.isSome)
  | inclusive, none => (inclusive, false)
  | some inclusive, some exclusive =>
      if exclusive.lt inclusive then (some inclusive, false) else (some exclusive, true)

/-- Pick the stronger upper restriction; a numerical tie keeps the open endpoint. -/
def effectiveUpper : Option Decimal → Option Decimal → Option Decimal × Bool
  | none, exclusive => (exclusive, exclusive.isSome)
  | inclusive, none => (inclusive, false)
  | some inclusive, some exclusive =>
      if inclusive.lt exclusive then (some inclusive, false) else (some exclusive, true)

/-- Lower all four restrictions into the existing interval vocabulary without loss. -/
def effectiveNumericBounds (bounds : AuthoredNumericBounds) : NumericBounds :=
  let lower := effectiveLower bounds.minimum bounds.exclusiveMinimum
  let upper := effectiveUpper bounds.maximum bounds.exclusiveMaximum
  { minimum := lower.1, exclusiveMinimum := lower.2
    maximum := upper.1, exclusiveMaximum := upper.2 }

private theorem decimal_le_of_not_lt {a b : Decimal} (h : a.lt b = false) : b.le a = true := by
  simp only [Decimal.lt, Decimal.le, decide_eq_false_iff_not, decide_eq_true_eq] at *
  omega

private theorem decimal_le_of_lt {a b : Decimal} (h : a.lt b = true) : a.le b = true := by
  simp only [Decimal.lt, Decimal.le, decide_eq_true_eq] at *
  omega

private theorem effectiveLower_iff (inclusive exclusive : Option Decimal) (value : Decimal) :
    (effectiveLower inclusive exclusive).1.all (fun lower =>
      if (effectiveLower inclusive exclusive).2 then lower.lt value else lower.le value) = true ↔
    (∀ lower ∈ inclusive, lower.le value = true) ∧
    (∀ lower ∈ exclusive, lower.lt value = true) := by
  cases inclusive with
  | none => cases exclusive <;> simp [effectiveLower]
  | some a =>
    cases exclusive with
    | none => simp [effectiveLower]
    | some b =>
      cases h : b.lt a with
      | true =>
        simp only [effectiveLower, h, ↓reduceIte, Option.all_some,
          Option.mem_some_iff, forall_eq']
        exact ⟨fun ha => ⟨ha, decimal_lt_of_lt_of_le h ha⟩, And.left⟩
      | false =>
        simp only [effectiveLower, h,
          Option.mem_some_iff, forall_eq']
        exact ⟨fun hb => ⟨decimal_le_of_lt (decimal_lt_of_le_of_lt
          (decimal_le_of_not_lt h) hb), hb⟩, And.right⟩

private theorem effectiveUpper_iff (inclusive exclusive : Option Decimal) (value : Decimal) :
    (effectiveUpper inclusive exclusive).1.all (fun upper =>
      if (effectiveUpper inclusive exclusive).2 then value.lt upper else value.le upper) = true ↔
    (∀ upper ∈ inclusive, value.le upper = true) ∧
    (∀ upper ∈ exclusive, value.lt upper = true) := by
  cases inclusive with
  | none => cases exclusive <;> simp [effectiveUpper]
  | some a =>
    cases exclusive with
    | none => simp [effectiveUpper]
    | some b =>
      cases h : a.lt b with
      | true =>
        simp only [effectiveUpper, h, ↓reduceIte, Option.all_some,
          Option.mem_some_iff, forall_eq']
        exact ⟨fun ha => ⟨ha, decimal_lt_of_le_of_lt ha h⟩, And.left⟩
      | false =>
        simp only [effectiveUpper, h,
          Option.mem_some_iff, forall_eq']
        exact ⟨fun hb => ⟨decimal_le_of_lt (decimal_lt_of_lt_of_le hb
          (decimal_le_of_not_lt h)), hb⟩, And.right⟩

/-- For every Decimal and all four optional authored bounds, including contradictory
intervals, the existing executable interval check equals the authored conjunction. -/
theorem effectiveNumericBounds_iff (bounds : AuthoredNumericBounds) (value : Decimal) :
    numberAllowed (effectiveNumericBounds bounds) value = true ↔ bounds.Allows value := by
  unfold numberAllowed effectiveNumericBounds
  dsimp only
  rw [Bool.and_eq_true, effectiveLower_iff, effectiveUpper_iff]
  unfold AuthoredNumericBounds.Allows
  grind only

/-- The known inclusive-five/exclusive-one overwrite admits three incorrectly. -/
theorem legacy_lower_overwrite_counterexample :
    let authored : AuthoredNumericBounds :=
      { minimum := some ⟨5, 0⟩, exclusiveMinimum := some ⟨1, 0⟩ }
    numberAllowed (legacyNumericLowering authored) ⟨3, 0⟩ = true ∧
      ¬ authored.Allows ⟨3, 0⟩ ∧
      numberAllowed (effectiveNumericBounds authored) ⟨3, 0⟩ = false := by
  dsimp only
  refine ⟨by decide, ?_, by decide⟩
  simp [AuthoredNumericBounds.Allows, Decimal.le, Decimal.fraction]

/-- The dual maximum overwrite is independently unsound. -/
theorem legacy_upper_overwrite_counterexample :
    let authored : AuthoredNumericBounds :=
      { maximum := some ⟨1, 0⟩, exclusiveMaximum := some ⟨5, 0⟩ }
    numberAllowed (legacyNumericLowering authored) ⟨3, 0⟩ = true ∧
      ¬ authored.Allows ⟨3, 0⟩ ∧
      numberAllowed (effectiveNumericBounds authored) ⟨3, 0⟩ = false := by
  dsimp only
  refine ⟨by decide, ?_, by decide⟩
  simp [AuthoredNumericBounds.Allows, Decimal.le, Decimal.fraction]

/-- Numerically equal endpoints need not have equal Decimal representations. -/
theorem equivalent_decimal_ties_remain_open :
    effectiveNumericBounds { minimum := some ⟨10, -1⟩, exclusiveMinimum := some ⟨1, 0⟩, maximum := some ⟨20, -1⟩, exclusiveMaximum := some ⟨2, 0⟩ } =
      { minimum := some ⟨1, 0⟩, maximum := some ⟨2, 0⟩,
        exclusiveMinimum := true, exclusiveMaximum := true } := by decide

/-- Boundary controls include an admitted contradictory interval. -/
theorem effective_numeric_boundary_controls :
    numberAllowed (effectiveNumericBounds {}) ⟨-17, -2⟩ = true ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨1, 0⟩, maximum := some ⟨10, -1⟩ }) ⟨1, 0⟩ = true ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨1, 0⟩, exclusiveMaximum := some ⟨10, -1⟩ }) ⟨1, 0⟩ = false ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨5, 0⟩, maximum := some ⟨1, 0⟩ }) ⟨3, 0⟩ = false ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨-2, 0⟩, exclusiveMinimum := some ⟨-15, -1⟩,
        maximum := some ⟨0, 0⟩, exclusiveMaximum := some ⟨1, 0⟩ }) ⟨-1, 0⟩ = true ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨5, 0⟩, exclusiveMinimum := some ⟨1, 0⟩,
        maximum := some ⟨9, 0⟩, exclusiveMaximum := some ⟨8, 0⟩ }) ⟨5, 0⟩ = true ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨1, 0⟩, exclusiveMinimum := some ⟨5, 0⟩,
        maximum := some ⟨8, 0⟩, exclusiveMaximum := some ⟨9, 0⟩ }) ⟨5, 0⟩ = false := by decide

end ValueContract.Candidate
