import ValueContract.SourceBodySpec
import ValueContract.ResolverProofs
import ValueContract.SourceNullProofs

namespace ValueContract.Candidate

private theorem mapEntries_stripHost (input : Input) :
    mapEntries input = mapEntries (stripHostInput input) := by
  cases input <;> simp only [mapEntries, stripHostInput]
  case host identity payload => exact mapEntries_stripHost payload
termination_by sizeOf input

private theorem objectEntries_stripHost (input : Input) :
    objectEntries input = objectEntries (stripHostInput input) := by
  cases input <;> simp only [objectEntries, stripHostInput]
  case host identity payload => exact objectEntries_stripHost payload
termination_by sizeOf input

private theorem stripHost_not_host (input : Input) (identity : Nat) (payload : Input) :
    stripHostInput input ≠ .host identity payload := by
  cases input <;> simp only [stripHostInput, ne_eq, reduceCtorEq, not_false_eq_true]
  case host actual child => exact stripHost_not_host child identity payload
termination_by sizeOf input

/-- Executable map input extraction exactly preserves the independently
specified shape and supplied entries, including transparent host wrappers. -/
theorem mapEntries_iff (input : Input) (entries : List (Scalar × Input)) :
    mapEntries input = some entries ↔ MapInputEntries input entries := by
  rw [mapEntries_stripHost]
  cases shape : stripHostInput input <;> simp [mapEntries, MapInputEntries, shape, eq_comm]
  case host identity payload =>
    exact False.elim (stripHost_not_host input identity payload shape)

private theorem objectMapEntries_iff (inputs : List (Scalar × Input))
    (entries : List (String × Input)) :
    inputs.mapM (fun entry => match entry.1 with
      | .string name => some (name, entry.2)
      | _ => none) = some entries ↔
      All₂ (fun source actual => source.1 = .string actual.1 ∧ source.2 = actual.2) inputs entries := by
  induction inputs generalizing entries with
  | nil => cases entries <;> simp [All₂]
  | cons head tail ih =>
    rcases head with ⟨key, value⟩
    cases entries with
    | nil => cases key <;> simp [List.mapM_cons, All₂, Option.bind_eq_some_iff]
    | cons entry rest =>
      rcases entry with ⟨name, child⟩
      cases key <;> simp [List.mapM_cons, All₂, Option.bind_eq_some_iff, ih, and_assoc, and_left_comm]

/-- Executable object extraction admits precisely objects and all-string-key
maps; a nonstring key cannot disappear during conversion. -/
theorem objectEntries_iff (input : Input) (entries : List (String × Input)) :
    objectEntries input = some entries ↔ ObjectInputEntries input entries := by
  rw [objectEntries_stripHost]
  cases shape : stripHostInput input <;> simp [objectEntries, ObjectInputEntries, shape]
  case object fields => exact eq_comm
  case map inputs => exact objectMapEntries_iff inputs entries
  case host identity payload =>
    exact False.elim (stripHost_not_host input identity payload shape)

/-- Key normalization agrees with the independent coercion/validation rule. -/
theorem mapKeyResolution_iff (checks : ExternalScalarChecks) (kind : MapKeyKind)
    (rules : ScalarRules) (source actual : Scalar × Input) :
    (match kind with
    | .builtin =>
      if mapKeyCompatible kind source.1 = false ∨ scalarAllowed checks rules source.1 = false then
        (.error Failure.invalid : Except Failure (Scalar × Input)) else .ok source
    | .scalar expected => match coerceScalar expected source.1 with
      | some key => if mapKeyCompatible kind key = false ∨ scalarAllowed checks rules key = false then
        .error Failure.invalid else .ok (key, source.2)
      | none => .error Failure.invalid) = .ok actual ↔
      MapKeyResolution checks kind rules source actual := by
  rcases source with ⟨original, input⟩
  rcases actual with ⟨key, child⟩
  cases kind with
  | builtin =>
    simp only [MapKeyResolution]
    split <;> simp_all
    all_goals intro same; subst key; try simp_all
    all_goals intro compatible valid; simp_all
  | scalar expected =>
    cases coerced : coerceScalar expected original with
    | none => simp [coerced, MapKeyResolution, ← coercion_iff]
    | some converted =>
      simp only [coerced, MapKeyResolution, ← coercion_iff, Option.some.injEq]
      split <;> simp_all
      all_goals intro same; subst key; try simp_all
      all_goals intro compatible valid; simp_all

