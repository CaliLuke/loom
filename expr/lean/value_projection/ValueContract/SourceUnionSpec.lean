import ValueContract.ResolutionSpec

namespace ValueContract.Candidate

/-- A candidate retains the declaration's branch identity, preference, and
entire child outcome. Ambiguity is evidence, not a missing candidate. -/
def CandidateInterpretation (child : Identity → Input → ResolveResult → Prop)
    (preference : Alternative → Bool) (input : Input)
    (alternative : Alternative) (candidate : BranchCandidate) : Prop :=
  candidate.branch = alternative.identity ∧ candidate.preferred = preference alternative ∧
    child alternative.child input candidate.result

/-- The selected payload is wrapped only with the chosen declared occurrence
and branch. Failure—including a nested ambiguity—is preserved. -/
def WrappedCandidate (occurrence : Identity) (candidate : BranchCandidate)
    (result : ResolveResult) : Prop :=
  match candidate.result with
  | .error failure => result = .error failure
  | .ok payload => result = .ok {payload with value := .union occurrence candidate.branch payload.value}

/-- Implicit union matching first constructs every complete interpretation.
Fallback interpretations are required only when that complete set is empty.
Complete-only roles never gain eligibility from a partial interpretation. -/
def ImplicitUnionOutcome
    (full incomplete : Identity → Input → ResolveResult → Prop)
    (preference : Alternative → Bool) (completeOnly : Bool)
    (occurrence : Identity) (alternatives : List Alternative) (input : Input)
    (result : ResolveResult) : Prop :=
  ∃ candidates, All₂ (CandidateInterpretation full preference input) alternatives candidates ∧
    let complete := candidates.filter BranchCandidate.complete
    ((complete ≠ [] ∧ ∃ ranked, RankedCandidates complete ranked ∧
      match ranked with
      | .error failure => result = .error failure
      | .ok chosen => WrappedCandidate occurrence chosen result) ∨
    (complete = [] ∧ ∃ fallback,
      ((completeOnly = true ∧ fallback = []) ∨
        (completeOnly = false ∧ All₂ (CandidateInterpretation incomplete preference input) alternatives fallback)) ∧
      ∃ ranked, RankedCandidates fallback ranked ∧
        match ranked with
        | .error failure => result = .error failure
        | .ok chosen => WrappedCandidate occurrence chosen result))

/-- Positional declaration lookup, independent of executable find?. The
well-formed graph separately rules out duplicate branch identities. -/
def FirstAlternative (branch : Identity) (alternatives : List Alternative)
    (alternative : Alternative) : Prop :=
  alternative.identity = branch ∧ ∃ before after,
    alternatives = before ++ alternative :: after ∧
      ∀ earlier ∈ before, earlier.identity ≠ branch

/-- Trusted selection still checks occurrence and branch ownership, then
validates its payload at the selected declaration. It cannot reselect. -/
def SelectedUnionResolution (child : Identity → Input → Resolution → Prop)
    (occurrence : Identity) (alternatives : List Alternative) (input : Input)
    (result : Resolution) : Prop :=
  ∃ branch payload alternative value,
    stripHostInput input = .selected occurrence branch payload ∧
    FirstAlternative branch alternatives alternative ∧ child alternative.child payload value ∧
    result = {value with value := .union occurrence branch value.value}

/-- Implicit matching excludes explicit selection and the explicit unsupported
or cyclic source markers. Individual branches still validate all other shapes. -/
def ImplicitUnionInput (input : Input) : Prop :=
  match stripHostInput input with
  | .selected _ _ _ | .cycle _ | .opaque _ => False
  | _ => True

end ValueContract.Candidate
