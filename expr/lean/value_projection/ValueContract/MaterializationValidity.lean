import ValueContract.MaterializationProofs

namespace ValueContract.Candidate

private theorem all₂_right {R : α → β → Prop} {left : List α} {right : List β}
    (relation : All₂ R left right) {value : β} (member : value ∈ right) :
    ∃ source, source ∈ left ∧ R source value := by
  induction left generalizing right with
  | nil => cases right <;> simp_all [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp at member
    | cons first rest =>
      have pair : R head first ∧ All₂ R tail rest := by
        simpa [All₂, and_left_comm] using relation
      rcases List.mem_cons.mp member with same | inside
      · subst value; exact ⟨head, List.mem_cons_self, pair.1⟩
      · obtain ⟨source, present, related⟩ := ih pair.2 inside
        exact ⟨source, List.mem_cons_of_mem _ present, related⟩

/-- Built-in materialization always produces valid JSON. This is separate from
its canonical spelling correspondence and does not assume projector success. -/
theorem Materializes_JSONValid {codecs value wire}
    (materialized : Materializes codecs value wire) : JSONValid codecs.numbers wire := by
  obtain ⟨depth, materialized⟩ := materialized
  induction depth generalizing value wire with
  | zero => exact False.elim materialized
  | succ depth ih =>
    apply (jsonValid_iff_JSONValid ..).mp
    simp only [materializesAt] at materialized
    split at materialized <;> try contradiction
    all_goals try exact (jsonValid_iff_JSONValid ..).mpr (ih materialized)
    all_goals try exact (jsonValid_iff_JSONValid ..).mpr materialized.2
    all_goals try simp [jsonValid]

    case h_7 =>
      intro child member
      obtain ⟨source, _, related⟩ := all₂_right materialized member
      exact (jsonValid_iff_JSONValid ..).mpr (ih related)
    all_goals
      obtain ⟨unsorted, related, unique, same⟩ := materialized
      rw [same]
      have permutation := List.mergeSort_perm unsorted (fun left right : String × Wire => left.1 ≤ right.1)
      constructor
      · exact (permutation.map Prod.fst).nodup_iff.mpr unique
      · intro name child member
        have original := permutation.mem_iff.mp member
        obtain ⟨source, _, relation⟩ := all₂_right related original
        exact (jsonValid_iff_JSONValid ..).mpr (ih relation.2)

end ValueContract.Candidate