private theorem all₂_cons (relation : α → β → Prop) (a : α) (b : β)
    (left : List α) (right : List β) :
    All₂ relation (a :: left) (b :: right) ↔ relation a b ∧ All₂ relation left right := by
  simp [All₂, and_left_comm]

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

private theorem all₂_map_left (f : α → β) (relation : β → γ → Prop)
    (left : List α) (right : List γ) :
    All₂ relation (left.map f) right ↔ All₂ (fun a b => relation (f a) b) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons other rest => simp only [List.map_cons, all₂_cons, ih]

private theorem pair_exists_ok (name : α) (computation : Except Failure β) (pair : α × β) :
    (∃ value, computation = .ok value ∧ (name, value) = pair) ↔
      name = pair.1 ∧ computation = .ok pair.2 := by
  rcases pair with ⟨key, child⟩
  constructor
  · rintro ⟨value, resolved, same⟩
    cases same
    exact ⟨rfl, resolved⟩
  · rintro ⟨same, resolved⟩
    exact ⟨child, resolved, by simp [same]⟩

private theorem ite_result_ok (condition : Prop) [Decidable condition]
    (yes no : Except Failure α) (value : α) :
    (if condition then yes else no) = .ok value ↔
      (condition ∧ yes = .ok value) ∨ (¬ condition ∧ no = .ok value) := by
  split <;> simp_all

set_option maxRecDepth 4096 in
/-- The declared map worker succeeds exactly for the independent map relation,
for arbitrary child callbacks; key coercion, collisions and missing paths are
all retained in both directions. -/
theorem resolveBody_map_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (kind : MapKeyKind)
    (rules : ScalarRules) (child : Identity) (bounds : LengthBounds) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : Resolution) :
    resolveBody declarations checks keys depth rank complete (.map kind rules child bounds)
      input same descend = .ok result ↔
      MapResolution checks keys kind rules bounds
        (fun input value => descend complete child input = .ok value) input result := by
  have maps := mapEntries_stripHost input
  cases shape : stripHostInput input <;> rw [shape] at maps <;>
    simp only [mapEntries] at maps
  all_goals try exact False.elim (stripHost_not_host _ _ _ shape)
  all_goals simp only [resolveBody, shape, maps, MapResolution, MapInputEntries]
  all_goals simp [ite_result_ok, exceptBind_ok,
    combineChecked_map_iff, nameKeys_iff, exists_and_left, and_assoc]
  all_goals try (intro _; exact eq_comm)
  all_goals intro _
  all_goals simp only [Bind.bind, Except.bind, exceptThrow]
  all_goals simp only [pair_exists_ok, all₂_map_left]
  all_goals simp only [All₂]
  all_goals simp only [← mapKeyResolution_iff]
  all_goals cases kind <;> dsimp only
  all_goals simp only [eq_comm]
  all_goals simp only [← and_assoc, exists_and_right]
  all_goals rfl

