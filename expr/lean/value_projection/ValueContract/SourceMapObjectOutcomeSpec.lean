import ValueContract.SourceBodyOutcomeSpec

namespace ValueContract.Candidate

/-- Extraction has only a shape rejection. The successful entries retain all
supplied names and child inputs, including duplicates for later validation. -/
def EntriesOutcome (entries : List α → Prop) : Except Failure (List α) → Prop
  | .ok values => entries values
  | .error failure => failure = .invalid ∧ ¬ ∃ values, entries values

/-- Local key normalization can only reject invalid coercion/constraints;
recursive child failures are evaluated later and retain their own identity. -/
def MapKeyOutcome (checks : ExternalScalarChecks) (kind : MapKeyKind) (rules : ScalarRules)
    (entry : Scalar × Input) : Except Failure (Scalar × Input) → Prop
  | .ok actual => MapKeyResolution checks kind rules entry actual
  | .error failure => failure = .invalid ∧ ¬ ∃ actual, MapKeyResolution checks kind rules entry actual

/-- Complete map outcomes keep every intermediate key before spelling and
semantic collision checks, then retain every child outcome and missing path. -/
def MapOutcome (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (bounds : LengthBounds) (child : Input → ResolveResult → Prop)
    (input : Input) (result : ResolveResult) : Prop :=
  match stripHostInput input with
  | .nilMap => GuardedOutcome (lengthAllowed bounds 0 = true)
      (fun output => output = .ok ⟨.nilMap, []⟩) result
  | _ => SequencedOutcome (EntriesOutcome (MapInputEntries input))
      (fun entries => GuardedOutcome (lengthAllowed bounds entries.length = true)
        (SequencedOutcome (ChildrenOutcome (MapKeyOutcome checks kind rules) entries)
          (fun normalized => GuardedOutcome (AdmissibleKeys keys (normalized.map Prod.fst))
            (GuardedOutcome (normalized.Pairwise (fun left right => scalarEqual left.1 right.1 = false))
              (MapChildOutcome (fun values : List (Scalar × Resolution) => {
                value := .map (values.map (fun entry => (entry.1, entry.2.value)))
                missing := values.flatMap (fun entry => entry.2.missing) })
                (ChildrenOutcome (fun entry => MapChildOutcome (entry.1, ·) (child entry.2)) normalized)))))) result

/-- Grouped reduction compares all field and extra failures before constructing
values. Payload erasure is relational; failures retain their exact identity. -/
def GroupedOutcome (fields : List (Except Failure α)) (extras : List (Except Failure β))
    (pack : List α → List β → γ) (result : Except Failure γ) : Prop :=
  ∃ fieldUnits extraUnits,
    All₂ (MappedOutcome (fun _ : α => ())) fields fieldUnits ∧
    All₂ (MappedOutcome (fun _ : β => ())) extras extraUnits ∧
    SequencedOutcome (CombinedOutcome (fieldUnits ++ extraUnits))
      (fun _ => SequencedOutcome (CombinedOutcome fields)
        (fun values => MapChildOutcome (pack values) (CombinedOutcome extras))) result

/-- Full object outcomes preserve all supplied alias validation, open-field raw
validation, deterministic error priority and exact ordered missing-member paths. -/
def ObjectOutcome (keys : KeyCodec) (rawDepth : Nat) (complete : Bool)
    (members : List Member) (isOpen : Bool)
    (child : Identity → Input → ResolveResult → Prop) (input : Input) (result : ResolveResult) : Prop :=
  SequencedOutcome (EntriesOutcome (ObjectInputEntries input))
    (fun entries => GuardedOutcome (entries.map Prod.fst).Nodup
      (GuardedOutcome (isOpen = true ∨ AdditionalEntries members entries = [])
        (fun output => ∃ fieldOutcomes extraOutcomes,
          All₂ (fun member => MemberOutcome child complete member entries) members fieldOutcomes ∧
          All₂ (fun entry => MapChildOutcome (entry.1, ·) (RawOutcomeAt keys rawDepth entry.2))
            (AdditionalEntries members entries) extraOutcomes ∧
          GroupedOutcome fieldOutcomes extraOutcomes
            (fun fields extras => {
              value := .object (fields.map (fun entry => (entry.1, entry.2.value))) extras
              missing := fields.flatMap (fun entry => entry.2.missing) }) output))) result

end ValueContract.Candidate
