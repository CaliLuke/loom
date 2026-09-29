import ValueContract.ResolverProofs
import ValueContract.SourceEqualityBudgetProofs

namespace ValueContract.Candidate

variable {keys : KeyCodec}

/-- Raw resolution preserves null even through host evidence. Typed nil
containers/Bytes are not null, and evidence cannot hide a cycle or absence. -/
theorem rawResolution_null_iff {keys : KeyCodec} {depth : Nat} {input : Input} {value : Value}
    (related : RawResolutionAt keys depth input value) :
    valueIsNull value = true ↔ stripHostInput input = .null := by
  induction depth generalizing input value with
  | zero => simp [RawResolutionAt] at related
  | succ depth ih =>
    cases input <;> cases value <;>
      simp [RawResolutionAt, NativeByteValue] at related <;> simp [valueIsNull, stripHostInput]
    case host.host => exact ih related.2

/-- Whole-node enum acceptance has both soundness and progress at the derived
budget. It does not assume that a failed finite comparison means inequality. -/
theorem enumValueAllowed_iff (enumeration : Option (List Value)) (value : Value) :
    enumValueAllowed keys enumeration value = true ↔ EnumAllows keys enumeration value := by
  cases enumeration with
  | none => simp [enumValueAllowed, EnumAllows]
  | some values =>
    simp only [enumValueAllowed, Option.all_some, List.any_eq_true, EnumAllows]
    have adequate (member : Value) :
        enumEquivalentAt keys (valueDepth value + valueDepth member + 1) value member ↔
          EnumEquivalent keys value member := (enumEquivalentAt_budget (Nat.le_refl _)).symm
    simp only [valueEqualAt_enum_iff, adequate]

private theorem anyResult_null_reflect (keys : KeyCodec) (depth : Nat)
    (input : Input) (result : Resolution)
    (success : (do let value ← resolveRawAt keys depth input
                   pure ({value := .any value, missing := []} : Resolution)) = .ok result)
    (isNull : valueIsNull result.value = true) : stripHostInput input = .null := by
  simp only [exceptBind_ok, exceptReturn_ok] at success
  obtain ⟨value, raw, same⟩ := success
  subst result
  exact (rawResolution_null_iff ((resolveRawAt_iff _ _ _ _).mp raw)).mp isNull

private theorem wrapBranch_not_null (occurrence : Identity) (candidate : BranchCandidate)
    (result : Resolution) (success : wrapBranch occurrence candidate = .ok result) :
    valueIsNull result.value = false := by
  simp only [wrapBranch, exceptBind_ok, exceptReturn_ok] at success
  obtain ⟨value, _, same⟩ := success
  subst result
  rfl


/-- Absence belongs only to an omitted object slot, never a resolved value at
a declaration. Even alias/wrapper chains cannot derive a supplied absence. -/
theorem typedAt_not_absent {depth : Nat} {declarations : Declarations}
    {checks : ExternalScalarChecks} {complete : Bool} {identity : Identity} {value : Value}
    (typed : typedAt keys depth declarations checks complete identity value) : value ≠ .absent := by
  intro same
  subst value
  induction depth generalizing identity with
  | zero => simp [typedAt] at typed
  | succ depth ih =>
    obtain ⟨declaration, _, _, _, body⟩ := typed
    cases contract : declaration.contract <;> simp only [contract] at body
    all_goals first | exact body | exact ih body | exact ih body.2

theorem Typed_not_absent {declarations : Declarations} {checks : ExternalScalarChecks}
    {complete : Bool} {identity : Identity} {value : Value}
    (typed : Typed keys declarations checks complete identity value) : value ≠ .absent := by
  obtain ⟨_, typed⟩ := typed
  exact typedAt_not_absent typed

end ValueContract.Candidate
