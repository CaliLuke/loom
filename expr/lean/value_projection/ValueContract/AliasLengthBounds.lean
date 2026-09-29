/-
Independent normalization of cumulative byte-length constraints along an alias
chain. This concerns present, non-null byte values only. Alias identity, source
selection, nullable bypass, enums and codec ownership remain separate contracts.
-/
import ValueContract.ScalarSemantics

namespace ValueContract.Candidate

/-- Intersect semantic decoded-length bounds without clamping negative limits.
An empty intersection remains a constraint that rejects every natural length. -/
def intersectLengthBounds (left right : LengthBounds) : LengthBounds :=
  { minimum := match left.minimum, right.minimum with
      | some a, some b => some (max a b)
      | some a, none => some a
      | none, bound => bound
    maximum := match left.maximum, right.maximum with
      | some a, some b => some (min a b)
      | some a, none => some a
      | none, bound => bound }

/-- Intersection preserves exactly both authored constraints for every decoded
length, including absent, negative and mutually inconsistent bounds. -/
theorem intersectLengthBounds_iff (left right : LengthBounds) (length : Nat) :
    lengthAllowed (intersectLengthBounds left right) length = true ↔
      lengthAllowed left length = true ∧ lengthAllowed right length = true := by
  rcases left with ⟨lmin, lmax⟩
  rcases right with ⟨rmin, rmax⟩
  cases lmin <;> cases lmax <;> cases rmin <;> cases rmax <;>
    simp [intersectLengthBounds, lengthAllowed] <;> omega

/-- Fold the independently captured occurrence-local constraints in an alias
chain. The input must come from declarations, never a successful resolved value. -/
def aliasLengthBounds (constraints : List LengthBounds) : LengthBounds :=
  constraints.foldr intersectLengthBounds {}

/-- Normalizing an arbitrary finite alias chain neither drops a local bound nor
introduces an additional constraint on decoded byte length. -/
theorem aliasLengthBounds_iff (constraints : List LengthBounds) (length : Nat) :
    lengthAllowed (aliasLengthBounds constraints) length = true ↔
      ∀ bounds ∈ constraints, lengthAllowed bounds length = true := by
  induction constraints with
  | nil => simp [aliasLengthBounds, lengthAllowed]
  | cons bounds rest ih =>
    simp only [aliasLengthBounds, List.foldr_cons, intersectLengthBounds_iff]
    change (lengthAllowed bounds length = true ∧
      lengthAllowed (aliasLengthBounds rest) length = true) ↔ _
    simp [ih]

/-- Reordering independently conjoined local length constraints is harmless. -/
theorem aliasLengthBounds_reverse (constraints : List LengthBounds) (length : Nat) :
    lengthAllowed (aliasLengthBounds constraints.reverse) length = true ↔
      lengthAllowed (aliasLengthBounds constraints) length = true := by
  simp [aliasLengthBounds_iff]

theorem aliasLengthBounds_negativeLower :
    lengthAllowed (aliasLengthBounds [{ minimum := some (-3) }]) 0 = true := by
  decide

theorem aliasLengthBounds_emptyIntersection :
    ∀ length, lengthAllowed (aliasLengthBounds
      [{ minimum := some 3 }, { maximum := some 2 }]) length = false := by
  intro length
  simp [aliasLengthBounds, intersectLengthBounds, lengthAllowed]
  omega

theorem aliasLengthBounds_negativeUpper :
    ∀ length, lengthAllowed (aliasLengthBounds [{ maximum := some (-1) }])
      length = false := by
  intro length
  simp [aliasLengthBounds, intersectLengthBounds, lengthAllowed]
  omega

end ValueContract.Candidate
