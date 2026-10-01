import ValueContract.SourceSelection

namespace ValueContract.OrderedExampleGroups

open SourceSelection

/-- The complete first authored group. Returning the original list preserves its
order, multiplicity, provenance and any detached metadata carried by `β`.
This is source-selection algebra only: it makes no extraction, copying,
resolution, codec or target-plan claim. -/
def firstNonemptyGroup : List (List β) → Option (List β)
  | [] => none
  | [] :: rest => firstNonemptyGroup rest
  | group@(_ :: _) :: _ => some group

/-- Group selection has the same reachability and suppression states as singular
example selection, but retains the complete authored group. -/
inductive GroupChoice (β : Type) where
  | excluded
  | suppressed
  | absent
  | selected (group : List β)
  deriving Repr

/-- Reachability precedes authored source use. Suppression blocks only synthesis;
it does not discard a present authored group. -/
def selectExampleGroup (reachable suppressGenerated : Bool)
    (sources : ExampleSources α) : GroupChoice (Supplied α) :=
  if reachable then
    match firstNonemptyGroup sources.groups with
    | some group => .selected group
    | none => if suppressGenerated then .suppressed else .absent
  else .excluded

/-- A selected group is the exact first nonempty input group. The decomposition
retains the original list itself; no flattening, sorting or reconstruction is
permitted. -/
theorem firstNonemptyGroupExact {groups : List (List β)} {selected : List β}
    (found : firstNonemptyGroup groups = some selected) :
    ∃ preceding suffix,
      groups = preceding ++ selected :: suffix ∧
      (∀ group ∈ preceding, group = []) ∧ selected ≠ [] := by
  induction groups with
  | nil => simp [firstNonemptyGroup] at found
  | cons group rest ih =>
    cases group with
    | nil =>
      simp only [firstNonemptyGroup] at found
      obtain ⟨preceding, suffix, layout, empty, nonempty⟩ := ih found
      refine ⟨[] :: preceding, suffix, ?_, ?_, nonempty⟩
      · simp [layout]
      · intro current member
        simp only [List.mem_cons] at member
        cases member with
        | inl isEmpty => exact isEmpty
        | inr inPrefix => exact empty current inPrefix
    | cons head tail =>
      simp only [firstNonemptyGroup] at found
      cases found
      exact ⟨[], rest, by simp, by simp, by simp⟩

/-- Detached metadata is part of each entry and survives unchanged with the
selected group's exact order and multiplicity. -/
theorem detachedMetadataPreserved (head : Supplied α × μ)
    (tail : List (Supplied α × μ)) (rest : List (List (Supplied α × μ))) :
    firstNonemptyGroup ((head :: tail) :: rest) = some (head :: tail) := rfl

/-- Empty prefixes are neutral for complete-group selection. -/
theorem firstNonemptyGroupEmptyPrefix (preceding rest : List (List β))
    (empty : ∀ group ∈ preceding, group = []) :
    firstNonemptyGroup (preceding ++ rest) = firstNonemptyGroup rest := by
  induction preceding with
  | nil => rfl
  | cons group tail ih =>
    have groupEmpty : group = [] := empty group (by simp)
    have tailEmpty : ∀ current ∈ tail, current = [] := by
      intro current member
      exact empty current (List.mem_cons_of_mem group member)
    simpa [groupEmpty, firstNonemptyGroup] using ih tailEmpty

/-- Complete-group absence means exactly that every extracted group is empty. -/
theorem firstNonemptyGroupAbsent (groups : List (List β)) :
    firstNonemptyGroup groups = none ↔ ∀ group ∈ groups, group = [] := by
  induction groups with
  | nil => simp [firstNonemptyGroup]
  | cons group rest ih =>
    cases group with
    | nil => simpa [firstNonemptyGroup] using ih
    | cons head tail => simp [firstNonemptyGroup]

/-- Taking the last member of the retained first group reproduces the existing
singular source selector exactly. -/
theorem firstNonemptyGroupLastAgrees (groups : List (List (Supplied α))) :
    (firstNonemptyGroup groups).bind List.getLast? = firstAuthored groups := by
  induction groups with
  | nil => rfl
  | cons group rest ih =>
    cases group with
    | nil => simpa [firstNonemptyGroup, firstAuthored] using ih
    | cons head tail =>
      have lastExists := List.getLast?_eq_some_getLast (List.cons_ne_nil head tail)
      simp [firstNonemptyGroup, firstAuthored, lastExists]

/-- Empty extracted groups do not become selected when the target is unreachable. -/
theorem groupReachabilityFirst (suppressed : Bool) (sources : ExampleSources α) :
    selectExampleGroup false suppressed sources = .excluded := rfl

/-- A present authored group survives either generation-suppression setting. -/
theorem groupAuthoredSuppressionException (suppressed : Bool)
    (sources : ExampleSources α) (group : List (Supplied α))
    (found : firstNonemptyGroup sources.groups = some group) :
    selectExampleGroup true suppressed sources = .selected group := by
  simp [selectExampleGroup, found]

/-- Group-aware absence uses the existing synthesis-eligibility policy. It is
stronger than group absence alone because reachability and suppression remain
part of the policy. -/
theorem groupSynthesisEligibility (reachable suppressed : Bool)
    (sources : ExampleSources α) :
    selectExampleGroup reachable suppressed sources = .absent ↔
      MaySynthesize reachable suppressed sources := by
  cases reachable <;> cases suppressed <;>
    simp [selectExampleGroup, MaySynthesize]
  all_goals
    cases found : firstNonemptyGroup sources.groups <;>
      simp [found, (firstNonemptyGroupAbsent sources.groups).symm]

/-- Selection does not inspect payloads. Null, cyclic, opaque and otherwise
invalid authored payloads remain in the exact retained group for later semantic
resolution. -/
theorem arbitraryPayloadGroupPreserved (source : Source) (payload : RawValue)
    (suppressed : Bool) :
    selectExampleGroup true suppressed
      ⟨[⟨source, payload⟩], [], [], []⟩ =
      .selected [⟨source, payload⟩] := by
  simp [selectExampleGroup, ExampleSources.groups, firstNonemptyGroup]

example (source : Source) (suppressed : Bool) :
    selectExampleGroup true suppressed
      ⟨[⟨source, RawValue.null⟩], [], [], []⟩ =
      .selected [⟨source, RawValue.null⟩] :=
  arbitraryPayloadGroupPreserved source .null suppressed

example (source : Source) (suppressed : Bool) :
    selectExampleGroup true suppressed
      ⟨[⟨source, RawValue.cycle 7⟩], [], [], []⟩ =
      .selected [⟨source, RawValue.cycle 7⟩] :=
  arbitraryPayloadGroupPreserved source (.cycle 7) suppressed

example (source : Source) (suppressed : Bool) :
    selectExampleGroup true suppressed
      ⟨[⟨source, RawValue.opaque 9⟩], [], [], []⟩ =
      .selected [⟨source, RawValue.opaque 9⟩] :=
  arbitraryPayloadGroupPreserved source (.opaque 9) suppressed

/-- Mutation control: flattening all groups and choosing the global last entry
changes the established policy. The first group must win intact. -/
theorem flattenAllGroupsLastWinsIsWrong :
    firstNonemptyGroup ([[1], [2]] : List (List Nat)) = some [1] ∧
      ([[1], [2]] : List (List Nat)).flatten.getLast? = some 2 := by
  decide

end ValueContract.OrderedExampleGroups
