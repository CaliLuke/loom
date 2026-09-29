import ValueContract.SchemaValidation
import ValueContract.MaterializationProofs

namespace ValueContract.Candidate

private theorem schemaFuel_successor (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (fuel : Nat) (identity : Identity) (wire : Wire) :
    schemaFuel codecs checks targets (fuel + 1) identity wire = (do
      let some declaration := findTarget targets identity | .error .malformedDeclaration
      let accepted ← schemaBody codecs checks (schemaFuel codecs checks targets fuel) declaration wire
      return accepted && schemaEnumAllowed codecs.numbers declaration.schemaEnumeration wire) := by
  simp only [schemaFuel]
  rfl

private theorem bind_ok {input : Except ε α} {next : α → Except ε β} {result : β} :
    (input >>= next) = .ok result ↔ ∃ value, input = .ok value ∧ next value = .ok result := by
  cases input <;> simp [Bind.bind, Except.bind]

private theorem all₂_cons {R : α → β → Prop} {a : α} {b : β}
    {left : List α} {right : List β} :
    All₂ R (a :: left) (b :: right) ↔ R a b ∧ All₂ R left right := by
  simp [All₂, and_left_comm]

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

private theorem schemaBody_sound {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {recur : Identity → Wire → Except Failure Bool}
    (sound : ∀ identity wire accepted, recur identity wire = .ok accepted →
      SchemaEvaluation codecs checks targets identity wire accepted)
    {declaration : TargetDeclaration} {wire : Wire} {accepted : Bool}
    (success : schemaBody codecs checks recur declaration wire = .ok accepted) :
    SchemaNode codecs checks targets declaration.schemaAllowsUnknown declaration.target wire accepted := by
  cases shape : declaration.target with
  | scalar encoding kind rules =>
    simp only [schemaBody, shape, Except.ok.injEq] at success
    subst accepted
    exact SchemaNode.scalar
  | nullable child =>
    cases wire <;> simp only [schemaBody, shape] at success ⊢
    case null => cases success; exact SchemaNode.nullableNull
    all_goals exact SchemaNode.nullableValue rfl (sound _ _ _ success)
  | nonNull child =>
    cases wire <;> simp only [schemaBody, shape, wireIsNull, Bool.false_eq_true, ite_false, ite_true] at success ⊢
    case null => cases success; exact SchemaNode.nonNullRejected
    all_goals exact SchemaNode.nonNullValue rfl (sound _ _ _ success)
  | alias child =>
    simp only [schemaBody, shape] at success ⊢
    exact SchemaNode.alias (sound _ _ _ success)
  | select field child =>
    simp only [schemaBody, shape] at success ⊢
    exact SchemaNode.select (sound _ _ _ success)
  | any =>
    simp only [schemaBody, shape, Except.ok.injEq] at success
    subst accepted
    cases valid : jsonValid codecs.numbers wire with
    | true => exact SchemaNode.anyValid ((jsonValid_iff_JSONValid ..).mp valid)
    | false =>
      apply SchemaNode.anyInvalid
      intro relation
      have yes := (jsonValid_iff_JSONValid ..).mpr relation
      simp [valid] at yes
  | custom codec => simp [schemaBody, shape] at success
  | array child bounds =>
    cases wire <;> simp only [schemaBody, shape] at success
    all_goals try (cases success; exact SchemaNode.mismatch rfl)
    case array items =>
      obtain ⟨outcomes, done, same⟩ := bind_ok.mp success
      cases same
      have related := (mapM_ok ..).mp done
      exact SchemaNode.array related.1 (fun pair member => sound _ _ _ (related.2 pair member))
  | map key rules child bounds =>
    cases wire <;> simp only [schemaBody, shape] at success
    all_goals try (cases success; exact SchemaNode.mismatch rfl)
    case object items =>
      obtain ⟨outcomes, done, same⟩ := bind_ok.mp success
      cases same
      have related := (mapM_ok ..).mp done
      exact SchemaNode.map related.1 (fun pair member => sound _ _ _ (related.2 pair member))
  | object members additional =>
    cases wire <;> simp only [schemaBody, shape] at success
    all_goals try (cases success; exact SchemaNode.mismatch rfl)
    case object items =>
      obtain ⟨outcomes, done, same⟩ := bind_ok.mp success
      cases same
      have related := (mapM_ok ..).mp done
      apply SchemaNode.object related.1
      · intro pair member missing
        have one := related.2 pair member
        simp only [missing, Except.ok.injEq] at one
        exact one.symm
      · intro pair member value present
        have one := related.2 pair member
        simp only [present] at one
        exact sound _ _ _ one
  | union occurrence style alternatives =>
    cases style with
    | untagged =>
      simp only [schemaBody, shape] at success
      obtain ⟨outcomes, done, same⟩ := bind_ok.mp success
      cases same
      have related := (mapM_ok ..).mp done
      exact SchemaNode.untagged related.1 (fun pair member => sound _ _ _ (related.2 pair member))
    | protobuf =>
      cases wire <;> simp only [schemaBody, shape] at success
      all_goals try (cases success; exact SchemaNode.mismatch rfl)
      case oneof name value =>
        cases chosen : alternatives.find? (fun alternative => alternative.wireName == name) with
        | none =>
          simp only [chosen] at success
          cases success
          exact SchemaNode.protobufMalformed chosen
        | some alternative =>
          simp only [chosen] at success
          exact SchemaNode.protobuf chosen (sound _ _ _ success)
    | tagged tagKey valueKey =>
      cases wire <;> simp only [schemaBody, shape] at success
      all_goals try (cases success; exact SchemaNode.mismatch rfl)
      case object items =>
        cases tag : wireMember items tagKey with
        | none =>
          simp only [tag] at success
          cases success
          exact SchemaNode.taggedMalformed (by simp [tag])
        | some tagValue =>
          cases tagValue <;> simp only [tag] at success
          all_goals try (cases success; exact SchemaNode.taggedMalformed (by simp [tag]))
          case text name =>
            cases payload : wireMember items valueKey with
            | none =>
              simp only [payload] at success
              cases success
              exact SchemaNode.taggedMalformed (by simp [payload])
            | some value =>
              simp only [payload] at success
              cases chosen : alternatives.find? (fun alternative => alternative.wireName == name) with
              | none =>
                simp only [chosen] at success
                cases success
                exact SchemaNode.taggedMalformed (by simp [tag, chosen])
              | some alternative =>
                simp only [chosen] at success
                obtain ⟨result, done, same⟩ := bind_ok.mp success
                cases same
                exact SchemaNode.tagged tag payload chosen (sound _ _ _ done)

/-- Every successful finite-budget evaluation has an independent derivation.
Exhaustion and unsupported codecs are errors, never negative branch evidence. -/
theorem schemaFuel_sound {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {fuel : Nat} {identity : Identity} {wire : Wire} {accepted : Bool}
    (success : schemaFuel codecs checks targets fuel identity wire = .ok accepted) :
    SchemaEvaluation codecs checks targets identity wire accepted := by
  induction fuel generalizing identity wire accepted with
  | zero => simp [schemaFuel] at success
  | succ fuel ih =>
    rw [schemaFuel_successor] at success
    cases found : findTarget targets identity with
    | none => simp [found] at success
    | some declaration =>
      simp only [found] at success
      obtain ⟨outcome, body, same⟩ := bind_ok.mp success
      cases same
      exact SchemaEvaluation.node declaration (findTarget_mem found) (findTarget_identity found)
        (schemaBody_sound (fun identity wire accepted done => ih done) body)

private theorem foldMax_initial (f : α → Nat) (items : List α) (initial : Nat) :
    initial ≤ items.foldl (fun n item => max n (f item)) initial := by
  induction items generalizing initial with
  | nil => exact Nat.le_refl _
  | cons head tail ih =>
    exact Nat.le_trans (Nat.le_max_left _ _) (ih (max initial (f head)))

private theorem foldMax_member (f : α → Nat) {items : List α} {item : α}
    (member : item ∈ items) (initial : Nat) :
    f item ≤ items.foldl (fun n entry => max n (f entry)) initial := by
  induction items generalizing initial with
  | nil => simp at member
  | cons head tail ih =>
    rcases List.mem_cons.mp member with same | inside
    · subst item
      exact Nat.le_trans (Nat.le_max_right initial (f head)) (foldMax_initial f tail _)
    · exact ih inside _

/-- Every table member lies below the executable maximum rank. -/
theorem targetRank_le_maximum {targets : Targets} {declaration : TargetDeclaration}
    (member : declaration ∈ targets) : declaration.expansionRank ≤ maximumTargetRank targets :=
  foldMax_member TargetDeclaration.expansionRank member 0

/-- An array element strictly decreases the executable finite wire height. -/
theorem array_wireHeight_lt {items : List Wire} {child : Wire} (member : child ∈ items) :
    wireHeight child < wireHeight (.array items) := by
  have bound := foldMax_member (fun entry : { item // item ∈ items } => wireHeight entry.val)
    (item := ⟨child, member⟩) (List.mem_attach _ _) 0
  dsimp at bound
  simp only [wireHeight]
  omega

/-- An object entry strictly decreases the executable finite wire height. -/
theorem object_wireHeight_lt {items : List (String × Wire)} {entry : String × Wire}
    (member : entry ∈ items) : wireHeight entry.2 < wireHeight (.object items) := by
  have bound := foldMax_member (fun entry : { item // item ∈ items } => wireHeight entry.val.2)
    (item := ⟨entry, member⟩) (List.mem_attach _ _) 0
  dsimp at bound
  simp only [wireHeight]
  omega

/-- A successfully selected object member is a consuming wire edge. -/
theorem wireMember_wireHeight_lt {items : List (String × Wire)} {key : String} {child : Wire}
    (found : wireMember items key = some child) : wireHeight child < wireHeight (.object items) := by
  unfold wireMember at found
  cases selected : items.find? (fun entry => entry.1 == key) with
  | none => simp [selected] at found
  | some entry =>
    simp only [selected, Option.map_some, Option.some.injEq] at found
    rw [← found]
    exact object_wireHeight_lt (List.mem_of_find?_eq_some selected)

/-- The per-node depth bound spends rank only on same-wire expansion; consuming
edges reset rank and strictly decrease the finite wire height. -/
def targetNodeBudget (targets : Targets) (declaration : TargetDeclaration) (wire : Wire) : Nat :=
  wireHeight wire * (maximumTargetRank targets + 1) + declaration.expansionRank + 1

/-- A consuming wire edge decreases the shared schema/decoder budget even when rank resets. -/
theorem targetNodeBudget_consuming_lt {targets : Targets} {declaration child : TargetDeclaration}
    {wire childWire : Wire} (member : child ∈ targets)
    (smaller : wireHeight childWire < wireHeight wire) :
    targetNodeBudget targets child childWire < targetNodeBudget targets declaration wire := by
  have rank := targetRank_le_maximum member
  have heights := Nat.mul_le_mul_right (maximumTargetRank targets + 1) smaller
  change (wireHeight childWire + 1) * (maximumTargetRank targets + 1) ≤ _ at heights
  simp only [Nat.add_mul, Nat.one_mul] at heights
  unfold targetNodeBudget
  omega

/-- A same-wire edge decreases the shared budget when its validated expansion rank decreases. -/
theorem targetNodeBudget_nonconsuming_lt {targets : Targets} {declaration child : TargetDeclaration}
    {wire : Wire} (smaller : child.expansionRank < declaration.expansionRank) :
    targetNodeBudget targets child wire < targetNodeBudget targets declaration wire := by
  unfold targetNodeBudget
  omega

/-- The public wire budget covers every declaration in its finite target table. -/
theorem targetNodeBudget_le_wireBudget {targets : Targets} {declaration : TargetDeclaration}
    {wire : Wire} (member : declaration ∈ targets) :
    targetNodeBudget targets declaration wire ≤ wireBudget targets wire := by
  have bound := targetRank_le_maximum member
  unfold targetNodeBudget wireBudget
  simp only [Nat.add_mul, Nat.one_mul]
  omega

private theorem schemaBody_complete {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {recur : Identity → Wire → Except Failure Bool}
    {allows : Bool} {target : Target} {wire : Wire} {accepted : Bool}
    (body : SchemaNode codecs checks targets allows target wire accepted)
    (declaration : TargetDeclaration) (shape : declaration.target = target)
    (unknown : declaration.schemaAllowsUnknown = allows)
    (sameWire : ∀ child ∈ targetNonconsumingChildren target, ∀ outcome,
      SchemaEvaluation codecs checks targets child wire outcome → recur child wire = .ok outcome)
    (childWire : ∀ child ∈ targetConsumingChildren target, ∀ value outcome,
      wireHeight value < wireHeight wire → SchemaEvaluation codecs checks targets child value outcome →
        recur child value = .ok outcome) :
    schemaBody codecs checks recur declaration wire = .ok accepted := by
  cases body
  case scalar => simp [schemaBody, shape]
  case nullableNull => simp [schemaBody, shape]
  case nonNullRejected => simp [schemaBody, shape, wireIsNull]
  case alias child result =>
    simpa only [schemaBody, shape] using
      sameWire child (by simp [targetNonconsumingChildren]) accepted result
  case select child field result =>
    simpa only [schemaBody, shape] using
      sameWire child (by simp [targetNonconsumingChildren]) accepted result
  case nullableValue child nonnull result =>
    have done := sameWire child (by simp [targetNonconsumingChildren]) accepted result
    cases wire <;> simp_all [schemaBody, wireIsNull]
  case nonNullValue child nonnull result =>
    simpa only [schemaBody, shape, nonnull, Bool.false_eq_true, ite_false] using
      sameWire child (by simp [targetNonconsumingChildren]) accepted result
  case array child bounds items outcomes lengths results =>
    have done : items.mapM (recur child) = .ok outcomes := by
      apply (mapM_ok ..).mpr
      exact ⟨lengths, fun pair member => childWire child (by simp [targetConsumingChildren]) _ _
        (array_wireHeight_lt (List.of_mem_zip member).1) (results pair member)⟩
    simp [schemaBody, shape, done]
  case map child key rules bounds items outcomes lengths results =>
    have done : items.mapM (fun entry => recur child entry.2) = .ok outcomes := by
      apply (mapM_ok ..).mpr
      exact ⟨lengths, fun pair member => childWire child (by simp [targetConsumingChildren]) _ _
        (object_wireHeight_lt (List.of_mem_zip member).1) (results pair member)⟩
    simp [schemaBody, shape, done]
  case object items members additional outcomes lengths missing present =>
    simp only [schemaBody, shape, unknown]
    apply bind_ok.mpr
    refine ⟨outcomes, ?_, rfl⟩
    apply (mapM_ok ..).mpr
    refine ⟨lengths, ?_⟩
    intro pair member
    cases found : wireMember items pair.1.wireName with
    | none => simp [missing pair member found]
    | some value =>
      exact childWire pair.1.child
        (by simpa [targetConsumingChildren] using (List.mem_map_of_mem (f := TargetMember.child)
          (List.of_mem_zip member).1)) value pair.2
        (wireMember_wireHeight_lt found) (present pair member value found)
  case untagged occurrence alternatives outcomes lengths results =>
    have done : alternatives.mapM (fun alternative => recur alternative.child wire) = .ok outcomes := by
      apply (mapM_ok ..).mpr
      refine ⟨lengths, ?_⟩
      intro pair member
      exact sameWire pair.1.child
        (by simpa [targetNonconsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
          (List.of_mem_zip member).1)) pair.2 (results pair member)
    simp [schemaBody, shape, done]
  case tagged items tagKey name valueKey value alternative outcome occurrence alternatives tag payload chosen result =>
    have done := childWire alternative.child
      (by simpa [targetConsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
        (List.mem_of_find?_eq_some chosen))) value outcome (wireMember_wireHeight_lt payload) result
    simp [schemaBody, shape, unknown, tag, payload, chosen, done]
  case taggedMalformed items tagKey valueKey occurrence alternatives malformed =>
    simp only [schemaBody, shape]
    cases tag : wireMember items tagKey with
    | none => rfl
    | some tagValue =>
      cases tagValue <;> try rfl
      case text name =>
        cases payload : wireMember items valueKey with
        | none => rfl
        | some value =>
          cases chosen : alternatives.find? (fun alternative => alternative.wireName == name) with
          | none => simp [chosen]
          | some alternative => exact False.elim (malformed ⟨name, value, alternative, tag, payload, chosen⟩)
  case protobuf alternative value occurrence alternatives name chosen result =>
    have done := childWire alternative.child
      (by simpa [targetConsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
        (List.mem_of_find?_eq_some chosen))) value accepted (by simp [wireHeight]) result
    simp [schemaBody, shape, chosen, done]
  case protobufMalformed occurrence alternatives name value missing => simp [schemaBody, shape, missing]
  case anyValid valid => simp [schemaBody, shape, (jsonValid_iff_JSONValid ..).mpr valid]
  case anyInvalid invalid =>
    have rejected : jsonValid codecs.numbers wire = false := by
      cases found : jsonValid codecs.numbers wire with
      | false => rfl
      | true => exact False.elim (invalid ((jsonValid_iff_JSONValid ..).mp found))
    simp [schemaBody, shape, rejected]
  case mismatch mismatch =>
    cases target <;> cases wire <;> simp_all [schemaBody, structuralMismatch]
    all_goals cases ‹UnionStyle› <;> simp_all

private theorem evaluation_lookup {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {accepted : Bool}
    (wellFormed : WellFormedTargets targets)
    (evaluated : SchemaEvaluation codecs checks targets identity wire accepted) :
    ∃ declaration, findTarget targets identity = some declaration := by
  cases evaluated with
  | node declaration member same body =>
    exact ⟨declaration, same ▸ findTarget_of_mem wellFormed member⟩

private theorem schemaFuel_complete_node {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} (wellFormed : WellFormedTargets targets)
    {fuel : Nat} {identity : Identity} {wire : Wire} {accepted : Bool}
    {declaration : TargetDeclaration}
    (found : findTarget targets identity = some declaration)
    (enough : targetNodeBudget targets declaration wire ≤ fuel)
    (evaluated : SchemaEvaluation codecs checks targets identity wire accepted) :
    schemaFuel codecs checks targets fuel identity wire = .ok accepted := by
  induction fuel generalizing identity wire accepted declaration with
  | zero => unfold targetNodeBudget at enough; omega
  | succ fuel ih =>
    cases evaluated with
    | @node identity wire outcome actual member same body =>
      have actualFound : findTarget targets identity = some actual :=
        same ▸ findTarget_of_mem wellFormed member
      have equal : declaration = actual := Option.some.inj (found.symm.trans actualFound)
      subst declaration
      rw [schemaFuel_successor]
      simp only [actualFound]
      have done : schemaBody codecs checks (schemaFuel codecs checks targets fuel) actual wire =
          .ok outcome := by
        apply schemaBody_complete body actual rfl rfl
        · intro child edge outcome relation
          obtain ⟨next, nextFound, smaller⟩ := target_nonconsuming_rank wellFormed actualFound edge
          apply ih nextFound _ relation
          have decrease := targetNodeBudget_nonconsuming_lt (targets := targets) (wire := wire) smaller
          omega
        · intro child edge value outcome smaller relation
          obtain ⟨next, nextFound⟩ := target_consuming_exists wellFormed actualFound edge
          apply ih nextFound _ relation
          have decrease := targetNodeBudget_consuming_lt (declaration := actual)
            (findTarget_mem nextFound) smaller
          omega
      simp [done]

/-- Every independent positive or negative derivation is realized once the
budget reaches the finite graph/wire bound. No fixed input-depth cap is assumed. -/
theorem schemaFuel_complete {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {accepted : Bool} {fuel : Nat}
    (wellFormed : WellFormedTargets targets)
    (evaluated : SchemaEvaluation codecs checks targets identity wire accepted)
    (enough : wireBudget targets wire ≤ fuel) :
    schemaFuel codecs checks targets fuel identity wire = .ok accepted := by
  obtain ⟨declaration, found⟩ := evaluation_lookup wellFormed evaluated
  exact schemaFuel_complete_node wellFormed found
    (Nat.le_trans (targetNodeBudget_le_wireBudget (findTarget_mem found)) enough) evaluated

/-- On a well-formed target graph, the public evaluator's derived budget is
adequate for every independent derivation, including rejection outcomes.
Unsupported custom codecs and missing roots have no such derivation. -/
theorem schema_iff_SchemaEvaluation {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {accepted : Bool}
    (wellFormed : WellFormedTargets targets) :
    schema codecs checks targets identity wire = .ok accepted ↔
      SchemaEvaluation codecs checks targets identity wire accepted := by
  simp only [schema, (validateTargets_iff targets).mpr wellFormed, ite_true]
  exact ⟨schemaFuel_sound, fun evaluated => schemaFuel_complete wellFormed evaluated (Nat.le_refl _)⟩

/-- At any sufficient budget, success is exactly the independent schema
outcome. This equivalence never treats evaluation exhaustion as rejection. -/
theorem schemaFuel_iff_SchemaEvaluation {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {accepted : Bool} {fuel : Nat}
    (wellFormed : WellFormedTargets targets) (enough : wireBudget targets wire ≤ fuel) :
    schemaFuel codecs checks targets fuel identity wire = .ok accepted ↔
      SchemaEvaluation codecs checks targets identity wire accepted :=
  ⟨schemaFuel_sound, fun evaluated => schemaFuel_complete wellFormed evaluated enough⟩

/-- Zero budget remains a plan/evaluation error, including inside untagged
branch enumeration; it cannot provide a negative schema outcome. -/
theorem schemaFuel_zero_is_error (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (identity : Identity) (wire : Wire) :
    schemaFuel codecs checks targets 0 identity wire = .error .malformedDeclaration := by
  simp [schemaFuel]

end ValueContract.Candidate
