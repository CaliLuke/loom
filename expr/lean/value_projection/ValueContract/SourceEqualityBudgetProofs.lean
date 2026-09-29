import ValueContract.ValueEquality

namespace ValueContract.Candidate

variable {keys : KeyCodec}

/-- A raw comparison layer only uses its recursive relation positively. -/
theorem rawAnyLayer_mono {first second : Value → Value → Prop}
    (lift : ∀ left right, first left right → second left right)
    (left right : Value) : rawAnyLayer first left right → rawAnyLayer second left right := by
  cases left <;> cases right <;>
    simp [rawAnyLayer, rawNil, rawObject, rawSlice]
  all_goals repeat first | (simp_all; done) | split
  all_goals grind [All₂]

theorem rawAnyEquivalentAt_step (depth : Nat) (left right : Value) :
    rawAnyEquivalentAt depth left right → rawAnyEquivalentAt (depth + 1) left right := by
  induction depth generalizing left right with
  | zero => simp [rawAnyEquivalentAt]
  | succ depth ih => exact rawAnyLayer_mono ih left right

theorem rawAnyEquivalentAt_mono {small large : Nat} (bound : small ≤ large)
    (left right : Value) : rawAnyEquivalentAt small left right →
      rawAnyEquivalentAt large left right := by
  induction bound with
  | refl => exact id
  | step bound ih => exact fun h => rawAnyEquivalentAt_step _ _ _ (ih h)

theorem valueDepth_pos (value : Value) : 0 < valueDepth value := by
  cases value <;> simp [valueDepth] <;> omega

private theorem rawMap_member (entries : List (Scalar × Value))
    (output : List (String × Value))
    (mapped : entries.mapM (fun entry => match entry.1 with
      | .string name => some (name, entry.2) | _ => none) = some output)
    (entry : String × Value) (inside : entry ∈ output) :
    ∃ original ∈ entries, original.2 = entry.2 := by
  induction entries generalizing output with
  | nil => simp at mapped; subst output; simp at inside
  | cons head tail ih =>
    rcases head with ⟨key, value⟩
    cases key <;> simp [List.mapM_cons, Option.bind] at mapped
    case string name =>
      split at mapped
      · simp at mapped
      · rename_i values tailEq
        simp at mapped
        subst output
        rcases List.mem_cons.mp inside with same | rest
        · subst entry; exact ⟨(.string name, value), by simp, rfl⟩
        · obtain ⟨original, member, equal⟩ := ih _ tailEq rest
          exact ⟨original, by simp [member], equal⟩

theorem rawObject_valueDepth_lt {value : Value} {entries : List (String × Value)}
    (shape : rawObject value = some entries) {entry : String × Value}
    (inside : entry ∈ entries) : valueDepth entry.2 < valueDepth value := by
  induction value using rawObject.induct generalizing entries with
  | case1 identity payload ih =>
    have small := ih shape inside
    simp only [valueDepth]
    omega
  | case2 values =>
    simp only [rawObject, Option.some.injEq] at shape
    subst entries
    exact extra_valueDepth_lt inside
  | case3 values =>
    obtain ⟨original, member, equal⟩ := rawMap_member values entries shape entry inside
    rw [← equal]
    exact map_valueDepth_lt member
  | case4 value mismatch => simp [rawObject] at shape

/-- Raw bytes expose numeric elements only for the legacy slice comparison;
those synthetic scalars have depth one, no larger than the byte scalar. -/
theorem rawSlice_valueDepth_le {value : Value} {items : List Value}
    (shape : rawSlice value = some items) {item : Value} (inside : item ∈ items) :
    valueDepth item ≤ valueDepth value := by
  induction value using rawSlice.induct generalizing items with
  | case1 identity payload ih =>
    have small := ih shape inside
    simp only [valueDepth]
    omega
  | case2 values =>
    simp only [rawSlice, Option.some.injEq] at shape
    subst items
    exact Nat.le_of_lt (array_valueDepth_lt inside)
  | case3 bytes =>
    simp only [rawSlice, Option.some.injEq] at shape
    subst items
    obtain ⟨byte, _, same⟩ := List.mem_map.mp inside
    subst item
    simp [valueDepth]
  | case4 value mismatch => simp [rawSlice] at shape

