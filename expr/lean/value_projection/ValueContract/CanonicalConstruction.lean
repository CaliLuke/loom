import ValueContract.ProjectionObservation

namespace ValueContract.Candidate

def requireWire : Option Wire → Except BuildFailure Wire
  | none => .error .incomplete
  | some wire => .ok wire

def constructExtra (codecs : ScalarCodecs) (value : Value) : Except BuildFailure Wire :=
  match value with
  | .jsonSnapshot wire => if jsonValid codecs.numbers wire then .ok wire else .error .unrepresentable
  | _ => materializeBuilt codecs value

/-- Canonical emission consumes an already observed value. A discarded child is
never recursively constructed or checked for required wire fields. -/
def constructFuel (codecs : ScalarCodecs) (targets : Targets) :
    Nat → Identity → Value → Except BuildFailure (Option Wire)
  | 0, _, _ => .error .invalidPlan
  | fuel + 1, identity, value => do
    let some declaration := findTarget targets identity | .error .invalidPlan
    match declaration.target, value with
    | _, .absent => return none
    | .scalar encoding kind _, .scalar scalar =>
      if scalar.kind = kind then return some (encodeScalar codecs encoding scalar)
      else .error .unrepresentable
    | .nullable _, .null => return some .null
    | .nullable child, value | .alias child, value | .select _ child, value =>
      constructFuel codecs targets fuel child value
    | .nonNull child, value =>
      if valueIsNull value then .error .unrepresentable
      else constructFuel codecs targets fuel child value
    | .array child _, .array values => do
      let wires ← values.mapM (fun value => do
        requireWire (← constructFuel codecs targets fuel child value))
      return some (.array wires)
    | .map _ _ child _, .map entries => do
      let names ← match nameKeys codecs.numbers (entries.map Prod.fst) with
        | .ok names => .ok names
        | .error _ => .error .unrepresentable
      let wires ← entries.mapM (fun entry => do
        requireWire (← constructFuel codecs targets fuel child entry.2))
      return some (.object (orderedMembers (names.zip wires)))
    | .object members preserveAdditional, .object fields extra => do
      let fragments ← members.mapM fun member => do
        let value := memberValue fields member.identity
        let wire ← if valueAbsent value || implicitDefaultOmitted member.presence value then .ok none
          else constructFuel codecs targets fuel member.child value
        if member.required && wire.isNone then .error .incomplete
        else return (member.wireName, wire)
      let additional ← if preserveAdditional then extra.mapM (fun entry => do
          let wire ← constructExtra codecs entry.2
          return (entry.1, wire)) else .ok []
      let combined := retainedMembers fragments ++ additional
      if !(decide (combined.map Prod.fst).Nodup) then .error .unrepresentable
      else return some (.object (orderedMembers combined))
    | .union occurrence style alternatives, .union actual branch payload => do
      if actual != occurrence then .error .unrepresentable
      else
        let some alternative := alternatives.find? (·.identity == branch) | .error .unrepresentable
        let wire ← requireWire (← constructFuel codecs targets fuel alternative.child payload)
        match style with
        | .untagged => return some wire
        | .tagged tagKey valueKey => return some (.object
            (orderedMembers [(tagKey, .text alternative.wireName), (valueKey, wire)]))
        | .protobuf => return some (.oneof alternative.wireName wire)
    | .any, .jsonSnapshot wire =>
      if jsonValid codecs.numbers wire then return some wire else .error .unrepresentable
    | .custom _, _ => .error .unsupported
    | _, _ => .error .unrepresentable

def construct (codecs : ScalarCodecs) (targets : Targets) (identity : Identity)
    (observed : Value) : Except BuildFailure (Option Wire) :=
  if validateTargets targets then constructFuel codecs targets (valueBudget targets observed) identity observed
  else .error .invalidPlan

end ValueContract.Candidate
