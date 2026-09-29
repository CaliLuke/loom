import ValueContract.DecoderSemantics
import ValueContract.SchemaValidation

namespace ValueContract.Candidate

/-- One target node's runtime computation. Whole-node enum filtering happens
only after this helper returns, for every target constructor. -/
def decodeBody (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (recurse : Identity → Wire → Except Failure (Option Value))
    (declaration : TargetDeclaration) (wire : Wire) : Except Failure (Option Value) :=
  match declaration.target, wire with
  | .scalar encoding kind rules, wire =>
    .ok (((decodeScalar codecs encoding kind rules.numericFormat rules.integerFormat wire).filter (scalarAllowed checks rules)).map Value.scalar)
  | .nullable _, .null => .ok (some .null)
  | .nullable child, wire | .alias child, wire | .select _ child, wire =>
    recurse child wire
  | .nonNull child, wire =>
    if wireIsNull wire then .ok none else recurse child wire
  | .array child bounds, .array items => do
    let outcomes ← items.mapM (recurse child)
    return if lengthAllowed bounds items.length then (collectDecoded outcomes).map Value.array else none
  | .map kind rules child bounds, .object items => do
    let outcomes ← items.mapM (fun entry => recurse child entry.2)
    let keys := items.map (fun entry => decodeKey codecs.numbers kind rules.numericFormat rules.integerFormat entry.1)
    return if (items.map Prod.fst).Nodup then mapDecoded checks rules bounds keys outcomes else none
  | .object members preserveAdditional, .object items => do
    let outcomes ← members.mapM fun member => match wireMember items member.wireName with
      | none => .ok (missingField member)
      | some value => recurse member.child value
    return objectDecoded declaration.decoderRejectsUnknown preserveAdditional
      codecs.numbers members items outcomes
  | .union occurrence .untagged alternatives, wire => do
    let outcomes ← alternatives.mapM (fun alternative =>
      recurse alternative.child wire)
    return unionDecoded occurrence alternatives outcomes
  | .union occurrence (.tagged tagKey valueKey) alternatives, .object items =>
    match wireMember items tagKey, wireMember items valueKey with
    | some (.text tag), some value =>
      match alternatives.find? (fun alternative => alternative.wireName == tag) with
      | some alternative => do
        let outcome ← recurse alternative.child value
        return if (items.map Prod.fst).Nodup && (!declaration.decoderRejectsUnknown ||
            items.all (fun entry => entry.1 == tagKey || entry.1 == valueKey)) then
          outcome.map (Value.union occurrence alternative.identity) else none
      | none => .ok none
    | _, _ => .ok none
  | .union occurrence .protobuf alternatives, .oneof tag value =>
    match alternatives.find? (fun alternative => alternative.wireName == tag) with
    | some alternative => do
      let outcome ← recurse alternative.child value
      return outcome.map (Value.union occurrence alternative.identity)
    | none => .ok none
  | .any, wire => .ok (if jsonValid codecs.numbers wire then some (.jsonSnapshot wire) else none)
  | .custom _, _ => .error .unsupported
  | _, _ => .ok none

/-- Runtime matching is independent of schema acceptance. Every alternative is
visited before untagged uniqueness is decided, with fatal errors propagated. -/
def decodeFuel (codecs : ScalarCodecs) (checks : ExternalScalarChecks) (targets : Targets) :
    Nat → Identity → Wire → Except Failure (Option Value)
  | 0, _, _ => .error .malformedDeclaration
  | fuel + 1, identity, wire => do
    let some declaration := findTarget targets identity | .error .malformedDeclaration
    let result ← decodeBody codecs checks (decodeFuel codecs checks targets fuel) declaration wire
    return result.filter (targetEnumAllowed codecs.numbers declaration.enumeration)

def decode (codecs : ScalarCodecs) (checks : ExternalScalarChecks) (targets : Targets)
    (identity : Identity) (wire : Wire) : Except Failure (Option Value) :=
  if validateTargets targets then decodeFuel codecs checks targets (wireBudget targets wire) identity wire
  else .error .malformedDeclaration

end ValueContract.Candidate
