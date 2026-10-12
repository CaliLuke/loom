import ValueContract.NumericIntervals

open ValueContract.Candidate.NumericIntervals

-- Expected failure: matching rounded lower endpoints do not prove equality.
example : (enclosure 10 (.leaf 1001)).1 = (enclosure 10 (.leaf 1000)).1 →
    value (.leaf 1001) = value (.leaf 1000) := by
  decide
