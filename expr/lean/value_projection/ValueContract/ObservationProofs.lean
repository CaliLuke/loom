import ValueContract.Canonical
import ValueContract.GraphValidation
import ValueContract.MaterializationProofs

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

theorem all₂_mono {R S : α → β → Prop} {left : List α} {right : List β}
    (relation : All₂ R left right) (step : ∀ a b, R a b → S a b) : All₂ S left right :=
  ⟨relation.1, fun pair member => step _ _ (relation.2 pair member)⟩

/-- Observation depths bound a derivation, never the admitted finite-value domain. -/
theorem observeAt_mono {depth larger : Nat} (bound : depth ≤ larger)
    {codecs targets role identity value observed}
    (relation : observeAt depth codecs targets role identity value observed) :
    observeAt larger codecs targets role identity value observed := by
  induction depth generalizing larger identity value observed with
  | zero => simp [observeAt] at relation
  | succ depth ih =>
    cases larger with
    | zero => omega
    | succ larger =>
      have smaller : depth ≤ larger := by omega
      simp only [observeAt] at relation ⊢
      obtain ⟨declaration, member, same, relation⟩ := relation
      refine ⟨declaration, member, same, ?_⟩
      split at relation <;> try simp_all only
      all_goals try exact relation
      all_goals try exact ih smaller relation
      all_goals try exact ⟨relation.1, ih smaller relation.2⟩
      all_goals try exact all₂_mono relation (fun _ _ child => ih smaller child)
      all_goals try simp
      case h_10 =>
        exact all₂_mono relation (fun _ _ child => ⟨child.1, ih smaller child.2⟩)
      case h_11 =>
        constructor
        · apply all₂_mono relation.1
          intro member result child
          obtain ⟨same, childObserved, observed, presence⟩ := child
          exact ⟨same, childObserved, ih smaller observed, presence⟩
        · split
          · rename_i chosen
            simp only [chosen, ite_true] at relation
            apply all₂_mono relation.2
            intro entry result related
            exact ⟨related.1, Nat.lt_of_lt_of_le related.2.1 smaller, related.2.2⟩
          · rename_i chosen
            simp only [chosen] at relation
            exact relation.2
      case h_12 =>
        obtain ⟨_, _, _, alternative, member, same, observed⟩ := relation
        exact ⟨alternative, member, same, ih smaller observed⟩

/-- Canonical wire witnesses have unbounded finite derivations. -/
theorem canonicalAt_mono {depth larger : Nat} (bound : depth ≤ larger)
    {codecs targets identity value wire}
    (relation : canonicalAt depth codecs targets identity value wire) :
    canonicalAt larger codecs targets identity value wire := by
  induction depth generalizing larger identity value wire with
  | zero => simp [canonicalAt] at relation
  | succ depth ih =>
    cases larger with
    | zero => omega
    | succ larger =>
      have smaller : depth ≤ larger := by omega
      simp only [canonicalAt] at relation ⊢
      obtain ⟨declaration, member, same, relation⟩ := relation
      refine ⟨declaration, member, same, ?_⟩
      split at relation <;> try simp_all only
      all_goals try exact relation
      all_goals try exact ih smaller relation
      all_goals try exact ⟨relation.1, ih smaller relation.2⟩
      all_goals try exact all₂_mono relation (fun _ _ child => ih smaller child)
      all_goals try simp

      case h_8 =>
        obtain ⟨fragments, related, unique, wire⟩ := relation
        exact ⟨fragments, all₂_mono related (fun _ _ child => ⟨child.1, ih smaller child.2⟩),
          unique, wire⟩
      case h_9 =>
        obtain ⟨fragments, additional, related, extra, unique, wire⟩ := relation
        refine ⟨fragments, ?_, additional, extra, ?_, wire⟩
        · apply all₂_mono related
          intro member fragment child
          refine ⟨child.1, ?_⟩
          cases found : fragment.2 with
          | none => simpa only [found] using child.2
          | some childWire =>
            simp only [found] at child ⊢
            exact ⟨child.2.1, child.2.2.1, ih smaller child.2.2.2⟩
        · simpa only [List.map_append] using unique
      case h_10 =>
        obtain ⟨_, alternative, member, same, childWire, child, wire⟩ := relation
        exact ⟨alternative, member, same, childWire, ih smaller child, wire⟩

