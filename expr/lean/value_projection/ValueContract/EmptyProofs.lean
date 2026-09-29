import ValueContract.ProjectionObservation
import ValueContract.ObservationProofs
import ValueContract.ValueBudgetProofs

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

theorem emptyObservedAt_mono {depth larger : Nat} (bound : depth ≤ larger)
    {targets identity value} (relation : emptyObservedAt depth targets identity value) :
    emptyObservedAt larger targets identity value := by
  induction depth generalizing larger identity value with
  | zero => exact False.elim relation
  | succ depth ih =>
    cases larger with
    | zero => omega
    | succ larger =>
      have smaller : depth ≤ larger := by omega
      obtain ⟨declaration, member, same, relation⟩ := relation
      refine ⟨declaration, member, same, ?_⟩
      split at relation <;> try simp_all only
      all_goals try exact relation
      all_goals try exact ih smaller relation
      all_goals try simp
      obtain ⟨_, alternative, member, same, relation⟩ := relation
      exact ⟨alternative, member, same, ih smaller relation⟩

theorem valueAbsent_iff (value : Value) : valueAbsent value = true ↔ value = .absent := by
  cases value <;> simp [valueAbsent]

/-- The successful positive decision produces an independent finite emptiness
witness; zero budget is an error rather than a false emptiness result. -/
theorem emptyFuel_sound {targets fuel identity value}
    (accepted : emptyFuel targets fuel identity value = .ok true) :
    emptyObservedAt fuel targets identity value := by
  induction fuel generalizing identity value with
  | zero => simp [emptyFuel] at accepted
  | succ fuel ih =>
    simp only [emptyFuel] at accepted
    cases found : findTarget targets identity with
    | none => simp [found] at accepted
    | some declaration =>
      simp only [found] at accepted
      refine ⟨declaration, findTarget_mem found, findTarget_identity found, ?_⟩
      cases shape : declaration.target <;> cases value <;>
        simp only [shape, valueAbsent] at accepted ⊢
      all_goals try contradiction
      all_goals try exact ih accepted
      all_goals try simp_all

      case scalar.scalar encoding kind rules scalar =>
        cases kind <;> cases scalar <;> simp_all
      case object.object members additional fields extra =>
        intro member inside
        rcases accepted.2 member inside with absent | implicitDefault
        · exact Or.inl ((valueAbsent_iff _).mp absent)
        · exact Or.inr implicitDefault
      case union.union occurrence style alternatives actual branch payload =>
        cases style <;> simp_all only
        all_goals try simp_all
        split at accepted <;> try simp_all
        cases chosen : alternatives.find? (·.identity == branch) with
        | none => simp [chosen] at accepted
        | some alternative =>
          simp only [chosen] at accepted
          refine ⟨alternative, List.mem_of_find?_eq_some chosen, ?_, ih accepted⟩
          exact beq_iff_eq.mp (List.find?_some
            (p := fun candidate : TargetAlternative => candidate.identity == branch) chosen)

theorem findAlternative_of_mem {alternatives : List TargetAlternative} {alternative : TargetAlternative}
    (unique : (alternatives.map TargetAlternative.identity).Nodup)
    (member : alternative ∈ alternatives) :
    alternatives.find? (·.identity == alternative.identity) = some alternative := by
  induction alternatives with
  | nil => simp at member
  | cons head tail ih =>
    by_cases same : head.identity = alternative.identity
    · have equal := unique_key unique (List.mem_cons_self) member same
      subst head
      simp
    · simp only [List.map_cons, List.nodup_cons] at unique
      have inside : alternative ∈ tail := by
        rcases List.mem_cons.mp member with equal | inside
        · subst head; exact False.elim (same rfl)
        · exact inside
      simp [same, ih unique.2 inside]

