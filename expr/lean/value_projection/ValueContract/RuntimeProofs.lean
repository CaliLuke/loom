import ValueContract.Decoding
import ValueContract.MaterializationProofs
import ValueContract.SchemaProofs

namespace ValueContract.Candidate

private theorem all₂_cons_iff {R : α → β → Prop} {a : α} {b : β}
    {left : List α} {right : List β} :
    All₂ R (a :: left) (b :: right) ↔ R a b ∧ All₂ R left right := by
  simp [All₂, and_left_comm]

private theorem all₂_mono {R S : α → β → Prop} {left : List α} {right : List β}
    (implies : ∀ a b, R a b → S a b) (related : All₂ R left right) : All₂ S left right :=
  ⟨related.1, fun pair member => implies pair.1 pair.2 (related.2 pair member)⟩

private theorem except_bind_ok {input : Except ε α} {next : α → Except ε β} {result : β} :
    (input >>= next) = .ok result ↔ ∃ value, input = .ok value ∧ next value = .ok result := by
  cases input <;> simp [Bind.bind, Except.bind]

private theorem mapM_ok_iff (f : α → Except ε β) (left : List α) (right : List β) :
    left.mapM f = .ok right ↔ All₂ (fun a b => f a = .ok b) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂, pure, Except.pure]
  | cons head tail ih =>
    simp only [List.mapM_cons, except_bind_ok]
    cases right with
    | nil => simp [All₂, pure, Except.pure]
    | cons result rest =>
      simp only [all₂_cons_iff]
      constructor
      · rintro ⟨value, first, values, remaining, same⟩
        cases same
        exact ⟨first, (ih _).mp remaining⟩
      · rintro ⟨first, remaining⟩
        exact ⟨result, first, rest, (ih _).mpr remaining, rfl⟩

private abbrev decodeStep := decodeBody