private theorem groupedChecked_iff (fields : List (Except Failure α))
    (extras : List (Except Failure β)) (next : List α → List β → Except Failure γ) (result : γ) :
    (do
      let _ ← combineChecked ((fields.map (Except.map (fun _ => ()))) ++
        (extras.map (Except.map (fun _ => ()))))
      let fieldValues ← combineChecked fields
      let extraValues ← combineChecked extras
      next fieldValues extraValues) = .ok result ↔
      ∃ fieldValues extraValues, combineChecked fields = .ok fieldValues ∧
        combineChecked extras = .ok extraValues ∧ next fieldValues extraValues = .ok result := by
  simp only [exceptBind_ok]
  constructor
  · rintro ⟨_, _, fieldValues, fieldsOK, extraValues, extrasOK, done⟩
    exact ⟨fieldValues, extraValues, fieldsOK, extrasOK, done⟩
  · rintro ⟨fieldValues, extraValues, fieldsOK, extrasOK, done⟩
    have fieldsSame := (combineChecked_ok_iff ..).mp fieldsOK
    have extrasSame := (combineChecked_ok_iff ..).mp extrasOK
    refine ⟨fieldValues.map (fun _ => ()) ++ extraValues.map (fun _ => ()), ?_,
      fieldValues, fieldsOK, extraValues, extrasOK, done⟩
    rw [combineChecked_ok_iff]
    simp only [fieldsSame, extrasSame, List.map_map, List.map_append]
    rfl

/-- Executable unknown-field partition equals name-and-alias exclusion; it
never treats an overlap between two declared spellings as unknown data. -/
theorem additionalEntries_iff (members : List Member) (entries : List (String × Input)) :
    entries.filter (fun entry => !members.any (fun member =>
      entry.1 == member.sourceName || entry.1 == member.wireAlias)) =
      AdditionalEntries members entries := by
  unfold AdditionalEntries
  apply List.filter_congr
  intro entry _
  apply Bool.eq_iff_iff.mpr
  simp [List.any_eq_false]

/-- The object worker matches the independent member/extra relation, including
validation of losing aliases and exact retention of incomplete member paths. -/
theorem resolveBody_object_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (members : List Member)
    (isOpen : Bool) (input : Input) (same descend : Bool → Identity → Input → ResolveResult)
    (result : Resolution) :
    resolveBody declarations checks keys depth rank complete (.object members isOpen)
      input same descend = .ok result ↔
      ObjectResolution keys depth complete members isOpen
        (fun identity input value => descend complete identity input = .ok value) input result := by
  have entryShape := objectEntries_stripHost input
  cases shape : stripHostInput input <;> rw [shape] at entryShape <;>
    simp only [objectEntries] at entryShape
  all_goals try exact False.elim (stripHost_not_host _ _ _ shape)
  all_goals try solve | simp [resolveBody, shape, entryShape, ObjectResolution,
    ObjectInputEntries, Bind.bind, Except.bind]
  all_goals simp only [resolveBody, shape, ObjectResolution, ← objectEntries_iff,
    additionalEntries_iff]
  all_goals cases entriesFound : objectEntries input with
  | none => simp [Bind.bind, Except.bind]
  | some entries =>
    simp only [Option.some.injEq, Bind.bind, Except.bind]
    simp [ite_result_ok, exceptThrow, exists_and_left]
    intro unique
    have law := groupedChecked_iff
      (members.map (objectMemberResult complete (descend complete) entries))
      ((AdditionalEntries members entries).map (fun entry => do
        let value ← resolveRawAt keys depth entry.2
        pure (entry.1, value)))
      (fun fields extras => (pure {
        value := .object (fields.map (fun entry => (entry.1, entry.2.value))) extras
        missing := fields.flatMap (fun entry => entry.2.missing) } : ResolveResult)) result
    simp only [Bind.bind, Except.bind, Pure.pure, Except.pure, List.map_map] at law
    rw [law]
    simp only [combineChecked_map_iff, objectMemberResult_iff]
    have rawLaw := fun (name : String) (value : Input) (actual : String × Value) =>
      pairResult_ok name (resolveRawAt keys depth value) actual
    simp only [Bind.bind, Except.bind, Pure.pure, Except.pure, resolveRawAt_iff] at rawLaw
    simp only [All₂, rawLaw]
    cases isOpen <;> simp [exists_and_left, and_assoc]
    all_goals simp only [eq_comm]
    all_goals simp

