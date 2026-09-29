import ValueContract.SourceRawOutcomeSpec
import ValueContract.SourceOutcomeProofs
import ValueContract.SourceBudgetProofs

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

private theorem mapM_only_invalid (f : α → Except Failure β) (inputs : List α)
    (localFailure : ∀ input ∈ inputs, ∀ failure, f input = .error failure → failure = .invalid)
    {failure : Failure} (failed : inputs.mapM f = .error failure) : failure = .invalid := by
  induction inputs with
  | nil => simp at failed
  | cons head tail ih =>
    cases first : f head with
    | error error =>
      simp [List.mapM_cons, first, Bind.bind, Except.bind] at failed
      subst failure
      exact localFailure head (by simp) error first
    | ok value =>
      cases rest : tail.mapM f with
      | ok values => simp [List.mapM_cons, first, rest, Bind.bind, Except.bind] at failed
      | error error =>
        simp [List.mapM_cons, first, rest, Bind.bind, Except.bind] at failed
        subst failure
        exact ih (fun input inside => localFailure input (by simp [inside])) rest

theorem nameKeys_error_invalid {keys : KeyCodec} {values : List Scalar} {failure : Failure}
    (failed : nameKeys keys values = .error failure) : failure = .invalid := by
  unfold nameKeys at failed
  generalize encoded : values.mapM (fun key => match encodeKey keys key with
    | some name => (Except.ok name : Except Failure String)
    | none => .error .invalid) = result at failed
  cases result with
  | error error =>
    simp only [Bind.bind, Except.bind, Except.error.injEq] at failed
    subst failure
    apply mapM_only_invalid _ values _ encoded
    intro key _ failure failed
    cases spelling : encodeKey keys key with
    | none => simpa only [spelling, Except.error.injEq] using failed.symm
    | some name => simp only [spelling] at failed; cases failed

  | ok names =>
    simp only [Bind.bind, Except.bind] at failed
    split at failed
    · simp at failed
    · simpa using failed.symm

/-- Failure to name a whole key list is exactly failure of independent
spelling/uniqueness evidence. No codec-specific rejection is invented. -/
theorem nameKeys_invalid_iff (keys : KeyCodec) (values : List Scalar) :
    nameKeys keys values = .error .invalid ↔ ¬ AdmissibleKeys keys values := by
  constructor
  · intro failed ⟨names, admitted⟩
    have success := (nameKeys_iff keys values names).mpr admitted
    simp [failed] at success
  · intro rejected
    cases actual : nameKeys keys values with
    | ok names => exact False.elim (rejected ⟨names, (nameKeys_iff _ _ _).mp actual⟩)
    | error failure =>
      have same := nameKeys_error_invalid actual
      subst failure
      rfl

private theorem nameKeys_guard (keys : KeyCodec) (values : List Scalar)
    (next : Except Failure α) (result : Except Failure α) :
    (do let _ ← nameKeys keys values; next) = result ↔
      GuardedOutcome (AdmissibleKeys keys values) (fun output => next = output) result := by
  cases actual : nameKeys keys values with
  | error failure =>
    have same := nameKeys_error_invalid actual
    subst failure
    have rejected := (nameKeys_invalid_iff _ _).mp actual
    simp [GuardedOutcome, rejected, Bind.bind, Except.bind, eq_comm]
  | ok names =>
    have admitted : AdmissibleKeys keys values := ⟨names, (nameKeys_iff _ _ _).mp actual⟩
    simp [GuardedOutcome, admitted, Bind.bind, Except.bind]

theorem resolveRawAt_outcome_iff (keys : KeyCodec) (depth : Nat) (input : Input)
    (result : Except Failure Value) :
    resolveRawAt keys depth input = result ↔ RawOutcomeAt keys depth input result := by
  induction depth generalizing input result with
  | zero => simp [resolveRawAt, RawOutcomeAt, eq_comm]
  | succ depth ih =>
    have children : RawOutcomeAt keys depth =
        (fun input result => resolveRawAt keys depth input = result) := by
      funext input result
      exact propext (ih input result).symm
    cases input <;> simp only [resolveRawAt, RawOutcomeAt]
    all_goals first
      | (simp only [eq_comm]; done)
      | skip
    case byteSequence sequence =>
      simp only [← nativeByteValue_iff, exists_eq_left, eq_comm]
    case host identity payload =>
      simpa only [children] using mapChildOutcome_iff (Value.host identity)
        (resolveRawAt keys depth payload) result
    case array items =>
      simp only [children, childrenOutcome_graph, mapChildOutcome_graph]
    case object fields =>
      by_cases unique : (fields.map Prod.fst).Nodup
      · simp only [unique, decide_true, Bool.not_true, Bool.false_eq_true, ↓reduceIte,
          GuardedOutcome, true_and, not_true_eq_false, false_and, or_false]
        simp only [children, mapChildOutcome_graph, childrenOutcome_graph]
      · simp only [unique, decide_false, Bool.not_false, ↓reduceIte, GuardedOutcome,
          false_and, not_false_eq_true, true_and, false_or, Bind.bind, Except.bind]
        simp only [exceptThrow, eq_comm]
    case map entries =>
      rw [nameKeys_guard]
      simp only [children, mapChildOutcome_graph, childrenOutcome_graph]

