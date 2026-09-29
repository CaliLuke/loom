import ValueContract.SourceUnionSpec
import ValueContract.ResolverProofs

namespace ValueContract.Candidate

theorem all₂_graph_iff (f : α → β) (inputs : List α) (outputs : List β) :
    All₂ (fun input output => f input = output) inputs outputs ↔ inputs.map f = outputs := by
  constructor
  · intro related
    simpa using all₂_map_eq related f id (fun _ _ same => same)
  · intro same
    subst outputs
    induction inputs with
    | nil => simp [All₂]
    | cons head tail ih => simpa [All₂] using ih

theorem candidateInterpretation_graph (child : Identity → Input → ResolveResult)
    (preference : Alternative → Bool) (input : Input)
    (alternative : Alternative) (candidate : BranchCandidate) :
    CandidateInterpretation (fun identity input result => child identity input = result)
      preference input alternative candidate ↔
      ({branch := alternative.identity, preferred := preference alternative,
        result := child alternative.child input} : BranchCandidate) = candidate := by
  cases candidate
  simp [CandidateInterpretation, BranchCandidate.mk.injEq, eq_comm]

theorem candidateList_graph (child : Identity → Input → ResolveResult)
    (preference : Alternative → Bool) (input : Input)
    (alternatives : List Alternative) (candidates : List BranchCandidate) :
    All₂ (CandidateInterpretation (fun identity input result => child identity input = result)
      preference input) alternatives candidates ↔
      alternatives.map (fun alternative => {
        branch := alternative.identity, preferred := preference alternative,
        result := child alternative.child input : BranchCandidate }) = candidates := by
  simp only [All₂, candidateInterpretation_graph]
  simpa only [All₂] using all₂_graph_iff (fun alternative => {
    branch := alternative.identity, preferred := preference alternative,
    result := child alternative.child input : BranchCandidate }) alternatives candidates

theorem wrapBranch_iff (occurrence : Identity) (candidate : BranchCandidate)
    (result : ResolveResult) : wrapBranch occurrence candidate = result ↔
      WrappedCandidate occurrence candidate result := by
  unfold wrapBranch WrappedCandidate
  cases candidate.result <;> simp [Bind.bind, Except.bind, Pure.pure, Except.pure, eq_comm]

theorem rankedWrapped_iff (candidates : List BranchCandidate) (occurrence : Identity)
    (result : ResolveResult) :
    (do let chosen ← rankCandidates candidates; wrapBranch occurrence chosen) = result ↔
      ∃ ranked, RankedCandidates candidates ranked ∧ match ranked with
        | .error failure => result = .error failure
        | .ok chosen => WrappedCandidate occurrence chosen result := by
  simp only [← rankCandidates_iff]
  cases selected : rankCandidates candidates <;>
    simp only [Bind.bind, Except.bind, exists_eq_left']
  · exact eq_comm
  · exact wrapBranch_iff _ _ _

/-- Full child outcomes are consumed before preference. Neither incomplete nor
ambiguous candidates can displace a nonempty complete set. The proof covers
success and every propagated error, not only a pre-assumed chosen branch. -/
theorem implicitUnionOutcome_iff
    (full incomplete : Identity → Input → ResolveResult)
    (preference : Alternative → Bool) (completeOnly : Bool)
    (occurrence : Identity) (alternatives : List Alternative) (input : Input)
    (result : ResolveResult) :
    (do
      let chosen ← rankCompleteFirst
        ((alternatives.map fun alternative => {
          branch := alternative.identity, preferred := preference alternative,
          result := full alternative.child input : BranchCandidate }).filter BranchCandidate.complete)
        (fun _ => if completeOnly then [] else alternatives.map fun alternative => {
          branch := alternative.identity, preferred := preference alternative,
          result := incomplete alternative.child input : BranchCandidate })
      wrapBranch occurrence chosen) = result ↔
    ImplicitUnionOutcome
      (fun identity input result => full identity input = result)
      (fun identity input result => incomplete identity input = result)
      preference completeOnly occurrence alternatives input result := by
  unfold ImplicitUnionOutcome
  simp only [candidateList_graph, exists_eq_left']
  cases completeSet : (alternatives.map fun alternative => {
      branch := alternative.identity, preferred := preference alternative,
      result := full alternative.child input : BranchCandidate }).filter BranchCandidate.complete with
  | nil =>
    cases completeOnly <;> simp [rankCompleteFirst]
    all_goals exact rankedWrapped_iff _ _ _
  | cons head tail =>
    simp [rankCompleteFirst]
    exact rankedWrapped_iff _ _ _

theorem firstAlternative_iff (branch : Identity) (alternatives : List Alternative)
    (alternative : Alternative) :
    alternatives.find? (fun candidate => candidate.identity == branch) = some alternative ↔
      FirstAlternative branch alternatives alternative := by
  simp [List.find?_eq_some_iff_append, FirstAlternative]

theorem resolveBody_selected_union_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (occurrence : Identity)
    (alternatives : List Alternative) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult)
    (actual branch : Identity) (payload : Input)
    (shape : stripHostInput input = .selected actual branch payload) (result : Resolution) :
    resolveBody declarations checks keys depth rank complete (.union occurrence alternatives)
      input same descend = .ok result ↔
      SelectedUnionResolution (fun child input value => descend complete child input = .ok value)
        occurrence alternatives input result := by
  by_cases sameOccurrence : actual = occurrence
  · subst actual
    cases found : alternatives.find? (fun candidate => candidate.identity == branch) <;>
      simp [resolveBody, shape, SelectedUnionResolution, ← firstAlternative_iff, found,
        wrapBranch, exceptBind_ok, and_assoc]
    simp [eq_comm]
  · simp [resolveBody, shape, sameOccurrence, SelectedUnionResolution]

theorem resolveBody_implicit_union_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (occurrence : Identity)
    (alternatives : List Alternative) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult)
    (implicit : ImplicitUnionInput input) (result : ResolveResult) :
    resolveBody declarations checks keys depth rank complete (.union occurrence alternatives)
      input same descend = result ↔
      ImplicitUnionOutcome
        (fun identity input result => same true identity input = result)
        (fun identity input result => same false identity input = result)
        (fun alternative => prefersObjectInput declarations (rank + 1) alternative.child input)
        complete occurrence alternatives input result := by
  cases shape : stripHostInput input <;> simp [ImplicitUnionInput, shape] at implicit
  all_goals simp only [resolveBody, shape]
  all_goals exact implicitUnionOutcome_iff _ _ _ _ _ _ _ _

