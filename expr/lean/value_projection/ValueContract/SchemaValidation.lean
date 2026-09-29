import ValueContract.SchemaSemantics
import ValueContract.GraphValidation

namespace ValueContract.Candidate

/-- One target node's schema computation. Keeping this in its own function
prevents branch-local return from bypassing the uniform whole-node enum gate. -/
def schemaBody (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (recur : Identity → Wire → Except Failure Bool)
    (declaration : TargetDeclaration) (wire : Wire) : Except Failure Bool :=
  match declaration.target, wire with
  | .scalar encoding kind rules, wire =>
    .ok (scalarSchema codecs checks encoding kind rules wire)
  | .nullable _, .null => .ok true
  | .nullable child, wire | .alias child, wire | .select _ child, wire =>
    recur child wire
  | .nonNull child, wire =>
    if wireIsNull wire then .ok false else recur child wire
  | .array child bounds, .array items => do
    let outcomes ← items.mapM (recur child)
    return lengthAllowed bounds items.length && outcomes.all id
  | .map _ _ child bounds, .object items => do
    let outcomes ← items.mapM (fun entry => recur child entry.2)
    return decide (items.map Prod.fst).Nodup &&
      lengthAllowed bounds items.length && outcomes.all id
  | .object members _, .object items => do
    let outcomes ← members.mapM fun member => match wireMember items member.wireName with
      | none => .ok (!member.required)
      | some value => recur member.child value
    return decide (items.map Prod.fst).Nodup && outcomes.all id &&
      (declaration.schemaAllowsUnknown ||
        items.all (fun entry => members.any (·.wireName == entry.1)))
  | .union _ .untagged alternatives, wire => do
    let outcomes ← alternatives.mapM (fun alternative =>
      recur alternative.child wire)
    return (outcomes.filter id).length == 1
  | .union _ (.tagged tagKey valueKey) alternatives, .object items =>
    match wireMember items tagKey, wireMember items valueKey with
    | some (.text tag), some value =>
      match alternatives.find? (fun alternative => alternative.wireName == tag) with
      | some alternative => do
        let result ← recur alternative.child value
        return result && decide (items.map Prod.fst).Nodup &&
          (declaration.schemaAllowsUnknown ||
            items.all (fun entry => entry.1 == tagKey || entry.1 == valueKey))
      | none => .ok false
    | _, _ => .ok false
  | .union _ .protobuf alternatives, .oneof tag value =>
    match alternatives.find? (fun alternative => alternative.wireName == tag) with
    | some alternative => recur alternative.child value
    | none => .ok false
  | .any, wire => .ok (jsonValid codecs.numbers wire)
  | .custom _, _ => .error .unsupported
  | _, _ => .ok false

/-- Exhaustion is an evaluation error, never a rejected union alternative. -/
def schemaFuel (codecs : ScalarCodecs) (checks : ExternalScalarChecks) (targets : Targets) :
    Nat → Identity → Wire → Except Failure Bool
  | 0, _, _ => .error .malformedDeclaration
  | fuel + 1, identity, wire => do
    let some declaration := findTarget targets identity | .error .malformedDeclaration
    let accepted ← schemaBody codecs checks (schemaFuel codecs checks targets fuel) declaration wire
    return accepted && schemaEnumAllowed codecs.numbers declaration.schemaEnumeration wire

/-- Upper rank is not a fixed semantic domain bound; each validated plan carries
its own finite expansion ranks, independently checked by GraphValidation. -/
def maximumTargetRank (targets : Targets) : Nat :=
  targets.foldl (fun rank declaration => max rank declaration.expansionRank) 0

def wireBudget (targets : Targets) (wire : Wire) : Nat :=
  (wireHeight wire + 1) * (maximumTargetRank targets + 1)

def schema (codecs : ScalarCodecs) (checks : ExternalScalarChecks) (targets : Targets)
    (identity : Identity) (wire : Wire) : Except Failure Bool :=
  if validateTargets targets then schemaFuel codecs checks targets (wireBudget targets wire) identity wire
  else .error .malformedDeclaration

end ValueContract.Candidate