/-- Declared-map resolution preserves the declared key constraints and produces
a typed child at every returned entry. Child typing is an explicit induction
premise, separate from the body worker's success equivalence. -/
theorem MapResolution_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {complete : Bool} {declaration : Declaration}
    {kind : MapKeyKind} {rules : ScalarRules} {child : Identity} {bounds : LengthBounds}
    {relation : Input → Resolution → Prop} {input : Input} {result : Resolution}
    (present : declaration ∈ declarations) (contract : declaration.contract = .map kind rules child bounds)
    (enumeration : EnumAllows declaration.enumeration result.value)
    (children : ∀ raw value, relation raw value → Typed declarations checks complete child value.value)
    (matched : MapResolution checks keys kind rules bounds relation input result) :
    Typed declarations checks complete declaration.identity result.value := by
  rcases matched with ⟨_, length, shape⟩ |
    ⟨entries, normalized, values, _, length, normalizedKeys, _, unique, related, shape⟩
  · subst result
    exact ⟨1, declaration, present, rfl, enumeration, by simpa only [contract] using length⟩
  · subst result
    have sameKeys := all₂_map_eq related Prod.fst Prod.fst (fun _ _ h => h.1)
    apply Typed_map_intro present contract enumeration
    · rw [normalizedKeys.1, related.1] at length
      simpa using length
    · have keyUnique : (normalized.map Prod.fst).Pairwise (fun a b => scalarEqual a b = false) := by
        simpa [List.pairwise_map] using unique
      rw [sameKeys] at keyUnique
      simpa [List.pairwise_map] using keyUnique
    · intro value member
      obtain ⟨resolution, inside, same⟩ := List.mem_map.mp member
      obtain ⟨normalized, normalizedInside, matching⟩ := all₂_right related inside
      obtain ⟨original, _, keyValid⟩ := all₂_right normalizedKeys normalizedInside
      subst value
      rw [← matching.1]
      exact ⟨keyValid.2.1, keyValid.2.2.1⟩
    · intro value member
      obtain ⟨resolution, inside, same⟩ := List.mem_map.mp member
      obtain ⟨original, _, matching⟩ := all₂_right related inside
      subst value
      exact children _ _ matching.2

private theorem all₂_left {relation : α → β → Prop} {left : List α} {right : List β}
    (matched : All₂ relation left right) {value : α} (member : value ∈ left) :
    ∃ actual ∈ right, relation value actual := by
  induction left generalizing right with
  | nil => simp at member
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂] at matched
    | cons actual rest =>
      have pair := (all₂_cons relation head actual tail rest).mp matched
      rcases List.mem_cons.mp member with rfl | member
      · exact ⟨actual, by simp, pair.1⟩
      · obtain ⟨other, inside, related⟩ := ih pair.2 member
        exact ⟨other, by simp [inside], related⟩

private theorem memberResolution_identity {child : Identity → Input → Resolution → Prop}
    {complete : Bool} {member : Member} {entries : List (String × Input)}
    {result : Identity × Resolution} (matched : MemberResolution child complete member entries result) :
    result.1 = member.identity := by
  rcases matched.2 with ⟨_, _, shape⟩ | ⟨_, _, _, _, shape⟩ <;> simp [shape]

private theorem nodup_map_injective {items : List α} {key : α → β}
    (unique : (items.map key).Nodup) {left right : α}
    (lmem : left ∈ items) (rmem : right ∈ items) (same : key left = key right) : left = right := by
  induction items generalizing left right with
  | nil => simp at lmem
  | cons head tail ih =>
    have unique := List.nodup_cons.mp unique
    rcases List.mem_cons.mp lmem with rfl | leftTail
    · rcases List.mem_cons.mp rmem with rfl | rmem
      · rfl
      · exact False.elim (unique.1 (same ▸ List.mem_map_of_mem rmem))
    · rcases List.mem_cons.mp rmem with rfl | rmem
      · exact False.elim (unique.1 (same ▸ List.mem_map_of_mem leftTail))
      · exact ih unique.2 leftTail rmem same

