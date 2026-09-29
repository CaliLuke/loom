import ValueContract.CanonicalConstruction
import ValueContract.CanonicalProofs
import ValueContract.ValueBudgetProofs
import ValueContract.EmptyProofs

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

private theorem requireWire_ok (value : Option Wire) (wire : Wire) :
    requireWire value = .ok wire ↔ value = some wire := by
  cases value <;> simp [requireWire]

private theorem materializeBuilt_correct {codecs value wire} :
    materializeBuilt codecs value = .ok wire ↔ Materializes codecs value wire := by
  rw [← materialize_iff_Materializes]
  unfold materializeBuilt
  cases found : materialize codecs value with
  | ok result => simp
  | error failure => cases failure <;> simp

private theorem ite_ok (condition : Prop) [Decidable condition]
    (yes no : Except ε α) (result : α) :
    (if condition then yes else no) = .ok result ↔
      (condition ∧ yes = .ok result) ∨ (¬condition ∧ no = .ok result) := by
  split <;> simp_all

/-- Additional construction validates snapshots and materializes other finite
built-in values, without assuming the target builder has already succeeded. -/
theorem constructExtra_iff (codecs : ScalarCodecs) (value : Value) (wire : Wire) :
    constructExtra codecs value = .ok wire ↔
      match value with
      | .jsonSnapshot snapshot => snapshot = wire ∧ JSONValid codecs.numbers snapshot
      | _ => Materializes codecs value wire := by
  cases value <;> simp only [constructExtra]
  all_goals try exact materializeBuilt_correct
  split <;> simp_all [jsonValid_iff_JSONValid]

/-- Successful omission is possible only for the explicit absent constructor;
no exhausted or invalid child is mistaken for an omitted value. -/
theorem constructFuel_none_sound {codecs targets fuel identity value}
    (success : constructFuel codecs targets fuel identity value = .ok none) : value = .absent := by
  induction fuel generalizing identity value with
  | zero => simp [constructFuel] at success
  | succ fuel ih =>
    cases found : findTarget targets identity with
    | none => simp [constructFuel, found] at success
    | some declaration =>
      simp only [constructFuel, found] at success
      split at success <;> try rfl
      all_goals try exact ih success
      all_goals try simp [bind_ok, pure, Except.pure] at success
      all_goals try (split at success <;> simp_all [bind_ok, ite_ok])
      all_goals try (have absent := ih success; contradiction)
      all_goals try simp [ite_ok] at success
      all_goals try (split at success <;> simp_all [bind_ok])
      all_goals
        obtain ⟨_, _, _, _, success⟩ := success
        split at success <;> simp at success

private theorem all₂_zip_join {R : α → β → Prop} {S : α → γ → Prop}
    {inputs : List α} {left : List β} {right : List γ}
    (one : All₂ R inputs left) (two : All₂ S inputs right) :
    All₂ (fun input pair => R input pair.1 ∧ S input pair.2) inputs (left.zip right) := by
  induction inputs generalizing left right with
  | nil => cases left <;> cases right <;> simp_all [All₂]
  | cons head tail ih =>
    cases left with
    | nil => simp [All₂] at one
    | cons l ls =>
      cases right with
      | nil => simp [All₂] at two
      | cons r rs =>
        obtain ⟨headOne, tailOne⟩ := all₂_cons.mp one
        obtain ⟨headTwo, tailTwo⟩ := all₂_cons.mp two
        exact all₂_cons.mpr ⟨⟨headOne, headTwo⟩, ih tailOne tailTwo⟩

private theorem all₂_map_left (f : α → β) (relation : β → γ → Prop)
    (left : List α) (right : List γ) :
    All₂ relation (left.map f) right ↔ All₂ (fun a b => relation (f a) b) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons other rest => simp only [List.map_cons, all₂_cons, ih]

private theorem all₂_map_right (f : β → γ) (relation : α → γ → Prop)
    (left : List α) (right : List β) :
    All₂ relation left (right.map f) ↔ All₂ (fun a b => relation a (f b)) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons other rest => simp only [List.map_cons, all₂_cons, ih]

