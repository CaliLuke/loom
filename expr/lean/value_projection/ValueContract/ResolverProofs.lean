import ValueContract.Resolve

namespace ValueContract.Candidate

variable {keys : KeyCodec}

/-- Collection succeeds exactly when every supplied child succeeded. No
invalid losing alias can disappear from this relation. -/
theorem combineChecked_ok_iff (results : List (Except Failure α)) (values : List α) :
    combineChecked results = .ok values ↔ results = values.map Except.ok := by
  induction results generalizing values with
  | nil => cases values <;> simp [combineChecked]
  | cons head tail ih =>
    cases head with
    | error failure =>
      cases rest : combineChecked tail <;> cases values <;>
        simp [combineChecked, rest]
    | ok value =>
      cases rest : combineChecked tail with
      | error failure =>
        have none (values : List α) : tail ≠ values.map Except.ok := by
          intro same
          have succeeds := (ih values).mpr same
          simp [rest] at succeeds
        cases values <;> simp [combineChecked, rest, none]
      | ok tailValues =>
        have same := (ih tailValues).mp rest
        simp only [combineChecked, rest]
        cases values <;> simp [same, List.map_inj_right (fun _ _ h => Except.ok.inj h)]

theorem mapOk_iff (f : α → Except Failure β) (inputs : List α) (values : List β) :
    inputs.map f = values.map Except.ok ↔
      All₂ (fun input value => f input = .ok value) inputs values := by
  induction inputs generalizing values with
  | nil => cases values <;> simp [All₂]
  | cons head tail ih =>
    cases values with
    | nil => simp [All₂]
    | cons value rest => simp [All₂, ih, and_left_comm]

theorem combineChecked_map_iff (f : α → Except Failure β)
    (inputs : List α) (values : List β) :
    combineChecked (inputs.map f) = .ok values ↔
      All₂ (fun input value => f input = .ok value) inputs values := by
  rw [combineChecked_ok_iff, mapOk_iff]

theorem exceptBind_ok (result : Except Failure α) (next : α → Except Failure β)
    (value : β) :
    (result >>= next) = .ok value ↔ ∃ intermediate,
      result = .ok intermediate ∧ next intermediate = .ok value := by
  cases result <;> simp [Bind.bind, Except.bind]

theorem exceptReturn_ok (value other : α) :
    (pure value : Except Failure α) = .ok other ↔ value = other := by
  simp [Pure.pure, Except.pure]

/-- Pair construction retains the original key/name as well as the recursively
resolved value; no insertion can silently rename or overwrite an entry. -/
theorem pairResult_ok (name : α) (result : Except Failure β) (pair : α × β) :
    (do let value ← result; pure (name, value)) = .ok pair ↔
      name = pair.1 ∧ result = .ok pair.2 := by
  cases pair
  cases result <;> simp [Bind.bind, Except.bind, Pure.pure, Except.pure, and_comm]

theorem exceptMap_ok (result : Except Failure α) (f : α → β) (value : β) :
    (f <$> result) = .ok value ↔ ∃ intermediate,
      result = .ok intermediate ∧ f intermediate = value := by
  cases result <;> simp [Functor.map, Except.map]

theorem resolveRaw_array_iff (keys : KeyCodec) (depth : Nat)
    (inputs : List Input) (values : List Value) :
    resolveRawAt keys (depth + 1) (.array inputs) = .ok (.array values) ↔
      All₂ (fun input value => resolveRawAt keys depth input = .ok value) inputs values := by
  simp only [resolveRawAt]
  simp only [exceptBind_ok, exceptReturn_ok, Value.array.injEq, exists_eq_right]
  exact combineChecked_map_iff _ _ _

theorem resolveRaw_object_iff (keys : KeyCodec) (depth : Nat)
    (inputs : List (String × Input)) (values : List (String × Value)) :
    resolveRawAt keys (depth + 1) (.object inputs) = .ok (.object [] values) ↔
      (inputs.map Prod.fst).Nodup ∧ All₂ (fun input value => input.1 = value.1 ∧
        resolveRawAt keys depth input.2 = .ok value.2) inputs values := by
  by_cases unique : (inputs.map Prod.fst).Nodup
  · simp only [resolveRawAt, unique, decide_true, Bool.not_true, Bool.false_eq_true, ↓reduceIte]
    simp only [exceptBind_ok, exceptReturn_ok, Value.object.injEq, true_and, exists_eq_right]
    simp only [combineChecked_map_iff, pairResult_ok]
  · simp [resolveRawAt, unique, Bind.bind, Except.bind, throw]

