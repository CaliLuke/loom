import ValueContract.SourceBodyOutcomeSpec
import ValueContract.SourceMemberOutcomeProofs
import ValueContract.SourceRawOutcomeProofs
import ValueContract.SourceBodyProofs

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

theorem resolveBody_array_outcome_iff (declarations : Declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (child : Identity) (bounds : LengthBounds) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.array child bounds) input same descend = result ↔
      SourceGuard input
        (ArrayOutcome (fun raw outcome => descend complete child raw = outcome) bounds input) result := by
  cases shape : stripHostInput input <;>
    simp only [resolveBody, SourceGuard, ArrayOutcome, shape, arrayInput, ArrayInputEntries]
  all_goals try (simp only [reduceCtorEq, false_and, false_or, exists_false,
    not_false_eq_true, true_and, eq_comm]; done)
  case byteSequence sequence =>
    cases nilShape : sequence.isNil <;>
      simp only [← nativeBytesNil_iff, nilShape, Bool.false_eq_true, ↓reduceIte,
        false_and, true_and, not_false_eq_true, not_true_eq_false, false_or,
        or_false, reduceCtorEq]
    all_goals simp only [Option.some.injEq, exists_eq_left, exists_eq, exists_false,
      not_true_eq_false, false_and, or_false]
    · simp only [childrenOutcome_graph, mapChildOutcome_graph]
      cases allowed : lengthAllowed bounds sequence.inputs.length <;>
        simp only [GuardedOutcome, Bool.not_true, Bool.not_false, Bool.false_eq_true,
          ↓reduceIte, true_and, false_and, not_true_eq_false, not_false_eq_true,
          false_or, or_false, exceptThrow, Bind.bind, Except.bind]
      all_goals exact eq_comm
    · cases allowed : lengthAllowed bounds 0 <;>
        simp only [GuardedOutcome, Bool.false_eq_true, ↓reduceIte, true_and,
          false_and, not_true_eq_false, not_false_eq_true, false_or, or_false]
      all_goals exact eq_comm
  case nilArray =>
    simp only [reduceCtorEq, false_and, true_and, exists_false, exists_eq,
      not_true_eq_false, or_false]
    cases allowed : lengthAllowed bounds 0 <;>
      simp only [GuardedOutcome, Bool.false_eq_true, ↓reduceIte, true_and,
        false_and, not_true_eq_false, not_false_eq_true, false_or, or_false]
    all_goals exact eq_comm
  case array items =>
    simp only [reduceCtorEq, false_and, false_or, Option.some.injEq, exists_eq_left,
      exists_eq, not_true_eq_false, or_false, childrenOutcome_graph, mapChildOutcome_graph]
    cases allowed : lengthAllowed bounds items.length <;>
      simp only [GuardedOutcome, Bool.not_true, Bool.not_false, Bool.false_eq_true,
        ↓reduceIte, true_and, false_and, not_true_eq_false, not_false_eq_true,
        false_or, or_false, exceptThrow, Bind.bind, Except.bind]
    all_goals exact eq_comm

theorem resolveBody_any_outcome_iff (declarations : Declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (input : Input) (same descend : Bool → Identity → Input → ResolveResult)
    (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete .any input same descend = result ↔
      SourceGuard input (MapChildOutcome (fun value => ⟨.any value, []⟩)
        (RawOutcomeAt keys (depth + 1) input)) result := by
  have child : RawOutcomeAt keys (depth + 1) input =
      (fun output => resolveRawAt keys (depth + 1) input = output) := by
    funext output
    exact propext (resolveRawAt_outcome_iff _ _ _ _).symm
  cases shape : stripHostInput input <;>
    simp only [resolveBody, SourceGuard, shape, child, mapChildOutcome_graph]
  all_goals simp only [eq_comm]

theorem resolveBody_scalar_outcome_iff (declarations : Declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (kind : ScalarKind) (rules : ScalarRules) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.scalar kind rules) input same descend = result ↔
      SourceGuard input (ScalarOutcome checks keys kind rules input) result := by
  have success := resolveBody_scalar_iff declarations checks keys depth rank complete kind rules
    input same descend
  cases shape : stripHostInput input
  case cycle | «opaque» => simp [SourceGuard, resolveBody, shape, eq_comm]
  all_goals
    simp only [SourceGuard, shape]
    have matched := invalidOtherwise_iff _ _ success (by
      intro error failed
      simp only [resolveBody, shape] at failed
      repeat first
        | (simpa using failed.symm)
        | (contradiction)
        | (split at failed)) result
    cases result <;> simpa only [ScalarOutcome] using matched

theorem resolveBody_selected_outcome_iff (declarations : Declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (occurrence : Identity) (alternatives : List Alternative) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult)
    (actual branch : Identity) (payload : Input)
    (shape : stripHostInput input = .selected actual branch payload) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.union occurrence alternatives)
      input same descend = result ↔
      SelectedUnionOutcome (fun identity raw output => descend complete identity raw = output)
        occurrence alternatives actual branch payload result := by
  simp only [resolveBody, shape, SelectedUnionOutcome]
  by_cases owned : actual = occurrence
  · simp only [owned, bne_self_eq_false, Bool.false_eq_true, ↓reduceIte, GuardedOutcome,
      true_and, not_true_eq_false, false_and, or_false]
    simp only [← firstAlternative_iff]
    cases found : alternatives.find? (fun alternative => alternative.identity == branch) with
    | none => simp [eq_comm]
    | some alternative =>
      simp only [Option.some.injEq, exists_eq_left', exists_eq', not_true_eq_false, false_and, or_false]
      unfold wrapBranch
      exact mapChildOutcome_iff _ _ _
  · have different : (actual != occurrence) = true := by simp [owned]
    simp only [different, ↓reduceIte, GuardedOutcome, owned, false_and, not_false_eq_true,
      true_and, false_or]
    exact eq_comm

end ValueContract.Candidate