theorem wrappedCandidate_success {occurrence : Identity} {candidate : BranchCandidate}
    {result : Resolution} (wrapped : WrappedCandidate occurrence candidate (.ok result)) :
    ∃ payload, candidate.result = .ok payload ∧
      result = {payload with value := .union occurrence candidate.branch payload.value} := by
  cases outcome : candidate.result <;> simp [WrappedCandidate, outcome] at wrapped ⊢
  exact wrapped

/-- A successful implicit match consumes an actual declared candidate. It
retains that candidate's branch and payload; partial evidence is eligible only
for a role that permits it and only through the fallback arm of the judgment. -/
theorem implicitUnion_success
    {full incomplete : Identity → Input → ResolveResult → Prop}
    {preference : Alternative → Bool} {completeOnly : Bool}
    {occurrence : Identity} {alternatives : List Alternative} {input : Input}
    {result : Resolution}
    (matched : ImplicitUnionOutcome full incomplete preference completeOnly occurrence
      alternatives input (.ok result)) :
    ∃ alternative ∈ alternatives, ∃ payload,
      (full alternative.child input (.ok payload) ∨
        (completeOnly = false ∧ incomplete alternative.child input (.ok payload))) ∧
      result = {payload with value := .union occurrence alternative.identity payload.value} := by
  obtain ⟨candidates, interpretations, chosen⟩ := matched
  rcases chosen with complete | fallback
  · obtain ⟨_, ranked, ranking, wrapped⟩ := complete
    cases ranked with
    | error failure => contradiction
    | ok chosen =>
      obtain ⟨payload, succeeded, resultShape⟩ := wrappedCandidate_success wrapped
      have selected := rankCandidates_chosen (rankCandidates_progress ranking)
      have member := (List.mem_filter.mp selected).1
      obtain ⟨alternative, declared, branch, _, child⟩ := all₂_right interpretations member
      exact ⟨alternative, declared, payload, Or.inl (succeeded ▸ child), branch ▸ resultShape⟩
  · obtain ⟨_, fallback, eligibility, ranked, ranking, wrapped⟩ := fallback
    cases ranked with
    | error failure => contradiction
    | ok chosen =>
      obtain ⟨payload, succeeded, resultShape⟩ := wrappedCandidate_success wrapped
      have member := rankCandidates_chosen (rankCandidates_progress ranking)
      rcases eligibility with ⟨_, empty⟩ | ⟨partialRole, interpretations⟩
      · subst fallback; simp at member
      · obtain ⟨alternative, declared, branch, _, child⟩ := all₂_right interpretations member
        exact ⟨alternative, declared, payload, Or.inr ⟨partialRole, succeeded ▸ child⟩,
          branch ▸ resultShape⟩

theorem resolveBody_union_not_null (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (occurrence : Identity)
    (alternatives : List Alternative) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : Resolution)
    (success : resolveBody declarations checks keys depth rank complete (.union occurrence alternatives)
      input same descend = .ok result) : valueIsNull result.value = false := by
  cases shape : stripHostInput input
  case cycle => simp [resolveBody, shape] at success
  case «opaque» => simp [resolveBody, shape] at success
  case selected actual branch payload =>
    have matched := (resolveBody_selected_union_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _ shape _).mp success
    obtain ⟨branch, payload, alternative, value, _, _, _, resultShape⟩ := matched
    rw [resultShape]
    rfl
  all_goals
    have implicit : ImplicitUnionInput input := by simp [ImplicitUnionInput, shape]
    have matched := (resolveBody_implicit_union_iff _ _ _ _ _ _ _ _ _ _ _ implicit _).mp success
    obtain ⟨alternative, _, payload, _, resultShape⟩ := implicitUnion_success matched
    rw [resultShape]
    rfl

end ValueContract.Candidate
