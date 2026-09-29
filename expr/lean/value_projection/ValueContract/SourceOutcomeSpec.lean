import ValueContract.Resolve
import ValueContract.SourceBodySpec
import ValueContract.SourceUnionSpec

namespace ValueContract.Candidate

/-- Independent error reduction validates all entries. A failure must occur in
the original list and have no lower-priority competitor; success preserves the
whole ordered value list. No executable reduction appears in this judgment. -/
def CombinedOutcome (inputs : List (Except Failure α)) : Except Failure (List α) → Prop
  | .ok values => All₂ (fun input value => input = .ok value) inputs values
  | .error failure => .error failure ∈ inputs ∧
      ∀ other, .error other ∈ inputs → failurePriority failure ≤ failurePriority other

/-- Every child contributes an outcome, including nested ambiguity and failure.
A failed child cannot disappear when successes are collected. -/
def ChildrenOutcome (child : α → Except Failure β → Prop) (inputs : List α)
    (result : Except Failure (List β)) : Prop :=
  ∃ outcomes, All₂ child inputs outcomes ∧ CombinedOutcome outcomes result

/-- Mapping a successful payload changes neither the existence nor the exact
identity of a failure. This relation does not execute an Except computation. -/
def MappedOutcome (transform : α → β) (source : Except Failure α)
    (result : Except Failure β) : Prop :=
  match source with
  | .error failure => result = .error failure
  | .ok value => result = .ok (transform value)

def MapChildOutcome (transform : α → β) (child : Except Failure α → Prop)
    (result : Except Failure β) : Prop :=
  ∃ source, child source ∧ MappedOutcome transform source result

/-- A local contract guard has its own rejection before child evaluation. -/
def GuardedOutcome (valid : Prop) (next : Except Failure α → Prop)
    (result : Except Failure α) : Prop :=
  (valid ∧ next result) ∨ (¬ valid ∧ result = .error .invalid)

/-- A prerequisite's failure is observable before the subsequent rule. Its
successful payload supplies evidence for the next rule without resynthesis. -/
def SequencedOutcome (first : Except Failure α → Prop)
    (next : α → Except Failure β → Prop) (result : Except Failure β) : Prop :=
  ∃ initial, first initial ∧ match initial with
    | .error failure => result = .error failure
    | .ok value => next value result

/-- An enum constrains a successfully interpreted whole node. Body failures
propagate before enum comparison; no branch may bypass the whole-node gate. -/
def NodeEnumOutcome (enumeration : Option (List Value)) (body : ResolveResult)
    (result : ResolveResult) : Prop :=
  match body with
  | .error failure => result = .error failure
  | .ok value => (EnumAllows enumeration value.value ∧ result = .ok value) ∨
      (¬ EnumAllows enumeration value.value ∧ result = .error .invalid)

/-- Source preference follows same-input declaration wrappers and tests known
object names. Non-object declarations are preferred for object-shaped input,
matching the established Any/map/nested-union rule. The index is a derivation
height, not a fixed implementation cutoff. -/
def ObjectPreferenceAt : Nat → Declarations → Identity → Input → Prop
  | 0, _, _, _ => False
  | depth + 1, declarations, identity, input =>
    ∃ entries, ObjectInputEntries input entries ∧
      ∃ declaration ∈ declarations, declaration.identity = identity ∧
        match declaration.contract with
        | .alias child | .nullable child | .nonNull child =>
          ObjectPreferenceAt depth declarations child input
        | .object members _ => ∃ member ∈ members, ∃ entry ∈ entries,
            entry.1 = member.sourceName ∨ entry.1 = member.wireAlias
        | _ => True

def ObjectPreference (declarations : Declarations) (identity : Identity) (input : Input) : Prop :=
  ∃ depth, ObjectPreferenceAt depth declarations identity input

end ValueContract.Candidate
