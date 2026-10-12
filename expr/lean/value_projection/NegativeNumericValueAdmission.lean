import ValueContract.NumericValueAdmission

open ValueContract.Candidate

-- Deliberately false: source identity alone cannot prove carrier membership.
example : sourceAdmits .int32 .builtinInt = true →
    integerCarrierAllows (.signed 32) 1099511627776 = true := by
  decide