theorem resolveRaw_map_iff (keys : KeyCodec) (depth : Nat)
    (inputs : List (SourceScalar × Input)) (values : List (Scalar × Value)) :
    resolveRawAt keys (depth + 1) (.map inputs) = .ok (.map values) ↔
      (∃ names, names.Nodup ∧ All₂ (KeySpelling keys) (inputs.map (fun entry => entry.1.value)) names) ∧
      All₂ (fun input value => input.1.value = value.1 ∧
        resolveRawAt keys depth input.2 = .ok value.2) inputs values := by
  have pair (input : SourceScalar × Input) (value : Scalar × Value) :
      (∃ child, resolveRawAt keys depth input.2 = .ok child ∧
        (input.1.value, child) = value) ↔
      input.1.value = value.1 ∧ resolveRawAt keys depth input.2 = .ok value.2 := by
    rcases value with ⟨key, child⟩
    constructor
    · rintro ⟨actual, resolved, same⟩
      cases same
      exact ⟨rfl, resolved⟩
    · rintro ⟨same, resolved⟩
      exact ⟨child, resolved, by simp [same]⟩
  simp only [resolveRawAt, exceptBind_ok, exceptReturn_ok, Value.map.injEq,
    exists_eq_right, combineChecked_map_iff, nameKeys_iff]
  simp only [pair, exists_and_right]

/-- Full raw Any/open-member correspondence over arbitrary finite structures.
The depth is a derivation index, not an assumed success or a fixed test bound. -/
theorem resolveRawAt_iff (keys : KeyCodec) (depth : Nat) (input : Input) (value : Value) :
    resolveRawAt keys depth input = .ok value ↔ RawResolutionAt keys depth input value := by
  induction depth generalizing input value with
  | zero => simp [resolveRawAt, RawResolutionAt]
  | succ depth ih =>
    cases input <;> cases value <;>
      simp [resolveRawAt, RawResolutionAt, exceptBind_ok,
        combineChecked_map_iff, nameKeys_iff, All₂, ih, exists_and_right, ← nativeByteValue_iff]
    all_goals first
      | exact and_comm
      | (simp [and_comm, and_left_comm]; done)
      | (split <;> simp_all [exceptBind_ok, combineChecked_map_iff, All₂, throw])
    all_goals simp [and_comm, and_left_comm]

theorem all₂_right {relation : α → β → Prop} {left : List α} {right : List β}
    (matched : All₂ relation left right) {value : β} (member : value ∈ right) :
    ∃ original ∈ left, relation original value := by
  have mapRight := List.map_snd_zip (Nat.le_of_eq matched.1.symm)
  rw [← mapRight] at member
  obtain ⟨⟨original, actual⟩, pairMember, same⟩ := List.mem_map.mp member
  change actual = value at same
  subst actual
  exact ⟨original, (List.of_mem_zip pairMember).1, matched.2 _ pairMember⟩

theorem all₂_map_eq {relation : α → β → Prop} {left : List α} {right : List β}
    (matched : All₂ relation left right) (lkey : α → γ) (rkey : β → γ)
    (same : ∀ l r, relation l r → lkey l = rkey r) :
    left.map lkey = right.map rkey := by
  induction left generalizing right with
  | nil => cases right <;> simp_all [All₂]
  | cons head tail ih =>
    cases right with
    | nil => simp [All₂] at matched
    | cons value rest =>
      have pair : relation head value := matched.2 (head, value) (by simp)
      have remaining : All₂ relation tail rest := by
        constructor
        · simpa using matched.1
        · intro pair member
          exact matched.2 pair (List.mem_cons_of_mem _ member)
      simp [same head value pair, ih remaining]

/-- Raw resolution produces independently well-formed finite built-in values;
source inputs cannot forge target snapshots or selected-union evidence. -/
theorem rawResolution_free {keys : KeyCodec} {depth : Nat} {input : Input} {value : Value}
    (matched : RawResolutionAt keys depth input value) : freeValueAt keys depth value := by
  induction depth generalizing input value with
  | zero => simp [RawResolutionAt] at matched
  | succ depth ih =>
    cases input <;> cases value <;> simp [RawResolutionAt] at matched <;>
      simp only [freeValueAt]
    case array.array inputs values =>
      intro value member
      obtain ⟨original, _, related⟩ := all₂_right matched member
      exact ih related
    case object.object inputs fields values =>
      obtain ⟨empty, unique, pairs⟩ := matched
      subst fields
      have names := all₂_map_eq pairs Prod.fst Prod.fst (fun _ _ h => h.1)
      refine ⟨rfl, names ▸ unique, ?_⟩
      intro value member
      obtain ⟨original, _, related⟩ := all₂_right pairs member
      exact ih related.2
    case map.map inputs values =>
      obtain ⟨admissible, pairs⟩ := matched
      have names := all₂_map_eq pairs (fun entry => entry.1.value) Prod.fst (fun _ _ h => h.1)
      refine ⟨?_, ?_⟩
      · exact names ▸ admissible
      · intro value member
        obtain ⟨original, _, related⟩ := all₂_right pairs member
        exact ih related.2
    case host.host => exact ih matched.2
    all_goals first | trivial | simp [NativeByteValue] at matched

