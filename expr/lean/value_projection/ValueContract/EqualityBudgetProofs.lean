import ValueContract.SchemaProofs
import ValueContract.TargetEquality
import ValueContract.SourceEqualityBudgetProofs

namespace ValueContract.Candidate

/-- Unbounded schema equality admits a finite derivation of any depth. -/
def WireEquivalent (numbers : NumericCodec) (left right : Wire) : Prop :=
  ∃ depth, wireEquivalentAt depth numbers left right

/-- Unbounded target observation equality; raw source equality remains separate. -/
def TargetEquivalent (numbers : NumericCodec) (left right : Value) : Prop :=
  ∃ depth, targetEquivalentAt depth numbers left right

private theorem all₂_lift {R S : α → β → Prop} {left : List α} {right : List β}
    (lift : ∀ a b, R a b → S a b) (related : All₂ R left right) : All₂ S left right :=
  ⟨related.1, fun pair member => lift pair.1 pair.2 (related.2 pair member)⟩

/-- Extra derivation depth preserves wire equality, including independently
matched object entries and exact-oneof labels. -/
theorem wireEquivalentAt_mono {numbers : NumericCodec} {low high : Nat} {left right : Wire}
    (increase : low ≤ high) (related : wireEquivalentAt low numbers left right) :
    wireEquivalentAt high numbers left right := by
  induction low generalizing high left right with
  | zero => simp [wireEquivalentAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases left <;> cases right <;> simp only [wireEquivalentAt] at related ⊢
      all_goals try assumption
      case array.array left right => exact all₂_lift (fun _ _ rel => ih smaller rel) related
      case object.object left right =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
        exact ⟨other, inside, same, ih smaller child⟩
      case oneof.oneof name value other child => exact ⟨related.1, ih smaller related.2⟩

/-- Finite wire height is positive even for empty containers. -/
theorem wireHeight_pos (wire : Wire) : 0 < wireHeight wire := by
  cases wire <;> simp [wireHeight] <;> omega

/-- A successful independent wire derivation needs no more than the left
wire's structural height. The source derivation depth is unrestricted. -/
theorem wireEquivalentAt_rebudget {numbers : NumericCodec} {depth budget : Nat}
    {left right : Wire} (related : wireEquivalentAt depth numbers left right)
    (enough : wireHeight left ≤ budget) : wireEquivalentAt budget numbers left right := by
  induction depth generalizing budget left right with
  | zero => simp [wireEquivalentAt] at related
  | succ depth ih =>
    cases budget with
    | zero => have positive := wireHeight_pos left; omega
    | succ budget =>
      cases left <;> cases right <;> simp only [wireEquivalentAt] at related ⊢
      all_goals try assumption
      case array.array left right =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have smaller := array_wireHeight_lt (List.of_mem_zip member).1
        omega
      case object.object left right =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
        refine ⟨other, inside, same, ih child ?_⟩
        have smaller := object_wireHeight_lt member
        omega
      case oneof.oneof name value other child =>
        refine ⟨related.1, ih related.2 ?_⟩
        simp only [wireHeight] at enough
        omega

/-- The public derived budget decides unbounded wire equality. A false result
therefore proves non-equality rather than merely exhausting a configured cap. -/
theorem wireEqual_iff_WireEquivalent (numbers : NumericCodec) (left right : Wire) :
    wireEqual numbers left right = true ↔ WireEquivalent numbers left right := by
  rw [wireEqual, wireEqualAt_iff]
  constructor
  · intro related; exact ⟨_, related⟩
  · rintro ⟨depth, related⟩
    exact wireEquivalentAt_rebudget related (by omega)

/-- Rejection by the public wire comparator excludes every finite equality derivation. -/
theorem wireEqual_false_iff (numbers : NumericCodec) (left right : Wire) :
    wireEqual numbers left right = false ↔ ¬ WireEquivalent numbers left right := by
  rw [← wireEqual_iff_WireEquivalent]
  cases wireEqual numbers left right <;> simp

/-- Exact snapshot equality is monotone too; it retains lexical number tokens
and must not be substituted for schema numerical equality. -/
theorem WireSameAt_mono {low high : Nat} {left right : Wire}
    (increase : low ≤ high) (related : WireSameAt low left right) : WireSameAt high left right := by
  induction low generalizing high left right with
  | zero => simp [WireSameAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases left <;> cases right <;> simp only [WireSameAt] at related ⊢
      all_goals try assumption
      case array.array left right => exact all₂_lift (fun _ _ rel => ih smaller rel) related
      case object.object left right =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
        exact ⟨other, inside, same, ih smaller child⟩
      case oneof.oneof name value other child => exact ⟨related.1, ih smaller related.2⟩

/-- Exact snapshot comparison is also adequate at finite wire height; no
semantic value-depth approximation is assumed for opaque JSON snapshots. -/
theorem WireSameAt_rebudget {depth budget : Nat} {left right : Wire}
    (related : WireSameAt depth left right) (enough : wireHeight left ≤ budget) :
    WireSameAt budget left right := by
  induction depth generalizing budget left right with
  | zero => simp [WireSameAt] at related
  | succ depth ih =>
    cases budget with
    | zero => have positive := wireHeight_pos left; omega
    | succ budget =>
      cases left <;> cases right <;> simp only [WireSameAt] at related ⊢
      all_goals try assumption
      case array.array left right =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have smaller := array_wireHeight_lt (List.of_mem_zip member).1
        omega
      case object.object left right =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
        refine ⟨other, inside, same, ih child ?_⟩
        have smaller := object_wireHeight_lt member
        omega
      case oneof.oneof name value other child =>
        refine ⟨related.1, ih related.2 ?_⟩
        simp only [wireHeight] at enough
        omega

/-- A sufficient exact-wire comparison budget decides the unbounded exact
snapshot relation. This is a separate lexical-preservation obligation. -/
theorem wireSameAt_budget_iff {budget : Nat} {left right : Wire}
    (enough : wireHeight left ≤ budget) :
    wireSameAt budget left right = true ↔ ∃ depth, WireSameAt depth left right := by
  rw [wireSameAt_iff]
  exact ⟨fun related => ⟨_, related⟩, fun ⟨_, related⟩ => WireSameAt_rebudget related enough⟩

/-- Additional derivation depth preserves target observation equality,
including the independently proved source-enum fallback for raw Any. -/
theorem targetEquivalentAt_mono
    {numbers : NumericCodec} {low high : Nat} {left right : Value}
    (increase : low ≤ high) (related : targetEquivalentAt low numbers left right) :
    targetEquivalentAt high numbers left right := by
  induction low generalizing high left right with
  | zero => simp [targetEquivalentAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases left <;> cases right <;> simp only [targetEquivalentAt] at related ⊢
      all_goals try assumption
      all_goals try exact enumEquivalentAt_mono increase related
      case array.array left right => exact all₂_lift (fun _ _ rel => ih smaller rel) related
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

/-- Every finite target-equality derivation fits the structural value-depth budget. Raw Any fallback uses the proved faithful source-enum bound. -/
theorem targetEquivalentAt_rebudget
    {numbers : NumericCodec} {depth budget : Nat} {left right : Value}
    (related : targetEquivalentAt depth numbers left right)
    (enough : valueDepth left + valueDepth right + 1 ≤ budget) :
    targetEquivalentAt budget numbers left right := by
  induction depth generalizing budget left right with
  | zero => simp [targetEquivalentAt] at related
  | succ depth ih =>
    cases budget with
    | zero => omega
    | succ budget =>
      cases left <;> cases right <;> simp only [targetEquivalentAt] at related ⊢
      all_goals try assumption
      all_goals try exact (enumEquivalentAt_budget enough).mp ⟨depth + 1, related⟩
      case array.array left right =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have l := array_valueDepth_lt (List.of_mem_zip member).1
        have r := array_valueDepth_lt (List.of_mem_zip member).2
        omega
      case object.object left le right re =>
        refine ⟨related.1, ?_, related.2.2.1, ?_⟩
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.1 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have l := object_valueDepth_lt (extra := le) member
          have r := object_valueDepth_lt (extra := re) inside
          omega
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have l := extra_valueDepth_lt (fields := left) member
          have r := extra_valueDepth_lt (fields := right) inside
          omega
      case map.map left right =>
        refine ⟨related.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2 entry member
        refine ⟨other, inside, same, ih child ?_⟩
        have l := map_valueDepth_lt member
        have r := map_valueDepth_lt inside
        omega
      case union.union identity branch value other otherBranch child =>
        refine ⟨related.1, related.2.1, ih related.2.2 ?_⟩
        simp only [valueDepth] at enough
        omega

/-- Schema enum membership uses the unbounded numerical wire relation; no
finite-budget miss can silently remove an otherwise equal member. -/
theorem schemaEnumAllowed_iff_unbounded (numbers : NumericCodec)
    (members : Option (List Wire)) (value : Wire) :
    schemaEnumAllowed numbers members value = true ↔
      match members with
      | none => True
      | some entries => ∃ member ∈ entries, WireEquivalent numbers value member := by
  cases members <;>
    simp [schemaEnumAllowed, List.any_eq_true, wireEqual_iff_WireEquivalent]

/-- Rejected schema enumeration means every listed member is unequal under
unbounded wire semantics, rather than only unequal at one guessed depth. -/
theorem schemaEnumAllowed_false_iff (numbers : NumericCodec)
    (members : Option (List Wire)) (value : Wire) :
    schemaEnumAllowed numbers members value = false ↔
      ∃ entries, members = some entries ∧ ∀ member ∈ entries, ¬ WireEquivalent numbers value member := by
  cases members with
  | none => simp [schemaEnumAllowed]
  | some entries =>
    have iff := schemaEnumAllowed_iff_unbounded numbers (some entries) value
    cases result : schemaEnumAllowed numbers (some entries) value <;> simp_all

/-- The derived target comparison budget decides the unbounded target observation relation. -/
theorem targetEqual_iff_TargetEquivalent
    (numbers : NumericCodec) (left right : Value) :
    targetEqualAt (valueDepth left + valueDepth right + 1) numbers left right = true ↔
      TargetEquivalent numbers left right := by
  rw [targetEqualAt_iff]
  constructor
  · intro related; exact ⟨_, related⟩
  · rintro ⟨depth, related⟩
    exact targetEquivalentAt_rebudget related (Nat.le_refl _)

/-- The existing target enum predicate is equivalent to genuinely unbounded membership, not merely one indexed comparison. -/
theorem targetEnumAllows_iff_unbounded
    (numbers : NumericCodec) (members : Option (List Value)) (value : Value) :
    TargetEnumAllows numbers members value ↔
      match members with
      | none => True
      | some entries => ∃ member ∈ entries, TargetEquivalent numbers value member := by
  cases members with
  | none => rfl
  | some entries =>
    simp only [TargetEnumAllows]
    apply exists_congr
    intro member
    apply and_congr_right
    intro inside
    exact (targetEqualAt_iff ..).symm.trans (targetEqual_iff_TargetEquivalent numbers value member)

/-- Executable target enum acceptance has an unbounded independent equality witness among the actual members. -/
theorem targetEnumAllowed_iff_unbounded
    (numbers : NumericCodec) (members : Option (List Value)) (value : Value) :
    targetEnumAllowed numbers members value = true ↔
      match members with
      | none => True
      | some entries => ∃ member ∈ entries, TargetEquivalent numbers value member :=
  (targetEnumAllowed_iff ..).trans (targetEnumAllows_iff_unbounded numbers members value)

/-- Target enum rejection excludes all unbounded equality witnesses for every listed member. -/
theorem targetEnumAllowed_false_iff
    (numbers : NumericCodec) (members : Option (List Value)) (value : Value) :
    targetEnumAllowed numbers members value = false ↔
      ∃ entries, members = some entries ∧ ∀ member ∈ entries, ¬ TargetEquivalent numbers value member := by
  cases members with
  | none => simp [targetEnumAllowed]
  | some entries =>
    have iff := targetEnumAllowed_iff_unbounded numbers (some entries) value
    cases result : targetEnumAllowed numbers (some entries) value <;> simp_all

/-- A false result at the derived target budget excludes all finite target
equality derivations, including deeper raw Any comparisons. -/
theorem targetEqual_false_iff (numbers : NumericCodec) (left right : Value) :
    targetEqualAt (valueDepth left + valueDepth right + 1) numbers left right = false ↔
      ¬ TargetEquivalent numbers left right := by
  rw [← targetEqual_iff_TargetEquivalent]
  cases targetEqualAt (valueDepth left + valueDepth right + 1) numbers left right <;> simp

end ValueContract.Candidate