/-- Every indexed independent emptiness derivation is executable at that depth.
Well-formed identity/branch lookup prevents declaration-order dependence. -/
theorem emptyFuel_complete_at {targets fuel identity value}
    (wellFormed : WellFormedTargets targets)
    (relation : emptyObservedAt fuel targets identity value) :
    emptyFuel targets fuel identity value = .ok true := by
  induction fuel generalizing identity value with
  | zero => exact False.elim relation
  | succ fuel ih =>
    obtain ⟨declaration, member, same, relation⟩ := relation
    have found := same ▸ findTarget_of_mem wellFormed member
    simp only [emptyFuel, found]
    split at relation <;> try simp_all only
    all_goals try contradiction
    all_goals try rfl
    all_goals try exact ih relation
    all_goals try simp_all

    case h_14 =>
      intro member inside
      rcases relation.2 member inside with absent | implicitDefault
      · exact Or.inl ((valueAbsent_iff _).mpr absent)
      · exact Or.inr implicitDefault
    case h_15 =>
      obtain ⟨_, alternative, inside, branch, child⟩ := relation
      have names := (wellFormed.2 declaration member).2.2
      simp_all only
      have chosen := findAlternative_of_mem names.1 inside
      rw [branch] at chosen
      simp [chosen, ih child]

