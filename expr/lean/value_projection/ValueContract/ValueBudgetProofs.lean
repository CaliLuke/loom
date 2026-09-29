import ValueContract.ProjectionObservation
import ValueContract.SchemaProofs

namespace ValueContract.Candidate

/-- A per-node structural budget; ranks are used only when no value child is
consumed. The public budget dominates every declaration in the finite graph. -/
def valueNodeBudget (targets : Targets) (declaration : TargetDeclaration) (value : Value) : Nat :=
  valueDepth value * (maximumTargetRank targets + 1) + declaration.expansionRank + 1

theorem valueDepth_positive (value : Value) : 0 < valueDepth value := by
  cases value <;> simp only [valueDepth] <;> omega

theorem valueNodeBudget_le_valueBudget {targets : Targets} {declaration : TargetDeclaration}
    {value : Value} (member : declaration ∈ targets) :
    valueNodeBudget targets declaration value ≤ valueBudget targets value := by
  have rank := targetRank_le_maximum member
  simp only [valueNodeBudget, valueBudget, Nat.add_mul]
  omega

theorem valueNodeBudget_nonconsuming_lt {targets : Targets} {parent child : TargetDeclaration}
    {value next : Value} (rank : child.expansionRank < parent.expansionRank)
    (height : valueDepth next ≤ valueDepth value) :
    valueNodeBudget targets child next < valueNodeBudget targets parent value := by
  have product := Nat.mul_le_mul_right (maximumTargetRank targets + 1) height
  unfold valueNodeBudget
  omega

theorem valueNodeBudget_consuming_lt {targets : Targets} {parent child : TargetDeclaration}
    {value next : Value} (member : child ∈ targets)
    (height : valueDepth next < valueDepth value) :
    valueNodeBudget targets child next < valueNodeBudget targets parent value := by
  have rank := targetRank_le_maximum member
  have product := Nat.mul_le_mul_right (maximumTargetRank targets + 1)
    (Nat.add_one_le_iff.mpr height)
  simp only [Nat.add_mul, Nat.one_mul] at product
  unfold valueNodeBudget
  omega

theorem memberValue_valueDepth_le (fields : List (Identity × Value))
    (extra : List (String × Value)) (identity : Identity) :
    valueDepth (memberValue fields identity) ≤ valueDepth (.object fields extra) := by
  unfold memberValue
  cases found : fields.find? (fun entry => entry.1 == identity) with
  | none => simp only [valueDepth]; omega
  | some entry => exact Nat.le_of_lt (object_valueDepth_lt (List.mem_of_find?_eq_some found))

theorem memberValue_present_depth_lt {fields : List (Identity × Value)}
    {extra : List (String × Value)} {identity : Identity}
    (present : valueAbsent (memberValue fields identity) = false) :
    valueDepth (memberValue fields identity) < valueDepth (.object fields extra) := by
  unfold memberValue at present ⊢
  cases found : fields.find? (fun entry => entry.1 == identity) with
  | none => simp [found, valueAbsent] at present
  | some entry => exact object_valueDepth_lt (List.mem_of_find?_eq_some found)

end ValueContract.Candidate
