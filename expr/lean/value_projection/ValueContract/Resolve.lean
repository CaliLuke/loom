import ValueContract.ResolutionSpec
import ValueContract.ValueEquality
import ValueContract.MapKeys

namespace ValueContract.Candidate

/-- Select only after recursive validation. An ambiguity obstruction remains
eligible for fallback ranking and is never mistaken for an invalid branch. -/
def rankCandidates (candidates : List BranchCandidate) : Except Failure BranchCandidate :=
  match candidates.filter (fun c => c.viable && c.preferred) with
  | [chosen] => .ok chosen
  | [] =>
    match candidates.filter BranchCandidate.viable with
    | [] => .error .invalid
    | [chosen] => .ok chosen
    | _ => .error .ambiguous
  | _ => .error .ambiguous

/-- The fallback thunk is deliberately not evaluated when complete matching
found any candidates, including an ambiguous complete set. -/
def rankCompleteFirst (complete : List BranchCandidate)
    (fallback : Unit → List BranchCandidate) : Except Failure BranchCandidate :=
  if complete.isEmpty then rankCandidates (fallback ()) else rankCandidates complete

/-- Error precedence is independent of input entry order. Validate every
supplied entry before this reduction: a nested ambiguity cannot conceal an
invalid alias, and an invalid value cannot conceal an explicit input cycle. -/
def failurePriority : Failure → Nat
  | .malformedDeclaration => 0
  | .cyclic => 1
  | .unsupported => 2
  | .invalid => 3
  | .ambiguous => 4

def combineChecked : List (Except Failure α) → Except Failure (List α)
  | [] => .ok []
  | head :: tail =>
    match head, combineChecked tail with
    | .ok value, .ok values => .ok (value :: values)
    | .error left, .error right =>
      .error (if failurePriority left ≤ failurePriority right then left else right)
    | .error failure, .ok _ | .ok _, .error failure => .error failure

def objectEntries : Input → Option (List (String × Input))
  | .host _ payload => objectEntries payload
  | .object fields => some fields
  | .map entries => entries.mapM (fun entry => match entry.1 with
      | .string name => some (name, entry.2)
      | _ => none)
  | _ => none

def mapEntries : Input → Option (List (Scalar × Input))
  | .host _ payload => mapEntries payload
  | .map entries => some entries
  | .object fields => some (fields.map (fun field => (.string field.1, field.2)))
  | _ => none

/-- Free built-in data retains scalar kinds and typed nil markers. The depth
argument is supplied from the actual finite input size, never a fixed limit. -/
def resolveRawAt (keys : KeyCodec) : Nat → Input → Except Failure Value
  | 0, _ => .error .malformedDeclaration
  | depth + 1, input =>
    match input with
    | .host identity payload => do
      let value ← resolveRawAt keys depth payload
      return .host identity value
    | .absent => .error .invalid
    | .null => .ok .null
    | .nilBytes => .ok .nilBytes
    | .nilArray => .ok .nilArray
    | .nilMap => .ok .nilMap
    | .scalar scalar => .ok (.scalar scalar)
    | .cycle _ => .error .cyclic
    | .opaque _ | .selected _ _ _ => .error .unsupported
    | .array items => do
      let values ← combineChecked (items.map (resolveRawAt keys depth))
      return .array values
    | .object fields => do
      if !(fields.map Prod.fst).Nodup then throw .invalid
      let values ← combineChecked (fields.map fun entry => do
        let value ← resolveRawAt keys depth entry.2
        return (entry.1, value))
      return .object [] values
    | .map entries => do
      let _ ← nameKeys keys (entries.map Prod.fst)
      if !(decide (entries.Pairwise (fun left right => scalarEqual left.1 right.1 = false))) then
        throw .invalid
      let values ← combineChecked (entries.map fun entry => do
        let value ← resolveRawAt keys depth entry.2
        return (entry.1, value))
      return .map values

def maximumExpansionRank (declarations : Declarations) : Nat :=
  declarations.foldl (fun largest declaration => max largest declaration.expansionRank) 0

/-- Preserve the existing preference rule: aliases and occurrence wrappers
expose their object shape, but a non-object alternative is preferred for object
input. In particular Any/map/nested-union alternatives are not demoted. -/
def prefersObjectInput (declarations : Declarations) : Nat → Identity → Input → Bool
  | 0, _, _ => false
  | rank + 1, identity, input =>
    match objectEntries input with
    | none => false
    | some entries =>
      match findDeclaration declarations identity with
      | none => false
      | some declaration =>
        match declaration.contract with
        | .alias child | .nullable child | .nonNull child =>
          prefersObjectInput declarations rank child input
        | .object members _ => members.any (fun member =>
            entries.any (fun entry =>
              entry.1 == member.sourceName || entry.1 == member.wireAlias))
        | _ => true

