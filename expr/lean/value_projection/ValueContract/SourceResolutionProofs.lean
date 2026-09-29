import ValueContract.SourceNullProofs
import ValueContract.SourceRawProofs
import ValueContract.SourceUnionProofs
import ValueContract.SourceBodyProofs

namespace ValueContract.Candidate

variable {keys : KeyCodec}

theorem Coerces_kind {codec format} {kind : ScalarKind} {input output : Scalar}
    (coercion : Coerces codec format kind input output) : output.kind = kind := by
  cases coercion with
  | builtin _ basic => cases basic <;> first | assumption | rfl
  | floating => rfl

/-- Scalar source interpretation produces the declaration's kind and enforces
its constraints. Host evidence and supported coercion do not change that owner. -/
theorem ScalarResolution_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {kind : ScalarKind} {rules : ScalarRules}
    {input : Input} {result : Resolution}
    (present : declaration ∈ declarations) (contract : declaration.contract = .scalar kind rules)
    (enumeration : EnumAllows keys declaration.enumeration result.value)
    (matched : ScalarResolution checks keys kind rules input result) :
    Typed keys declarations checks complete declaration.identity result.value := by
  obtain ⟨scalar, shape, allowed, resultShape⟩ := matched
  have scalarKind : scalar.kind = kind := by
    rcases shape with ⟨_, _, coercion⟩ | ⟨_, sameKind, sameValue⟩ |
        ⟨_, bytes, _, sameKind, _, _, sameValue⟩
    · exact Coerces_kind coercion.2
    · subst kind; subst scalar; rfl
    · subst kind; subst scalar; rfl
  subst result
  exact ⟨1, declaration, present, rfl, enumeration, by simpa only [contract] using ⟨scalarKind, allowed⟩⟩

/-- Every returned array element comes from its corresponding child derivation;
finite child witnesses lift to a common independent typing depth. -/
theorem ArrayResolution_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {child : Identity} {bounds : LengthBounds}
    {relation : Input → Resolution → Prop} {input : Input} {result : Resolution}
    (present : declaration ∈ declarations) (contract : declaration.contract = .array child bounds)
    (enumeration : EnumAllows keys declaration.enumeration result.value)
    (children : ∀ raw value, relation raw value → Typed keys declarations checks complete child value.value)
    (matched : ArrayResolution relation bounds input result) :
    Typed keys declarations checks complete declaration.identity result.value := by
  rcases matched with ⟨_, length, shape⟩ | ⟨inputs, values, _, length, related, shape⟩
  · subst result
    exact ⟨1, declaration, present, rfl, enumeration, by simpa only [contract] using length⟩
  · subst result
    apply Typed_array_intro present contract enumeration
    · rw [related.1] at length
      simpa using length
    · intro value member
      obtain ⟨resolution, inside, same⟩ := List.mem_map.mp member
      obtain ⟨original, _, interpreted⟩ := all₂_right related inside
      subst value
      exact children original resolution interpreted

/-- Supplied branch evidence preserves the exact declared branch identity. -/
theorem SelectedUnionResolution_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {occurrence : Identity}
    {alternatives : List Alternative} {relation : Identity → Input → Resolution → Prop}
    {input : Input} {result : Resolution}
    (present : declaration ∈ declarations)
    (contract : declaration.contract = .union occurrence alternatives)
    (enumeration : EnumAllows keys declaration.enumeration result.value)
    (children : ∀ identity raw value, relation identity raw value →
      Typed keys declarations checks complete identity value.value)
    (matched : SelectedUnionResolution relation occurrence alternatives input result) :
    Typed keys declarations checks complete declaration.identity result.value := by
  obtain ⟨branch, payload, alternative, value, _, chosen, interpreted, shape⟩ := matched
  have inside : alternative ∈ alternatives :=
    List.mem_of_find?_eq_some ((firstAlternative_iff _ _ _).mpr chosen)
  have sameBranch := chosen.1
  subst result
  rw [← sameBranch] at enumeration ⊢
  exact Typed_union_intro present contract enumeration inside (children _ _ _ interpreted)