private theorem fragment_sound {codecs targets fuel fields}
    (ih : ∀ {identity value wire},
      constructFuel codecs targets fuel identity value = .ok (some wire) →
      canonicalAt fuel codecs targets identity value wire)
    (member : TargetMember) (fragment : String × Option Wire)
    (success : ((do
      let value := memberValue fields member.identity
      let wire ← if valueAbsent value || implicitDefaultOmitted member.presence value then .ok none
        else constructFuel codecs targets fuel member.child value
      if member.required && wire.isNone then .error .incomplete
      else pure (member.wireName, wire)) : Except BuildFailure (String × Option Wire)) = .ok fragment) :
    fragment.1 = member.wireName ∧
      match fragment.2 with
      | none => member.required = false ∧
          (memberValue fields member.identity = .absent ∨
            implicitDefaultOmitted member.presence (memberValue fields member.identity) = true)
      | some wire => memberValue fields member.identity ≠ .absent ∧
          implicitDefaultOmitted member.presence (memberValue fields member.identity) = false ∧
          canonicalAt fuel codecs targets member.child (memberValue fields member.identity) wire := by
  dsimp only at success
  split at success
  · simp only [Bind.bind, Except.bind] at success
    split at success <;> try simp_all only [reduceCtorEq]
    simp only [pure, Except.pure, Except.ok.injEq] at success
    subst fragment
    simp_all [valueAbsent]
    split at * <;> simp_all
  · rename_i present
    simp only [bind_ok] at success
    obtain ⟨optional, built, success⟩ := success
    split at success <;> try simp_all only [reduceCtorEq]
    simp only [pure, Except.pure, Except.ok.injEq] at success
    subst fragment
    refine ⟨rfl, ?_⟩
    cases optional with
    | none =>
      have absent := constructFuel_none_sound built
      simp_all [valueAbsent]
    | some wire =>
      refine ⟨?_, ?_, ih built⟩
      · intro absent
        simp [absent, valueAbsent] at present
      · simpa using (Bool.or_eq_false_iff.mp (Bool.eq_false_iff.mpr present)).2

