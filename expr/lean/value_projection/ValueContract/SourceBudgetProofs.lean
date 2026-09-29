import ValueContract.SourceRawProofs

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

/-- Fuel exhaustion and missing declarations share this explicit internal
failure; valid graph/budget theorems must exclude it, including rejection paths. -/
def NoMalformed (result : Except Failure α) : Prop := result ≠ .error .malformedDeclaration

theorem NoMalformed_bind {first : Except Failure α} {next : α → Except Failure β}
    (safe : NoMalformed first) (steps : ∀ value, NoMalformed (next value)) :
    NoMalformed (first >>= next) := by
  cases first with
  | error failure => simpa [NoMalformed, Bind.bind, Except.bind] using safe
  | ok value => exact steps value

theorem NoMalformed_map {first : Except Failure α} (f : α → β)
    (safe : NoMalformed first) : NoMalformed (f <$> first) := by
  cases first with
  | error failure => simpa [NoMalformed, Functor.map, Except.map] using safe
  | ok value => simp [NoMalformed, Functor.map, Except.map]

theorem NoMalformed_mapM (f : α → Except Failure β) (items : List α)
    (safe : ∀ item ∈ items, NoMalformed (f item)) : NoMalformed (items.mapM f) := by
  induction items with
  | nil => simp [NoMalformed]
  | cons head tail ih =>
    have headSafe := safe head (by simp)
    have tailSafe := ih (fun item inside => safe item (by simp [inside]))
    cases first : f head <;> cases rest : tail.mapM f <;>
      simp_all [NoMalformed, List.mapM_cons, Bind.bind, Except.bind]

theorem combineChecked_noMalformed (results : List (Except Failure α))
    (safe : ∀ result ∈ results, NoMalformed result) : NoMalformed (combineChecked results) := by
  induction results with
  | nil => simp [combineChecked, NoMalformed]
  | cons head tail ih =>
    have headSafe := safe head (by simp)
    have tailSafe := ih (fun result inside => safe result (by simp [inside]))
    cases head <;> cases rest : combineChecked tail <;>
      simp only [combineChecked, rest]
    all_goals first
      | (simp [NoMalformed]; done)
      | simpa [NoMalformed] using headSafe
      | simpa only [rest] using tailSafe
      | skip
    split
    · simpa [NoMalformed] using headSafe
    · simpa only [rest] using tailSafe

theorem nameKeys_noMalformed (keys : KeyCodec) (values : List Scalar) :
    NoMalformed (nameKeys keys values) := by
  unfold nameKeys
  apply NoMalformed_bind
  · apply NoMalformed_mapM
    intro value _
    cases encodeKey keys value <;> simp [NoMalformed]
  · intro names
    split <;> simp [NoMalformed]

/-- Invalid, cyclic and unsupported raw data still report their semantic
failure at the derived budget; exhaustion cannot masquerade as rejection. -/
theorem resolveRaw_noMalformed {keys : KeyCodec} {budget : Nat} {input : Input}
    (enough : inputDepth input ≤ budget) : NoMalformed (resolveRawAt keys budget input) := by
  induction budget generalizing input with
  | zero => have positive := inputDepth_pos input; omega
  | succ budget ih =>
    cases input <;> simp only [resolveRawAt]
    all_goals try (simp [NoMalformed]; done)
    case host identity payload =>
      apply NoMalformed_bind
      · apply ih
        simp only [inputDepth] at enough
        omega
      · intro value; simp [NoMalformed]
    case array items =>
      apply NoMalformed_bind
      · apply combineChecked_noMalformed
        intro result member
        obtain ⟨input, inside, same⟩ := List.mem_map.mp member
        subst result
        apply ih
        have smaller := array_inputDepth_lt inside
        omega
      · intro value; simp [NoMalformed]
    case object entries =>
      split
      · simp [NoMalformed, exceptThrow, Bind.bind, Except.bind]
      · apply NoMalformed_bind
        · apply combineChecked_noMalformed
          intro result member
          obtain ⟨entry, inside, same⟩ := List.mem_map.mp member
          subst result
          apply NoMalformed_bind
          · apply ih
            have smaller := object_inputDepth_lt inside
            omega
          · intro value; simp [NoMalformed]
        · intro values; simp [NoMalformed]
    case map entries =>
      apply NoMalformed_bind (nameKeys_noMalformed keys _)
      intro names
      apply NoMalformed_bind
      · apply combineChecked_noMalformed
        intro result member
        obtain ⟨entry, inside, same⟩ := List.mem_map.mp member
        subst result
        apply NoMalformed_bind
        · apply ih
          have smaller := map_inputDepth_lt inside
          omega
        · intro value; simp [NoMalformed]
      · intro values; simp [NoMalformed]

/-- Resetting the graph counter after input descent admits every declaration
in the validated finite table; it is not a separately chosen fuel constant. -/
theorem declaration_rank_le_maximum {declarations : Declarations} {declaration : Declaration}
    (member : declaration ∈ declarations) :
    declaration.expansionRank ≤ maximumExpansionRank declarations := by
  have bound := foldMax_member (List.mem_map_of_mem (f := Declaration.expansionRank) member) 0
  simpa only [maximumExpansionRank, List.foldl_map] using bound

end ValueContract.Candidate