/-- Complete matching can be weakened for examples; fallback matching cannot
be strengthened for enum/default roles. Neither step changes its chosen value. -/
theorem ImplicitUnionOutcome_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {occurrence : Identity}
    {alternatives : List Alternative}
    {full incomplete : Identity → Input → ResolveResult → Prop}
    {preference : Alternative → Bool} {input : Input} {result : Resolution}
    (present : declaration ∈ declarations)
    (contract : declaration.contract = .union occurrence alternatives)
    (enumeration : EnumAllows keys declaration.enumeration result.value)
    (fullTyped : ∀ identity raw value, full identity raw (.ok value) →
      Typed keys declarations checks true identity value.value)
    (partialTyped : ∀ identity raw value, incomplete identity raw (.ok value) →
      Typed keys declarations checks false identity value.value)
    (matched : ImplicitUnionOutcome full incomplete preference complete occurrence alternatives
      input (.ok result)) :
    Typed keys declarations checks complete declaration.identity result.value := by
  obtain ⟨alternative, inside, value, interpreted, shape⟩ := implicitUnion_success matched
  subst result
  apply Typed_union_intro present contract enumeration inside
  rcases interpreted with full | ⟨partialRole, incomplete⟩
  · have typed := fullTyped _ _ _ full
    cases complete
    · exact Typed_partial typed
    · exact typed
  · subst complete
    exact partialTyped _ _ _ incomplete

