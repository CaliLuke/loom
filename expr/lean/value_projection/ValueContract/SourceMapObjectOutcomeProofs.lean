import ValueContract.SourceMapObjectOutcomeSpec
import ValueContract.SourceBodyProofs
import ValueContract.SourceMemberOutcomeProofs
import ValueContract.SourceRawOutcomeProofs

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

private theorem sequenced_graph (first : Except Failure α) (next : α → Except Failure β) :
    SequencedOutcome (fun output => first = output) (fun value output => next value = output) =
      (fun output => (first >>= next) = output) := by
  funext output
  exact propext (sequencedOutcome_iff first next output).symm

private theorem guarded_graph (condition : Prop) [Decidable condition] (next : Except Failure α) :
    GuardedOutcome condition (fun output => next = output) =
      (fun output => (if condition then next else .error .invalid) = output) := by
  funext output
  split <;> simp_all [GuardedOutcome, eq_comm]

private theorem combined_graph (inputs : List (Except Failure α)) :
    CombinedOutcome inputs = (fun output => combineChecked inputs = output) := by
  funext output
  exact propext (combineChecked_outcome_iff inputs output).symm

private theorem mapped_graph (transform : α → β) (input : Except Failure α) :
    MappedOutcome transform input = (fun output => Except.map transform input = output) := by
  funext output
  cases input <;> simp [MappedOutcome, Except.map, eq_comm]

private def extractEntries (input : Option (List α)) : Except Failure (List α) :=
  match input with
  | some values => .ok values
  | none => .error .invalid

private theorem entries_graph (input : Option (List α)) (meaning : List α → Prop)
    (correct : ∀ values, input = some values ↔ meaning values) :
    EntriesOutcome meaning = (fun output => extractEntries input = output) := by
  funext output
  apply propext
  symm
  cases input with
  | none =>
    have absent (values : List α) : ¬ meaning values := by
      rw [← correct]
      simp
    cases output <;> simp [extractEntries, EntriesOutcome, absent, eq_comm]
  | some values =>
    have chosen (other : List α) : meaning other ↔ values = other := by
      rw [← correct]
      simp
    cases output <;> simp [extractEntries, EntriesOutcome, chosen, eq_comm]