theorem hasSuppliedValue_iff (input : Input) :
    hasSuppliedValue input = true ↔ input ≠ .absent := by
  cases input <;> simp [hasSuppliedValue]

theorem hasSuppliedValue_false_iff (input : Input) :
    hasSuppliedValue input = false ↔ input = .absent := by
  cases input <;> simp [hasSuppliedValue]

theorem findNamed_some_iff (name : String) (entries : List (String × Input)) (input : Input) :
    (entries.find? (fun entry => entry.1 == name && hasSuppliedValue entry.2)).map Prod.snd =
      some input ↔ FirstSupplied name entries input := by
  simp [List.find?_eq_some_iff_append, hasSuppliedValue_iff, hasSuppliedValue_false_iff,
    FirstSupplied, exists_and_left, exists_and_right, and_assoc]

theorem findNamed_none_iff (name : String) (entries : List (String × Input)) :
    (entries.find? (fun entry => entry.1 == name && hasSuppliedValue entry.2)).map Prod.snd =
      none ↔ NoSupplied name entries := by
  simp [List.find?_eq_none, hasSuppliedValue_iff, NoSupplied]

theorem selectedMemberInput_iff (member : Member) (entries : List (String × Input))
    (input : Option Input) : selectedMemberInput member entries = input ↔
      MemberChoice member entries input := by
  have selection : selectedMemberInput member entries =
      ((entries.find? (fun entry => entry.1 == member.wireAlias && hasSuppliedValue entry.2)).map Prod.snd).orElse
        (fun _ => (entries.find? (fun entry => entry.1 == member.sourceName && hasSuppliedValue entry.2)).map Prod.snd) := by
    unfold selectedMemberInput
    cases entries.find? (fun entry => entry.1 == member.wireAlias && hasSuppliedValue entry.2) <;> rfl
  rw [selection]
  cases wire : (entries.find? (fun entry => entry.1 == member.wireAlias && hasSuppliedValue entry.2)).map Prod.snd <;>
    cases input <;>
    simp [MemberChoice, Option.orElse, ← findNamed_some_iff, ← findNamed_none_iff, wire]

theorem exists_all₂_iff (relation : α → β → Prop) (inputs : List α) :
    (∃ values, All₂ relation inputs values) ↔ ∀ input ∈ inputs, ∃ value, relation input value := by
  induction inputs with
  | nil => simp [All₂]; exact ⟨[], rfl⟩
  | cons head tail ih =>
    constructor
    · rintro ⟨values, matched⟩
      cases values with
      | nil => simp [All₂] at matched
      | cons value rest =>
        have pair := matched.2 (head, value) (by simp)
        have remaining : All₂ relation tail rest := by
          constructor
          · simpa using matched.1
          · intro pair member
            exact matched.2 pair (List.mem_cons_of_mem _ member)
        intro input member
        rcases List.mem_cons.mp member with rfl | member
        · exact ⟨value, pair⟩
        · exact ih.mp ⟨rest, remaining⟩ input member
    · intro each
      obtain ⟨value, first⟩ := each head (by simp)
      obtain ⟨rest, remaining⟩ := ih.mpr (fun input member => each input (by simp [member]))
      refine ⟨value :: rest, ?_⟩
      simpa [All₂] using And.intro remaining.1 (And.intro first remaining.2)

theorem combineChecked_map_succeeds_iff (f : α → Except Failure β) (inputs : List α) :
    (∃ values, combineChecked (inputs.map f) = .ok values) ↔
      ∀ input ∈ inputs, ∃ value, f input = .ok value := by
  simp only [combineChecked_map_iff, exists_all₂_iff]

