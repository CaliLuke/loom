import ValueContract.SourceAdmissionControls

namespace ValueContract.Candidate.NumericValueAdmission

/-- Retain the former extraction path as a counterexample: source type plus
normalization alone accepted an integer outside its declared carrier. -/
theorem former_integer_carrier_omission :
    sourceAdmits .int32 .builtinInt = true ∧
    coerceScalar NumericCoercionControls.numbers .exact .integer (.integer 1099511627776) =
      some (.integer 1099511627776) ∧
    integerCarrierAllows (.signed 32) 1099511627776 = false := by
  decide

theorem successful_source_retains_carrier (codec rules kind source output)
    (success : coerceSourceScalar codec rules kind source = some output) :
    SourceAdmits (rules.sourcePrimitive.getD (.abstract kind)) source.primitive ∧
      sourceCarrierAllows rules kind source.value = true := by
  obtain ⟨admitted, carrier, _⟩ := (sourceCoercion_iff ..).mp success
  exact ⟨admitted, carrier⟩

theorem source_rejection_cannot_be_repaired_by_range (codec rules kind source)
    (rejected : sourceAdmits (rules.sourcePrimitive.getD (.abstract kind)) source.primitive = false) :
    coerceSourceScalar codec rules kind source = none := by
  simp [coerceSourceScalar, rejected]

theorem original_integer_failures_rejected :
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int32, integerFormat := .signed 32}
      .integer ⟨.integer 1099511627776, .builtinInt⟩ = none ∧
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .uint, integerFormat := .unsigned 64}
      .integer ⟨.integer (-5), .builtinInt⟩ = none := by
  decide

theorem exact_boundaries_and_source_controls :
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int32, integerFormat := .signed 32}
      .integer ⟨.integer 2147483647, .builtinInt⟩ = some (.integer 2147483647) ∧
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .uint64, integerFormat := .unsigned 64}
      .integer ⟨.integer 18446744073709551615, .builtinUInt64⟩ = some (.integer 18446744073709551615) ∧
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int32, integerFormat := .signed 32}
      .integer ⟨.integer 1, .builtinInt64⟩ = none := by
  decide

/-- Width extraction is an independently tested Go boundary, not always 64. -/
theorem machine_width_changes_admission :
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int, integerFormat := .signed 32}
      .integer ⟨.integer 2147483648, .builtinInt⟩ = none ∧
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int, integerFormat := .signed 64}
      .integer ⟨.integer 2147483648, .builtinInt⟩ = some (.integer 2147483648) := by
  decide

/-- Carrier admission is not a constraint on the authored bound's range. -/
theorem redundant_bound_does_not_invalidate_value :
    coerceSourceScalar NumericCoercionControls.numbers
      {sourcePrimitive := some .int32, integerFormat := .signed 32,
        numeric := {maximum := some ⟨1000000000000, 0⟩}}
      .integer ⟨.integer 1, .builtinInt⟩ = some (.integer 1) ∧
    numberAllowed {maximum := some ⟨1000000000000, 0⟩} ⟨1, 0⟩ = true ∧
    integerCarrierAllows (.signed 32) 1000000000000 = false := by
  decide

end ValueContract.Candidate.NumericValueAdmission