/-- The independently stated outcome is total and unique, including rejection. -/
theorem rawOutcomeAt_total_unique (keys : KeyCodec) (depth : Nat) (input : Input) :
    ∃ result, RawOutcomeAt keys depth input result ∧
      ∀ other, RawOutcomeAt keys depth input other → other = result := by
  refine ⟨resolveRawAt keys depth input, (resolveRawAt_outcome_iff _ _ _ _).mp rfl, ?_⟩
  intro other specified
  exact ((resolveRawAt_outcome_iff _ _ _ _).mpr specified).symm

/-- At any adequate input-derived height, no independent raw derivation can
report fuel exhaustion, including inputs rejected for a semantic reason. -/
theorem rawOutcomeAt_noMalformed {keys : KeyCodec} {depth : Nat} {input : Input}
    {result : Except Failure Value} (enough : inputDepth input ≤ depth)
    (specified : RawOutcomeAt keys depth input result) : NoMalformed result := by
  rw [← (resolveRawAt_outcome_iff _ _ _ _).mpr specified]
  exact resolveRaw_noMalformed enough

/-- All raw outcomes are invariant above the structural input bound. This
covers rejected inputs as well as success and permits safe existential semantics. -/
theorem resolveRawAt_stable {keys : KeyCodec} {first second : Nat} {input : Input}
    (firstEnough : inputDepth input ≤ first) (secondEnough : inputDepth input ≤ second) :
    resolveRawAt keys first input = resolveRawAt keys second input := by
  induction first generalizing second input with
  | zero => have positive := inputDepth_pos input; omega
  | succ first ih =>
    cases second with
    | zero => have positive := inputDepth_pos input; omega
    | succ second =>
      cases input <;> simp only [resolveRawAt]
      all_goals try rfl
      case host identity payload =>
        simp only [inputDepth] at firstEnough secondEnough
        rw [ih (input := payload) (second := second) (by omega) (by omega)]
      case array items =>
        have children : items.map (resolveRawAt keys first) = items.map (resolveRawAt keys second) := by
          apply List.map_congr_left
          intro item member
          have smaller := array_inputDepth_lt member
          exact ih (second := second) (by omega) (by omega)
        rw [children]
      case object entries =>
        have children : entries.map (fun entry => do
            let value ← resolveRawAt keys first entry.2
            pure (entry.1, value)) = entries.map (fun entry => do
            let value ← resolveRawAt keys second entry.2
            pure (entry.1, value)) := by
          apply List.map_congr_left
          intro entry member
          have smaller := object_inputDepth_lt member
          rw [ih (input := entry.2) (second := second) (by omega) (by omega)]
        rw [children]
      case map entries =>
        have children : entries.map (fun entry => do
            let value ← resolveRawAt keys first entry.2
            pure (entry.1.value, value)) = entries.map (fun entry => do
            let value ← resolveRawAt keys second entry.2
            pure (entry.1.value, value)) := by
          apply List.map_congr_left
          intro entry member
          have smaller := map_inputDepth_lt member
          rw [ih (input := entry.2) (second := second) (by omega) (by omega)]
        rw [children]


theorem resolveRaw_outcome_iff {keys : KeyCodec} {depth : Nat} {input : Input}
    (enough : inputDepth input ≤ depth) (result : Except Failure Value) :
    resolveRawAt keys depth input = result ↔ RawOutcome keys input result := by
  constructor
  · intro computed
    exact ⟨depth, enough, (resolveRawAt_outcome_iff _ _ _ _).mp computed⟩
  · rintro ⟨other, otherEnough, specified⟩
    rw [resolveRawAt_stable enough otherEnough]
    exact (resolveRawAt_outcome_iff _ _ _ _).mpr specified

theorem rawOutcome_unique {keys : KeyCodec} {input : Input} {left right : Except Failure Value}
    (first : RawOutcome keys input left) (second : RawOutcome keys input right) : left = right := by
  exact ((resolveRaw_outcome_iff (Nat.le_refl _) left).mpr first).symm.trans
    ((resolveRaw_outcome_iff (Nat.le_refl _) right).mpr second)

end ValueContract.Candidate
