import ValueContract.Decoding

/- Rejected candidate: canonical child construction ran before parent omission.
Retained for the checked progress counterexample, never called by project. -/

namespace ValueContract.Candidate.LegacyBuild

inductive BuildFailure where
  | incomplete | unsupported | invalidPlan | unrepresentable
  deriving DecidableEq, Repr

abbrev BuiltValue := Value × Option Wire

def valueAbsent : Value → Bool
  | .absent => true
  | _ => false

abbrev valueNull := valueIsNull

def applyPresence (presence : Presence) (built : BuiltValue) : BuiltValue :=
  match presence with
  | .explicit => built
  | .implicitDefault value =>
    if valueAbsent built.1 then (.scalar value, none)
    else if implicitDefaultOmitted presence built.1 then (built.1, none) else built
  | .omitEmpty =>
    if built.2.any snapshotEmpty then (.absent, none) else built

def builtWire (built : BuiltValue) : Except BuildFailure Wire :=
  match built.2 with
  | none => .error .incomplete
  | some wire => .ok wire

def materializeBuilt (codecs : ScalarCodecs) (value : Value) : Except BuildFailure Wire :=
  match materialize codecs value with
  | .ok wire => .ok wire
  | .error .unsupported => .error .unsupported
  | .error _ => .error .unrepresentable

/-- Executable observation plus canonical construction. Validation is separate:
this function cannot certify schema acceptance or runtime union uniqueness. -/
def buildFuel (codecs : ScalarCodecs) (targets : Targets) (role : Role) :
    Nat → Identity → Value → Except BuildFailure BuiltValue
  | 0, _, _ => .error .invalidPlan
  | fuel + 1, identity, value => do
    let some declaration := findTarget targets identity | .error .invalidPlan
    match declaration.target, value with
    | _, .absent => return (.absent, none)
    | .scalar encoding kind _, .scalar scalar =>
      if scalar.kind = kind then return (value, some (encodeScalar codecs encoding scalar))
      else .error .unrepresentable
    | .nullable _, .null => return (.null, some .null)
    | .nullable child, value | .alias child, value => buildFuel codecs targets role fuel child value
    | .nonNull child, value =>
      if valueNull value then .error .unrepresentable
      else buildFuel codecs targets role fuel child value
    | .array _ _, .nilArray =>
      if contractRole role then return (.array [], some (.array [])) else .error .incomplete
    | .map _ _ _ _, .nilMap =>
      if contractRole role then return (.map [], some (.object [])) else .error .incomplete
    | .array child _, .array values => do
      let built ← values.mapM (buildFuel codecs targets role fuel child)
      let wires ← built.mapM builtWire
      return (.array (built.map Prod.fst), some (.array wires))
    | .map kind _ child _, .map entries => do
      let names ← match nameKeys codecs.numbers (entries.map Prod.fst) with
        | .ok names => .ok names
        | .error _ => .error .unrepresentable
      let keys ← entries.mapM (fun entry => match observeKey codecs.numbers kind entry.1 with
        | some key => .ok key
        | none => .error .unrepresentable)
      let built ← entries.mapM (fun entry => buildFuel codecs targets role fuel child entry.2)
      let wires ← built.mapM builtWire
      return (.map (keys.zip (built.map Prod.fst)), some (.object (orderedMembers (names.zip wires))))
    | .object members preserveAdditional, .object fields extra => do
      let built ← members.mapM fun member => do
        let child ← buildFuel codecs targets role fuel member.child (memberValue fields member.identity)
        let represented := applyPresence member.presence child
        if member.required && represented.2.isNone then .error .incomplete
        else return represented
      let extraWires ← if preserveAdditional then extra.mapM (fun entry => do
          let wire ← materializeBuilt codecs entry.2
          return (entry.1, wire)) else .ok []
      let fragments := (members.zip built).map (fun pair => (pair.1.wireName, pair.2.2))
      let combined := retainedMembers fragments ++ extraWires
      if !(decide (combined.map Prod.fst).Nodup) then .error .unrepresentable
      else return (.object ((members.map TargetMember.identity).zip (built.map Prod.fst))
        (extraWires.map (fun entry => (entry.1, .jsonSnapshot entry.2))),
        some (.object (orderedMembers combined)))
    | .union occurrence style alternatives, .union actual branch payload => do
      if actual != occurrence then .error .unrepresentable
      else
        let some alternative := alternatives.find? (·.identity == branch) | .error .unrepresentable
        let built ← buildFuel codecs targets role fuel alternative.child payload
        let wire ← builtWire built
        let observed := Value.union occurrence branch built.1
        match style with
        | .untagged => return (observed, some wire)
        | .tagged tagKey valueKey => return (observed,
            some (.object (orderedMembers [(tagKey, .text alternative.wireName), (valueKey, wire)])))
        | .protobuf => return (observed, some (.oneof alternative.wireName wire))
    | .select field child, .object fields _ =>
      buildFuel codecs targets role fuel child (memberValue fields field)
    | .any, .any payload => do
      let wire ← materializeBuilt codecs payload
      return (.jsonSnapshot wire, some wire)
    | .custom _, _ => .error .unsupported
    | _, _ => .error .unrepresentable

def valueBudget (targets : Targets) (value : Value) : Nat :=
  (valueDepth value + 1) * (maximumTargetRank targets + 1)

def build (codecs : ScalarCodecs) (targets : Targets) (role : Role)
    (identity : Identity) (value : Value) : Except BuildFailure BuiltValue :=
  if validateTargets targets then buildFuel codecs targets role (valueBudget targets value) identity value
  else .error .invalidPlan

end ValueContract.Candidate.LegacyBuild