theorem resolveBody_any_iff (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (input : Input)
    (same descend : Bool → Identity → Input → ResolveResult) (result : Resolution) :
    resolveBody declarations checks keys depth rank complete .any input same descend = .ok result ↔
      ∃ raw, RawResolutionAt keys (depth + 1) input raw ∧ result = ⟨.any raw, []⟩ := by
  cases shape : stripHostInput input
  case cycle | «opaque» =>
    simp [resolveBody, shape]
    intro raw related
    have supported := rawResolution_inputSupported related
    simp [RawInputSupported, shape] at supported
  all_goals
    simp only [resolveBody, shape, exceptBind_ok, exceptReturn_ok, resolveRawAt_iff]
    simp [eq_comm]

theorem resolveBody_any_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {depth rank : Nat} {complete : Bool} {declaration : Declaration}
    {input : Input} {same descend : Bool → Identity → Input → ResolveResult} {result : Resolution}
    (present : declaration ∈ declarations) (contract : declaration.contract = .any)
    (enumeration : EnumAllows keys declaration.enumeration result.value)
    (success : resolveBody declarations checks keys depth rank complete .any input same descend = .ok result) :
    Typed keys declarations checks complete declaration.identity result.value := by
  obtain ⟨raw, interpreted, shape⟩ := (resolveBody_any_iff _ _ _ _ _ _ _ _ _ _).mp success
  subst result
  exact ⟨1, declaration, present, rfl, enumeration,
    by simpa only [contract] using (show FreeValue keys raw from ⟨_, rawResolution_free interpreted⟩)⟩

theorem Typed_alias_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {child : Identity} {value : Value}
    (present : declaration ∈ declarations) (contract : declaration.contract = .alias child)
    (enumeration : EnumAllows keys declaration.enumeration value)
    (typed : Typed keys declarations checks complete child value) :
    Typed keys declarations checks complete declaration.identity value := by
  obtain ⟨depth, typed⟩ := typed
  exact ⟨depth + 1, declaration, present, rfl, enumeration, by simpa only [contract] using typed⟩

theorem Typed_nullable_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {child : Identity} {value : Value}
    (present : declaration ∈ declarations) (contract : declaration.contract = .nullable child)
    (enumeration : EnumAllows keys declaration.enumeration value)
    (typed : Typed keys declarations checks complete child value) :
    Typed keys declarations checks complete declaration.identity value := by
  obtain ⟨depth, typed⟩ := typed
  refine ⟨depth + 1, declaration, present, rfl, enumeration, ?_⟩
  cases value <;> simp only [contract]
  all_goals first | trivial | exact typed

theorem Typed_nonNull_intro {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {declaration : Declaration} {child : Identity} {value : Value}
    (present : declaration ∈ declarations) (contract : declaration.contract = .nonNull child)
    (enumeration : EnumAllows keys declaration.enumeration value) (notNull : valueIsNull value = false)
    (typed : Typed keys declarations checks complete child value) :
    Typed keys declarations checks complete declaration.identity value := by
  obtain ⟨depth, typed⟩ := typed
  exact ⟨depth + 1, declaration, present, rfl, enumeration,
    by simpa only [contract] using And.intro notNull typed⟩

/-- Null reflection is structural across every body shape. In particular,
non-null occurrence validation can trust source null without a second pass. -/
theorem resolveBody_null_reflect (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (complete : Bool) (contract : Contract)
    (input : Input) (same descend : Bool → Identity → Input → ResolveResult)
    (sameNull : ∀ mode child raw result, same mode child raw = .ok result →
      valueIsNull result.value = true → stripHostInput raw = .null)
    (result : Resolution)
    (success : resolveBody declarations checks keys depth rank complete contract input same descend = .ok result)
    (isNull : valueIsNull result.value = true) : stripHostInput input = .null := by
  cases contract with
  | scalar kind rules =>
    obtain ⟨_, _, _, shape⟩ := (resolveBody_scalar_iff _ _ _ _ _ _ _ _ _ _ _ _).mp success
    subst result
    simp [valueIsNull] at isNull
  | array child bounds =>
    have matched := (resolveBody_array_iff _ _ _ _ _ _ _ _ _ _ _ _).mp success
    rcases matched with ⟨_, _, shape⟩ | ⟨_, _, _, _, _, shape⟩ <;>
      subst result <;> simp [valueIsNull] at isNull
  | map kind rules child bounds =>
    have matched := (resolveBody_map_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _).mp success
    rcases matched with ⟨_, _, shape⟩ | ⟨_, _, _, _, _, _, _, _, shape⟩ <;>
      subst result <;> simp [valueIsNull] at isNull
  | object members isOpen =>
    obtain ⟨_, _, _, _, _, _, _, _, shape⟩ :=
      (resolveBody_object_iff _ _ _ _ _ _ _ _ _ _ _ _).mp success
    subst result
    simp [valueIsNull] at isNull
  | union occurrence alternatives =>
    have notNull := resolveBody_union_not_null _ _ _ _ _ _ _ _ _ _ _ _ success
    simp [notNull] at isNull
  | any =>
    obtain ⟨raw, interpreted, shape⟩ := (resolveBody_any_iff _ _ _ _ _ _ _ _ _ _).mp success
    subst result
    exact (rawResolution_null_iff interpreted).mp isNull
  | alias child | nullable child | nonNull child =>
    cases shape : stripHostInput input <;> simp only [resolveBody, shape] at success
    all_goals first
      | contradiction
      | rfl
      | simpa only [shape] using sameNull _ _ _ _ success isNull
  | custom codec =>
    cases shape : stripHostInput input <;> simp [resolveBody, shape] at success

theorem resolveAt_null_reflect {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {depth rank : Nat} {complete : Bool} {identity : Identity}
    {input : Input} {result : Resolution}
    (success : resolveAt declarations checks keys depth rank complete identity input = .ok result)
    (isNull : valueIsNull result.value = true) : stripHostInput input = .null := by
  induction depth, rank, complete, identity, input using resolveAt.induct declarations checks keys
      generalizing result with
  | case1 => simp [resolveAt] at success
  | case2 depth complete identity input nonzero => cases depth <;> simp [resolveAt] at success
  | case3 depth rank complete identity input declaration found same children =>
    have body := ((resolveAt_body_iff _ _ _ _ _ _ _ _ _ _ found).mp success).1
    exact resolveBody_null_reflect _ _ _ _ _ _ _ _ _ _
      (fun mode child raw result resolved nullValue => same mode child raw resolved nullValue)
      result body isNull
  | case4 => simp [resolveAt, *] at success

/-- Every successful declared-source worker result is independently typed.
This induction covers both same-input graph expansion and finite input descent;
all child typing premises are discharged by the resolver's recursive calls. -/
theorem resolveAt_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {depth rank : Nat} {complete : Bool} {identity : Identity}
    {input : Input} {result : Resolution}
    (wellFormed : WellFormedDeclarations declarations)
    (resolved : resolveAt declarations checks keys depth rank complete identity input = .ok result) :
    Typed keys declarations checks complete identity result.value := by
  induction depth, rank, complete, identity, input using resolveAt.induct declarations checks keys
      generalizing result with
  | case1 => simp [resolveAt] at resolved
  | case2 depth complete identity input nonzero => cases depth <;> simp [resolveAt] at resolved
  | case3 depth rank complete identity input declaration found same children =>
    have pieces := (resolveAt_body_iff _ _ _ _ _ _ _ _ _ _ found).mp resolved
    have body := pieces.1
    have enumeration := enumValueAllowed_sound pieces.2
    have member := findDeclaration_mem found
    have sameIdentity := findDeclaration_identity found
    rw [← sameIdentity]
    cases contract : declaration.contract with
    | scalar kind rules =>
      rw [contract] at body
      exact ScalarResolution_typed member contract enumeration
        ((resolveBody_scalar_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | array child bounds =>
      rw [contract] at body
      exact ArrayResolution_typed member contract enumeration
        (fun raw value success => children complete child raw success)
        ((resolveBody_array_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | any =>
      rw [contract] at body
      exact resolveBody_any_typed member contract enumeration body
    | alias child =>
      cases shape : stripHostInput input <;> simp only [resolveBody, contract, shape] at body
      all_goals first
        | contradiction
        | exact Typed_alias_intro member contract enumeration (same complete child input body)
    | nullable child =>
      cases shape : stripHostInput input <;> simp only [resolveBody, contract, shape] at body
      case null =>
        have resultShape := Except.ok.inj body
        subst result
        exact ⟨1, declaration, member, rfl, enumeration, by simp [contract]⟩
      all_goals first
        | contradiction
        | exact Typed_nullable_intro member contract enumeration (same complete child input body)
    | nonNull child =>
      cases shape : stripHostInput input <;> simp only [resolveBody, contract, shape] at body
      all_goals try contradiction
      all_goals
        have notNull : valueIsNull result.value = false := by
          cases nullValue : valueIsNull result.value
          · rfl
          · have sourceNull := resolveAt_null_reflect body nullValue
            simp [shape] at sourceNull
        exact Typed_nonNull_intro member contract enumeration notNull (same complete child input body)
    | union occurrence alternatives =>
      rw [contract] at body
      cases shape : stripHostInput input
      case cycle | «opaque» => simp [resolveBody, shape] at body
      case selected actual branch payload =>
        exact SelectedUnionResolution_typed member contract enumeration
          (fun child raw value success => children complete child raw success)
          ((resolveBody_selected_union_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _ shape _).mp body)
      all_goals
        have implicit : ImplicitUnionInput input := by simp [ImplicitUnionInput, shape]
        exact ImplicitUnionOutcome_typed member contract enumeration
          (fun child raw value success => same true child raw success)
          (fun child raw value success => same false child raw success)
          ((resolveBody_implicit_union_iff _ _ _ _ _ _ _ _ _ _ _ implicit _).mp body)
    | custom codec =>
      cases shape : stripHostInput input <;> simp [resolveBody, contract, shape] at body
    | map kind rules child bounds =>
      rw [contract] at body
      exact MapResolution_typed member contract enumeration
        (fun raw value success => children complete child raw success)
        ((resolveBody_map_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | object members isOpen =>
      rw [contract] at body
      have names := (wellFormed.2 declaration member).2.2
      simp only [contract] at names
      exact ObjectResolution_typed member contract names.1 enumeration
        (fun child raw value success => children complete child raw success)
        ((resolveBody_object_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body)
  | case4 => simp [resolveAt, *] at resolved

/-- Public source resolution is target-independent and type sound for every
finite accepted input and every validated declaration graph. Enum/default roles
require complete typing; authored and synthesized examples permit omissions. -/
theorem resolve_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {role : Role} {effective : Identity} {input : Input} {result : Resolution}
    (resolved : resolve declarations checks keys role effective input = .ok result) :
    Typed keys declarations checks (role == .enumMember || role == .defaultValue)
      effective result.value := by
  by_cases valid : validateDeclarations declarations = true
  · simp only [resolve, valid, Bool.not_true, Bool.false_eq_true, ↓reduceIte] at resolved
    exact resolveAt_typed ((validateDeclarations_iff _).mp valid) resolved
  · simp [resolve, valid] at resolved

end ValueContract.Candidate
