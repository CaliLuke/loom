import ValueContract.EqualityBudgetProofs
import ValueContract.StrictValueDepth

namespace ValueContract.Candidate

variable {keys : KeyCodec}

private theorem array_strictDepth_lt {items : List Value} {child : Value}
    (member : child ∈ items) : strictValueDepth child < strictValueDepth (.array items) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := strictValueDepth) member) 0
  simp only [strictValueDepth]
  omega

private theorem object_strictDepth_lt {fields : List (Identity × Value)}
    {extra : List (String × Value)} {entry : Identity × Value}
    (member : entry ∈ fields) : strictValueDepth entry.2 < strictValueDepth (.object fields extra) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => strictValueDepth entry.2) member) 0
  simp only [strictValueDepth]
  omega

private theorem extra_strictDepth_lt {fields : List (Identity × Value)}
    {extra : List (String × Value)} {entry : String × Value}
    (member : entry ∈ extra) : strictValueDepth entry.2 < strictValueDepth (.object fields extra) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => strictValueDepth entry.2) member) 0
  simp only [strictValueDepth]
  omega

private theorem map_strictDepth_lt {entries : List (Scalar × Value)} {entry : Scalar × Value}
    (member : entry ∈ entries) : strictValueDepth entry.2 < strictValueDepth (.map entries) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => strictValueDepth entry.2) member) 0
  simp only [strictValueDepth]
  omega

/-- Strict observation equality remains valid with additional derivation depth,
including transparent host wrappers on only one side. -/
theorem strictEquivalentAt_mono {low high : Nat} {left right : Value}
    (increase : low ≤ high) (related : strictEquivalentAt keys low left right) :
    strictEquivalentAt keys high left right := by
  induction low generalizing high left right with
  | zero => simp [strictEquivalentAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases left <;> cases right <;> simp only [strictEquivalentAt] at related ⊢
      all_goals try assumption
      all_goals try exact ih smaller related
      case array.array left right =>
        exact ⟨related.1, fun pair member => ih smaller (related.2 pair member)⟩
      case object.object left le right re =>
        refine ⟨related.1, ?_, related.2.2.1, ?_⟩
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.1 entry member
          exact ⟨other, inside, same, ih smaller child⟩
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
          exact ⟨other, inside, same, ih smaller child⟩
      case map.map left right =>
        refine ⟨related.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2 entry member
        exact ⟨other, inside, same, ih smaller child⟩
      case union.union identity branch value other otherBranch child =>
        exact ⟨related.1, related.2.1, ih smaller related.2.2⟩
      case jsonSnapshot.jsonSnapshot left right => exact WireSameAt_mono smaller related

/-- The snapshot-aware structural bound realizes every finite strict-equality
derivation. It accounts for one-sided host stripping and arbitrary wire depth. -/
theorem strictEquivalentAt_rebudget {depth budget : Nat} {left right : Value}
    (related : strictEquivalentAt keys depth left right)
    (enough : strictValueDepth left + strictValueDepth right + 1 ≤ budget) :
    strictEquivalentAt keys budget left right := by
  induction depth generalizing budget left right with
  | zero => simp [strictEquivalentAt] at related
  | succ depth ih =>
    cases budget with
    | zero => omega
    | succ budget =>
      cases left <;> cases right <;> simp only [strictEquivalentAt] at related ⊢
      all_goals try assumption
      all_goals try exact ih related (by simp only [strictValueDepth] at enough ⊢; omega)
      case array.array left right =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have l := array_strictDepth_lt (List.of_mem_zip member).1
        have r := array_strictDepth_lt (List.of_mem_zip member).2
        omega
      case object.object left le right re =>
        refine ⟨related.1, ?_, related.2.2.1, ?_⟩
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.1 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have l := object_strictDepth_lt (extra := le) member
          have r := object_strictDepth_lt (extra := re) inside
          omega
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have l := extra_strictDepth_lt (fields := left) member
          have r := extra_strictDepth_lt (fields := right) inside
          omega
      case map.map left right =>
        refine ⟨related.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2 entry member
        refine ⟨other, inside, same, ih child ?_⟩
        have l := map_strictDepth_lt member
        have r := map_strictDepth_lt inside
        omega
      case union.union identity branch value other otherBranch child =>
        refine ⟨related.1, related.2.1, ih related.2.2 ?_⟩
        simp only [strictValueDepth] at enough
        omega
      case jsonSnapshot.jsonSnapshot left right =>
        apply WireSameAt_rebudget related
        simp only [strictValueDepth] at enough
        omega

/-- At any sufficient structural budget, the executable strict comparison
exactly decides the unbounded independent observation relation. -/
theorem valueEqualAt_strict_budget_iff {budget : Nat} {left right : Value}
    (enough : strictValueDepth left + strictValueDepth right + 1 ≤ budget) :
    valueEqualAt keys budget false left right = true ↔ StrictEquivalent keys left right := by
  rw [valueEqualAt_strict_iff]
  exact ⟨fun related => ⟨_, related⟩, fun ⟨_, related⟩ => strictEquivalentAt_rebudget related enough⟩

/-- Sufficient-budget rejection excludes every strict equality derivation; it
cannot arise merely because a JSON snapshot was deeper than semantic valueDepth. -/
theorem valueEqualAt_strict_false_iff {budget : Nat} {left right : Value}
    (enough : strictValueDepth left + strictValueDepth right + 1 ≤ budget) :
    valueEqualAt keys budget false left right = false ↔ ¬ StrictEquivalent keys left right := by
  rw [← valueEqualAt_strict_budget_iff enough]
  cases valueEqualAt keys budget false left right <;> simp

end ValueContract.Candidate