/-- Object resolution produces independently typed declared members and raw
extras. Unique declaration identities prevent a missing member from being
replaced by another occurrence; alias overlap remains permitted. -/
theorem ObjectResolution_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {rawDepth : Nat} {complete : Bool} {declaration : Declaration}
    {members : List Member} {isOpen : Bool}
    {relation : Identity → Input → Resolution → Prop} {input : Input} {result : Resolution}
    (present : declaration ∈ declarations) (contract : declaration.contract = .object members isOpen)
    (identities : (members.map Member.identity).Nodup)
    (enumeration : EnumAllows declaration.enumeration result.value)
    (children : ∀ identity raw value, relation identity raw value →
      Typed declarations checks complete identity value.value)
    (matched : ObjectResolution keys rawDepth complete members isOpen relation input result) :
    Typed declarations checks complete declaration.identity result.value := by
  obtain ⟨entries, fields, extras, _, uniqueNames, openPolicy, fieldRelation, extraRelation, shape⟩ := matched
  subst result
  have fieldKeys := all₂_map_eq fieldRelation Member.identity Prod.fst
    (fun _ _ related => (memberResolution_identity related).symm)
  have fieldUnique : (fields.map Prod.fst).Nodup := fieldKeys ▸ identities
  have extraKeys := all₂_map_eq extraRelation Prod.fst Prod.fst (fun _ _ related => related.1)
  apply Typed_object_intro present contract enumeration
  · simpa only [List.map_map, Function.comp_def] using fieldUnique
  · intro value member
    obtain ⟨field, inside, same⟩ := List.mem_map.mp member
    obtain ⟨declared, known, related⟩ := all₂_right fieldRelation inside
    subst value
    exact ⟨declared, known, memberResolution_identity related⟩
  · intro declared known
    obtain ⟨field, inside, related⟩ := all₂_left fieldRelation known
    rcases related.2 with ⟨_, mayMiss, fieldShape⟩ | ⟨raw, value, _, interpreted, fieldShape⟩
    · right
      refine ⟨mayMiss, ?_⟩
      intro value member
      obtain ⟨other, otherInside, same⟩ := List.mem_map.mp member
      have sameIdentity : other.1 = field.1 := by
        have names := congrArg Prod.fst same
        simpa [fieldShape] using names
      have sameField := nodup_map_injective fieldUnique otherInside inside sameIdentity
      subst other
      simpa [fieldShape] using (congrArg Prod.snd same).symm
    · left
      have typed := children _ _ _ interpreted
      refine ⟨value.value, ?_, Typed_not_absent typed, typed⟩
      exact List.mem_map.mpr ⟨field, inside, by simp [fieldShape]⟩
  · rcases openPolicy with isOpen | empty
    · exact Or.inl isOpen
    · right
      have sizes := extraRelation.1
      rw [empty] at sizes
      simpa using sizes.symm
  · have extraUnique : ((AdditionalEntries members entries).map Prod.fst).Nodup := by
      exact List.Nodup.sublist (List.Sublist.map Prod.fst List.filter_sublist) uniqueNames
    exact extraKeys ▸ extraUnique
  · intro value inside declared known
    obtain ⟨original, originalInside, related⟩ := all₂_right extraRelation inside
    have exclusion := (List.mem_filter.mp originalInside).2
    have excludes : ∀ member ∈ members,
        original.1 ≠ member.sourceName ∧ original.1 ≠ member.wireAlias := of_decide_eq_true exclusion
    rw [← related.1]
    exact excludes declared known
  · intro value inside
    obtain ⟨_, _, related⟩ := all₂_right extraRelation inside
    exact ⟨rawDepth, rawResolution_free related.2⟩

end ValueContract.Candidate
