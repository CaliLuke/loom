import ValueContract.ObservationProofs

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

theorem canonicalAt_null {depth : Nat} {codecs targets identity wire}
    (relation : canonicalAt depth codecs targets identity .null wire) : wire = .null := by
  induction depth generalizing identity wire with
  | zero => simp [canonicalAt] at relation
  | succ depth ih =>
    obtain ⟨declaration, _, _, relation⟩ := relation
    split at relation <;> try simp_all [valueIsNull]
    all_goals first
      | exact ih relation
      | exact False.elim (by have same := ih relation; contradiction)

theorem canonicalAt_absent {depth : Nat} {codecs targets identity wire} :
    ¬ canonicalAt depth codecs targets identity .absent wire := by
  induction depth generalizing identity wire with
  | zero => simp [canonicalAt]
  | succ depth ih =>
    rintro ⟨declaration, _, _, relation⟩
    split at relation <;> try simp_all [valueIsNull]
    all_goals first
      | exact ih relation
      | exact ih relation.2

/-- Canonical spelling is a function of the observed value and target plan,
independently of whether its schema or decoder accepts that spelling. -/
theorem canonicalAt_functional {depth : Nat} {codecs targets identity value first second}
    (wellFormed : WellFormedTargets targets)
    (one : canonicalAt depth codecs targets identity value first)
    (two : canonicalAt depth codecs targets identity value second) : first = second := by
  induction depth generalizing identity value first second with
  | zero => simp [canonicalAt] at one
  | succ depth ih =>
    simp only [canonicalAt] at one two
    obtain ⟨declaration, member, same, one⟩ := one
    obtain ⟨other, otherMember, otherSame, two⟩ := two
    have found := same ▸ findTarget_of_mem wellFormed member
    have otherFound := otherSame ▸ findTarget_of_mem wellFormed otherMember
    have equal : declaration = other := Option.some.inj (found.symm.trans otherFound)
    subst other
    split at one <;> try simp_all only
    all_goals try contradiction
    all_goals try (split at two <;> simp_all)
    all_goals try rfl
    all_goals try exact (canonicalAt_null two).symm
    all_goals try (have same := canonicalAt_null one; contradiction)
    all_goals try exact ih one.2 two.2
    all_goals try exact ih one two
    all_goals try exact ih one two.2
    all_goals try exact canonicalScalar_unique one two
    all_goals try exact canonicalScalar_unique one.2 two.2

    case h_7 =>
      apply all₂_functional (R := canonicalAt depth codecs targets _) _ one two
      intro value first second firstCanonical secondCanonical
      exact ih firstCanonical secondCanonical
    case h_8 =>
      obtain ⟨firstFragments, firstRelated, _, firstWire⟩ := one
      obtain ⟨secondFragments, secondRelated, _, secondWire⟩ := two
      have sameFragments : firstFragments = secondFragments := by
        apply all₂_functional _ firstRelated secondRelated
        intro entry first second firstRelation secondRelation
        exact Prod.ext (Option.some.inj ((keySpelling_progress firstRelation.1).symm.trans
          (keySpelling_progress secondRelation.1))) (ih firstRelation.2 secondRelation.2)
      rw [firstWire, secondWire, sameFragments]
    case h_9 =>
      obtain ⟨firstFragments, firstRelated, firstExtra, firstExtraRelated, _, firstWire⟩ := one
      obtain ⟨secondFragments, secondRelated, secondExtra, secondExtraRelated, _, secondWire⟩ := two
      have sameFragments : firstFragments = secondFragments := by
        apply all₂_functional _ firstRelated secondRelated
        intro member first second firstRelation secondRelation
        apply Prod.ext (firstRelation.1.trans secondRelation.1.symm)
        cases firstOutput : first.2 <;> cases secondOutput : second.2 <;>
          simp only [firstOutput, secondOutput] at firstRelation secondRelation ⊢
        · rcases firstRelation.2.2 with absent | implicitDefault
          · exact False.elim (secondRelation.2.1 absent)
          · simp [implicitDefault] at secondRelation
        · rcases secondRelation.2.2 with absent | implicitDefault
          · exact False.elim (firstRelation.2.1 absent)
          · simp [implicitDefault] at firstRelation
        · exact congrArg some (ih firstRelation.2.2.2 secondRelation.2.2.2)
      have sameExtra : firstExtra = secondExtra := by
        split at firstExtraRelated
        · rename_i included
          simp only [included, ite_true] at secondExtraRelated
          apply all₂_functional _ firstExtraRelated secondExtraRelated
          intro entry first second firstRelation secondRelation
          apply Prod.ext (firstRelation.1.symm.trans secondRelation.1)
          split at firstRelation <;> simp_all only
          exact Materializes_functional firstRelation.2 secondRelation.2
        · rename_i excluded
          simp only [excluded] at secondExtraRelated
          exact firstExtraRelated.trans secondExtraRelated.symm
      rw [firstWire, secondWire, sameFragments, sameExtra]
    case h_1 =>
      obtain ⟨_, firstAlt, firstMember, firstIdentity, firstCanonical⟩ := one
      obtain ⟨secondAlt, secondMember, secondIdentity, secondCanonical⟩ := two
      have names := (wellFormed.2 declaration otherMember).2.2
      simp_all only
      have sameAlt := unique_key names.1 firstMember secondMember
        (firstIdentity.trans secondIdentity.symm)
      subst secondAlt
      exact ih firstCanonical secondCanonical
    all_goals
      obtain ⟨_, firstAlt, firstMember, firstIdentity, firstPayload, firstCanonical, firstWire⟩ := one
      obtain ⟨secondAlt, secondMember, secondIdentity, secondPayload, secondCanonical, secondWire⟩ := two
      have names := (wellFormed.2 declaration otherMember).2.2
      simp_all only
      have sameAlt := unique_key names.1 firstMember secondMember
        (firstIdentity.trans secondIdentity.symm)
      subst secondAlt
      have payload := ih firstCanonical secondCanonical
      rw [payload]

theorem Canonical_functional {codecs targets identity value first second}
    (wellFormed : WellFormedTargets targets)
    (one : Canonical codecs targets identity value first)
    (two : Canonical codecs targets identity value second) : first = second := by
  obtain ⟨firstDepth, firstRelation⟩ := one
  obtain ⟨secondDepth, secondRelation⟩ := two
  exact canonicalAt_functional wellFormed
    (canonicalAt_mono (Nat.le_max_left firstDepth secondDepth) firstRelation)
    (canonicalAt_mono (Nat.le_max_right firstDepth secondDepth) secondRelation)

end ValueContract.Candidate
