import ValueContract.SourceOutcomeSpec
import ValueContract.SourceNullProofs
import ValueContract.SourceUnionProofs

namespace ValueContract.Candidate

theorem failurePriority_injective {left right : Failure}
    (same : failurePriority left = failurePriority right) : left = right := by
  cases left <;> cases right <;> simp_all [failurePriority]

private theorem noError_of_combine_ok {inputs : List (Except Failure α)}
    {values : List α} (success : combineChecked inputs = .ok values) (failure : Failure) :
    .error failure ∉ inputs := by
  rw [(combineChecked_ok_iff _ _).mp success]
  simp

/-- Failure reduction returns an actual failed child with minimal error
priority. This characterizes rejected outcomes, not merely successful lists. -/
theorem combineChecked_error_min {inputs : List (Except Failure α)} {failure : Failure}
    (failed : combineChecked inputs = .error failure) :
    .error failure ∈ inputs ∧
      ∀ other, .error other ∈ inputs → failurePriority failure ≤ failurePriority other := by
  induction inputs generalizing failure with
  | nil => simp [combineChecked] at failed
  | cons head tail ih =>
    cases rest : combineChecked tail with
    | ok values =>
      cases head with
      | ok value => simp [combineChecked, rest] at failed
      | error first =>
        simp only [combineChecked, rest, Except.error.injEq] at failed
        subst failure
        refine ⟨by simp, ?_⟩
        intro other member
        simp only [List.mem_cons, Except.error.injEq] at member
        rcases member with same | inside
        · subst other; exact Nat.le_refl _
        · exact False.elim (noError_of_combine_ok rest other inside)
    | error last =>
      have tailMinimum := ih rest
      cases head with
      | ok value =>
        simp only [combineChecked, rest, Except.error.injEq] at failed
        subst failure
        refine ⟨by simp [tailMinimum.1], ?_⟩
        intro other member
        exact tailMinimum.2 other (by simpa using member)
      | error first =>
        simp only [combineChecked, rest, Except.error.injEq] at failed
        split at failed
        next preferred =>
          subst failure
          refine ⟨by simp, ?_⟩
          intro other member
          simp only [List.mem_cons, Except.error.injEq] at member
          rcases member with same | inside
          · subst other; exact Nat.le_refl _
          · exact Nat.le_trans preferred (tailMinimum.2 other inside)
        next notPreferred =>
          subst failure
          refine ⟨by simp [tailMinimum.1], ?_⟩
          intro other member
          simp only [List.mem_cons, Except.error.injEq] at member
          rcases member with same | inside
          · subst other; omega
          · exact tailMinimum.2 other inside

/-- Complete result correspondence includes every rejection and preserves the
ordered success payload. No child-success hypothesis is assumed. -/
theorem combineChecked_outcome_iff (inputs : List (Except Failure α))
    (result : Except Failure (List α)) :
    combineChecked inputs = result ↔ CombinedOutcome inputs result := by
  cases result with
  | ok values =>
    simp only [CombinedOutcome, combineChecked_ok_iff]
    simpa only [List.map_id, id_eq] using (mapOk_iff id inputs values)
  | error failure =>
    constructor
    · exact combineChecked_error_min
    · intro specified
      cases actual : combineChecked inputs with
      | ok values => exact False.elim (noError_of_combine_ok actual failure specified.1)
      | error other =>
        have minimal := combineChecked_error_min actual
        have same := failurePriority_injective
          (Nat.le_antisymm (minimal.2 failure specified.1) (specified.2 other minimal.1))
        subst other
        rfl

