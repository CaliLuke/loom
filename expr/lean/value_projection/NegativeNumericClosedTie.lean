import ValueContract.NumericBounds

open ValueContract.Candidate

-- Deliberately false: equal-valued inclusive/exclusive endpoints cannot close.
example : numberAllowed (effectiveNumericBounds
    { minimum := some ⟨10, -1⟩, exclusiveMinimum := some ⟨1, 0⟩ }) ⟨1, 0⟩ = true := by
  decide
