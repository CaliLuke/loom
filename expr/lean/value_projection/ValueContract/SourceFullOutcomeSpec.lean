import ValueContract.SourceMapObjectOutcomeSpec

namespace ValueContract.Candidate

/-- Preference is decided from the independent source-shape derivation. The
budget theorem connects this indexed form to unbounded ObjectPreference. -/
noncomputable def sourcePreferenceAt (declarations : Declarations) (rank : Nat)
    (input : Input) (alternative : Alternative) : Bool := by
  classical
  exact decide (ObjectPreferenceAt rank declarations alternative.child input)

/-- A complete one-node judgment includes all recursive outcomes, not only
successful children. Scalar rules and external codecs remain explicit owners. -/
def SourceBodyOutcome (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (contract : Contract)
    (input : Input) (same descend : Bool → Identity → Input → ResolveResult → Prop)
    (result : ResolveResult) : Prop :=
  SourceGuard input (fun output => match contract with
    | .custom _ => output = .error .unsupported
    | .any => MapChildOutcome (fun value => ⟨.any value, []⟩)
        (RawOutcomeAt keys (depth + 1) input) output
    | .alias child => same complete child input output
    | .nullable child => match stripHostInput input with
      | .null => output = .ok ⟨.null, []⟩
      | _ => same complete child input output
    | .nonNull child => match stripHostInput input with
      | .null => output = .error .invalid
      | _ => same complete child input output
    | .scalar kind rules => ScalarOutcome checks keys kind rules input output
    | .array child bounds => ArrayOutcome (descend complete child) bounds input output
    | .map kind rules child bounds =>
        MapOutcome checks keys kind rules bounds (descend complete child) input output
    | .object members isOpen =>
        ObjectOutcome keys depth complete members isOpen (descend complete) input output
    | .union occurrence alternatives => match stripHostInput input with
      | .selected actual branch payload => SelectedUnionOutcome (descend complete)
          occurrence alternatives actual branch payload output
      | _ => ImplicitUnionOutcome (same true) (same false)
          (sourcePreferenceAt declarations (rank + 1) input)
          complete occurrence alternatives input output) result

/-- Internal indexed derivations expose exhausted runs explicitly. This is not
the public semantic relation: public derivations must satisfy structural bounds. -/
def SourceOutcomeAt (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) : Nat → Nat → Bool → Identity → Input → ResolveResult → Prop
  | 0, _, _, _, _, result => result = .error .malformedDeclaration
  | _, 0, _, _, _, result => result = .error .malformedDeclaration
  | depth + 1, rank + 1, complete, identity, input, result =>
    (∃ declaration ∈ declarations, declaration.identity = identity ∧ ∃ body,
      SourceBodyOutcome declarations checks keys depth rank complete declaration.contract input
        (SourceOutcomeAt declarations checks keys (depth + 1) rank)
        (fun mode child raw => SourceOutcomeAt declarations checks keys depth
          (maximumExpansionRank declarations + 1) mode child raw) body ∧
      NodeEnumOutcome keys declaration.enumeration body result) ∨
    ((¬ ∃ declaration ∈ declarations, declaration.identity = identity) ∧
      result = .error .malformedDeclaration)
termination_by depth rank _ _ _ _ => (depth, rank)

/-- The semantic source relation admits arbitrary finite values via adequate
derivations. Insufficient counters cannot contribute outcomes. Stability above
these bounds makes the existential answer independent of the chosen counters. -/
def SourceOutcome (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (complete : Bool) (identity : Identity) (input : Input)
    (result : ResolveResult) : Prop :=
  ∃ declaration ∈ declarations, declaration.identity = identity ∧ ∃ depth rank,
    inputDepth input ≤ depth ∧ declaration.expansionRank < rank ∧
    SourceOutcomeAt declarations checks keys depth rank complete identity input result

end ValueContract.Candidate
