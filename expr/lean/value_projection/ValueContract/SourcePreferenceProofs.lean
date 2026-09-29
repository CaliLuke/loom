import ValueContract.SourceOutcomeSpec
import ValueContract.SourceBodyProofs
import ValueContract.SourceBudgetProofs

namespace ValueContract.Candidate

/-- A positive preference decision yields an independent finite derivation.
Neither target projection nor successful branch resolution appears here. -/
theorem prefersObjectInput_sound {declarations budget identity input}
    (preferred : prefersObjectInput declarations budget identity input = true) :
    ObjectPreferenceAt budget declarations identity input := by
  induction budget generalizing identity with
  | zero => simp [prefersObjectInput] at preferred
  | succ budget ih =>
    simp only [prefersObjectInput] at preferred
    cases entries : objectEntries input with
    | none => simp [entries] at preferred
    | some values =>
      simp only [entries] at preferred
      cases found : findDeclaration declarations identity with
      | none => simp [found] at preferred
      | some declaration =>
        simp only [found] at preferred
        refine ⟨values, (objectEntries_iff ..).mp entries, declaration,
          findDeclaration_mem found, findDeclaration_identity found, ?_⟩
        cases shape : declaration.contract <;> simp only [shape] at preferred ⊢
        all_goals try trivial
        all_goals try exact ih preferred
        simpa only [List.any_eq_true, Bool.or_eq_true, beq_iff_eq] using preferred

/-- Every independent preference derivation executes at its own index. -/
theorem prefersObjectInput_complete_at {declarations budget identity input}
    (wellFormed : WellFormedDeclarations declarations)
    (relation : ObjectPreferenceAt budget declarations identity input) :
    prefersObjectInput declarations budget identity input = true := by
  induction budget generalizing identity with
  | zero => exact False.elim relation
  | succ budget ih =>
    obtain ⟨values, entries, declaration, member, same, relation⟩ := relation
    have found := same ▸ findDeclaration_of_mem wellFormed member
    have inputEntries := (objectEntries_iff ..).mpr entries
    simp only [prefersObjectInput, inputEntries, found]
    cases shape : declaration.contract <;> simp only [shape] at relation ⊢
    all_goals try rfl
    all_goals try exact ih relation
    simpa only [List.any_eq_true, Bool.or_eq_true, beq_iff_eq] using relation

/-- A declaration's same-input rank bounds every wrapper chain. This rebudgeting
uses the independent derivation, not the executable preference answer. -/
theorem ObjectPreferenceAt_rebudget {declarations depth budget identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (enough : declaration.expansionRank < budget)
    (relation : ObjectPreferenceAt depth declarations identity input) :
    ObjectPreferenceAt budget declarations identity input := by
  induction depth generalizing budget identity declaration with
  | zero => exact False.elim relation
  | succ depth ih =>
    cases budget with
    | zero => omega
    | succ budget =>
      obtain ⟨values, entries, actual, member, same, relation⟩ := relation
      have actualFound := same ▸ findDeclaration_of_mem wellFormed member
      have equal := Option.some.inj (found.symm.trans actualFound)
      subst declaration
      refine ⟨values, entries, actual, member, same, ?_⟩
      have childStep : ∀ child ∈ nonconsumingChildren actual.contract,
          ObjectPreferenceAt depth declarations child input →
            ObjectPreferenceAt budget declarations child input := by
        intro child edge relation
        obtain ⟨next, childFound, rank⟩ := declaration_nonconsuming_rank wellFormed actualFound edge
        exact ih childFound (by omega) relation
      cases shape : actual.contract <;> simp only [shape] at relation childStep ⊢
      all_goals try trivial
      all_goals try exact childStep _ (by simp [nonconsumingChildren]) relation

/-- The actual finite-table rank budget decides the unbounded preference rule,
including its established compatible non-object preference for object input. -/
theorem prefersObjectInput_iff {declarations budget identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (enough : declaration.expansionRank < budget) :
    prefersObjectInput declarations budget identity input = true ↔
      ObjectPreference declarations identity input := by
  constructor
  · intro preferred
    exact ⟨budget, prefersObjectInput_sound preferred⟩
  · rintro ⟨depth, relation⟩
    exact prefersObjectInput_complete_at wellFormed
      (ObjectPreferenceAt_rebudget wellFormed found enough relation)

/-- False excludes every independent preference derivation; a small traversal
cutoff cannot silently demote a supported candidate. -/
theorem prefersObjectInput_false_iff {declarations budget identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (enough : declaration.expansionRank < budget) :
    prefersObjectInput declarations budget identity input = false ↔
      ¬ ObjectPreference declarations identity input := by
  rw [← prefersObjectInput_iff wellFormed found enough]
  cases prefersObjectInput declarations budget identity input <;> simp

/-- The public resolver's table-derived rank always meets the preference budget
for every declared root; no input-size or fixed search bound is assumed. -/
theorem prefersObjectInput_maximum_iff {declarations identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration) :
    prefersObjectInput declarations (maximumExpansionRank declarations + 1) identity input = true ↔
      ObjectPreference declarations identity input :=
  prefersObjectInput_iff wellFormed found
    (Nat.lt_succ_of_le (declaration_rank_le_maximum (findDeclaration_mem found)))

end ValueContract.Candidate
