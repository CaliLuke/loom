import ValueContract.Model

namespace ValueContract.SourceSelection

/-- Ordered example groups supplied by the existing extraction boundary. Each
reference/base group is already extracted using its graph traversal and cycle
rules. This module proves selection over those groups, not DSL graph extraction.
Supplied source identity is preserved; a later resolver's effective occurrence
may differ and is not inferred from the authored source occurrence. -/
structure ExampleSources (α : Type) where
  localExamples : List (Supplied α)
  references : List (List (Supplied α))
  bases : List (List (Supplied α))
  typeExamples : List (Supplied α)

/-- The extraction precedence is local, ordered references, ordered bases, type.
The last example within the first nonempty group overrides earlier examples. -/
def ExampleSources.groups (sources : ExampleSources α) : List (List (Supplied α)) :=
  [sources.localExamples] ++ sources.references ++ sources.bases ++ [sources.typeExamples]

/-- Executable source selection only: no validation, synthesis or materialization
occurs. `absent` means no supplied source; only the separate example eligibility
predicate below permits a later synthesis attempt, never promises success. -/
inductive Choice (α : Type) where
  | excluded
  | suppressed
  | absent
  | selected (supplied : Supplied α)

/-- Select the last authored example from the first nonempty extracted group. -/
def firstAuthored : List (List (Supplied α)) → Option (Supplied α)
  | [] => none
  | group :: rest =>
      match group.getLast? with
      | some value => some value
      | none => firstAuthored rest

/-- Reachability wins before source use. An authored example retains the current
exception to generation-suppression metadata; suppression only blocks synthesis
when no authored source exists. Already-effective exclusion belongs in reachable,
not in suppressGenerated. No raw value, including explicit null, is inspected. -/
def selectExample (reachable suppressGenerated : Bool) (sources : ExampleSources α) :
    Choice α :=
  if reachable then
    match firstAuthored sources.groups with
    | some supplied => .selected supplied
    | none => if suppressGenerated then .suppressed else .absent
  else .excluded

/-- A contract value has no example-suppression input and no synthesis operation.
Call once per enum member or declared default. `none` is absent, whereas
`some ⟨source, RawValue.null⟩` is an explicit supplied contract value. -/
def selectContract (reachable : Bool) (supplied : Option (Supplied α)) : Choice α :=
  if reachable then
    match supplied with
    | some value => .selected value
    | none => .absent
  else .excluded

/-- Independent eligibility for attempting example synthesis. It says nothing
about a generator's termination or ability to produce a valid example. -/
def MaySynthesize (reachable suppressGenerated : Bool) (sources : ExampleSources α) : Prop :=
  reachable = true ∧ suppressGenerated = false ∧ ∀ group ∈ sources.groups, group = []

/-- Selection never manufactures or substitutes a supplied value or provenance. -/
theorem firstAuthoredFromInput {groups : List (List (Supplied α))} {selected : Supplied α}
    (found : firstAuthored groups = some selected) :
    ∃ group ∈ groups, selected ∈ group := by
  induction groups with
  | nil => simp [firstAuthored] at found
  | cons group rest ih =>
    simp only [firstAuthored] at found
    split at found
    next value last =>
      cases found
      exact ⟨group, by simp, List.mem_of_getLast? last⟩
    next =>
      obtain ⟨chosen, member, valueMember⟩ := ih found
      exact ⟨chosen, List.mem_cons_of_mem group member, valueMember⟩

/-- Absence depends on empty groups, not null/empty/invalid authored values. -/
theorem firstAuthoredAbsent (groups : List (List (Supplied α))) :
    firstAuthored groups = none ↔ ∀ group ∈ groups, group = [] := by
  induction groups with
  | nil => simp [firstAuthored]
  | cons group rest ih =>
    cases group with
    | nil => simpa [firstAuthored] using ih
    | cons value tail =>
      have lastExists := List.getLast?_eq_some_getLast (List.cons_ne_nil value tail)
      simp [firstAuthored, lastExists]

/-- Earlier empty extraction groups do not hide the next available source. -/
theorem firstAuthoredEmptyPrefix (precedingGroups rest : List (List (Supplied α)))
    (empty : ∀ group ∈ precedingGroups, group = []) :
    firstAuthored (precedingGroups ++ rest) = firstAuthored rest := by
  induction precedingGroups with
  | nil => rfl
  | cons group tail ih =>
    have groupEmpty : group = [] := empty group (by simp)
    have tailEmpty : ∀ value ∈ tail, value = [] := by
      intro value member
      exact empty value (List.mem_cons_of_mem group member)
    simpa [groupEmpty, firstAuthored] using ih tailEmpty

/-- The first nonempty group wins, and its last example wins within that group. -/
theorem firstAuthoredLastWins (precedingGroups suffix : List (List (Supplied α)))
    (earlier : List (Supplied α)) (latest : Supplied α)
    (empty : ∀ group ∈ precedingGroups, group = []) :
    firstAuthored (precedingGroups ++ (earlier ++ [latest]) :: suffix) = some latest := by
  rw [firstAuthoredEmptyPrefix precedingGroups _ empty]
  simp [firstAuthored]

/-- Excluded targets never request authored materialization or synthesis. -/
theorem exampleReachabilityFirst (suppressed : Bool) (sources : ExampleSources α) :
    selectExample false suppressed sources = .excluded := rfl

