import ValueContract.EmptyProofs
import ValueContract.MaterializationValidity

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

private theorem bind_ok {input : Except ε α} {next : α → Except ε β} {result : β} :
    (input >>= next) = .ok result ↔ ∃ value, input = .ok value ∧ next value = .ok result := by
  cases input <;> simp [Bind.bind, Except.bind]

private theorem mapM_ok (f : α → Except ε β) (left : List α) (right : List β) :
    left.mapM f = .ok right ↔ All₂ (fun a b => f a = .ok b) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂, pure, Except.pure]
  | cons head tail ih =>
    simp only [List.mapM_cons, bind_ok]
    cases right with
    | nil => simp [All₂, pure, Except.pure]
    | cons result rest =>
      simp only [all₂_cons]
      constructor
      · rintro ⟨value, first, values, remaining, same⟩
        cases same
        exact ⟨first, (ih _).mp remaining⟩
      · rintro ⟨first, remaining⟩
        exact ⟨result, first, rest, (ih _).mpr remaining, rfl⟩

theorem observeKey_iff {codec kind value result} :
    observeKey codec kind value = some result ↔ ObservedKey codec kind value result := by
  cases kind with
  | scalar kind =>
    simp only [observeKey]
    split
    · rename_i compatible
      constructor
      · intro same; cases same; exact .scalar compatible
      · intro relation; cases relation; rfl
    · rename_i incompatible
      constructor
      · intro same; contradiction
      · intro relation; cases relation; contradiction
  | builtin =>
    simp only [observeKey, Option.map_eq_some_iff]
    constructor
    · rintro ⟨name, spelling, same⟩
      subst result
      exact .builtin (keySpelling_sound spelling)
    · intro relation
      cases relation with
      | builtin spelling => exact ⟨_, keySpelling_progress spelling, rfl⟩

theorem materializeBuilt_iff {codecs value wire} :
    materializeBuilt codecs value = .ok wire ↔ Materializes codecs value wire := by
  rw [← materialize_iff_Materializes]
  unfold materializeBuilt
  cases found : materialize codecs value with
  | ok result => simp
  | error failure => cases failure <;> simp

private theorem observedMembers_sound {codecs targets role fuel depth declaration members fields results}
    (wellFormed : WellFormedTargets targets) (member : declaration ∈ targets)
    (shape : declaration.target = .object members preserve)
    (children : ∀ identity value observed,
      observeFuel codecs targets role fuel identity value = .ok observed →
      observeAt depth codecs targets role identity value observed)
    (done : members.mapM (fun member => do
      let value ← observeFuel codecs targets role fuel member.child (memberValue fields member.identity)
      let visible ← observePresence targets member value
      pure (member.identity, visible)) = .ok results) :
    All₂ (fun member result => member.identity = result.1 ∧
      ∃ childObserved, observeAt depth codecs targets role member.child
        (memberValue fields member.identity) childObserved ∧
        FieldPresence targets member.child member.presence childObserved result.2) members results := by
  have related := (mapM_ok ..).mp done
  refine ⟨related.1, ?_⟩
  intro pair inside
  have done := related.2 pair inside
  obtain ⟨childObserved, childDone, rest⟩ := bind_ok.mp done
  obtain ⟨visible, presenceDone, same⟩ := bind_ok.mp rest
  rw [← Except.ok.inj same]
  have edge : pair.1.child ∈ targetConsumingChildren declaration.target := by
    rw [shape]
    exact List.mem_map_of_mem (List.of_mem_zip inside).1
  obtain ⟨childDeclaration, childFound⟩ := target_consuming_exists wellFormed
    (findTarget_of_mem wellFormed member) edge
  exact ⟨rfl, childObserved, children _ _ _ childDone,
    (observePresence_iff wellFormed childFound).mp presenceDone⟩

private theorem observedExtras_sound {codecs depth extra results} (positive : 0 < depth)
    (done : extra.mapM (fun entry : String × Value => do
      let wire ← materializeBuilt codecs entry.2
      pure (entry.1, Value.jsonSnapshot wire)) = .ok results) :
    All₂ (fun entry result => entry.1 = result.1 ∧
      observeFreeAt depth codecs entry.2 result.2) extra results := by
  apply all₂_mono ((mapM_ok ..).mp done)
  intro entry result related
  obtain ⟨wire, built, same⟩ := bind_ok.mp related
  cases same
  have materialized := materializeBuilt_iff.mp built
  exact ⟨rfl, positive, wire, rfl, materialized, Materializes_JSONValid materialized⟩

