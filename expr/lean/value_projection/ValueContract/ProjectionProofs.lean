import ValueContract.Projection
import ValueContract.RuntimeProofs
import ValueContract.StrictEqualityBudgetProofs

namespace ValueContract.Candidate

/-- The final gate never substitutes schema uniqueness for actual decoding.
This component statement deliberately does not yet assert observation/build
correspondence; its observed value is the builder's output. -/
theorem project_emitted_checks {codecs checks targets use role identity resolved wire}
    (wellFormed : WellFormedTargets targets)
    (emitted : project codecs checks targets use role identity resolved = .emitted wire) :
    ∃ observed, build codecs targets role identity resolved.value = .ok (observed, some wire) ∧
      SchemaAccepts codecs checks targets identity wire ∧
      RuntimeObligation use codecs checks targets identity wire observed := by
  unfold project at emitted
  cases built : build codecs targets role identity resolved.value with
  | error failure => cases failure <;> simp [built, buildFailureResult] at emitted
  | ok result =>
    rcases result with ⟨observed, output⟩
    cases output with
    | none => simp [built] at emitted
    | some candidate =>
      simp only [built] at emitted
      cases checked : schema codecs checks targets identity candidate with
      | error failure => cases failure <;> simp [checked, evaluationFailureResult] at emitted
      | ok accepted =>
        cases accepted with
        | false => simp [checked] at emitted
        | true =>
          simp only [checked] at emitted
          have valid := (schema_iff_SchemaEvaluation wellFormed).mp checked
          cases use with
          | documentation =>
            cases emitted
            exact ⟨observed, rfl, valid, trivial⟩
          | runtime =>
            cases decoded : decode codecs checks targets identity candidate with
            | error failure => cases failure <;> simp [decoded, evaluationFailureResult] at emitted
            | ok result =>
              cases result with
              | none => simp [decoded] at emitted
              | some value =>
                simp only [decoded] at emitted
                split at emitted
                next same =>
                  cases emitted
                  exact ⟨observed, rfl, valid, value,
                    (decode_iff_RuntimeEvaluation wellFormed).mp decoded,
                    _, (valueEqualAt_strict_iff ..).mp same⟩
                next different => contradiction

/-- Once canonical construction has succeeded, independent same-wire validity
and preservation suffice for the final gate. No fixed equality depth is assumed. -/
theorem project_checks_complete {codecs checks targets use role identity resolved wire observed}
    (wellFormed : WellFormedTargets targets)
    (built : build codecs targets role identity resolved.value = .ok (observed, some wire))
    (valid : SchemaAccepts codecs checks targets identity wire)
    (runtime : RuntimeObligation use codecs checks targets identity wire observed) :
    project codecs checks targets use role identity resolved = .emitted wire := by
  have schemaResult := (schema_iff_SchemaEvaluation wellFormed).mpr valid
  unfold project
  simp only [built, schemaResult]
  cases use with
  | documentation => rfl
  | runtime =>
    obtain ⟨decoded, decodedRelation, preserved⟩ := runtime
    have decodedResult := (decode_iff_RuntimeEvaluation wellFormed).mpr decodedRelation
    have same := (valueEqualAt_strict_budget_iff (Nat.le_refl
      (strictValueDepth observed + strictValueDepth decoded + 1))).mpr preserved
    simp [decodedResult, same]

end ValueContract.Candidate