private theorem decodeFuel_successor (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (fuel : Nat) (identity : Identity) (wire : Wire) :
    decodeFuel codecs checks targets (fuel + 1) identity wire = (do
      let some declaration := findTarget targets identity | .error .malformedDeclaration
      let result ← decodeStep codecs checks (decodeFuel codecs checks targets fuel) declaration wire
      return result.filter (targetEnumAllowed codecs.numbers declaration.enumeration)) := by
  simp only [decodeFuel, decodeStep]
  rfl

private theorem runtimeEnumLift {codecs checks targets identity wire result}
    (declaration : TargetDeclaration) (member : declaration ∈ targets)
    (same : declaration.identity = identity)
    (node : RuntimeNode codecs checks targets declaration.decoderRejectsUnknown declaration.target wire result) :
    RuntimeEvaluation codecs checks targets identity wire
      (result.filter (targetEnumAllowed codecs.numbers declaration.enumeration)) := by
  cases result with
  | none => exact .rejected declaration member same node
  | some value =>
    by_cases allowed : targetEnumAllowed codecs.numbers declaration.enumeration value = true
    · simpa [Option.filter, allowed] using RuntimeEvaluation.accepted declaration member same node
        ((targetEnumAllowed_iff _ _ _).mp allowed)
    · have rejected : ¬ TargetEnumAllows codecs.numbers declaration.enumeration value := by
        simpa [← targetEnumAllowed_iff] using allowed
      have no : targetEnumAllowed codecs.numbers declaration.enumeration value = false :=
        Bool.eq_false_iff.mpr allowed
      simpa [Option.filter, no] using RuntimeEvaluation.enumRejected declaration member same node rejected

private theorem keyOutcome_actual (codec : NumericCodec) (kind : MapKeyKind) (name : String) :
    KeyOutcome codec kind name (decodeKey codec kind name) := by
  cases parsed : decodeKey codec kind name with
  | none =>
    intro key relation
    have result := keyDecoder_progress relation
    simp [parsed] at result
  | some key => exact keyDecoder_sound parsed

private theorem all₂_map_right (relation : α → β → Prop) (f : α → β)
    (valid : ∀ item, relation item (f item)) (items : List α) :
    All₂ relation items (items.map f) := by
  induction items with
  | nil => simp [All₂]
  | cons head tail ih =>
    exact all₂_cons_iff.mpr ⟨valid head, ih⟩

private theorem decodeStep_sound {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {recur : Identity → Wire → Except Failure (Option Value)}
    (sound : ∀ identity wire result, recur identity wire = .ok result →
      RuntimeEvaluation codecs checks targets identity wire result)
    {declaration : TargetDeclaration} {wire : Wire} {result : Option Value}
    (success : decodeStep codecs checks recur declaration wire = .ok result) :
    RuntimeNode codecs checks targets declaration.decoderRejectsUnknown declaration.target wire result := by
  cases shape : declaration.target with
  | scalar encoding kind rules =>
    cases parsed : decodeScalar codecs encoding kind wire with
    | none =>
      simp [decodeStep, decodeBody, shape, parsed, Option.filter] at success
      subst result
      apply RuntimeNode.scalarRejected
      intro value relation
      have accepted := scalarDecoder_progress relation
      simp [parsed] at accepted
    | some value =>
      by_cases allowed : scalarAllowed checks rules value = true
      · simp [decodeStep, decodeBody, shape, parsed, Option.filter, allowed] at success
        subst result
        exact RuntimeNode.scalar (scalarDecoder_sound parsed) allowed
      · have denied := Bool.eq_false_iff.mpr allowed
        simp [decodeStep, decodeBody, shape, parsed, Option.filter, denied] at success
        subst result
        apply RuntimeNode.scalarRejected
        intro other relation
        have accepted := scalarDecoder_progress relation
        have same : value = other := by simpa [parsed] using accepted
        subst other
        exact denied
  | nullable child =>
    cases wire <;> simp only [decodeStep, decodeBody, shape] at success ⊢
    case null => cases success; exact RuntimeNode.nullableNull
    all_goals exact RuntimeNode.nullableValue rfl (sound _ _ _ success)
  | nonNull child =>
    cases wire <;> simp only [decodeStep, decodeBody, shape, wireIsNull,
      Bool.false_eq_true, ite_false, ite_true] at success ⊢
    case null => cases success; exact RuntimeNode.nonNullRejected
    all_goals exact RuntimeNode.nonNullValue rfl (sound _ _ _ success)
  | alias child =>
    simp only [decodeStep, decodeBody, shape] at success ⊢
    exact RuntimeNode.alias (sound _ _ _ success)
  | select field child =>
    simp only [decodeStep, decodeBody, shape] at success ⊢
    exact RuntimeNode.select (sound _ _ _ success)
  | any =>
    cases valid : jsonValid codecs.numbers wire with
    | true =>
      simp [decodeStep, decodeBody, shape, valid] at success
      subst result
      exact RuntimeNode.anyValid ((jsonValid_iff_JSONValid ..).mp valid)
    | false =>
      simp [decodeStep, decodeBody, shape, valid] at success
      subst result
      apply RuntimeNode.anyInvalid
      intro relation
      have yes := (jsonValid_iff_JSONValid ..).mpr relation
      simp [valid] at yes
  | custom codec => simp [decodeStep, decodeBody, shape] at success
  | array child bounds =>
    cases wire <;> simp only [decodeStep, decodeBody, shape] at success
    all_goals try (cases success; exact RuntimeNode.mismatch rfl)
    case array items =>
      obtain ⟨outcomes, done, same⟩ := except_bind_ok.mp success
      cases same
      have related := (mapM_ok_iff ..).mp done
      exact RuntimeNode.array related.1 (fun pair member => sound _ _ _ (related.2 pair member))
  | map key rules child bounds =>
    cases wire <;> simp only [decodeStep, decodeBody, shape] at success
    all_goals try (cases success; exact RuntimeNode.mismatch rfl)
    case object items =>
      obtain ⟨outcomes, done, same⟩ := except_bind_ok.mp success
      cases same
      have related := (mapM_ok_iff ..).mp done
      have keys := all₂_map_right (fun entry decoded => KeyOutcome codecs.numbers key entry.1 decoded)
        (fun entry => decodeKey codecs.numbers key entry.1)
        (fun entry => keyOutcome_actual codecs.numbers key entry.1) items
      exact RuntimeNode.map keys.1 related.1 keys.2
        (fun pair member => sound _ _ _ (related.2 pair member))
  | object members additional =>
    cases wire <;> simp only [decodeStep, decodeBody, shape] at success
    all_goals try (cases success; exact RuntimeNode.mismatch rfl)
    case object items =>
      obtain ⟨outcomes, done, same⟩ := except_bind_ok.mp success
      cases same
      have related := (mapM_ok_iff ..).mp done
      apply RuntimeNode.object related.1
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
      simp only [decodeStep, decodeBody, shape] at success
      obtain ⟨outcomes, done, same⟩ := except_bind_ok.mp success
      cases same
      have related := (mapM_ok_iff ..).mp done
      exact RuntimeNode.untagged related.1 (fun pair member => sound _ _ _ (related.2 pair member))
    | protobuf =>
      cases wire <;> simp only [decodeStep, decodeBody, shape] at success
      all_goals try (cases success; exact RuntimeNode.mismatch rfl)
      case oneof name value =>
        cases chosen : alternatives.find? (fun alternative => alternative.wireName == name) with
        | none =>
          simp only [chosen] at success
          cases success
          exact RuntimeNode.protobufMalformed chosen
        | some alternative =>
          simp only [chosen] at success
          obtain ⟨outcome, done, same⟩ := except_bind_ok.mp success
          cases same
          exact RuntimeNode.protobuf chosen (sound _ _ _ done)
    | tagged tagKey valueKey =>
      cases wire <;> simp only [decodeStep, decodeBody, shape] at success
      all_goals try (cases success; exact RuntimeNode.mismatch rfl)
      case object items =>
        cases tag : wireMember items tagKey with
        | none =>
          simp only [tag] at success
          cases success
          exact RuntimeNode.taggedMalformed (by simp [tag])
        | some tagValue =>
          cases tagValue <;> simp only [tag] at success
          all_goals try (cases success; exact RuntimeNode.taggedMalformed (by simp [tag]))
          case text name =>
            cases payload : wireMember items valueKey with
            | none =>
              simp only [payload] at success
              cases success
              exact RuntimeNode.taggedMalformed (by simp [payload])
            | some value =>
              simp only [payload] at success
              cases chosen : alternatives.find? (fun alternative => alternative.wireName == name) with
              | none =>
                simp only [chosen] at success
                cases success
                exact RuntimeNode.taggedMalformed (by simp [tag, chosen])
              | some alternative =>
                simp only [chosen] at success
                obtain ⟨outcome, done, same⟩ := except_bind_ok.mp success
                cases same
                exact RuntimeNode.tagged tag payload chosen (sound _ _ _ done)

/-- Successful runtime evaluation, including a negative branch outcome, always
has an independent decoder derivation. No graph or depth restriction is needed
for this soundness direction; failed traversal never supplies negative evidence. -/
theorem decodeFuel_sound {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {fuel : Nat} {identity : Identity} {wire : Wire} {result : Option Value}
    (success : decodeFuel codecs checks targets fuel identity wire = .ok result) :
    RuntimeEvaluation codecs checks targets identity wire result := by
  induction fuel generalizing identity wire result with
  | zero => simp [decodeFuel] at success
  | succ fuel ih =>
    rw [decodeFuel_successor] at success
    cases found : findTarget targets identity with
    | none => simp [found] at success
    | some declaration =>
      simp only [found] at success
      obtain ⟨outcome, body, same⟩ := except_bind_ok.mp success
      cases same
      exact runtimeEnumLift declaration (findTarget_mem found) (findTarget_identity found)
        (decodeStep_sound (fun identity wire result done => ih done) body)

private theorem keyOutcome_complete {codec kind name result}
    (related : KeyOutcome codec kind name result) : decodeKey codec kind name = result := by
  cases result with
  | some key => exact keyDecoder_progress related
  | none =>
    cases parsed : decodeKey codec kind name with
    | none => rfl
    | some key => exact False.elim (related key (keyDecoder_sound parsed))

private theorem all₂_map_eq {left : List α} {right : List β} (f : α → β)
    (related : All₂ (fun a b => f a = b) left right) : left.map f = right := by
  induction left generalizing right with
  | nil => cases right <;> simp_all [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂] at related
    | cons result rest =>
      obtain ⟨first, remaining⟩ := all₂_cons_iff.mp related
      simp [first, ih remaining]

private theorem decodeBody_complete {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {recur : Identity → Wire → Except Failure (Option Value)}
    {reject : Bool} {target : Target} {wire : Wire} {result : Option Value}
    (body : RuntimeNode codecs checks targets reject target wire result)
    (declaration : TargetDeclaration) (shape : declaration.target = target)
    (unknown : declaration.decoderRejectsUnknown = reject)
    (sameWire : ∀ child ∈ targetNonconsumingChildren target, ∀ outcome,
      RuntimeEvaluation codecs checks targets child wire outcome → recur child wire = .ok outcome)
    (childWire : ∀ child ∈ targetConsumingChildren target, ∀ value outcome,
      wireHeight value < wireHeight wire → RuntimeEvaluation codecs checks targets child value outcome →
        recur child value = .ok outcome) :
    decodeBody codecs checks recur declaration wire = .ok result := by
  cases body
  case scalar encoding kind value rules valid decoded =>
    simp [decodeBody, shape, scalarDecoder_progress decoded, Option.filter, valid]
  case scalarRejected encoding kind rules rejected =>
    cases parsed : decodeScalar codecs encoding kind wire with
    | none => simp [decodeBody, shape, parsed, Option.filter]
    | some value =>
      have invalid := rejected value (scalarDecoder_sound parsed)
      simp [decodeBody, shape, parsed, Option.filter, invalid]
  case nullableNull => simp [decodeBody, shape]
  case nonNullRejected => simp [decodeBody, shape, wireIsNull]
  case alias child decoded =>
    simpa only [decodeBody, shape] using
      sameWire child (by simp [targetNonconsumingChildren]) result decoded
  case select child field decoded =>
    simpa only [decodeBody, shape] using
      sameWire child (by simp [targetNonconsumingChildren]) result decoded
  case nullableValue child nonnull decoded =>
    have done := sameWire child (by simp [targetNonconsumingChildren]) result decoded
    cases wire <;> simp_all [decodeBody, wireIsNull]
  case nonNullValue child nonnull decoded =>
    simpa only [decodeBody, shape, nonnull, Bool.false_eq_true, ite_false] using
      sameWire child (by simp [targetNonconsumingChildren]) result decoded
  case array child bounds items outcomes lengths results =>
    have done : items.mapM (recur child) = .ok outcomes := by
      apply (mapM_ok_iff ..).mpr
      exact ⟨lengths, fun pair member => childWire child (by simp [targetConsumingChildren]) _ _
        (array_wireHeight_lt (List.of_mem_zip member).1) (results pair member)⟩
    simp [decodeBody, shape, done]
  case map kind child rules bounds items keys outcomes keyLengths valueLengths keyResults results =>
    have done : items.mapM (fun entry => recur child entry.2) = .ok outcomes := by
      apply (mapM_ok_iff ..).mpr
      exact ⟨valueLengths, fun pair member => childWire child (by simp [targetConsumingChildren]) _ _
        (object_wireHeight_lt (List.of_mem_zip member).1) (results pair member)⟩
    have parsed : items.map (fun entry => decodeKey codecs.numbers kind entry.1) = keys :=
      all₂_map_eq _ ⟨keyLengths, fun pair member => keyOutcome_complete (keyResults pair member)⟩
    simp [decodeBody, shape, done, parsed]
  case object items members additional outcomes lengths missing present =>
    simp only [decodeBody, shape, unknown]
    apply except_bind_ok.mpr
    refine ⟨outcomes, ?_, rfl⟩
    apply (mapM_ok_iff ..).mpr
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
      apply (mapM_ok_iff ..).mpr
      refine ⟨lengths, ?_⟩
      intro pair member
      exact sameWire pair.1.child
        (by simpa [targetNonconsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
          (List.of_mem_zip member).1)) pair.2 (results pair member)
    simp [decodeBody, shape, done]
  case tagged items tagKey name valueKey value alternative outcome occurrence alternatives tag payload chosen decoded =>
    have done := childWire alternative.child
      (by simpa [targetConsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
        (List.mem_of_find?_eq_some chosen))) value outcome (wireMember_wireHeight_lt payload) decoded
    simp [decodeBody, shape, tag, payload, chosen, done]
    rw [unknown]
  case taggedMalformed items tagKey valueKey occurrence alternatives malformed =>
    simp only [decodeBody, shape]
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
  case protobuf alternative value outcome occurrence alternatives name chosen decoded =>
    have done := childWire alternative.child
      (by simpa [targetConsumingChildren] using (List.mem_map_of_mem (f := TargetAlternative.child)
        (List.mem_of_find?_eq_some chosen))) value outcome (by simp [wireHeight]) decoded
    simp [decodeBody, shape, chosen, done]
  case protobufMalformed occurrence alternatives name value missing => simp [decodeBody, shape, missing]
  case anyValid valid => simp [decodeBody, shape, (jsonValid_iff_JSONValid ..).mpr valid]
  case anyInvalid invalid =>
    have rejected : jsonValid codecs.numbers wire = false := by
      cases found : jsonValid codecs.numbers wire with
      | false => rfl
      | true => exact False.elim (invalid ((jsonValid_iff_JSONValid ..).mp found))
    simp [decodeBody, shape, rejected]
  case mismatch mismatch =>
    cases target <;> cases wire <;> simp_all [decodeBody, structuralMismatch]
    all_goals cases ‹UnionStyle› <;> simp_all

private theorem runtimeEvaluation_parts {codecs checks targets identity wire result}
    (evaluated : RuntimeEvaluation codecs checks targets identity wire result) :
    ∃ declaration ∈ targets, declaration.identity = identity ∧ ∃ bodyResult,
      RuntimeNode codecs checks targets declaration.decoderRejectsUnknown declaration.target wire bodyResult ∧
      result = bodyResult.filter (targetEnumAllowed codecs.numbers declaration.enumeration) := by
  cases evaluated with
  | rejected declaration member same body => exact ⟨declaration, member, same, none, body, rfl⟩
  | accepted declaration member same body allowed =>
    refine ⟨declaration, member, same, _, body, ?_⟩
    simp [Option.filter, (targetEnumAllowed_iff ..).mpr allowed]
  | enumRejected declaration member same body denied =>
    refine ⟨declaration, member, same, _, body, ?_⟩
    have no : targetEnumAllowed codecs.numbers declaration.enumeration _ = false :=
      Bool.eq_false_iff.mpr (fun yes => denied ((targetEnumAllowed_iff ..).mp yes))
    simp [Option.filter, no]

private theorem decodeFuel_complete_node {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} (wellFormed : WellFormedTargets targets)
    {fuel : Nat} {identity : Identity} {wire : Wire} {result : Option Value}
    {declaration : TargetDeclaration}
    (found : findTarget targets identity = some declaration)
    (enough : targetNodeBudget targets declaration wire ≤ fuel)
    (evaluated : RuntimeEvaluation codecs checks targets identity wire result) :
    decodeFuel codecs checks targets fuel identity wire = .ok result := by
  induction fuel generalizing identity wire result declaration with
  | zero => unfold targetNodeBudget at enough; omega
  | succ fuel ih =>
    obtain ⟨actual, member, same, outcome, body, final⟩ := runtimeEvaluation_parts evaluated
    have actualFound : findTarget targets identity = some actual :=
      same ▸ findTarget_of_mem wellFormed member
    have equal : declaration = actual := Option.some.inj (found.symm.trans actualFound)
    subst declaration
    rw [decodeFuel_successor]
    simp only [actualFound]
    have done : decodeBody codecs checks (decodeFuel codecs checks targets fuel) actual wire =
        .ok outcome := by
      apply decodeBody_complete body actual rfl rfl
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
    simp [decodeStep, done, final]

/-- Independent decoder derivations are realized at the finite graph/wire bound.
The runtime uniqueness judgment includes every competing branch outcome. -/
theorem decodeFuel_complete {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {result : Option Value} {fuel : Nat}
    (wellFormed : WellFormedTargets targets)
    (evaluated : RuntimeEvaluation codecs checks targets identity wire result)
    (enough : wireBudget targets wire ≤ fuel) :
    decodeFuel codecs checks targets fuel identity wire = .ok result := by
  obtain ⟨declaration, member, same, _, _, _⟩ := runtimeEvaluation_parts evaluated
  have found := same ▸ findTarget_of_mem wellFormed member
  exact decodeFuel_complete_node wellFormed found
    (Nat.le_trans (targetNodeBudget_le_wireBudget member) enough) evaluated

theorem decode_iff_RuntimeEvaluation {codecs : ScalarCodecs} {checks : ExternalScalarChecks}
    {targets : Targets} {identity : Identity} {wire : Wire} {result : Option Value}
    (wellFormed : WellFormedTargets targets) :
    decode codecs checks targets identity wire = .ok result ↔
      RuntimeEvaluation codecs checks targets identity wire result := by
  simp only [decode, (validateTargets_iff targets).mpr wellFormed, ite_true]
  exact ⟨decodeFuel_sound, fun evaluated => decodeFuel_complete wellFormed evaluated (Nat.le_refl _)⟩

end ValueContract.Candidate