/-- Every successful observation has an independent derivation. The extra
successor accounts for codec-owned additional members, whose materialization
is independent of the structural target recursion budget. -/
theorem observeFuel_sound {codecs targets role fuel identity value observed}
    (wellFormed : WellFormedTargets targets)
    (done : observeFuel codecs targets role fuel identity value = .ok observed) :
    observeAt (fuel + 1) codecs targets role identity value observed := by
  induction fuel generalizing identity value observed with
  | zero => simp [observeFuel] at done
  | succ fuel ih =>
    simp only [observeFuel] at done
    cases found : findTarget targets identity with
    | none => simp [found] at done
    | some declaration =>
      simp only [found] at done
      refine ⟨declaration, findTarget_mem found, findTarget_identity found, ?_⟩
      cases shape : declaration.target <;> cases value <;>
        simp only [shape] at done ⊢
      all_goals try contradiction
      all_goals try (cases done; trivial)
      all_goals try exact ih done
      all_goals try (split at done <;> simp_all [pure, Except.pure])
      all_goals try (split at done <;> try contradiction; exact ⟨by simp_all, ih done⟩)
      all_goals try (subst observed; simp_all)

      case array.array child bounds values =>
        obtain ⟨results, related, same⟩ := bind_ok.mp done
        cases same
        exact all₂_mono ((mapM_ok ..).mp related) (fun _ _ relation => ih relation)
      case map.map kind keyRules child bounds entries =>
        obtain ⟨results, related, same⟩ := bind_ok.mp done
        cases same
        apply all₂_mono ((mapM_ok ..).mp related)
        intro entry result related
        cases key : observeKey codecs.numbers kind entry.1 with
        | none => simp [key] at related
        | some keyValue =>
          simp only [key] at related
          obtain ⟨childValue, childDone, same⟩ := bind_ok.mp related
          cases same
          exact ⟨observeKey_iff.mp key, ih childDone⟩
      case any.any payload =>
        obtain ⟨wire, built, same⟩ := bind_ok.mp done
        cases same
        have materialized := materializeBuilt_iff.mp built
        exact ⟨materialized, Materializes_JSONValid materialized⟩
      case h_1 =>
        rename_i alternative selected
        obtain ⟨childValue, childDone, same⟩ := bind_ok.mp done
        cases same
        simp only
        refine ⟨trivial, trivial, trivial, alternative, List.mem_of_find?_eq_some selected, ?_, ih childDone⟩
        exact beq_iff_eq.mp (List.find?_some
          (p := fun candidate : TargetAlternative => candidate.identity == _) selected)
      case h_2 => contradiction
      case object.object.isTrue members preserve fields extra included =>
        obtain ⟨results, membersDone, rest⟩ := bind_ok.mp done
        obtain ⟨additional, extrasDone, same⟩ := bind_ok.mp rest
        cases same
        exact ⟨observedMembers_sound wellFormed (findTarget_mem found) shape
          (fun _ _ _ => ih) membersDone, observedExtras_sound (by omega) extrasDone⟩
      case object.object.isFalse members preserve fields extra exclude =>
        obtain ⟨results, membersDone, same⟩ := bind_ok.mp done
        cases same
        exact ⟨observedMembers_sound wellFormed (findTarget_mem found) shape
          (fun _ _ _ => ih) membersDone, rfl⟩

private theorem all₂_mono_mem {R S : α → β → Prop} {left : List α} {right : List β}
    (relation : All₂ R left right) (step : ∀ a ∈ left, ∀ b, R a b → S a b) : All₂ S left right :=
  ⟨relation.1, fun pair member => step _ (List.of_mem_zip member).1 _ (relation.2 pair member)⟩

