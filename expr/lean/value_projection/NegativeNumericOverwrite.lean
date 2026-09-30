import ValueContract.NumericBounds

open ValueContract.Candidate

-- Deliberately false: the old lowering cannot satisfy the authored conjunction.
example : let bounds : AuthoredNumericBounds :=
    { minimum := some ⟨5, 0⟩, exclusiveMinimum := some ⟨1, 0⟩ }
    numberAllowed (legacyNumericLowering bounds) ⟨3, 0⟩ = true ↔
      bounds.Allows ⟨3, 0⟩ := by
  simp [AuthoredNumericBounds.Allows, legacyNumericLowering, numberAllowed,
    Decimal.le, Decimal.lt, Decimal.fraction]