private theorem rawSlice_smaller_or_bytes {value : Value} {items : List Value}
    (shape : rawSlice value = some items) {item : Value} (inside : item ∈ items) :
    valueDepth item < valueDepth value ∨ ∃ bytes, value = .scalar (.bytes bytes) := by
  induction value using rawSlice.induct generalizing items with
  | case1 identity payload _ =>
    have bound := rawSlice_valueDepth_le (value := payload) shape inside
    left; simp only [valueDepth]; omega
  | case2 values =>
    simp only [rawSlice, Option.some.injEq] at shape
    subst items
    exact Or.inl (array_valueDepth_lt inside)
  | case3 bytes => exact Or.inr ⟨bytes, rfl⟩
  | case4 value mismatch => simp [rawSlice] at shape

private theorem rawFallback_rebudget {first second : Value → Value → Prop}
    (left right : Value)
    (notBoth : ¬ ∃ l r, left = .scalar l ∧ right = .scalar r)
    (lift : ∀ l r, valueDepth l + valueDepth r < valueDepth left + valueDepth right →
      first l r → second l r)
    (related : match rawObject left, rawObject right with
      | some l, some r => l.length = r.length ∧ ∀ entry ∈ l, ∃ other ∈ r,
          entry.1 = other.1 ∧ first entry.2 other.2
      | _, _ => match rawSlice left, rawSlice right with
        | some l, some r => All₂ first l r | _, _ => False) :
    (match rawObject left, rawObject right with
      | some l, some r => l.length = r.length ∧ ∀ entry ∈ l, ∃ other ∈ r,
          entry.1 = other.1 ∧ second entry.2 other.2
      | _, _ => match rawSlice left, rawSlice right with
        | some l, some r => All₂ second l r | _, _ => False) := by
  cases ls : rawObject left <;> cases rs : rawObject right <;> simp only [ls, rs] at related ⊢
  case some.some l r =>
    refine ⟨related.1, ?_⟩
    intro entry inside
    obtain ⟨other, member, same, child⟩ := related.2 entry inside
    refine ⟨other, member, same, lift _ _ ?_ child⟩
    have lb := rawObject_valueDepth_lt ls inside
    have rb := rawObject_valueDepth_lt rs member
    omega
  all_goals cases la : rawSlice left <;> cases ra : rawSlice right <;>
    simp only [la, ra] at related ⊢
  all_goals try exact False.elim related
  all_goals
    refine ⟨related.1, ?_⟩
    intro pair member
    have lm := (List.of_mem_zip member).1
    have rm := (List.of_mem_zip member).2
    apply lift _ _ ?_ (related.2 pair member)
    have lb := rawSlice_valueDepth_le la lm
    have rb := rawSlice_valueDepth_le ra rm
    rcases rawSlice_smaller_or_bytes la lm with strict | ⟨bytes, same⟩
    · omega
    · rcases rawSlice_smaller_or_bytes ra rm with strict | ⟨otherBytes, otherSame⟩
      · omega
      · exact False.elim (notBoth ⟨.bytes bytes, .bytes otherBytes, same, otherSame⟩)

/-- Every recursive comparison inside one raw layer decreases combined
structural depth, including transparent host wrappers and Bytes/slice fallback. -/
theorem rawAnyLayer_rebudget {first second : Value → Value → Prop}
    (left right : Value)
    (lift : ∀ l r, valueDepth l + valueDepth r < valueDepth left + valueDepth right →
      first l r → second l r)
    (related : rawAnyLayer first left right) : rawAnyLayer second left right := by
  cases left <;> cases right <;>
    simp only [rawAnyLayer, rawNil] at related ⊢
  all_goals try exact related
  all_goals try
    exact lift _ _ (by simp only [valueDepth]; omega) related
  all_goals try
    exact related.elim Or.inl (fun child => Or.inr
      (lift _ _ (by simp only [valueDepth]; omega) child))
  all_goals simp only [Bool.or_self, Bool.false_eq_true, ↓reduceIte] at related ⊢
  all_goals try exact related
  all_goals exact rawFallback_rebudget _ _ (by simp) lift related

/-- Any successful raw comparison can be replayed at a budget derived only
from the two finite structures, regardless of its original derivation depth. -/
theorem rawAnyEquivalentAt_rebudget {depth budget : Nat} {left right : Value}
    (related : rawAnyEquivalentAt depth left right)
    (enough : valueDepth left + valueDepth right + 1 ≤ budget) :
    rawAnyEquivalentAt budget left right := by
  induction depth generalizing budget left right with
  | zero => simp [rawAnyEquivalentAt] at related
  | succ depth ih =>
    cases budget with
    | zero => omega
    | succ budget =>
      apply rawAnyLayer_rebudget left right ?_ related
      intro l r smaller child
      exact ih child (by omega)

