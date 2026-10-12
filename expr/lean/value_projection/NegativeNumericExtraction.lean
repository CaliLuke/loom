import ValueContract.NumericExtraction

open ValueContract.Candidate

-- Expected failure: the rounded lower endpoint must not admit the integer
-- immediately below the authored 2^53+1 minimum.
example : numberAllowed (effectiveNumericBounds
    { minimum := some lostIntegerBound }) ⟨9007199254740992, 0⟩ = false := by
  decide