/-- Member validation is exactly the independent alias and missing-field rule,
for every recursive child implementation. Both directions quantify over all
supplied aliases; a resolver that ignores the losing spelling fails this law. -/
theorem objectMemberResult_iff (complete : Bool)
    (child : Identity → Input → ResolveResult) (entries : List (String × Input))
    (member : Member) (result : Identity × Resolution) :
    objectMemberResult complete child entries member = .ok result ↔
      MemberResolution (fun identity input value => child identity input = .ok value)
        complete member entries result := by
  have checked :
      (∃ values, combineChecked ((entries.filter (fun entry => hasSuppliedValue entry.2 &&
        (entry.1 == member.sourceName || entry.1 == member.wireAlias))).map
          (fun entry => child member.child entry.2)) = .ok values) ↔
      ∀ entry ∈ entries, entry.2 ≠ .absent →
        (entry.1 = member.sourceName ∨ entry.1 = member.wireAlias) →
          ∃ value, child member.child entry.2 = .ok value := by
    rw [combineChecked_map_succeeds_iff]
    simp [hasSuppliedValue_iff, and_imp]
  cases selected : selectedMemberInput member entries with
  | none =>
    have absent : MemberChoice member entries none := (selectedMemberInput_iff _ _ _).mp selected
    have noSome (input : Input) : ¬ MemberChoice member entries (some input) := by
      rw [← selectedMemberInput_iff, selected]
      simp
    simp only [objectMemberResult, selected, exceptBind_ok]
    cases complete <;> cases required : member.required <;>
      simp [required, MemberResolution, absent, noSome, exists_and_right, checked]
    all_goals simp [eq_comm, Bind.bind, Except.bind]
  | some input =>
    have choice (other : Input) : MemberChoice member entries (some other) ↔ input = other := by
      rw [← selectedMemberInput_iff, selected]
      simp
    have notAbsent : ¬ MemberChoice member entries none := by
      rw [← selectedMemberInput_iff, selected]
      simp
    simp only [objectMemberResult, selected, exceptBind_ok, exceptReturn_ok]
    simp [MemberResolution, choice, notAbsent, exists_and_right, checked]
    simp [eq_comm]

/-- Enum checking can establish membership only through the independently
defined enum relation. No generic semantic equality or nil policy is assumed. -/
theorem enumValueAllowed_sound {enumeration : Option (List Value)} {value : Value}
    (allowed : enumValueAllowed keys enumeration value = true) : EnumAllows keys enumeration value := by
  cases enumeration with
  | none => trivial
  | some members =>
    simp only [enumValueAllowed, Option.all_some, List.any_eq_true] at allowed
    obtain ⟨member, present, equal⟩ := allowed
    exact ⟨member, present, _, (valueEqualAt_enum_iff _ _ _).mp equal⟩

/-- A deeper derivation preserves independently established typing. This is a
proof-index law only; it imposes no bound on the depth of admitted values. -/
theorem typedAt_step {depth : Nat} {declarations : Declarations}
    {checks : ExternalScalarChecks} {complete : Bool} {identity : Identity} {value : Value}
    (typed : typedAt keys depth declarations checks complete identity value) :
    typedAt keys (depth + 1) declarations checks complete identity value := by
  induction depth generalizing identity value with
  | zero => simp [typedAt] at typed
  | succ depth ih =>
    rw [typedAt] at typed ⊢
    obtain ⟨declaration, member, same, enumeration, body⟩ := typed
    refine ⟨declaration, member, same, enumeration, ?_⟩
    cases contract : declaration.contract <;> cases value <;>
      simp only [contract] at body ⊢
    all_goals try first
      | exact body
      | exact ih body
      | exact False.elim body
    all_goals try
      exact ⟨body.1, ih body.2⟩
    case array.array child bounds values =>
      exact ⟨body.1, fun item inside => ih (body.2 item inside)⟩
    case map.map kind rules child bounds entries =>
      refine ⟨body.1, body.2.1, ?_⟩
      intro entry inside
      have valid := body.2.2 entry inside
      exact ⟨valid.1, valid.2.1, ih valid.2.2⟩
    case object.object members isOpen fields extra =>
      refine ⟨body.1, body.2.1, ?_, body.2.2.2⟩
      intro field inside
      rcases body.2.2.1 field inside with present | missing
      · obtain ⟨item, found, supplied, valid⟩ := present
        exact Or.inl ⟨item, found, supplied, ih valid⟩
      · exact Or.inr missing
    case union.union occurrence alternatives actual branch payload =>
      obtain ⟨sameOccurrence, alternative, inside, sameBranch, valid⟩ := body
      exact ⟨sameOccurrence, alternative, inside, sameBranch, ih valid⟩

theorem typedAt_mono {depth larger : Nat} {declarations : Declarations}
    {checks : ExternalScalarChecks} {complete : Bool} {identity : Identity} {value : Value}
    (le : depth ≤ larger) (typed : typedAt keys depth declarations checks complete identity value) :
    typedAt keys larger declarations checks complete identity value := by
  induction le with
  | refl => exact typed
  | step le ih => exact typedAt_step ih