theorem all₂_cons {R : α → β → Prop} {a : α} {b : β}
    {left : List α} {right : List β} :
    All₂ R (a :: left) (b :: right) ↔ R a b ∧ All₂ R left right := by
  simp [All₂, and_left_comm]

theorem all₂_functional {R : α → β → Prop} {left : List α} {first second : List β}
    (functional : ∀ a b c, R a b → R a c → b = c)
    (one : All₂ R left first) (two : All₂ R left second) : first = second := by
  induction left generalizing first second with
  | nil =>
    have firstEmpty : first = [] := by simpa using one.1.symm
    have secondEmpty : second = [] := by simpa using two.1.symm
    simp [firstEmpty, secondEmpty]
  | cons head tail ih =>
    cases first with
    | nil => simp [All₂] at one
    | cons first rest =>
      cases second with
      | nil => simp [All₂] at two
      | cons second remaining =>
        obtain ⟨oneHead, oneTail⟩ := all₂_cons.mp one
        obtain ⟨twoHead, twoTail⟩ := all₂_cons.mp two
        rw [functional _ _ _ oneHead twoHead, ih oneTail twoTail]

theorem FieldPresence_functional {targets child presence value first second}
    (one : FieldPresence targets child presence value first)
    (two : FieldPresence targets child presence value second) : first = second := by
  cases presence with
  | explicit => exact one.trans two.symm
  | implicitDefault scalar => exact one.trans two.symm
  | omitEmpty =>
    rcases one with ⟨empty, one⟩ | ⟨notEmpty, one⟩ <;>
      rcases two with ⟨emptyTwo, two⟩ | ⟨notEmptyTwo, two⟩
    · exact one.trans two.symm
    · exact False.elim (notEmptyTwo empty)
    · exact False.elim (notEmpty emptyTwo)
    · exact one.trans two.symm

theorem ObservedKey_functional {codec kind value first second}
    (one : ObservedKey codec kind value first) (two : ObservedKey codec kind value second) :
    first = second := by
  cases one with
  | scalar compatible => cases two; rfl
  | builtin spelling =>
    cases two with
    | builtin other =>
      exact congrArg Scalar.string (Option.some.inj
        ((keySpelling_progress spelling).symm.trans (keySpelling_progress other)))

theorem Materializes_functional {codecs value first second}
    (one : Materializes codecs value first) (two : Materializes codecs value second) :
    first = second :=
  Except.ok.inj (((materialize_iff_Materializes ..).mpr one).symm.trans
    ((materialize_iff_Materializes ..).mpr two))

theorem observeAt_absent {depth : Nat} {codecs targets role identity observed}
    (relation : observeAt depth codecs targets role identity .absent observed) :
    observed = .absent := by
  induction depth generalizing identity observed with
  | zero => simp [observeAt] at relation
  | succ depth ih =>
    obtain ⟨declaration, _, _, relation⟩ := relation
    split at relation <;> simp_all
    all_goals first
      | exact False.elim (by have same := ih relation; contradiction)
      | exact False.elim (by have same := ih relation.2; contradiction)

theorem observeAt_null {depth : Nat} {codecs targets role identity observed}
    (relation : observeAt depth codecs targets role identity .null observed) :
    observed = .null := by
  induction depth generalizing identity observed with
  | zero => simp [observeAt] at relation
  | succ depth ih =>
    obtain ⟨declaration, _, _, relation⟩ := relation
    split at relation <;> simp_all [valueIsNull]
    all_goals first
      | exact ih relation
      | exact False.elim (by have same := ih relation; contradiction)

theorem unique_key {entries : List α} {key : α → β} {first second : α}
    (unique : (entries.map key).Nodup) (one : first ∈ entries) (two : second ∈ entries)
    (same : key first = key second) : first = second := by
  induction entries with
  | nil => simp at one
  | cons head tail ih =>
    simp only [List.map_cons, List.nodup_cons] at unique
    rcases List.mem_cons.mp one with one | one <;>
      rcases List.mem_cons.mp two with two | two
    · exact one.trans two.symm
    · subst first
      exact False.elim (unique.1 (same ▸ List.mem_map_of_mem two))
    · subst second
      exact False.elim (unique.1 (same ▸ List.mem_map_of_mem one))
    · exact ih unique.2 one two