def enumValueAllowed (enumeration : Option (List Value)) (value : Value) : Bool :=
  enumeration.all (fun values => values.any (fun member =>
    valueEqualAt (valueDepth value + valueDepth member + 1) true value member))

def hasSuppliedValue : Input → Bool
  | .absent => false
  | _ => true

def selectedMemberInput (member : Member) (entries : List (String × Input)) : Option Input :=
  match entries.find? (fun entry => entry.1 == member.wireAlias && hasSuppliedValue entry.2) with
  | some entry => some entry.2
  | none => (entries.find? (fun entry =>
      entry.1 == member.sourceName && hasSuppliedValue entry.2)).map Prod.snd

def wrapBranch (occurrence : Identity) (candidate : BranchCandidate) : ResolveResult := do
  let resolution ← candidate.result
  return { resolution with value := .union occurrence candidate.branch resolution.value }

def objectMemberResult (completeOnly : Bool) (resolveChild : Identity → Input → ResolveResult)
    (entries : List (String × Input)) (member : Member) :
    Except Failure (Identity × Resolution) := do
  let aliases := entries.filter (fun entry => hasSuppliedValue entry.2 &&
    (entry.1 == member.sourceName || entry.1 == member.wireAlias))
  -- This validation is intentionally before precedence, including the losing alias.
  let _ ← combineChecked (aliases.map (fun entry => resolveChild member.child entry.2))
  match selectedMemberInput member entries with
  | none =>
    if completeOnly && member.required then throw .invalid
    return (member.identity, {
      value := .absent, missing := if member.required then [[member.identity]] else [] })
  | some input =>
    let resolution ← resolveChild member.child input
    return (member.identity, { resolution with
      missing := resolution.missing.map (member.identity :: ·) })