/-- Independent child outcomes compose without assuming that any child or the
collection succeeds. All supplied inputs are represented in the judgment. -/
theorem childrenOutcome_iff (child : α → Except Failure β) (inputs : List α)
    (result : Except Failure (List β)) :
    combineChecked (inputs.map child) = result ↔
      ChildrenOutcome (fun input output => child input = output) inputs result := by
  simp only [ChildrenOutcome, all₂_graph_iff, exists_eq_left', combineChecked_outcome_iff]

/-- The uniform whole-node enum gate preserves every body failure and rejects
exactly the successful semantic values outside the independent enum relation. -/
theorem nodeEnumOutcome_iff (enumeration : Option (List Value)) (body result : ResolveResult) :
    (do let resolution ← body
        if enumValueAllowed enumeration resolution.value then pure resolution
        else throw .invalid) = result ↔ NodeEnumOutcome enumeration body result := by
  cases body with
  | error failure => simp [NodeEnumOutcome, Bind.bind, Except.bind, eq_comm]
  | ok resolution =>
    by_cases allowed : enumValueAllowed enumeration resolution.value = true
    · have permitted := (enumValueAllowed_iff _ _).mp allowed
      simp [NodeEnumOutcome, Bind.bind, Except.bind, allowed, permitted,
        Pure.pure, Except.pure, eq_comm]
    · have forbidden : ¬ EnumAllows enumeration resolution.value :=
        fun permitted => allowed ((enumValueAllowed_iff _ _).mpr permitted)
      simp [NodeEnumOutcome, Bind.bind, Except.bind, allowed, forbidden, eq_comm]
      rfl

theorem mappedOutcome_iff (transform : α → β) (source : Except Failure α)
    (result : Except Failure β) :
    (do let value ← source; pure (transform value)) = result ↔
      MappedOutcome transform source result := by
  cases source <;> simp [MappedOutcome, Bind.bind, Except.bind, Pure.pure, Except.pure, eq_comm]

theorem mapChildOutcome_iff (transform : α → β) (source : Except Failure α)
    (result : Except Failure β) :
    (do let value ← source; pure (transform value)) = result ↔
      MapChildOutcome transform (fun output => source = output) result := by
  simp only [MapChildOutcome, exists_eq_left', mappedOutcome_iff]

theorem guardedOutcome_iff (valid : Bool) (next : Except Failure α)
    (result : Except Failure α) :
    (if valid then next else .error .invalid) = result ↔
      GuardedOutcome (valid = true) (fun output => next = output) result := by
  cases valid <;> simp [GuardedOutcome, eq_comm]

theorem mapChildOutcome_graph (transform : α → β) (source : Except Failure α) :
    MapChildOutcome transform (fun output => source = output) =
      (fun result => (do let value ← source; pure (transform value)) = result) := by
  funext result
  exact propext (mapChildOutcome_iff transform source result).symm

theorem childrenOutcome_graph (child : α → Except Failure β) (inputs : List α) :
    ChildrenOutcome (fun input output => child input = output) inputs =
      (fun result => combineChecked (inputs.map child) = result) := by
  funext result
  exact propext (childrenOutcome_iff child inputs result).symm

theorem sequencedOutcome_iff (first : Except Failure α) (next : α → Except Failure β)
    (result : Except Failure β) :
    (first >>= next) = result ↔
      SequencedOutcome (fun initial => first = initial)
        (fun value output => next value = output) result := by
  simp only [SequencedOutcome, exists_eq_left']
  cases first <;> simp [Bind.bind, Except.bind, eq_comm]

/-- For a local rule whose only rejection is invalid, complete success
correspondence also determines the exact complementary failure contract. -/
theorem invalidOtherwise_iff (computation : Except Failure α) (accepted : α → Prop)
    (success : ∀ value, computation = .ok value ↔ accepted value)
    (failure : ∀ error, computation = .error error → error = .invalid)
    (result : Except Failure α) :
    computation = result ↔ match result with
      | .ok value => accepted value
      | .error error => error = .invalid ∧ ¬ ∃ value, accepted value := by
  cases result with
  | ok value => exact success value
  | error error =>
    constructor
    · intro rejected
      refine ⟨failure error rejected, ?_⟩
      rintro ⟨value, permitted⟩
      have contradiction := (success value).mpr permitted
      simp [rejected] at contradiction
    · rintro ⟨rfl, rejected⟩
      cases actual : computation with
      | ok value => exact False.elim (rejected ⟨value, (success value).mp actual⟩)
      | error error =>
        have same := failure error actual
        subst error
        rfl

end ValueContract.Candidate