/-- Independent observation is deterministic for a well-formed target graph. -/
theorem observeAt_functional {depth : Nat} {codecs targets role identity value first second}
    (wellFormed : WellFormedTargets targets)
    (one : observeAt depth codecs targets role identity value first)
    (two : observeAt depth codecs targets role identity value second) : first = second := by
  induction depth generalizing identity value first second with
  | zero => simp [observeAt] at one
  | succ depth ih =>
    simp only [observeAt] at one two
    obtain ⟨declaration, member, same, one⟩ := one
    obtain ⟨other, otherMember, otherSame, two⟩ := two
    have found := same ▸ findTarget_of_mem wellFormed member
    have otherFound := otherSame ▸ findTarget_of_mem wellFormed otherMember
    have equal : declaration = other := Option.some.inj (found.symm.trans otherFound)
    subst other
    split at one <;> try simp_all only
    all_goals try contradiction
    all_goals try (split at two <;> simp_all)
    all_goals try exact (observeAt_absent two).symm
    all_goals try exact (observeAt_null two).symm
    all_goals try exact (observeAt_absent two.2).symm
    all_goals try (have absent := observeAt_absent one; contradiction)
    all_goals try (have absent := observeAt_absent one.2; contradiction)
    all_goals try (have null := observeAt_null one; contradiction)
    all_goals try rfl
    all_goals try exact ih one two
    case h_6 => exact ih one two.2
    case h_9 =>
      apply all₂_functional (R := observeAt depth codecs targets role _) _ one two
      intro value first second firstObs secondObs
      exact ih firstObs secondObs
    case h_10 =>
      apply all₂_functional _ one two
      intro entry first second one two
      exact Prod.ext (ObservedKey_functional one.1 two.1) (ih one.2 two.2)
    case h_11 =>
      constructor
      · apply all₂_functional _ one.1 two.1
        intro member first second firstRelation secondRelation
        obtain ⟨firstName, firstObserved, firstObs, firstPresence⟩ := firstRelation
        obtain ⟨secondName, secondObserved, secondObs, secondPresence⟩ := secondRelation
        have sameObserved := ih firstObs secondObs
        subst secondObserved
        exact Prod.ext (firstName.symm.trans secondName)
          (FieldPresence_functional firstPresence secondPresence)
      · have firstExtra := one.2
        have secondExtra := two.2
        split at firstExtra
        · rename_i chosen
          simp only [chosen, ite_true] at secondExtra
          apply all₂_functional _ firstExtra secondExtra
          intro entry first second firstRelation secondRelation
          obtain ⟨firstName, _, firstWire, firstValue, firstMaterialized, _⟩ := firstRelation
          obtain ⟨secondName, _, secondWire, secondValue, secondMaterialized, _⟩ := secondRelation
          apply Prod.ext (firstName.symm.trans secondName)
          rw [firstValue, secondValue, Materializes_functional firstMaterialized secondMaterialized]
        · rename_i chosen
          simp only [chosen] at secondExtra
          exact firstExtra.trans secondExtra.symm
    case h_12 =>
      obtain ⟨_, _, _, firstAlt, firstMember, firstIdentity, firstObs⟩ := one
      obtain ⟨_, _, _, secondAlt, secondMember, secondIdentity, secondObs⟩ := two
      have names := (wellFormed.2 declaration otherMember).2.2
      simp_all only
      have sameAlt := unique_key names.1 firstMember secondMember
        (firstIdentity.trans secondIdentity.symm)
      subst secondAlt
      exact ih firstObs secondObs
    case h_13 => exact Materializes_functional one.1 two.1


theorem Observe_functional {codecs targets role identity value first second}
    (wellFormed : WellFormedTargets targets)
    (one : Observe codecs targets role identity value first)
    (two : Observe codecs targets role identity value second) : first = second := by
  obtain ⟨firstDepth, firstRelation⟩ := one
  obtain ⟨secondDepth, secondRelation⟩ := two
  exact observeAt_functional wellFormed
    (observeAt_mono (Nat.le_max_left firstDepth secondDepth) firstRelation)
    (observeAt_mono (Nat.le_max_right firstDepth secondDepth) secondRelation)

end ValueContract.Candidate