/-- Evaluate one declared source body using explicit same-input and child
recursors. This function owns body-local early returns. The enclosing resolver
applies the declaration enum only after this call returns, for every shape. -/
def resolveBody (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (depth rank : Nat) (completeOnly : Bool) (contract : Contract)
    (input : Input) (same descend : Bool → Identity → Input → ResolveResult) : ResolveResult :=
  match stripHostInput input with
  | .cycle _ => .error .cyclic
  | .opaque _ => .error .unsupported
  | _ =>
    match contract with
    | .custom _ => .error .unsupported
    | .any => do
      let value ← resolveRawAt keys (depth + 1) input
      return { value := .any value, missing := [] }
    | .alias child =>
      same completeOnly child input
    | .nullable child =>
      match stripHostInput input with
      | .null => .ok { value := .null, missing := [] }
      | _ => same completeOnly child input
    | .nonNull child =>
      match stripHostInput input with
      | .null => .error .invalid
      | _ => same completeOnly child input
    | .scalar kind rules =>
      let scalar := match stripHostInput input with
        | .scalar value => coerceScalar kind value
        | .nilBytes => if kind == .bytes then some (.bytes []) else none
        | _ => none
      match scalar with
      | some value =>
        if scalarAllowed checks rules value then .ok { value := .scalar value, missing := [] }
        else .error .invalid
      | none => .error .invalid
    | .array child bounds =>
      match stripHostInput input with
      | .nilArray =>
        if lengthAllowed bounds 0 then .ok { value := .nilArray, missing := [] }
        else .error .invalid
      | .array items => do
        if !lengthAllowed bounds items.length then throw .invalid
        let values ← combineChecked (items.map
          (descend completeOnly child))
        return {
          value := .array (values.map Resolution.value)
          missing := values.flatMap Resolution.missing }
      | _ => .error .invalid
    | .map kind rules child bounds =>
      match stripHostInput input with
      | .nilMap =>
        if lengthAllowed bounds 0 then .ok { value := .nilMap, missing := [] }
        else .error .invalid
      | _ => do
        let entries ← match mapEntries input with
          | some entries => .ok entries
          | none => .error .invalid
        if !lengthAllowed bounds entries.length then throw .invalid
        let normalized ← combineChecked (entries.map fun entry => do
          let key ← match kind with
            | .builtin => .ok entry.1
            | .scalar expected => match coerceScalar expected entry.1 with
              | some key => .ok key
              | none => .error .invalid
          if !mapKeyCompatible kind key || !scalarAllowed checks rules key then throw .invalid
          return (key, entry.2))
        let _ ← nameKeys keys (normalized.map Prod.fst)
        if !(decide (normalized.Pairwise (fun left right => scalarEqual left.1 right.1 = false))) then
          throw .invalid
        let values ← combineChecked (normalized.map fun entry => do
          let value ← descend completeOnly child entry.2
          return (entry.1, value))
        return {
          value := .map (values.map (fun entry => (entry.1, entry.2.value)))
          missing := values.flatMap (fun entry => entry.2.missing) }
    | .object members isOpen => do
      let entries ← match objectEntries input with
        | some entries => .ok entries
        | none => .error .invalid
      if !(entries.map Prod.fst).Nodup then throw .invalid
      let additional := entries.filter (fun entry => !members.any (fun member =>
        entry.1 == member.sourceName || entry.1 == member.wireAlias))
      if !isOpen && !additional.isEmpty then throw .invalid
      let fields := members.map (objectMemberResult completeOnly
        (descend completeOnly) entries)
      let extras := additional.map (fun entry => do
        let value ← resolveRawAt keys depth entry.2
        return (entry.1, value))
      -- Evaluate both groups before reducing errors so unknown data cannot
      -- hide an invalid known field or a known-field ambiguity.
      let _ ← combineChecked ((fields.map (Except.map (fun _ => ()))) ++
        (extras.map (Except.map (fun _ => ()))))
      let fieldValues ← combineChecked fields
      let extraValues ← combineChecked extras
      return {
        value := .object (fieldValues.map (fun entry => (entry.1, entry.2.value))) extraValues
        missing := fieldValues.flatMap (fun entry => entry.2.missing) }
    | .union occurrence alternatives =>
      match stripHostInput input with
      | .selected actual branch payload =>
        if actual != occurrence then .error .invalid
        else match alternatives.find? (fun alternative => alternative.identity == branch) with
          | none => .error .invalid
          | some alternative => wrapBranch occurrence {
            branch := branch, preferred := false,
            result := descend completeOnly alternative.child payload }
      | _ => do
        let complete := (alternatives.map fun alternative => {
          branch := alternative.identity,
          preferred := prefersObjectInput declarations (rank + 1) alternative.child input,
          result := same true
            alternative.child input : BranchCandidate }).filter BranchCandidate.complete
        let selected ← rankCompleteFirst complete (fun _ =>
          if completeOnly then [] else alternatives.map fun alternative => {
            branch := alternative.identity,
            preferred := prefersObjectInput declarations (rank + 1) alternative.child input,
            result := same false
              alternative.child input })
        wrapBranch occurrence selected

/-- The outer counter descends with input children; the inner counter bounds
same-input graph expansion and resets on child descent. Both are derived from
the finite input and checked declaration graph by the public resolver. -/
def resolveAt (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) : Nat → Nat → Bool → Identity → Input → ResolveResult
  | 0, _, _, _, _ => .error .malformedDeclaration
  | _, 0, _, _, _ => .error .malformedDeclaration
  | depth + 1, rank + 1, completeOnly, identity, input => do
    let declaration ← match findDeclaration declarations identity with
      | some declaration => .ok declaration
      | none => .error .malformedDeclaration
    let resolution ← resolveBody declarations checks keys depth rank completeOnly declaration.contract input
      (fun mode child value => resolveAt declarations checks keys (depth + 1) rank mode child value)
      (fun mode child value => resolveAt declarations checks keys depth
        (maximumExpansionRank declarations + 1) mode child value)
    if enumValueAllowed declaration.enumeration resolution.value then pure resolution
    else throw .invalid
termination_by depth rank _ _ _ => (depth, rank)

/-- Computable input depth. Child descent decreases this measure regardless of
the declaration identity, admitting arbitrarily deep finite recursive values. -/
def inputDepth : Input → Nat
  | .array items => 1 + (items.map inputDepth).foldl max 0
  | .object fields => 1 + (fields.map (fun field => inputDepth field.2)).foldl max 0
  | .map entries => 1 + (entries.map (fun entry => inputDepth entry.2)).foldl max 0
  | .selected _ _ payload | .host _ payload => 1 + inputDepth payload
  | _ => 1
termination_by input => sizeOf input
decreasing_by
  all_goals simp_wf
  all_goals first
    | exact pairChildSize_lt ‹_ ∈ _›
    | exact Nat.lt_trans (List.sizeOf_lt_of_mem ‹_ ∈ _›) (by omega)
    | omega

/-- Target-independent entry point. No schema, selected body, rendering context,
or target branch list is available to affect semantic source selection. -/
def resolve (declarations : Declarations) (checks : ExternalScalarChecks)
    (keys : KeyCodec) (role : Role) (effective : Identity) (input : Input) : ResolveResult :=
  if !validateDeclarations declarations then .error .malformedDeclaration
  else resolveAt declarations checks keys (inputDepth input + 1)
    (maximumExpansionRank declarations + 1)
    (role == .enumMember || role == .defaultValue) effective input

end ValueContract.Candidate
