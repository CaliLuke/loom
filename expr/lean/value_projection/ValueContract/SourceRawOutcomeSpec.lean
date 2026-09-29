import ValueContract.SourceOutcomeSpec

namespace ValueContract.Candidate

/-- All raw-data outcomes, including failures, have independent recursive
derivations. Guards preserve their ordering, while children are reduced only
after all their outcomes are retained. The index is derivation height. -/
def RawOutcomeAt (keys : KeyCodec) : Nat → Input → Except Failure Value → Prop
  | 0, _, result => result = .error .malformedDeclaration
  | depth + 1, input, result =>
    match input with
    | .host identity payload => MapChildOutcome (.host identity)
        (RawOutcomeAt keys depth payload) result
    | .absent => result = .error .invalid
    | .null => result = .ok .null
    | .nilBytes => result = .ok .nilBytes
    | .nilArray => result = .ok .nilArray
    | .nilMap => result = .ok .nilMap
    | .scalar scalar => result = .ok (.scalar scalar.value)
    | .byteSequence sequence => ∃ value, NativeByteValue sequence value ∧ result = .ok value
    | .cycle _ => result = .error .cyclic
    | .opaque _ | .selected _ _ _ => result = .error .unsupported
    | .array items => MapChildOutcome Value.array
        (ChildrenOutcome (RawOutcomeAt keys depth) items) result
    | .object fields => GuardedOutcome (fields.map Prod.fst).Nodup
        (MapChildOutcome (Value.object [])
          (ChildrenOutcome (fun entry => MapChildOutcome (entry.1, ·)
            (RawOutcomeAt keys depth entry.2)) fields)) result
    | .map entries => GuardedOutcome (AdmissibleKeys keys (entries.map (fun entry => entry.1.value)))
        (MapChildOutcome Value.map
          (ChildrenOutcome (fun entry => MapChildOutcome (entry.1.value, ·)
            (RawOutcomeAt keys depth entry.2)) entries)) result

/-- Adequate derivations alone define semantic raw outcomes. Exhausted indexed
runs are excluded, and the exact answer is independent of the chosen height. -/
def RawOutcome (keys : KeyCodec) (input : Input) (result : Except Failure Value) : Prop :=
  ∃ depth, inputDepth input ≤ depth ∧ RawOutcomeAt keys depth input result


end ValueContract.Candidate