/-- Requiring completeness can only restrict acceptance; a complete selected
branch remains a valid partial interpretation without changing its value. -/
theorem typedAt_partial {depth : Nat} {declarations : Declarations}
    {checks : ExternalScalarChecks} {complete : Bool} {identity : Identity} {value : Value}
    (typed : typedAt keys depth declarations checks complete identity value) :
    typedAt keys depth declarations checks false identity value := by
  induction depth generalizing identity value with
  | zero => simp [typedAt] at typed
  | succ depth ih =>
    rw [typedAt] at typed ⊢
    obtain ⟨declaration, member, same, enumeration, body⟩ := typed
    refine ⟨declaration, member, same, enumeration, ?_⟩
    cases contract : declaration.contract <;> cases value <;>
      simp only [contract] at body ⊢
    all_goals try first
      | exact body
      | exact ih body
      | exact False.elim body
    all_goals try
      exact ⟨body.1, ih body.2⟩
    case array.array child bounds values =>
      exact ⟨body.1, fun item inside => ih (body.2 item inside)⟩
    case map.map kind rules child bounds entries =>
      refine ⟨body.1, body.2.1, ?_⟩
      intro entry inside
      have valid := body.2.2 entry inside
      exact ⟨valid.1, valid.2.1, ih valid.2.2⟩
    case object.object members isOpen fields extra =>
      refine ⟨body.1, body.2.1, ?_, body.2.2.2⟩
      intro field inside
      rcases body.2.2.1 field inside with present | missing
      · obtain ⟨item, found, supplied, valid⟩ := present
        exact Or.inl ⟨item, found, supplied, ih valid⟩
      · exact Or.inr ⟨Or.inl (by trivial), missing.2⟩
    case union.union occurrence alternatives actual branch payload =>
      obtain ⟨sameOccurrence, alternative, inside, sameBranch, valid⟩ := body
      exact ⟨sameOccurrence, alternative, inside, sameBranch, ih valid⟩

