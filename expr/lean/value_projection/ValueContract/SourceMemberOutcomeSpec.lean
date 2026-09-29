import ValueContract.SourceOutcomeSpec

namespace ValueContract.Candidate

/-- Every supplied spelling is retained for child validation, including losing
aliases and cross-member spelling overlaps. Bare absence alone is omitted. -/
noncomputable def SuppliedAliases (member : Member) (entries : List (String × Input)) :
    List (String × Input) := by
  classical
  exact entries.filter (fun entry => decide (entry.2 ≠ .absent ∧
    (entry.1 = member.sourceName ∨ entry.1 = member.wireAlias)))

/-- Full member outcomes first validate every supplied alias. Only then does
wire precedence choose the retained child. A missing required example field
records its identity; enum/default completeness instead rejects it. -/
def MemberOutcome (child : Identity → Input → ResolveResult → Prop)
    (complete : Bool) (member : Member) (entries : List (String × Input))
    (result : Except Failure (Identity × Resolution)) : Prop :=
  SequencedOutcome
    (ChildrenOutcome (fun entry => child member.child entry.2) (SuppliedAliases member entries))
    (fun _ output =>
      (MemberChoice member entries none ∧
        GuardedOutcome (complete = false ∨ member.required = false)
          (fun outcome => outcome = .ok (member.identity,
            ⟨.absent, if member.required then [[member.identity]] else []⟩)) output) ∨
      (∃ input, MemberChoice member entries (some input) ∧
        MapChildOutcome (fun value => (member.identity,
          {value with missing := value.missing.map (member.identity :: ·)}))
          (child member.child input) output)) result

end ValueContract.Candidate