theorem rawAnyEquivalentAt_budget {budget : Nat} {left right : Value}
    (enough : valueDepth left + valueDepth right + 1 ≤ budget) :
    (∃ depth, rawAnyEquivalentAt depth left right) ↔ rawAnyEquivalentAt budget left right :=
  ⟨fun ⟨_, related⟩ => rawAnyEquivalentAt_rebudget related enough, fun related => ⟨_, related⟩⟩

/-- Extra derivation depth preserves declared enum equality. -/
theorem enumEquivalentAt_mono {low high : Nat} {left right : Value}
    (increase : low ≤ high) (related : enumEquivalentAt keys low left right) :
    enumEquivalentAt keys high left right := by
  induction low generalizing high left right with
  | zero => simp [enumEquivalentAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases left <;> cases right <;> simp only [enumEquivalentAt] at related ⊢
      all_goals try assumption
      all_goals try exact rawAnyLayer_mono (rawAnyEquivalentAt_mono smaller) _ _ related
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
      case union.union identity branch payload other otherBranch otherPayload =>
        exact ⟨related.1, related.2.1, ih smaller related.2.2⟩
      case any.any left right => exact rawAnyEquivalentAt_mono smaller _ _ related
      case nilArray.array items => cases items <;> simp_all
      case array.nilArray items => cases items <;> simp_all
      case nilMap.map entries => cases entries <;> simp_all
      case map.nilMap entries => cases entries <;> simp_all

/-- Declared enum comparison has the same complete finite-depth bound. -/
theorem enumEquivalentAt_rebudget {depth budget : Nat} {left right : Value}
    (related : enumEquivalentAt keys depth left right)
    (enough : valueDepth left + valueDepth right + 1 ≤ budget) :
    enumEquivalentAt keys budget left right := by
  induction depth generalizing budget left right with
  | zero => simp [enumEquivalentAt] at related
  | succ depth ih =>
    cases budget with
    | zero => omega
    | succ budget =>
      cases left <;> cases right <;> simp only [enumEquivalentAt] at related ⊢
      all_goals try exact related
      all_goals try
        apply rawAnyLayer_rebudget _ _ ?_ related
        intro l r smaller child
        exact rawAnyEquivalentAt_rebudget child (by omega)
      case array.array left right =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have lb := array_valueDepth_lt (List.of_mem_zip member).1
        have rb := array_valueDepth_lt (List.of_mem_zip member).2
        omega
      case object.object left le right re =>
        refine ⟨related.1, ?_, related.2.2.1, ?_⟩
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.1 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have lb := object_valueDepth_lt (extra := le) member
          have rb := object_valueDepth_lt (extra := re) inside
          omega
        · intro entry member
          obtain ⟨other, inside, same, child⟩ := related.2.2.2 entry member
          refine ⟨other, inside, same, ih child ?_⟩
          have lb := extra_valueDepth_lt (fields := left) member
          have rb := extra_valueDepth_lt (fields := right) inside
          omega
      case map.map left right =>
        refine ⟨related.1, ?_⟩
        intro entry member
        obtain ⟨other, inside, same, child⟩ := related.2 entry member
        refine ⟨other, inside, same, ih child ?_⟩
        have lb := map_valueDepth_lt member
        have rb := map_valueDepth_lt inside
        omega
      case union.union identity branch payload other otherBranch otherPayload =>
        refine ⟨related.1, related.2.1, ih related.2.2 ?_⟩
        simp only [valueDepth] at enough
        omega
      case any.any left right =>
        apply rawAnyEquivalentAt_rebudget related
        simp only [valueDepth] at enough
        omega
      case nilArray.array items => cases items <;> simp_all
      case array.nilArray items => cases items <;> simp_all
      case nilMap.map entries => cases entries <;> simp_all
      case map.nilMap entries => cases entries <;> simp_all

/-- The derived budget is equivalent to the unbounded independent relation;
false therefore cannot be caused by a hidden fixed recursion cap. -/
theorem enumEquivalentAt_budget {budget : Nat} {left right : Value}
    (enough : valueDepth left + valueDepth right + 1 ≤ budget) :
    (∃ depth, enumEquivalentAt keys depth left right) ↔ enumEquivalentAt keys budget left right :=
  ⟨fun ⟨_, related⟩ => enumEquivalentAt_rebudget related enough, fun related => ⟨_, related⟩⟩

end ValueContract.Candidate