/-- Every successful finite construction has an independent canonical
representation derivation; no graph validity assumption is needed for soundness. -/
theorem constructFuel_some_sound {codecs targets fuel identity value wire}
    (success : constructFuel codecs targets fuel identity value = .ok (some wire)) :
    canonicalAt fuel codecs targets identity value wire := by
  induction fuel generalizing identity value wire with
  | zero => simp [constructFuel] at success
  | succ fuel ih =>
    cases found : findTarget targets identity with
    | none => simp [constructFuel, found] at success
    | some declaration =>
      refine ⟨declaration, findTarget_mem found, findTarget_identity found, ?_⟩
      simp only [constructFuel, found] at success
      split at success <;> simp_all only
      all_goals try contradiction
      all_goals try exact ih success
      all_goals try (split at success <;> simp_all only)
      all_goals try exact ⟨by assumption, ih success⟩
      all_goals try simp [bind_ok, pure, Except.pure] at success
      all_goals try contradiction
      all_goals try trivial
      case h_2.isTrue =>
        subst wire
        exact ⟨trivial, canonicalScalar_exists ..⟩
      case h_3 => subst wire; trivial
      case h_4 => split <;> simp_all
      case h_8 =>
        obtain ⟨wires, children, rfl⟩ := success
        apply all₂_mono ((mapM_ok ..).mp children)
        intro value wire child
        simp only [bind_ok, requireWire_ok] at child
        obtain ⟨_, constructed, rfl⟩ := child
        exact ih constructed
      case h_9 =>
        rename_i _ _ kind rules child bounds entries _
        cases encoded : nameKeys codecs.numbers (entries.map Prod.fst) with
        | error failure => simp [encoded] at success
        | ok names =>
          simp [encoded, bind_ok] at success
          obtain ⟨wires, built, rfl⟩ := success
          have children : All₂ (fun entry wire => canonicalAt fuel codecs targets child entry.2 wire)
              entries wires := by
            apply all₂_mono ((mapM_ok ..).mp built)
            intro entry wire childResult
            simp only [bind_ok, requireWire_ok] at childResult
            obtain ⟨_, constructed, rfl⟩ := childResult
            exact ih constructed
          have keyRelation := (nameKeys_iff ..).mp encoded
          have namesLength : names.length = wires.length := by
            have valuesLength := children.1
            have keysLength := keyRelation.2.1
            simp only [List.length_map] at keysLength
            omega
          refine ⟨names.zip wires,
            all₂_zip_join ((all₂_map_left ..).mp keyRelation.2) children, ?_, rfl⟩
          rw [List.map_fst_zip (Nat.le_of_eq namesLength)]
          exact keyRelation.1
      case h_10.isTrue =>
        obtain ⟨fragments, built, additional, extras, result⟩ := success
        split at result <;> try simp only [reduceCtorEq] at result
        simp only [Except.ok.injEq, Option.some.injEq] at result
        subst wire
        refine ⟨fragments, additional, ?_, ?_, ?_, rfl⟩
        · apply all₂_mono ((mapM_ok ..).mp built)
          intro member fragment success
          apply fragment_sound ih member fragment
          simpa using success
        · simp only [ite_true]
          apply all₂_mono ((mapM_ok ..).mp extras)
          intro entry fragment success
          simp only [bind_ok, Except.ok.injEq] at success
          obtain ⟨child, built, same⟩ := success
          cases same
          exact ⟨rfl, (constructExtra_iff ..).mp built⟩
        · simpa only [List.map_append] using ‹(List.map Prod.fst (retainedMembers fragments) ++ List.map Prod.fst additional).Nodup›
      case h_10.isFalse =>
        obtain ⟨fragments, built, result⟩ := success
        split at result <;> try simp only [reduceCtorEq] at result
        simp only [Except.ok.injEq, Option.some.injEq] at result
        subst wire
        refine ⟨fragments, [], ?_, rfl, ?_, ?_⟩
        · apply all₂_mono ((mapM_ok ..).mp built)
          intro member fragment success
          apply fragment_sound ih member fragment
          simpa using success
        · simpa using ‹(List.map Prod.fst (retainedMembers fragments)).Nodup›
        · simp
      case h_11 =>
        rename_i _ _ occurrence style alternatives actual branch payload _
        split at success
        · cases selected : alternatives.find? (fun alt => alt.identity == branch) with
          | none => simp [selected] at success
          | some alternative =>
            simp only [selected, bind_ok, requireWire_ok] at success
            obtain ⟨optional, built, childWire, same, result⟩ := success
            subst optional
            refine ⟨by symm; assumption, alternative, List.mem_of_find?_eq_some selected,
              beq_iff_eq.mp (List.find?_some (p := fun alt : TargetAlternative => alt.identity == branch) selected), childWire, ih built, ?_⟩
            cases style <;> simpa using result.symm
        · simp at success
      case h_12 =>
        simp only [ite_ok, reduceCtorEq, and_false, or_false, Except.ok.injEq, Option.some.injEq] at success
        exact ⟨success.2.symm, (jsonValid_iff_JSONValid ..).mp success.1⟩

private theorem all₂_mono_mem {R S : α → β → Prop} {left : List α} {right : List β}
    (relation : All₂ R left right) (step : ∀ a ∈ left, ∀ b, R a b → S a b) : All₂ S left right :=
  ⟨relation.1, fun pair member => step _ (List.of_mem_zip member).1 _ (relation.2 pair member)⟩