theorem Typed_partial {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {identity : Identity} {value : Value}
    (typed : Typed keys declarations checks complete identity value) :
    Typed keys declarations checks false identity value := by
  obtain ⟨depth, typed⟩ := typed
  exact ⟨depth, typedAt_partial typed⟩

/-- Finitely many recursive typing witnesses have a common derivation depth.
The chosen maximum depends on the actual witnesses, never a global value bound. -/
theorem commonDepth (items : List α) (predicate : Nat → α → Prop)
    (monotone : ∀ smaller larger item, smaller ≤ larger →
      predicate smaller item → predicate larger item)
    (each : ∀ item ∈ items, ∃ depth, predicate depth item) :
    ∃ depth, ∀ item ∈ items, predicate depth item := by
  induction items with
  | nil => exact ⟨0, by simp⟩
  | cons head tail ih =>
    obtain ⟨headDepth, headValid⟩ := each head (by simp)
    obtain ⟨tailDepth, tailValid⟩ := ih (fun item inside => each item (by simp [inside]))
    refine ⟨max headDepth tailDepth, ?_⟩
    intro item inside
    rcases List.mem_cons.mp inside with rfl | inside
    · exact monotone _ _ _ (Nat.le_max_left _ _) headValid
    · exact monotone _ _ _ (Nat.le_max_right _ _) (tailValid item inside)

theorem enumGate_iff (computation : ResolveResult) (enumeration : Option (List Value))
    (result : Resolution) :
    (do let resolved ← computation
        if enumValueAllowed keys enumeration resolved.value then pure resolved else throw Failure.invalid) =
      .ok result ↔ computation = .ok result ∧ enumValueAllowed keys enumeration result.value = true := by
  cases computation with
  | error failure => simp [Bind.bind, Except.bind]
  | ok resolved =>
    by_cases allowed : enumValueAllowed keys enumeration resolved.value = true
    · simp [Bind.bind, Except.bind, allowed, Pure.pure, Except.pure]
      intro equal
      cases equal
      simpa using allowed
    · simp [Bind.bind, Except.bind, allowed]
      intro equal
      cases equal
      simpa using allowed

/-- Uniform whole-node gate for EVERY contract shape. Factoring the body call
is semantically necessary: nested do/return branches must not exit this gate. -/
theorem resolveAt_body_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (identity : Identity)
    (input : Input) (declaration : Declaration) (result : Resolution)
    (found : findDeclaration declarations identity = some declaration) :
    resolveAt declarations checks keys (depth + 1) (rank + 1) complete identity input = .ok result ↔
      resolveBody declarations checks keys depth rank complete declaration.contract input
        (fun mode child value => resolveAt declarations checks keys (depth + 1) rank mode child value)
        (fun mode child value => resolveAt declarations checks keys depth
          (maximumExpansionRank declarations + 1) mode child value) = .ok result ∧
      enumValueAllowed keys declaration.enumeration result.value = true := by
  rw [resolveAt]
  simp only [found]
  exact enumGate_iff _ _ _

theorem resolveAt_enum_sound {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {depth rank : Nat} {complete : Bool} {identity : Identity}
    {input : Input} {declaration : Declaration} {result : Resolution}
    (found : findDeclaration declarations identity = some declaration)
    (resolved : resolveAt declarations checks keys depth rank complete identity input = .ok result) :
    EnumAllows keys declaration.enumeration result.value := by
  cases depth with
  | zero => simp [resolveAt] at resolved
  | succ depth =>
    cases rank with
    | zero => simp [resolveAt] at resolved
    | succ rank =>
      exact enumValueAllowed_sound
        ((resolveAt_body_iff _ _ _ _ _ _ _ _ _ _ found).mp resolved).2

theorem Typed_array_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {child : Identity} {bounds : LengthBounds}
    {values : List Value} (member : declaration ∈ declarations)
    (contract : declaration.contract = .array child bounds)
    (enumeration : EnumAllows keys declaration.enumeration (.array values))
    (length : lengthAllowed bounds values.length = true)
    (children : ∀ value ∈ values, Typed keys declarations checks complete child value) :
    Typed keys declarations checks complete declaration.identity (.array values) := by
  obtain ⟨depth, allTyped⟩ := commonDepth values
    (fun depth value => typedAt keys depth declarations checks complete child value)
    (fun _ _ _ le typed => typedAt_mono le typed) children
  refine ⟨depth + 1, declaration, member, rfl, enumeration, ?_⟩
  simpa [contract, All] using And.intro length allTyped

theorem Typed_map_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {kind : MapKeyKind} {rules : ScalarRules}
    {child : Identity} {bounds : LengthBounds} {values : List (Scalar × Value)}
    (member : declaration ∈ declarations) (contract : declaration.contract = .map kind rules child bounds)
    (enumeration : EnumAllows keys declaration.enumeration (.map values))
    (length : lengthAllowed bounds values.length = true)
    (unique : AdmissibleKeys keys (values.map Prod.fst))
    (keyRules : ∀ value ∈ values, mapKeyCompatible kind value.1 = true ∧
      scalarAllowed checks rules value.1 = true)
    (children : ∀ value ∈ values, Typed keys declarations checks complete child value.2) :
    Typed keys declarations checks complete declaration.identity (.map values) := by
  obtain ⟨depth, allTyped⟩ := commonDepth values
    (fun depth value => typedAt keys depth declarations checks complete child value.2)
    (fun _ _ _ le typed => typedAt_mono le typed) children
  refine ⟨depth + 1, declaration, member, rfl, enumeration, ?_⟩
  simp only [contract]
  exact ⟨length, unique, fun value inside => ⟨(keyRules value inside).1,
    (keyRules value inside).2, allTyped value inside⟩⟩

theorem Typed_union_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {occurrence : Identity}
    {alternatives : List Alternative} {alternative : Alternative} {value : Value}
    (member : declaration ∈ declarations)
    (contract : declaration.contract = .union occurrence alternatives)
    (enumeration : EnumAllows keys declaration.enumeration (.union occurrence alternative.identity value))
    (inside : alternative ∈ alternatives)
    (child : Typed keys declarations checks complete alternative.child value) :
    Typed keys declarations checks complete declaration.identity (.union occurrence alternative.identity value) := by
  obtain ⟨depth, typed⟩ := child
  refine ⟨depth + 1, declaration, member, rfl, enumeration, ?_⟩
  simp only [contract, true_and]
  exact ⟨alternative, inside, rfl, typed⟩

theorem Typed_object_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {members : List Member} {isOpen : Bool}
    {fields : List (Identity × Value)} {extra : List (String × Value)}
    (present : declaration ∈ declarations) (contract : declaration.contract = .object members isOpen)
    (enumeration : EnumAllows keys declaration.enumeration (.object fields extra))
    (unique : (fields.map Prod.fst).Nodup)
    (known : ∀ entry ∈ fields, ∃ member ∈ members, entry.1 = member.identity)
    (children : ∀ member ∈ members,
      (∃ value, (member.identity, value) ∈ fields ∧ value ≠ .absent ∧
        Typed keys declarations checks complete member.child value) ∨
      ((complete = false ∨ member.required = false) ∧
        ∀ value, (member.identity, value) ∈ fields → value = .absent))
    (openAllowed : isOpen = true ∨ extra = []) (extraUnique : (extra.map Prod.fst).Nodup)
    (extraNames : ∀ entry ∈ extra, ∀ member ∈ members,
      entry.1 ≠ member.sourceName ∧ entry.1 ≠ member.wireAlias)
    (extraTyped : ∀ entry ∈ extra, FreeValue keys entry.2) :
    Typed keys declarations checks complete declaration.identity (.object fields extra) := by
  let P := fun depth (member : Member) =>
    (∃ value, (member.identity, value) ∈ fields ∧ value ≠ .absent ∧
      typedAt keys depth declarations checks complete member.child value) ∨
    ((complete = false ∨ member.required = false) ∧
      ∀ value, (member.identity, value) ∈ fields → value = .absent)
  have each : ∀ member ∈ members, ∃ depth, P depth member := by
    intro member inside
    rcases children member inside with supplied | missing
    · obtain ⟨value, found, nonabsent, depth, typed⟩ := supplied
      exact ⟨depth, Or.inl ⟨value, found, nonabsent, typed⟩⟩
    · exact ⟨0, Or.inr missing⟩
  have monotone : ∀ smaller larger member, smaller ≤ larger → P smaller member → P larger member := by
    intro smaller larger member le matched
    rcases matched with supplied | missing
    · obtain ⟨value, found, nonabsent, typed⟩ := supplied
      exact Or.inl ⟨value, found, nonabsent, typedAt_mono le typed⟩
    · exact Or.inr missing
  obtain ⟨depth, allTyped⟩ := commonDepth members P monotone each
  refine ⟨depth + 1, declaration, present, rfl, enumeration, ?_⟩
  simp only [contract]
  exact ⟨unique, known, allTyped, openAllowed, extraUnique, extraNames, extraTyped⟩

theorem coercion_iff (keys : KeyCodec) (format : NumericFormat) (kind : ScalarKind) (input output : Scalar) :
    coerceScalar keys format kind input = some output ↔ Coerces keys format kind input output :=
  ⟨coercion_sound, coercion_progress⟩

theorem arrayInput_iff (input : Input) (items : Option (List Input)) :
    arrayInput input = some items ↔ ArrayInputEntries input items := by
  cases input <;> simp only [arrayInput, ArrayInputEntries]
  all_goals try simp only [reduceCtorEq, Option.some.injEq]
  all_goals try exact eq_comm
  all_goals try exact iff_false_intro id
  case byteSequence sequence =>
    simp only [← nativeBytesNil_iff]
    cases sequence.isNil <;> simp [eq_comm]

 theorem resolveBody_array_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (child : Identity)
    (bounds : LengthBounds) (input : Input) (same descend : Bool → Identity → Input → ResolveResult)
    (result : Resolution) :
    resolveBody declarations checks keys depth rank complete (.array child bounds) input same descend = .ok result ↔
      ArrayResolution (fun input value => descend complete child input = .ok value) bounds input result := by
  cases shape : stripHostInput input
  all_goals try (simp [resolveBody, ArrayResolution, arrayInput, ArrayInputEntries, shape]; done)
  case byteSequence sequence =>
    cases nilShape : sequence.isNil <;>
      simp [resolveBody, ArrayResolution, arrayInput, ArrayInputEntries, shape,
        ← nativeBytesNil_iff, nilShape]
    all_goals split <;> simp_all [exceptBind_ok, combineChecked_map_iff, throw]
    all_goals simp [eq_comm]
  case nilArray =>
    simp only [resolveBody, shape, arrayInput, ArrayResolution, ArrayInputEntries]
    split <;> simp_all
    all_goals simp [eq_comm]
  case array items =>
    simp only [resolveBody, shape, arrayInput, ArrayResolution, ArrayInputEntries]
    split <;> simp_all [exceptBind_ok, combineChecked_map_iff, throw]
    all_goals simp [eq_comm]

theorem resolveBody_scalar_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (kind : ScalarKind)
    (rules : ScalarRules) (input : Input) (same descend : Bool → Identity → Input → ResolveResult)
    (result : Resolution) :
    resolveBody declarations checks keys depth rank complete (.scalar kind rules) input same descend = .ok result ↔
      ScalarResolution checks keys kind rules input result := by
  cases shape : stripHostInput input <;> simp [resolveBody, ScalarResolution, shape, ← sourceCoercion_iff, ← nativeBytesAdmission_iff, ← nativeByteOctets_iff]
  case nilBytes =>
    cases kind <;> simp
    all_goals split <;> simp_all
    all_goals simp [eq_comm]
  case byteSequence sequence =>
    cases kind <;> simp
    all_goals cases sequence.bytesAdmission <;> simp
    all_goals split <;> simp_all
    all_goals simp [eq_comm]
  case scalar original =>
    cases encoded : coerceSourceScalar keys rules kind original with
    | none => simp
    | some value =>
      simp only [Option.some.injEq]
      split <;> simp_all
      all_goals simp [eq_comm]

private theorem noPreferredWhenNoViable (candidates : List BranchCandidate)
    (empty : candidates.filter BranchCandidate.viable = []) :
    candidates.filter (fun c => c.viable && c.preferred) = [] := by
  apply List.filter_eq_nil_iff.mpr
  intro candidate member
  have absent := List.filter_eq_nil_iff.mp empty candidate member
  simp [absent]

/-- Soundness covers every result, not only successful selection. In particular,
the implementation cannot use partial candidates after complete ambiguity. -/
theorem rankCandidates_sound (candidates : List BranchCandidate) :
    RankedCandidates candidates (rankCandidates candidates) := by
  cases preferred : candidates.filter (fun c => c.viable && c.preferred) with
  | nil =>
    cases viable : candidates.filter BranchCandidate.viable with
    | nil => simpa [rankCandidates, preferred, viable] using RankedCandidates.none viable
    | cons chosen rest =>
      cases rest with
      | nil =>
        simpa [rankCandidates, preferred, viable] using
          RankedCandidates.soleUnpreferred chosen preferred viable
      | cons next rest =>
        have many : 2 ≤ (candidates.filter BranchCandidate.viable).length := by
          simp [viable]
        simpa [rankCandidates, preferred, viable] using
          RankedCandidates.manyUnpreferred preferred many
  | cons chosen rest =>
    cases rest with
    | nil =>
      simpa [rankCandidates, preferred] using RankedCandidates.uniquePreferred chosen preferred
    | cons next rest =>
      have many : 2 ≤ (candidates.filter (fun c => c.viable && c.preferred)).length := by
        simp [preferred]
      simpa [rankCandidates, preferred] using RankedCandidates.manyPreferred many

/-- Progress follows an independent derivation. It rules out an always-reject
selector, including arbitrarily many invalid and nonpreferred competitors. -/
theorem rankCandidates_progress {candidates : List BranchCandidate} {result}
    (specified : RankedCandidates candidates result) :
    rankCandidates candidates = result := by
  cases specified with
  | uniquePreferred chosen preferred => simp [rankCandidates, preferred]
  | soleUnpreferred chosen preferred viable => simp [rankCandidates, preferred, viable]
  | none empty =>
    simp [rankCandidates, noPreferredWhenNoViable candidates empty, empty]
  | manyPreferred many =>
    cases preferred : candidates.filter (fun c => c.viable && c.preferred) with
    | nil => simp [preferred] at many
    | cons chosen rest =>
      cases rest with
      | nil => simp [preferred] at many
      | cons next rest => simp [rankCandidates, preferred]
  | manyUnpreferred preferred many =>
    cases viable : candidates.filter BranchCandidate.viable with
    | nil => simp [viable] at many
    | cons chosen rest =>
      cases rest with
      | nil => simp [viable] at many
      | cons next rest => simp [rankCandidates, preferred, viable]

theorem rankCandidates_iff (candidates : List BranchCandidate) (result) :
    rankCandidates candidates = result ↔ RankedCandidates candidates result := by
  constructor
  · intro equal
    rw [← equal]
    exact rankCandidates_sound candidates
  · exact rankCandidates_progress

theorem rankCompleteFirst_iff (complete fallback : List BranchCandidate) (result) :
    rankCompleteFirst complete (fun _ => fallback) = result ↔
      CompleteFirst complete fallback result := by
  constructor
  · intro equal
    cases complete with
    | nil =>
      exact .fallbackSet rfl (rankCandidates_iff fallback result |>.mp
        (by simpa [rankCompleteFirst] using equal))
    | cons head tail =>
      exact .completeSet (by simp) (rankCandidates_iff (head :: tail) result |>.mp
        (by simpa [rankCompleteFirst] using equal))
  · intro specified
    cases specified with
    | fallbackSet empty ranked =>
      subst complete
      simpa [rankCompleteFirst] using rankCandidates_progress ranked
    | completeSet present ranked =>
      cases complete with
      | nil => exact False.elim (present rfl)
      | cons head tail => simpa [rankCompleteFirst] using rankCandidates_progress ranked
/-- Ranking can only return a candidate from the validated input list. It
cannot synthesize an alternative or replace its retained branch identity. -/
theorem rankCandidates_chosen {candidates : List BranchCandidate} {chosen : BranchCandidate}
    (selected : rankCandidates candidates = .ok chosen) : chosen ∈ candidates := by
  have ranked := (rankCandidates_iff candidates (.ok chosen)).mp selected
  cases ranked with
  | uniquePreferred chosen preferred =>
    exact (List.mem_filter.mp (show chosen ∈ candidates.filter
      (fun c => c.viable && c.preferred) from preferred ▸ (by simp))).1
  | soleUnpreferred chosen _ viable =>
    exact (List.mem_filter.mp (show chosen ∈ candidates.filter BranchCandidate.viable
      from viable ▸ (by simp))).1

theorem rankCompleteFirst_chosen {complete fallback : List BranchCandidate}
    {chosen : BranchCandidate}
    (selected : rankCompleteFirst complete (fun _ => fallback) = .ok chosen) :
    chosen ∈ complete ∨ (complete = [] ∧ chosen ∈ fallback) := by
  cases complete with
  | nil =>
    exact Or.inr ⟨rfl, rankCandidates_chosen (by simpa [rankCompleteFirst] using selected)⟩
  | cons head tail =>
    exact Or.inl (rankCandidates_chosen (by simpa [rankCompleteFirst] using selected))

end ValueContract.Candidate