/-- A graph-derived budget realizes every independent finite emptiness witness. -/
theorem emptyObservedAt_rebudget {targets depth fuel identity value declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (enough : valueNodeBudget targets declaration value ≤ fuel)
    (relation : emptyObservedAt depth targets identity value) :
    emptyObservedAt fuel targets identity value := by
  induction depth generalizing fuel identity value declaration with
  | zero => exact False.elim relation
  | succ depth ih =>
    cases fuel with
    | zero => unfold valueNodeBudget at enough; omega
    | succ fuel =>
      obtain ⟨actual, member, same, relation⟩ := relation
      have actualFound := same ▸ findTarget_of_mem wellFormed member
      have equal : declaration = actual := Option.some.inj (found.symm.trans actualFound)
      subst declaration
      refine ⟨actual, member, same, ?_⟩
      have childStep : ∀ child ∈ targetNonconsumingChildren actual.target, ∀ next,
          valueDepth next ≤ valueDepth value → emptyObservedAt depth targets child next →
            emptyObservedAt fuel targets child next := by
        intro child edge next height relation
        obtain ⟨childDeclaration, childFound, rank⟩ := target_nonconsuming_rank wellFormed actualFound edge
        apply ih childFound _ relation
        have smaller := valueNodeBudget_nonconsuming_lt (targets := targets) rank height
        omega
      split at relation <;> try simp_all only
      all_goals try exact relation
      all_goals try simp
      all_goals try exact childStep _ (by simp [targetNonconsumingChildren]) _ (Nat.le_refl _) relation
      obtain ⟨_, alternative, inside, branch, child⟩ := relation
      refine ⟨alternative, inside, branch, ?_⟩
      apply childStep alternative.child (by simpa [targetNonconsumingChildren] using
        (List.mem_map_of_mem (f := TargetAlternative.child) inside)) _ _ child
      simp only [valueDepth]
      omega

/-- Emptiness evaluation cannot silently run out of fuel on a supported target
root. False is a completed decision, not an approximation from a depth cutoff. -/
theorem emptyFuel_total {targets fuel identity value declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (enough : valueNodeBudget targets declaration value ≤ fuel) :
    ∃ result, emptyFuel targets fuel identity value = .ok result := by
  induction fuel generalizing identity value declaration with
  | zero => unfold valueNodeBudget at enough; omega
  | succ fuel ih =>
    have childStep : ∀ child ∈ targetNonconsumingChildren declaration.target, ∀ next,
        valueDepth next ≤ valueDepth value → ∃ result, emptyFuel targets fuel child next = .ok result := by
      intro child edge next height
      obtain ⟨childDeclaration, childFound, rank⟩ := target_nonconsuming_rank wellFormed found edge
      apply ih childFound
      have smaller := valueNodeBudget_nonconsuming_lt (targets := targets) rank height
      omega
    simp only [emptyFuel, found]
    cases shape : declaration.target <;> cases value <;>
      simp only [shape] at childStep ⊢
    all_goals try exact ⟨_, rfl⟩
    all_goals try exact childStep _ (by simp [targetNonconsumingChildren]) _ (Nat.le_refl _)
    case scalar.scalar encoding kind rules scalar =>
      cases kind <;> cases scalar <;> exact ⟨_, rfl⟩
    case union.union occurrence style alternatives actual branch payload =>
      cases style <;> try exact ⟨_, rfl⟩
      simp only
      split <;> try exact ⟨_, rfl⟩
      cases selected : alternatives.find? (·.identity == branch) with
      | none => exact ⟨_, rfl⟩
      | some alternative =>
        apply childStep alternative.child (by simpa [targetNonconsumingChildren] using
          (List.mem_map_of_mem (f := TargetAlternative.child) (List.mem_of_find?_eq_some selected)))
        simp only [valueDepth]
        omega

/-- The public decision is equivalent to the unbounded independent predicate. -/
theorem emptyObserved_true_iff {targets identity value declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    emptyObserved targets identity value = .ok true ↔ EmptyObserved targets identity value := by
  have valid := (validateTargets_iff targets).mpr wellFormed
  simp only [emptyObserved, valid, ↓reduceIte]
  constructor
  · intro accepted
    exact ⟨_, emptyFuel_sound accepted⟩
  · rintro ⟨depth, relation⟩
    exact emptyFuel_complete_at wellFormed (emptyObservedAt_rebudget wellFormed found
      (valueNodeBudget_le_valueBudget (findTarget_mem found)) relation)

/-- A negative answer excludes every finite independent emptiness derivation;
it cannot arise from insufficient evaluation fuel. -/
theorem emptyObserved_false_iff {targets identity value declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    emptyObserved targets identity value = .ok false ↔ ¬ EmptyObserved targets identity value := by
  have positive := emptyObserved_true_iff (value := value) wellFormed found
  constructor
  · intro rejected relation
    have accepted := positive.mpr relation
    rw [rejected] at accepted
    cases accepted
  · intro absent
    obtain ⟨result, completed⟩ := emptyFuel_total (value := value) wellFormed found
      (valueNodeBudget_le_valueBudget (findTarget_mem found))
    have valid := (validateTargets_iff targets).mpr wellFormed
    have same : emptyObserved targets identity value = .ok result := by
      simpa only [emptyObserved, valid, ↓reduceIte] using completed
    cases result
    · exact same
    · exact False.elim (absent (positive.mp same))

/-- Field-local omission implements its independent rule after child
observation, with a complete positive and negative emptiness decision. -/
theorem observePresence_iff {targets member value result declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets member.child = some declaration) :
    observePresence targets member value = .ok result ↔
      FieldPresence targets member.child member.presence value result := by
  cases presence : member.presence with
  | explicit => simp [observePresence, FieldPresence, presence, eq_comm]
  | implicitDefault scalar =>
    cases value <;> simp [observePresence, FieldPresence, presence, presenceValue, valueAbsent, eq_comm]
  | omitEmpty =>
    have positive := emptyObserved_true_iff (value := value) wellFormed found
    have negative := emptyObserved_false_iff (value := value) wellFormed found
    have valid := (validateTargets_iff targets).mpr wellFormed
    obtain ⟨empty, completed⟩ := emptyFuel_total (value := value) wellFormed found
      (valueNodeBudget_le_valueBudget (findTarget_mem found))
    have same : emptyObserved targets member.child value = .ok empty := by
      simpa only [emptyObserved, valid, ↓reduceIte] using completed
    cases empty
    · have notEmpty := negative.mp same
      simp [observePresence, presence, same, FieldPresence, notEmpty, eq_comm]
    · have isEmpty := positive.mp same
      simp [observePresence, presence, same, FieldPresence, isEmpty, eq_comm]

end ValueContract.Candidate
