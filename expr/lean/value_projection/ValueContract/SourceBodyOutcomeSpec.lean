import ValueContract.SourceMemberOutcomeSpec
import ValueContract.SourceRawOutcomeSpec

namespace ValueContract.Candidate

/-- Explicit source cycle/opaque markers have precedence over the declared
contract, even when a shape mismatch would otherwise be invalid. -/
def SourceGuard (input : Input) (next : ResolveResult → Prop) (result : ResolveResult) : Prop :=
  match stripHostInput input with
  | .cycle _ => result = .error .cyclic
  | .opaque _ => result = .error .unsupported
  | _ => next result

/-- Scalar rejection is the complement of the established independent scalar
matching relation; no unsupported conversion is silently added. -/
def ScalarOutcome (checks : ExternalScalarChecks) (kind : ScalarKind) (rules : ScalarRules)
    (input : Input) : ResolveResult → Prop
  | .ok value => ScalarResolution checks kind rules input value
  | .error failure => failure = .invalid ∧ ¬ ∃ value, ScalarResolution checks kind rules input value

/-- Array failures retain all indexed child outcomes and their priority.
Success carries the exact ordered concatenation of missing paths. -/
def ArrayOutcome (child : Input → ResolveResult → Prop) (bounds : LengthBounds)
    (input : Input) (result : ResolveResult) : Prop :=
  match stripHostInput input with
  | .nilArray => GuardedOutcome (lengthAllowed bounds 0 = true)
      (fun output => output = .ok ⟨.nilArray, []⟩) result
  | .array items => GuardedOutcome (lengthAllowed bounds items.length = true)
      (MapChildOutcome (fun values => {
        value := .array (values.map Resolution.value)
        missing := values.flatMap Resolution.missing }) (ChildrenOutcome child items)) result
  | _ => result = .error .invalid

/-- Explicit branch evidence is checked against occurrence ownership and the
declared branch table. Its payload is validated without implicit reselection. -/
def SelectedUnionOutcome (child : Identity → Input → ResolveResult → Prop)
    (occurrence : Identity) (alternatives : List Alternative)
    (actual branch : Identity) (payload : Input) (result : ResolveResult) : Prop :=
  GuardedOutcome (actual = occurrence)
    (fun output =>
      (∃ alternative, FirstAlternative branch alternatives alternative ∧
        MapChildOutcome (fun value => {value with value := .union occurrence branch value.value})
          (child alternative.child payload) output) ∨
      ((¬ ∃ alternative, FirstAlternative branch alternatives alternative) ∧
        output = .error .invalid)) result

end ValueContract.Candidate