/-- Every independent finite observation executes within the graph/value-derived
budget. The derivation depth does not restrict the admitted finite values. -/
theorem observeFuel_complete {codecs targets role depth fuel identity value observed declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (enough : valueNodeBudget targets declaration value ≤ fuel)
    (relation : observeAt depth codecs targets role identity value observed) :
    observeFuel codecs targets role fuel identity value = .ok observed := by
  induction depth generalizing fuel identity value observed declaration with
  | zero => exact False.elim relation
  | succ depth ih =>
    cases fuel with
    | zero => unfold valueNodeBudget at enough; omega
    | succ fuel =>
      obtain ⟨actual, member, same, relation⟩ := relation
      have actualFound := same ▸ findTarget_of_mem wellFormed member
      have equal : declaration = actual := Option.some.inj (found.symm.trans actualFound)
      subst declaration
      have childStep : ∀ child ∈ targetNonconsumingChildren actual.target, ∀ next result,
          valueDepth next ≤ valueDepth value → observeAt depth codecs targets role child next result →
            observeFuel codecs targets role fuel child next = .ok result := by
        intro child edge next result height relation
        obtain ⟨childDeclaration, childFound, rank⟩ := target_nonconsuming_rank wellFormed actualFound edge
        apply ih childFound _ relation
        have smaller := valueNodeBudget_nonconsuming_lt (targets := targets) rank height
        omega
      have consumedStep : ∀ child ∈ targetConsumingChildren actual.target, ∀ next result,
          valueDepth next < valueDepth value → observeAt depth codecs targets role child next result →
            observeFuel codecs targets role fuel child next = .ok result := by
        intro child edge next result height relation
        obtain ⟨childDeclaration, childFound⟩ := target_consuming_exists wellFormed actualFound edge
        apply ih childFound _ relation
        have smaller := valueNodeBudget_consuming_lt (targets := targets) (parent := actual)
          (findTarget_mem childFound) height
        omega
      simp only [observeFuel, actualFound]
      split at relation <;> try simp_all only
      all_goals try contradiction
      all_goals try rfl
      all_goals try (cases relation.2; simp_all)
      all_goals try simp_all

      case h_4 =>
        rename_i input output target ignoredValue ignoredObserved child shape noAbsent noNull
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation
        have absent : input = .absent → output = .absent := fun same => observeAt_absent (same ▸ relation)
        have null : input = .null → output = .null := fun same => observeAt_null (same ▸ relation)
        cases input <;> simp_all
      case h_5 =>
        rename_i input output target ignoredValue ignoredObserved child shape noAbsent
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation
        have absent : input = .absent → output = .absent := fun same => observeAt_absent (same ▸ relation)
        cases input <;> simp_all
      case h_6 =>
        rename_i input output target ignoredValue ignoredObserved child shape noAbsent
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation.2
        have absent : input = .absent → output = .absent := fun same => observeAt_absent (same ▸ relation.2)
        cases input <;> simp_all
      case h_9 =>
        rename_i target ignoredValue ignoredObserved child bounds values results shape
        have done : values.mapM (observeFuel codecs targets role fuel child) = .ok results := by
          apply (mapM_ok ..).mpr
          apply all₂_mono_mem relation
          intro value inside result childRelation
          exact consumedStep child (by simp [targetConsumingChildren]) value result
            (array_valueDepth_lt inside) childRelation
        simp [done]

      case h_10 =>
        rename_i target ignoredValue ignoredObserved kind keyRules child bounds entries results shape
        have done : entries.mapM (fun entry => do
            let some key := observeKey codecs.numbers kind entry.1 | Except.error BuildFailure.unrepresentable
            let observed ← observeFuel codecs targets role fuel child entry.2
            pure (key, observed)) = .ok results := by
          apply (mapM_ok ..).mpr
          apply all₂_mono_mem relation
          intro entry inside result childRelation
          have key := observeKey_iff.mpr childRelation.1
          have value := consumedStep child (by simp [targetConsumingChildren]) entry.2 result.2
            (map_valueDepth_lt inside) childRelation.2
          simp [key, value]
        exact bind_ok.mpr ⟨results, done, rfl⟩
      case h_13.intro =>
        simp [materializeBuilt_iff.mpr relation.1]
      case h_14 =>
        rename_i output target ignoredValue ignoredObserved field child fields extra shape
        exact childStep child (by simp [targetNonconsumingChildren]) _ _
          (memberValue_valueDepth_le fields extra field) relation
      case h_12.intro =>
        rename_i target ignoredValue ignoredObserved occurrence style alternatives inputOccurrence branch payload
          outputOccurrence outputBranch outputPayload shape left right
        obtain ⟨_, alternative, inside, branchIdentity, childRelation⟩ := right
        have names := (wellFormed.2 actual member).2.2
        rw [shape] at names
        have selected := findAlternative_of_mem names.1 inside
        rw [branchIdentity] at selected
        have childDone : observeFuel codecs targets role fuel alternative.child payload = .ok outputPayload := by
          cases style with
          | untagged =>
            apply childStep alternative.child
              (by simpa [targetNonconsumingChildren] using
                (List.mem_map_of_mem (f := TargetAlternative.child) inside)) _ _ _ childRelation
            simp only [valueDepth]; omega
          | tagged tag valueKey =>
            apply consumedStep alternative.child
              (by simpa [targetConsumingChildren] using
                (List.mem_map_of_mem (f := TargetAlternative.child) inside)) _ _ _ childRelation
            simp only [valueDepth]; omega
          | protobuf =>
            apply consumedStep alternative.child
              (by simpa [targetConsumingChildren] using
                (List.mem_map_of_mem (f := TargetAlternative.child) inside)) _ _ _ childRelation
            simp only [valueDepth]; omega
        simp [selected, childDone]

      case h_11 =>
        rename_i target ignoredValue ignoredObserved members preserve fields extra results observedExtra shape
        have membersDone : members.mapM (fun targetMember => do
            let childObserved ← observeFuel codecs targets role fuel targetMember.child
              (memberValue fields targetMember.identity)
            let visible ← observePresence targets targetMember childObserved
            pure (targetMember.identity, visible)) = .ok results := by
          apply (mapM_ok ..).mpr
          apply all₂_mono_mem relation.1
          intro targetMember inside result related
          obtain ⟨sameIdentity, childObserved, childRelation, presence⟩ := related
          have edge : targetMember.child ∈ targetConsumingChildren actual.target := by
            rw [shape]
            exact List.mem_map_of_mem inside
          obtain ⟨childDeclaration, childFound⟩ := target_consuming_exists wellFormed found edge
          have childDone : observeFuel codecs targets role fuel targetMember.child
              (memberValue fields targetMember.identity) = .ok childObserved := by
            by_cases absent : memberValue fields targetMember.identity = .absent
            · have childAbsent := observeAt_absent (absent ▸ childRelation)
              have product := Nat.mul_pos (valueDepth_positive (Value.object fields extra))
                (Nat.zero_lt_succ (maximumTargetRank targets))
              change 0 < valueDepth (.object fields extra) * (maximumTargetRank targets + 1) at product
              have positive : 0 < fuel := by unfold valueNodeBudget at enough; omega
              cases fuel with
              | zero => omega
              | succ fuel => simp [observeFuel, childFound, absent, childAbsent]
            · have present : valueAbsent (memberValue fields targetMember.identity) = false := by
                cases eq : valueAbsent (memberValue fields targetMember.identity) with
                | false => rfl
                | true => exact False.elim (absent ((valueAbsent_iff _).mp eq))
              exact consumedStep targetMember.child
                (by simpa [shape] using edge) _ _
                (memberValue_present_depth_lt present) childRelation
          have visible := (observePresence_iff wellFormed childFound).mpr presence
          exact bind_ok.mpr ⟨childObserved, childDone,
            bind_ok.mpr ⟨result.2, visible, by simp [sameIdentity]⟩⟩
        apply bind_ok.mpr
        refine ⟨results, membersDone, ?_⟩
        cases preserve with
        | false => simp_all
        | true =>
          have extrasDone : extra.mapM (fun entry => do
              let wire ← materializeBuilt codecs entry.2
              pure (entry.1, Value.jsonSnapshot wire)) = .ok observedExtra := by
            apply (mapM_ok ..).mpr
            have related : All₂ (fun entry result => entry.1 = result.1 ∧
                observeFreeAt depth codecs entry.2 result.2) extra observedExtra := by
              simpa only [ite_true] using relation.2
            apply all₂_mono related
            intro entry result related
            obtain ⟨sameName, positive, wire, sameValue, materialized, valid⟩ := related
            have done := materializeBuilt_iff.mpr materialized
            exact bind_ok.mpr ⟨wire, done, congrArg Except.ok (Prod.ext sameName sameValue.symm)⟩
          exact bind_ok.mpr ⟨observedExtra, extrasDone, rfl⟩

/-- The executable observation implements the independent, unbounded target
observation for every finite value and every well-formed target graph. -/
theorem observeValue_iff {codecs targets role identity value observed declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    observeValue codecs targets role identity value = .ok observed ↔
      Observe codecs targets role identity value observed := by
  have valid := (validateTargets_iff targets).mpr wellFormed
  simp only [observeValue, valid, ↓reduceIte]
  constructor
  · intro done
    exact ⟨_, observeFuel_sound wellFormed done⟩
  · rintro ⟨depth, relation⟩
    exact observeFuel_complete wellFormed found
      (valueNodeBudget_le_valueBudget (findTarget_mem found)) relation

end ValueContract.Candidate