/-- Independent canonical derivations execute at the graph-derived structural
budget, without bounding the depth of supported finite values. -/
theorem constructFuel_complete {codecs targets depth fuel identity value wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (enough : valueNodeBudget targets declaration value ≤ fuel)
    (relation : canonicalAt depth codecs targets identity value wire) :
    constructFuel codecs targets fuel identity value = .ok (some wire) := by
  induction depth generalizing fuel identity value wire declaration with
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
          valueDepth next ≤ valueDepth value → canonicalAt depth codecs targets child next result →
            constructFuel codecs targets fuel child next = .ok (some result) := by
        intro child edge next result height relation
        obtain ⟨childDeclaration, childFound, rank⟩ := target_nonconsuming_rank wellFormed actualFound edge
        apply ih childFound _ relation
        have smaller := valueNodeBudget_nonconsuming_lt (targets := targets) rank height
        omega
      have consumedStep : ∀ child ∈ targetConsumingChildren actual.target, ∀ next result,
          valueDepth next < valueDepth value → canonicalAt depth codecs targets child next result →
            constructFuel codecs targets fuel child next = .ok (some result) := by
        intro child edge next result height relation
        obtain ⟨childDeclaration, childFound⟩ := target_consuming_exists wellFormed actualFound edge
        apply ih childFound _ relation
        have smaller := valueNodeBudget_consuming_lt (targets := targets) (parent := actual)
          (findTarget_mem childFound) height
        omega
      simp only [constructFuel, actualFound]
      split at relation <;> try simp_all only
      all_goals try contradiction
      all_goals try rfl
      case h_1 => simpa using canonicalScalar_realized relation.2
      case h_3 =>
        rename_i input output _ _ _ child _ _
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation
        have notAbsent : input ≠ .absent := fun same => canonicalAt_absent (same ▸ relation)
        have null : input = .null → output = .null := fun same => canonicalAt_null (same ▸ relation)
        cases input <;> simp_all
      case h_4 =>
        rename_i input output _ _ _ child _
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation
        have notAbsent : input ≠ .absent := fun same => canonicalAt_absent (same ▸ relation)
        cases input <;> simp_all
      case h_5 =>
        rename_i input output _ _ _ field child _
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation
        have notAbsent : input ≠ .absent := fun same => canonicalAt_absent (same ▸ relation)
        cases input <;> simp_all
      case h_6 =>
        rename_i input output _ _ _ child _
        have done := childStep child (by simp [targetNonconsumingChildren]) input output (Nat.le_refl _) relation.2
        have notAbsent : input ≠ .absent := fun same => canonicalAt_absent (same ▸ relation.2)
        cases input <;> simp_all
      case h_7 =>
        rename_i _ _ _ child bounds values wires _
        have done : values.mapM (fun value => do
            requireWire (← constructFuel codecs targets fuel child value)) = .ok wires := by
          apply (mapM_ok ..).mpr
          apply all₂_mono_mem relation
          intro value inside wire childRelation
          have built := consumedStep child (by simp [targetConsumingChildren]) value wire
            (array_valueDepth_lt inside) childRelation
          simp [built, requireWire]
        simp [done]
      case h_8 =>
        rename_i _ _ _ kind rules child bounds entries wires _
        obtain ⟨fragments, children, uniqueNames, rfl⟩ := relation
        have named : nameKeys codecs.numbers (entries.map Prod.fst) = .ok (fragments.map Prod.fst) := by
          apply (nameKeys_iff ..).mpr
          refine ⟨uniqueNames, ?_⟩
          apply (all₂_map_left ..).mpr
          apply (all₂_map_right ..).mpr
          exact all₂_mono children (fun _ _ related => related.1)
        have built : entries.mapM (fun entry => do
            requireWire (← constructFuel codecs targets fuel child entry.2)) = .ok (fragments.map Prod.snd) := by
          apply (mapM_ok ..).mpr
          apply (all₂_map_right ..).mpr
          apply all₂_mono_mem children
          intro entry inside fragment related
          have done := consumedStep child (by simp [targetConsumingChildren]) entry.2 fragment.2
            (map_valueDepth_lt inside) related.2
          simp [done, requireWire]
        have zipped : (fragments.map Prod.fst).zip (fragments.map Prod.snd) = fragments :=
          (List.zip_of_prod rfl rfl).symm
        simp [named, built, zipped]
      case h_9 =>
        rename_i _ _ _ members preserve fields extra wires shape
        obtain ⟨fragments, additional, membersRelated, extrasRelated, uniqueNames, rfl⟩ := relation
        have built : members.mapM (fun member => do
            let value := memberValue fields member.identity
            let wire ← if valueAbsent value || implicitDefaultOmitted member.presence value then (Except.ok none : Except BuildFailure (Option Wire))
              else constructFuel codecs targets fuel member.child value
            if member.required && wire.isNone then .error .incomplete
            else pure (member.wireName, wire)) = .ok fragments := by
          apply (mapM_ok ..).mpr
          apply all₂_mono_mem membersRelated
          intro targetMember inside fragment related
          obtain ⟨name, related⟩ := related
          cases fragment with
          | mk name optional =>
            simp only at name
            subst name
            cases optional with
            | none =>
              obtain ⟨required, absent | omitted⟩ := related
              · simp [absent, valueAbsent, required]
              · simp [omitted, required]
            | some childWire =>
              obtain ⟨notAbsent, notOmitted, relation⟩ := related
              have present : valueAbsent (memberValue fields targetMember.identity) = false := by
                cases eq : valueAbsent (memberValue fields targetMember.identity) with
                | false => rfl
                | true => exact False.elim (notAbsent ((valueAbsent_iff _).mp eq))
              have done := consumedStep targetMember.child
                (by simpa [targetConsumingChildren] using
                  (List.mem_map_of_mem (f := TargetMember.child) inside)) _ _
                (memberValue_present_depth_lt present) relation
              simp [present, notOmitted, done]
        rw [built]
        simp only [Bind.bind, Except.bind]
        cases preserve with
        | false =>
          simp only [Bool.false_eq_true, ↓reduceIte] at extrasRelated
          subst additional
          simp only [List.append_nil] at uniqueNames
          simp [uniqueNames]
        | true =>
          simp only [ite_true] at extrasRelated
          have extrasDone : extra.mapM (fun entry => do
              let wire ← constructExtra codecs entry.2
              pure (entry.1, wire)) = .ok additional := by
            apply (mapM_ok ..).mpr
            apply all₂_mono extrasRelated
            intro entry result related
            have done := (constructExtra_iff ..).mpr related.2
            simp [done, related.1]
          simp only [Bind.bind, Except.bind, Pure.pure, Except.pure] at extrasDone
          simp only [List.map_append] at uniqueNames
          simp [extrasDone, uniqueNames]
      case h_10 =>
        rename_i output _ _ _ occurrence style alternatives actualOccurrence branch payload shape
        obtain ⟨_, alternative, inside, branchIdentity, childWire, childRelation, envelope⟩ := relation
        have names := (wellFormed.2 actual member).2.2
        rw [shape] at names
        have selected := findAlternative_of_mem names.1 inside
        rw [branchIdentity] at selected
        have childDone : constructFuel codecs targets fuel alternative.child payload = .ok (some childWire) := by
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
        cases style <;> simp_all [requireWire]
      case h_11 => simp [(jsonValid_iff_JSONValid ..).mpr relation.2]

/-- Construction succeeds with a wire exactly when the independently specified
canonical representation exists, using the public graph/value-derived budget. -/
theorem construct_some_iff {codecs targets identity value wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    construct codecs targets identity value = .ok (some wire) ↔
      Canonical codecs targets identity value wire := by
  have valid := (validateTargets_iff targets).mpr wellFormed
  simp only [construct, valid, ↓reduceIte]
  constructor
  · intro done
    exact ⟨_, constructFuel_some_sound done⟩
  · rintro ⟨depth, relation⟩
    exact constructFuel_complete wellFormed found
      (valueNodeBudget_le_valueBudget (findTarget_mem found)) relation

/-- A well-formed, present root returns omission exactly for explicit absence.
Invalid plans and exhausted recursion never count as successful omission. -/
theorem construct_none_iff {codecs targets identity value declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    construct codecs targets identity value = .ok none ↔ value = .absent := by
  have valid := (validateTargets_iff targets).mpr wellFormed
  simp only [construct, valid, ↓reduceIte]
  constructor
  · exact constructFuel_none_sound
  · intro absent
    subst value
    have positive : 0 < valueBudget targets .absent := by
      unfold valueBudget
      exact Nat.mul_pos (Nat.zero_lt_succ _) (Nat.zero_lt_succ _)
    cases fuel : valueBudget targets .absent with
    | zero => omega
    | succ fuel => simp [constructFuel, found]

end ValueContract.Candidate