private def normalizeKey (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (entry : SourceScalar × Input) : Except Failure (Scalar × Input) := do
  let key ← match kind with
    | .builtin => .ok entry.1.value
    | .scalar expected => match coerceSourceScalar keys rules expected entry.1 with
      | some key => .ok key
      | none => .error .invalid
  if !mapKeyCompatible kind key || !scalarAllowed checks rules key then throw .invalid
  return (key, entry.2)

private theorem normalizedKey_graph (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (entry : SourceScalar × Input) :
    MapKeyOutcome checks keys kind rules entry = (fun output => normalizeKey checks keys kind rules entry = output) := by
  have success : ∀ value, normalizeKey checks keys kind rules entry = .ok value ↔
      MapKeyResolution checks keys kind rules entry value := by
    intro value
    rw [← mapKeyResolution_iff]
    cases kind <;> simp [normalizeKey, Bind.bind, Except.bind, exceptThrow]
    split <;> simp_all
  have failure : ∀ failure, normalizeKey checks keys kind rules entry = .error failure → failure = .invalid := by
    intro failure failed
    cases kind <;> simp only [normalizeKey, Bind.bind, Except.bind] at failed
    all_goals split at failed <;> try simp_all only [exceptThrow, Except.error.injEq]
    all_goals try (split at failed <;> simp_all)
    all_goals cases failed
  funext output
  apply propext
  symm
  have correct := invalidOtherwise_iff (normalizeKey checks keys kind rules entry)
    (MapKeyResolution checks keys kind rules entry) success failure output
  cases output <;> exact correct

private theorem keys_guard_graph (keys : KeyCodec) (values : List Scalar) (next : Except Failure α) :
    GuardedOutcome (AdmissibleKeys keys values) (fun output => next = output) =
      (fun output => (do let _ ← nameKeys keys values; next) = output) := by
  funext output
  cases result : nameKeys keys values with
  | error failure =>
    have same := nameKeys_error_invalid result
    subst failure
    have rejected := (nameKeys_invalid_iff _ _).mp result
    simp [GuardedOutcome, rejected, Bind.bind, Except.bind, eq_comm]
  | ok names =>
    have accepted : AdmissibleKeys keys values := ⟨names, (nameKeys_iff _ _ _).mp result⟩
    simp [GuardedOutcome, accepted, Bind.bind, Except.bind]

private def mapWork (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (bounds : LengthBounds) (input : Input)
    (child : Input → ResolveResult) : ResolveResult := do
  let entries ← match mapEntries input with
    | some entries => .ok entries
    | none => .error .invalid
  if !lengthAllowed bounds entries.length then throw .invalid
  let normalized ← combineChecked (entries.map (normalizeKey checks keys kind rules))
  let _ ← nameKeys keys (normalized.map Prod.fst)
  let values ← combineChecked (normalized.map fun entry => do
    let value ← child entry.2
    return (entry.1, value))
  return {
    value := .map (values.map (fun entry => (entry.1, entry.2.value)))
    missing := values.flatMap (fun entry => entry.2.missing) }

private theorem mapWork_graph (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (bounds : LengthBounds) (input : Input) (child : Input → ResolveResult) :
    SequencedOutcome (EntriesOutcome (MapInputEntries input))
      (fun entries => GuardedOutcome (lengthAllowed bounds entries.length = true)
        (SequencedOutcome (ChildrenOutcome (MapKeyOutcome checks keys kind rules) entries)
          (fun normalized => GuardedOutcome (AdmissibleKeys keys (normalized.map Prod.fst))
            (MapChildOutcome (fun values : List (Scalar × Resolution) => {
                value := .map (values.map (fun entry => (entry.1, entry.2.value)))
                missing := values.flatMap (fun entry => entry.2.missing) })
                (ChildrenOutcome (fun entry => MapChildOutcome (entry.1, ·)
                  (fun output => child entry.2 = output)) normalized))))) =
      (fun output => mapWork checks keys kind rules bounds input child = output) := by
  have extraction := entries_graph (mapEntries input) (MapInputEntries input) (mapEntries_iff input)
  have keyMeaning : MapKeyOutcome checks keys kind rules =
      (fun entry output => normalizeKey checks keys kind rules entry = output) := by
    funext entry; exact normalizedKey_graph checks keys kind rules entry
  simp only [extraction, keyMeaning, mapChildOutcome_graph, childrenOutcome_graph,
    guarded_graph, keys_guard_graph, sequenced_graph]
  funext output
  apply congrArg (fun computation : ResolveResult => computation = output)
  unfold mapWork extractEntries
  cases extracted : mapEntries input with
  | none => simp [Bind.bind, Except.bind]
  | some entries =>
    cases length : lengthAllowed bounds entries.length <;>
      simp [length, exceptThrow, Bind.bind, Except.bind]

/-- Every map-body outcome agrees with independent staged extraction, key
normalization/collision evidence and complete child-outcome reduction. -/
theorem resolveBody_map_outcome_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (kind : MapKeyKind)
    (rules : ScalarRules) (child : Identity) (bounds : LengthBounds) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.map kind rules child bounds)
      input same descend = result ↔
      SourceGuard input (MapOutcome checks keys kind rules bounds
        (fun input output => descend complete child input = output) input) result := by
  unfold SourceGuard MapOutcome
  simp only [mapWork_graph]
  cases shape : stripHostInput input <;>
    simp only [resolveBody, shape]
  all_goals try exact eq_comm
  all_goals try rfl
  case nilMap =>
    cases length : lengthAllowed bounds 0 <;> simp [GuardedOutcome, eq_comm]

private theorem grouped_graph (fields : List (Except Failure α)) (extras : List (Except Failure β))
    (pack : List α → List β → γ) :
    GroupedOutcome fields extras pack = (fun result => (do
      let _ ← combineChecked ((fields.map (Except.map (fun _ => ()))) ++
        (extras.map (Except.map (fun _ => ()))))
      let fieldValues ← combineChecked fields
      let extraValues ← combineChecked extras
      pure (pack fieldValues extraValues)) = result) := by
  have fieldsMeaning : MappedOutcome (fun _ : α => ()) =
      (fun input output => Except.map (fun _ => ()) input = output) := by
    funext input; exact mapped_graph _ input
  have extrasMeaning : MappedOutcome (fun _ : β => ()) =
      (fun input output => Except.map (fun _ => ()) input = output) := by
    funext input; exact mapped_graph _ input
  funext result
  simp only [GroupedOutcome, fieldsMeaning, extrasMeaning, all₂_graph_iff, combined_graph,
    mapChildOutcome_graph, sequenced_graph]
  apply propext
  constructor
  · rintro ⟨fieldUnits, extraUnits, rfl, rfl, done⟩
    exact done
  · intro done
    exact ⟨_, _, rfl, rfl, done⟩

private theorem objectOutcomes_graph (keys : KeyCodec) (depth : Nat) (complete : Bool)
    (members : List Member) (entries : List (String × Input)) (child : Identity → Input → ResolveResult) :
    (fun output : ResolveResult => ∃ fieldOutcomes extraOutcomes,
      All₂ (fun member => MemberOutcome (fun identity input result => child identity input = result)
        complete member entries) members fieldOutcomes ∧
      All₂ (fun entry => MapChildOutcome (entry.1, ·) (RawOutcomeAt keys depth entry.2))
        (AdditionalEntries members entries) extraOutcomes ∧
      GroupedOutcome fieldOutcomes extraOutcomes
        (fun fields extras => {
          value := .object (fields.map (fun entry => (entry.1, entry.2.value))) extras
          missing := fields.flatMap (fun entry => entry.2.missing) }) output) =
    (fun output : ResolveResult => (do
      let fields := members.map (objectMemberResult complete child entries)
      let extras := (AdditionalEntries members entries).map (fun entry => do
        let value ← resolveRawAt keys depth entry.2
        return (entry.1, value))
      let _ ← combineChecked ((fields.map (Except.map (fun _ => ()))) ++
        (extras.map (Except.map (fun _ => ()))))
      let fieldValues ← combineChecked fields
      let extraValues ← combineChecked extras
      return {
        value := .object (fieldValues.map (fun entry => (entry.1, entry.2.value))) extraValues
        missing := fieldValues.flatMap (fun entry => entry.2.missing) }) = output) := by
  have memberGraph (member : Member) :
      MemberOutcome (fun identity input result => child identity input = result) complete member entries =
        (fun result => objectMemberResult complete child entries member = result) := by
    funext result
    exact propext (objectMemberResult_outcome_iff complete child entries member result).symm
  have rawGraph (input : Input) : RawOutcomeAt keys depth input =
      (fun output => resolveRawAt keys depth input = output) := by
    funext output
    exact propext (resolveRawAt_outcome_iff keys depth input output).symm
  funext output
  simp only [memberGraph, rawGraph, mapChildOutcome_graph, all₂_graph_iff, grouped_graph]
  apply propext
  constructor
  · rintro ⟨fields, extras, rfl, rfl, done⟩
    exact done
  · intro done
    exact ⟨_, _, rfl, rfl, done⟩

/-- Every object-body outcome validates all aliases and additional entries
before selecting the global error, preserving exact member identities and paths. -/
theorem resolveBody_object_outcome_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (members : List Member) (isOpen : Bool)
    (input : Input) (same descend : Bool → Identity → Input → ResolveResult) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.object members isOpen)
      input same descend = result ↔
      SourceGuard input (ObjectOutcome keys depth complete members isOpen
        (fun identity input output => descend complete identity input = output) input) result := by
  have extraction := entries_graph (objectEntries input) (ObjectInputEntries input) (objectEntries_iff input)
  unfold SourceGuard ObjectOutcome
  simp only [extraction, objectOutcomes_graph, guarded_graph, sequenced_graph]
  cases shape : stripHostInput input
  case cycle | «opaque» => simp [resolveBody, shape, eq_comm]
  all_goals
    simp only [resolveBody, shape]
    apply Iff.of_eq
    apply congrArg (fun computation : ResolveResult => computation = result)
    simp only [extractEntries]
    cases extracted : objectEntries input with
    | none => rfl
    | some entries =>
      simp only [Bind.bind, Except.bind]
      by_cases unique : (entries.map Prod.fst).Nodup
      · simp only [unique, decide_true, Bool.not_true, Bool.false_eq_true, ↓reduceIte]
        rw [additionalEntries_iff]
        by_cases admitted : isOpen = true ∨ AdditionalEntries members entries = []
        · have openGuard : (!isOpen && !(AdditionalEntries members entries).isEmpty) = false := by
            rcases admitted with openValue | noExtra <;> simp [*]
          simp [admitted, openGuard]
        · have openGuard : (!isOpen && !(AdditionalEntries members entries).isEmpty) = true := by
            simp_all
          simp [admitted, openGuard, exceptThrow]
      · simp [unique, exceptThrow]

end ValueContract.Candidate
