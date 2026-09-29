import ValueContract.SourceResolutionProofs

namespace ValueContract.Candidate

theorem MemberResolution_complete_missing
    {child : Identity → Input → Resolution → Prop} {member : Member}
    {entries : List (String × Input)} {result : Identity × Resolution}
    (children : ∀ identity input value, child identity input value → value.missing = [])
    (matched : MemberResolution child true member entries result) : result.2.missing = [] := by
  rcases matched.2 with ⟨_, completeness, rfl⟩ | ⟨input, value, _, childResult, rfl⟩
  · have optional : member.required = false := by simpa using completeness
    simp [optional]
  · simp [children _ _ _ childResult]

theorem ArrayResolution_complete_missing {child : Input → Resolution → Prop}
    {bounds : LengthBounds} {input : Input} {result : Resolution}
    (children : ∀ input value, child input value → value.missing = [])
    (matched : ArrayResolution child bounds input result) : result.missing = [] := by
  rcases matched with ⟨_, _, rfl⟩ | ⟨inputs, values, _, _, related, rfl⟩
  · rfl
  · apply List.flatMap_eq_nil_iff.mpr
    intro value member
    obtain ⟨raw, _, childResult⟩ := all₂_right related member
    exact children raw value childResult

theorem MapResolution_complete_missing {child : Input → Resolution → Prop}
    {checks : ExternalScalarChecks} {keys : KeyCodec} {kind : MapKeyKind}
    {rules : ScalarRules} {bounds : LengthBounds} {input : Input} {result : Resolution}
    (children : ∀ input value, child input value → value.missing = [])
    (matched : MapResolution checks keys kind rules bounds child input result) : result.missing = [] := by
  rcases matched with ⟨_, _, rfl⟩ | ⟨inputs, normalized, values, _, _, _, _, related, rfl⟩
  · rfl
  · apply List.flatMap_eq_nil_iff.mpr
    intro value member
    obtain ⟨raw, _, childResult⟩ := all₂_right related member
    exact children raw.2 value.2 childResult.2

theorem ObjectResolution_complete_missing {child : Identity → Input → Resolution → Prop}
    {keys : KeyCodec} {depth : Nat} {members : List Member} {isOpen : Bool}
    {input : Input} {result : Resolution}
    (children : ∀ identity input value, child identity input value → value.missing = [])
    (matched : ObjectResolution keys depth true members isOpen child input result) : result.missing = [] := by
  obtain ⟨_, fields, _, _, _, _, related, _, rfl⟩ := matched
  apply List.flatMap_eq_nil_iff.mpr
  intro field member
  obtain ⟨_, _, childResult⟩ := all₂_right related member
  exact MemberResolution_complete_missing children childResult

/-- Complete-role source success cannot retain an omitted required-member
path, including through recursive collections, aliases and selected unions. -/
theorem resolveAt_complete_missing {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {depth rank : Nat} {complete : Bool} {identity : Identity}
    {input : Input} {result : Resolution}
    (isComplete : complete = true)
    (resolved : resolveAt declarations checks keys depth rank complete identity input = .ok result) :
    result.missing = [] := by
  induction depth, rank, complete, identity, input using resolveAt.induct declarations checks keys
      generalizing result with
  | case1 => simp [resolveAt] at resolved
  | case2 depth complete identity input nonzero => cases depth <;> simp [resolveAt] at resolved
  | case3 depth rank complete identity input declaration found same children =>
    subst complete
    have body := ((resolveAt_body_iff _ _ _ _ _ _ _ _ _ _ found).mp resolved).1
    cases contract : declaration.contract with
    | scalar kind rules =>
      rw [contract] at body
      obtain ⟨_, _, _, rfl⟩ := (resolveBody_scalar_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body
      rfl
    | array child bounds =>
      rw [contract] at body
      exact ArrayResolution_complete_missing
        (fun raw value success => children true child raw rfl success)
        ((resolveBody_array_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | map kind rules child bounds =>
      rw [contract] at body
      exact MapResolution_complete_missing
        (fun raw value success => children true child raw rfl success)
        ((resolveBody_map_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | object members isOpen =>
      rw [contract] at body
      exact ObjectResolution_complete_missing
        (fun child raw value success => children true child raw rfl success)
        ((resolveBody_object_iff _ _ _ _ _ _ _ _ _ _ _ _).mp body)
    | any =>
      rw [contract] at body
      obtain ⟨_, _, rfl⟩ := (resolveBody_any_iff _ _ _ _ _ _ _ _ _ _).mp body
      rfl
    | alias child | nonNull child | nullable child =>
      cases shape : stripHostInput input <;> simp only [resolveBody, contract, shape] at body
      all_goals first
        | contradiction
        | exact same true child input rfl body
        | (have equal := Except.ok.inj body; subst result; rfl)
    | custom codec =>
      cases shape : stripHostInput input <;> simp [resolveBody, contract, shape] at body
    | union occurrence alternatives =>
      rw [contract] at body
      cases shape : stripHostInput input
      case cycle | «opaque» => simp [resolveBody, shape] at body
      case selected actual branch payload =>
        have selected := (resolveBody_selected_union_iff _ _ _ _ _ _ _ _ _ _ _ _ _ _ shape _).mp body
        obtain ⟨_, raw, alternative, value, _, _, childResult, rfl⟩ := selected
        exact children true alternative.child raw (result := value) rfl childResult
      all_goals
        have implicit : ImplicitUnionInput input := by simp [ImplicitUnionInput, shape]
        have matched := (resolveBody_implicit_union_iff _ _ _ _ _ _ _ _ _ _ _ implicit _).mp body
        obtain ⟨alternative, _, value, childResult, rfl⟩ := implicitUnion_success matched
        rcases childResult with full | ⟨impossible, _⟩
        · exact same true alternative.child input (result := value) rfl full
        · contradiction
  | case4 => simp [resolveAt, *] at resolved

/-- Enum/default public success retains no missing-required paths. Examples
remain free to retain them; this does not fill or resynthesize absent fields. -/
theorem resolve_complete_missing {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {role : Role} {identity : Identity} {input : Input} {result : Resolution}
    (complete : (role == .enumMember || role == .defaultValue) = true)
    (resolved : resolve declarations checks keys role identity input = .ok result) :
    result.missing = [] := by
  by_cases valid : validateDeclarations declarations = true
  · simp only [resolve, valid, Bool.not_true, Bool.false_eq_true, ↓reduceIte] at resolved
    exact resolveAt_complete_missing complete resolved
  · simp [resolve, valid] at resolved

end ValueContract.Candidate
