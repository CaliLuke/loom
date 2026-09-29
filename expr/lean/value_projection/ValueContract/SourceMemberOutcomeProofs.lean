import ValueContract.SourceMemberOutcomeSpec
import ValueContract.SourceOutcomeProofs

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

theorem suppliedAliases_eq (member : Member) (entries : List (String × Input)) :
    SuppliedAliases member entries = entries.filter (fun entry => hasSuppliedValue entry.2 &&
      (entry.1 == member.sourceName || entry.1 == member.wireAlias)) := by
  apply List.filter_congr
  intro entry _
  apply Bool.eq_iff_iff.mpr
  simp [hasSuppliedValue_iff]

/-- Full member correspondence includes rejected losing aliases and nested
ambiguity. Required omission and retained-child path prefixes are exact. -/
theorem objectMemberResult_outcome_iff (complete : Bool)
    (child : Identity → Input → ResolveResult) (entries : List (String × Input))
    (member : Member) (result : Except Failure (Identity × Resolution)) :
    objectMemberResult complete child entries member = result ↔
      MemberOutcome (fun identity input output => child identity input = output)
        complete member entries result := by
  unfold MemberOutcome
  rw [suppliedAliases_eq, childrenOutcome_graph]
  unfold objectMemberResult
  rw [sequencedOutcome_iff]
  apply Iff.of_eq
  congr 2
  funext values output
  apply propext
  cases selected : selectedMemberInput member entries with
  | none =>
    have absent : MemberChoice member entries none := (selectedMemberInput_iff _ _ _).mp selected
    have noSome (input : Input) : ¬ MemberChoice member entries (some input) := by
      rw [← selectedMemberInput_iff, selected]
      simp
    cases complete <;> cases required : member.required <;>
      simp [absent, noSome, GuardedOutcome, Bind.bind, Except.bind,
        exceptThrow, Pure.pure, Except.pure, eq_comm]
  | some input =>
    have choice (other : Input) : MemberChoice member entries (some other) ↔ input = other := by
      rw [← selectedMemberInput_iff, selected]
      simp
    have notAbsent : ¬ MemberChoice member entries none := by
      rw [← selectedMemberInput_iff, selected]
      simp
    simp only [notAbsent, false_and, false_or, choice, exists_eq_left', mapChildOutcome_graph]

end ValueContract.Candidate
