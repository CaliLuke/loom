import ValueContract.NumericBounds

set_option exponentiation.threshold 1024

namespace ValueContract.Candidate

/-- Concrete extraction witness, not an implementation of IEEE conversion.
The independent Go tests establish that int64 2^53+1 is rounded to this value. -/
def lostIntegerBound : Decimal := ⟨9007199254740992, 0⟩

/-- Rounding before effective-bound selection admits an integer below the
authored minimum; the existing exact bound owner correctly rejects it. -/
theorem lossy_integer_extraction_counterexample :
    numberAllowed (effectiveNumericBounds
      { minimum := some lostIntegerBound }) ⟨9007199254740992, 0⟩ = true ∧
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨9007199254740993, 0⟩ }) ⟨9007199254740992, 0⟩ = false := by
  decide

/-- The same loss changes an exclusive upper endpoint in the other direction. -/
theorem lossy_exclusive_upper_extraction_counterexample :
    numberAllowed (effectiveNumericBounds
      { exclusiveMaximum := some lostIntegerBound }) ⟨9007199254740992, 0⟩ = false ∧
    numberAllowed (effectiveNumericBounds
      { exclusiveMaximum := some ⟨9007199254740993, 0⟩ }) ⟨9007199254740992, 0⟩ = true := by
  decide

/-- A compact exact decimal retains a nonzero exponent even when a binary
float parser would underflow. Concrete parsing is a separate Go obligation. -/
theorem nonzero_underflow_bound :
    numberAllowed (effectiveNumericBounds
      { minimum := some ⟨1, -400⟩ }) ⟨0, 0⟩ = false := by
  decide

/-- Binary float64 0.1 and authored decimal 0.1 are different mathematical
endpoints. The binary coefficient is independently checked by Go big.Rat. -/
theorem binary_float_is_not_decimal_text :
    Decimal.lt ⟨1, -1⟩ ⟨1000000000000000055511151231257827021181583404541015625, -55⟩ = true := by
  decide

/-- Float32 0.1 retains a different exact boundary from Float64 0.1. -/
theorem float32_boundary_is_not_float64_boundary :
    Decimal.lt ⟨1000000000000000055511151231257827021181583404541015625, -55⟩
      ⟨100000001490116119384765625, -27⟩ = true := by
  decide

/-- Existing ValidationTypesDSL's exclusive 100.1 bound changes acceptance if
short decimal spelling silently replaces the supplied binary float meaning. -/
theorem existing_float_fixture_spelling_changes_exclusive_admission :
    let binary : Decimal := ⟨100099999999999994315658113919198513031005859375, -45⟩
    numberAllowed (effectiveNumericBounds
      { exclusiveMaximum := some binary }) binary = false ∧
    numberAllowed (effectiveNumericBounds
      { exclusiveMaximum := some ⟨1001, -1⟩ }) binary = true := by
  decide

/-- Positive compact decimal comparison. Magnitudes use arbitrary integers;
coefficient alignment depends on input digit lengths, never exponent magnitude.
Production can compare virtual padded digits instead of forming these small
alignment powers. Zero/sign handling and concrete parsing remain separate. -/
def compactPositiveLT (left right : Decimal) : Bool :=
  let ld := (toString left.coefficient.natAbs).length
  let rd := (toString right.coefficient.natAbs).length
  let lm := (ld : Int) + left.exponent
  let rm := (rd : Int) + right.exponent
  if lm == rm then
    left.coefficient * Int.ofNat (10 ^ rd) < right.coefficient * Int.ofNat (10 ^ ld)
  else lm < rm

/-- Bounded independent comparison with the exact rational Decimal owner:
different spellings, both ordering directions, equal magnitude and exponent. -/
theorem compact_decimal_comparison_controls :
    let values : List Decimal := [⟨1, -2⟩, ⟨10, -3⟩, ⟨125, -2⟩,
      ⟨13, -1⟩, ⟨9007199254740993, 0⟩, ⟨9007199254740992, 0⟩]
    values.all (fun left => values.all (fun right =>
      compactPositiveLT left right == left.lt right)) = true := by
  decide

/-- Huge exponents remain compact in the candidate comparison. These are
algorithm controls, not a proof relating all exponents to Decimal.fraction. -/
theorem compact_decimal_huge_exponent_controls :
    compactPositiveLT ⟨1, -1000000000000000000000000000000⟩
      ⟨1, -100000000000000000000000000000⟩ = true ∧
    compactPositiveLT ⟨10, -1000000000000000000000000000001⟩
      ⟨1, -1000000000000000000000000000000⟩ = false ∧
    compactPositiveLT ⟨1, -1000000000000000000000000000000⟩
      ⟨10, -1000000000000000000000000000001⟩ = false := by
  decide

/-- Preserving extraction makes the existing conjunction theorem apply with
no renderer-specific tolerance. This explicitly assumes extraction equality;
the theorem does not prove a Go parser or a JSON/YAML encoder. -/
theorem exact_extraction_preserves_authored_contract
    (authored extracted : AuthoredNumericBounds) (value : Decimal)
    (preserved : extracted = authored) :
    numberAllowed (effectiveNumericBounds extracted) value = true ↔ authored.Allows value := by
  rw [preserved]
  exact effectiveNumericBounds_iff authored value

/-- The old annotation bypass emits a native float endpoint whose shortest
JSON token fails the exact schema. This checks the artifact obligation rather
than claiming equivalence between raw JSON and native decoding. -/
theorem required_annotation_binary_boundary_counterexample :
    let binary : Decimal := ⟨1000000000000000055511151231257827021181583404541015625, -55⟩
    numberAllowed (effectiveNumericBounds { minimum := some binary }) binary = true ∧
    numberAllowed (effectiveNumericBounds { minimum := some binary }) ⟨1, -1⟩ = false := by
  decide

/-- Clause constants remain predicates when a derived minimum excludes one
member. Checking the base declaration and final default are distinct judgments. -/
theorem inherited_enum_constant_is_not_final_default :
    numberAllowed (effectiveNumericBounds {}) ⟨1, 0⟩ = true ∧
    numberAllowed (effectiveNumericBounds { minimum := some ⟨15, -1⟩ }) ⟨1, 0⟩ = false := by
  decide

/-- A blanket external-codec fallback hides a known builtin failure. Correct
composition requires every known subtree to pass independently. This bounded
control does not prove concrete Go traversal or codec behavior. -/
theorem opaque_annotation_cannot_override_known_failure :
    (true || [false, true].all id) = true ∧ [false, true].all id = false := by
  decide

/-- Nested enum constants carry a declaring field identity. Filtering them by
later child predicates would silently alter the inherited clause. Projection
may rename the wire field, but the independent final conjunction still rejects
1 (minimum) and 3 (enum), and admits 2. Concrete graph extraction is not proved. -/
theorem nested_enum_constants_survive_refinement :
    let constants : List (String × Nat) := [("n", 1), ("n", 2)]
    let projected := constants.map (fun value => ("number", value.2))
    projected = [("number", 1), ("number", 2)] ∧
    (constants.filter (fun value => value.2 ≥ 2)).length = 1 ∧
    ([1, 2, 3].map (fun n => n ≥ 2 && constants.any (fun value => value.2 == n))) =
      [false, true, false] := by
  decide

/-- Distinct captured declaration and target identities must not be compared as
though they were the same occurrence. Target decoding translates a constant's
wire representation to the target identity before native enum comparison. -/
theorem enum_identity_translation_counterexample :
    ((11, "number", 2) == (23, "number", 2)) = false ∧
    ((23, "number", 2) == (23, "number", 2)) = true := by
  decide

end ValueContract.Candidate
