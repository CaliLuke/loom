import ValueContract.CandidateModel

namespace ValueContract.Candidate

/-- Nested list recursion decreases through the value component of an entry. -/
theorem pairChildSize_lt [SizeOf α] [SizeOf β] {p : α × β}
    {items : List (α × β)} (member : p ∈ items) :
    sizeOf p.2 < 1 + sizeOf items := by
  have smaller := List.sizeOf_lt_of_mem member
  cases p
  simp at smaller ⊢
  omega

/-- Computable structural depth; runtime traversal never evaluates Lean's
proof-only generated sizeOf definitions for nested inductive values. -/
def valueDepth : Value → Nat
  | .array items => 1 + (items.map valueDepth).foldl max 0
  | .object fields additional => 1 + max
      ((fields.map (fun field => valueDepth field.2)).foldl max 0)
      ((additional.map (fun field => valueDepth field.2)).foldl max 0)
  | .map entries => 1 + (entries.map (fun entry => valueDepth entry.2)).foldl max 0
  | .union _ _ payload | .any payload | .host _ payload => 1 + valueDepth payload
  | _ => 1
termination_by value => sizeOf value
decreasing_by
  all_goals simp_wf
  all_goals first
    | exact pairChildSize_lt ‹_ ∈ _›
    | exact Nat.lt_trans (List.sizeOf_lt_of_mem ‹_ ∈ _›) (by omega)
    | have smaller := pairChildSize_lt ‹_ ∈ _›; omega
    | omega

theorem foldMax_initial (items : List Nat) (initial : Nat) :
    initial ≤ items.foldl max initial := by
  induction items generalizing initial with
  | nil => exact Nat.le_refl _
  | cons head tail ih => exact Nat.le_trans (Nat.le_max_left _ _) (ih (max initial head))

theorem foldMax_member {items : List Nat} {item : Nat}
    (member : item ∈ items) (initial : Nat) : item ≤ items.foldl max initial := by
  induction items generalizing initial with
  | nil => simp at member
  | cons head tail ih =>
    rcases List.mem_cons.mp member with same | inside
    · subst item
      exact Nat.le_trans (Nat.le_max_right initial head) (foldMax_initial tail _)
    · exact ih inside _

theorem array_valueDepth_lt {items : List Value} {child : Value}
    (member : child ∈ items) : valueDepth child < valueDepth (.array items) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := valueDepth) member) 0
  simp only [valueDepth]
  omega

theorem object_valueDepth_lt {fields : List (Identity × Value)}
    {extra : List (String × Value)} {entry : Identity × Value}
    (member : entry ∈ fields) : valueDepth entry.2 < valueDepth (.object fields extra) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => valueDepth entry.2) member) 0
  simp only [valueDepth]
  omega

theorem extra_valueDepth_lt {fields : List (Identity × Value)}
    {extra : List (String × Value)} {entry : String × Value}
    (member : entry ∈ extra) : valueDepth entry.2 < valueDepth (.object fields extra) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => valueDepth entry.2) member) 0
  simp only [valueDepth]
  omega

theorem map_valueDepth_lt {entries : List (Scalar × Value)} {entry : Scalar × Value}
    (member : entry ∈ entries) : valueDepth entry.2 < valueDepth (.map entries) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => valueDepth entry.2) member) 0
  simp only [valueDepth]
  omega


end ValueContract.Candidate
