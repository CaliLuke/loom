import ValueContract.SourceFullOutcomeSpec
import ValueContract.SourceBodyOutcomeProofs
import ValueContract.SourceMapObjectOutcomeProofs
import ValueContract.SourcePreferenceProofs

namespace ValueContract.Candidate

theorem sourcePreferenceAt_eq {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations) (rank : Nat) (input : Input)
    (alternative : Alternative) :
    sourcePreferenceAt declarations rank input alternative =
      prefersObjectInput declarations rank alternative.child input := by
  apply Bool.eq_iff_iff.mpr
  simp only [sourcePreferenceAt, decide_eq_true_eq]
  exact ⟨prefersObjectInput_complete_at wellFormed, prefersObjectInput_sound⟩

/-- Complete body correspondence for arbitrary recursive implementations. The
relation preserves failures and full resolution records, including missing paths. -/
theorem resolveBody_outcome_iff {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (contract : Contract) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete contract input same descend = result ↔
      SourceBodyOutcome declarations checks keys depth rank complete contract input
        (fun mode identity raw output => same mode identity raw = output)
        (fun mode identity raw output => descend mode identity raw = output) result := by
  cases contract with
  | scalar kind rules => exact resolveBody_scalar_outcome_iff _ _ _ _ _ _ _ _ _ _ _ _
  | array child bounds => exact resolveBody_array_outcome_iff _ _ _ _ _ _ _ _ _ _ _ _
  | any => exact resolveBody_any_outcome_iff _ _ _ _ _ _ _ _ _ _
  | map kind rules child bounds => exact resolveBody_map_outcome_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _
  | object members isOpen => exact resolveBody_object_outcome_iff _ _ _ _ _ _ _ _ _ _ _ _
  | alias child =>
    cases shape : stripHostInput input <;> simp [resolveBody, SourceBodyOutcome, SourceGuard, shape, eq_comm]
  | nullable child =>
    cases shape : stripHostInput input <;> simp [resolveBody, SourceBodyOutcome, SourceGuard, shape, eq_comm]
  | nonNull child =>
    cases shape : stripHostInput input <;> simp [resolveBody, SourceBodyOutcome, SourceGuard, shape, eq_comm]
  | custom codec =>
    cases shape : stripHostInput input <;> simp [resolveBody, SourceBodyOutcome, SourceGuard, shape, eq_comm]
  | union occurrence alternatives =>
    have preferences : sourcePreferenceAt declarations (rank + 1) input =
        (fun alternative => prefersObjectInput declarations (rank + 1) alternative.child input) := by
      funext alternative
      exact sourcePreferenceAt_eq wellFormed _ _ _
    cases shape : stripHostInput input
    case cycle | «opaque» => simp [resolveBody, SourceBodyOutcome, SourceGuard, shape, eq_comm]
    case selected actual branch payload =>
      simpa only [SourceBodyOutcome, SourceGuard, shape] using
        resolveBody_selected_outcome_iff declarations checks keys depth rank complete occurrence
          alternatives input same descend actual branch payload shape result
    all_goals
      have implicit : ImplicitUnionInput input := by simp [ImplicitUnionInput, shape]
      simpa only [SourceBodyOutcome, SourceGuard, shape, preferences] using
        resolveBody_implicit_union_iff declarations checks keys depth rank complete occurrence
          alternatives input same descend implicit result

private theorem declaration_selected {declarations : Declarations} {identity : Identity}
    {selected : Declaration} (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some selected)
    {declaration : Declaration} (member : declaration ∈ declarations)
    (same : declaration.identity = identity) : declaration = selected := by
  have other := same ▸ findDeclaration_of_mem wellFormed member
  exact Option.some.inj (other.symm.trans found)

/-- Every indexed worker outcome corresponds to the independent full source
derivation. This includes failure classification, nested ambiguity, exact source
branch identities and the complete ordered missing-path list. -/
theorem resolveAt_outcome_iff {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations)
    (checks : ExternalScalarChecks) (keys : KeyCodec) (depth rank : Nat) (complete : Bool)
    (identity : Identity) (input : Input) (result : ResolveResult) :
    resolveAt declarations checks keys depth rank complete identity input = result ↔
      SourceOutcomeAt declarations checks keys depth rank complete identity input result := by
  induction depth, rank, complete, identity, input using resolveAt.induct declarations checks keys
      generalizing result with
  | case1 => simp [resolveAt, SourceOutcomeAt, eq_comm]
  | case2 depth complete identity input nonzero =>
    cases depth <;> simp [resolveAt, SourceOutcomeAt, eq_comm]
  | case3 depth rank complete identity input declaration found same children =>
    have sameMeaning : SourceOutcomeAt declarations checks keys (depth + 1) rank =
        (fun mode child raw output => resolveAt declarations checks keys (depth + 1) rank
          mode child raw = output) := by
      funext mode child raw output
      exact propext (same mode child raw output).symm
    have childMeaning : SourceOutcomeAt declarations checks keys depth (maximumExpansionRank declarations + 1) =
        (fun mode child raw output => resolveAt declarations checks keys depth
          (maximumExpansionRank declarations + 1) mode child raw = output) := by
      funext mode child raw output
      exact propext (children mode child raw output).symm
    have selected : SourceOutcomeAt declarations checks keys (depth + 1) (rank + 1)
        complete identity input result ↔
        ∃ body, SourceBodyOutcome declarations checks keys depth rank complete declaration.contract input
          (SourceOutcomeAt declarations checks keys (depth + 1) rank)
          (SourceOutcomeAt declarations checks keys depth (maximumExpansionRank declarations + 1)) body ∧
          NodeEnumOutcome keys declaration.enumeration body result := by
      rw [SourceOutcomeAt]
      constructor
      · rintro (⟨actual, member, identitySame, body, matched, enumeration⟩ | ⟨missing, _⟩)
        · have equal := declaration_selected wellFormed found member identitySame
          subst actual
          exact ⟨body, matched, enumeration⟩
        · exact False.elim (missing ⟨declaration, findDeclaration_mem found, findDeclaration_identity found⟩)
      · rintro ⟨body, matched, enumeration⟩
        exact Or.inl ⟨declaration, findDeclaration_mem found, findDeclaration_identity found,
          body, matched, enumeration⟩
    rw [selected, sameMeaning, childMeaning]
    simp only [← resolveBody_outcome_iff wellFormed, exists_eq_left']
    simp only [resolveAt, found, Bind.bind, Except.bind]
    exact nodeEnumOutcome_iff _ _ _
  | case4 depth rank complete identity input found _ _ =>
    have missing : ¬ ∃ declaration ∈ declarations, declaration.identity = identity := by
      rintro ⟨declaration, member, same⟩
      have other := same ▸ findDeclaration_of_mem wellFormed member
      simp [found] at other
    simp only [resolveAt, found, SourceOutcomeAt, Bind.bind, Except.bind]
    constructor
    · intro failed
      exact Or.inr ⟨missing, failed.symm⟩
    · rintro (⟨declaration, member, same, _⟩ | ⟨_, failed⟩)
      · exact False.elim (missing ⟨declaration, member, same⟩)
      · exact failed.symm

end ValueContract.Candidate