theorem contractReachabilityFirst (supplied : Option (Supplied α)) :
    selectContract false supplied = .excluded := rfl

/-- Any selected authored value survives either generation-suppression setting. -/
theorem authoredSuppressionException (suppressed : Bool) (sources : ExampleSources α)
    (supplied : Supplied α) (found : firstAuthored sources.groups = some supplied) :
    selectExample true suppressed sources = .selected supplied := by
  simp [selectExample, found]

/-- Local explicit examples, including transport-local overrides supplied here,
win over every inherited source; earlier local entries are overridden. -/
theorem localLastExampleWins (earlier : List (Supplied α)) (latest : Supplied α)
    (references bases : List (List (Supplied α))) (typeExamples : List (Supplied α))
    (suppressed : Bool) :
    selectExample true suppressed ⟨earlier ++ [latest], references, bases, typeExamples⟩ =
      .selected latest := by
  simp [selectExample, ExampleSources.groups, firstAuthored]

/-- First reference/base/type group precedence is universal, rather than a finite
list of samples. The empty precedingGroups and arbitrary suffix express every tier. -/
theorem inheritedGroupPrecedence (sources : ExampleSources α)
    (precedingGroups suffix : List (List (Supplied α))) (earlier : List (Supplied α))
    (latest : Supplied α) (suppressed : Bool)
    (layout : sources.groups = precedingGroups ++ (earlier ++ [latest]) :: suffix)
    (empty : ∀ group ∈ precedingGroups, group = []) :
    selectExample true suppressed sources = .selected latest := by
  apply authoredSuppressionException
  rw [layout]
  exact firstAuthoredLastWins precedingGroups suffix earlier latest empty

/-- Explicit null is a selected authored value, not an absent-source sentinel. -/
theorem authoredNullPreserved (source : Source) (earlier : List (Supplied RawValue))
    (references bases : List (List (Supplied RawValue)))
    (typeExamples : List (Supplied RawValue)) (suppressed : Bool) :
    selectExample true suppressed
      ⟨earlier ++ [⟨source, .null⟩], references, bases, typeExamples⟩ =
      .selected ⟨source, .null⟩ :=
  localLastExampleWins earlier ⟨source, .null⟩ references bases typeExamples suppressed

/-- Even invalid/cyclic/opaque authored inputs remain selected for resolution;
selection cannot replace them with a synthetic value. -/
theorem selectedFromInput {reachable suppressed : Bool} {sources : ExampleSources α}
    {supplied : Supplied α} (chosen : selectExample reachable suppressed sources =
      .selected supplied) : ∃ group ∈ sources.groups, supplied ∈ group := by
  cases reachable <;> simp [selectExample] at chosen
  split at chosen
  next value found =>
    cases chosen
    exact firstAuthoredFromInput found
  next => cases suppressed <;> simp at chosen

/-- Source role, origin and occurrence are preserved together with the exact
value. Role validity is an explicit input obligation, not silently repaired. -/
theorem selectedProvenance {reachable suppressed : Bool} {sources : ExampleSources α}
    {supplied : Supplied α} (chosen : selectExample reachable suppressed sources =
      .selected supplied)
    (authored : ∀ group ∈ sources.groups, ∀ value ∈ group,
      value.source.role = .authoredExample) : supplied.source.role = .authoredExample := by
  obtain ⟨group, member, valueMember⟩ := selectedFromInput chosen
  exact authored group member supplied valueMember

/-- Suppression applies exactly when no authored source was extracted. -/
theorem missingExampleSuppressed (sources : ExampleSources α)
    (empty : ∀ group ∈ sources.groups, group = []) :
    selectExample true true sources = .suppressed := by
  simp [selectExample, (firstAuthoredAbsent sources.groups).mpr empty]

/-- The executable absent result agrees with independently specified eligibility;
an always-omit or always-absent selector cannot satisfy this characterization. -/
theorem synthesisEligibility (reachable suppressed : Bool) (sources : ExampleSources α) :
    selectExample reachable suppressed sources = .absent ↔
      MaySynthesize reachable suppressed sources := by
  cases reachable <;> cases suppressed <;>
    simp [selectExample, MaySynthesize]
  all_goals
    cases found : firstAuthored sources.groups <;>
      simp [found, (firstAuthoredAbsent sources.groups).symm]

/-- Enum/default source selection cannot fill an absent declaration. -/
theorem absentContractPreserved : selectContract true (none : Option (Supplied α)) =
    .absent := rfl

/-- Contract source selection retains the exact supplied role, identity and value;
it neither inspects nor depends on example-generation suppression metadata. -/
theorem contractSourcePreserved (supplied : Supplied α) :
    selectContract true (some supplied) = .selected supplied := rfl

theorem contractNullPreserved (source : Source) :
    selectContract true (some ⟨source, RawValue.null⟩) = .selected ⟨source, .null⟩ := rfl

/-- A contract selection is supplied input, never an invented replacement. -/
theorem contractNoReplacement {reachable : Bool} {input : Option (Supplied α)}
    {supplied : Supplied α} (chosen : selectContract reachable input = .selected supplied) :
    input = some supplied := by
  cases reachable <;> cases input <;> simp [selectContract] at chosen ⊢
  exact chosen

end ValueContract.SourceSelection
