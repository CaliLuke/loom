import ValueContract.Decoding

namespace ValueContract.Candidate

inductive BuildFailure where
  | incomplete | unsupported | invalidPlan | unrepresentable
  deriving DecidableEq, Repr

abbrev BuiltValue := Value × Option Wire

def valueAbsent : Value → Bool
  | .absent => true
  | _ => false

def valueBudget (targets : Targets) (value : Value) : Nat :=
  (valueDepth value + 1) * (maximumTargetRank targets + 1)

def materializeBuilt (codecs : ScalarCodecs) (value : Value) : Except BuildFailure Wire :=
  match materialize codecs value with
  | .ok wire => .ok wire
  | .error .unsupported => .error .unsupported
  | .error _ => .error .unrepresentable

/-- Decides field-local emptiness of an already observed value. Exhaustion is an
error, never evidence that a field is nonempty. Requiredness belongs to retained
canonical construction, not to this visibility/presence decision. -/
def emptyFuel (targets : Targets) : Nat → Identity → Value → Except BuildFailure Bool
  | 0, _, _ => .error .invalidPlan
  | fuel + 1, identity, value => do
    let some declaration := findTarget targets identity | .error .invalidPlan
    match declaration.target, value with
    | _, .null | _, .nilArray | _, .nilMap | _, .nilBytes => return false
    | .scalar _ .string _, .scalar (.string value) => return value == ""
    | .scalar _ .bytes _, .scalar (.bytes value) => return value.isEmpty
    | .array _ _, .array values => return values.isEmpty
    | .map _ _ _ _, .map entries => return entries.isEmpty
    | .any, .jsonSnapshot wire => return snapshotEmpty wire
    | .nullable child, value | .nonNull child, value | .alias child, value |
        .select _ child, value => emptyFuel targets fuel child value
    | .object members _, .object fields extra => return extra.isEmpty && members.all (fun member =>
        valueAbsent (memberValue fields member.identity) ||
        implicitDefaultOmitted member.presence (memberValue fields member.identity))
    | .union occurrence .untagged alternatives, .union actual branch payload =>
      if occurrence != actual then return false
      let some alternative := alternatives.find? (·.identity == branch) | return false
      emptyFuel targets fuel alternative.child payload
    | _, _ => return false

def emptyObserved (targets : Targets) (identity : Identity) (value : Value) : Except BuildFailure Bool :=
  if validateTargets targets then emptyFuel targets (valueBudget targets value) identity value
  else .error .invalidPlan

def observePresence (targets : Targets) (member : TargetMember) (value : Value) : Except BuildFailure Value :=
  match member.presence with
  | .explicit => .ok value
  | .implicitDefault scalar => if valueAbsent value then .ok (.scalar scalar) else .ok value
  | .omitEmpty => do
    let empty ← emptyObserved targets member.child value
    return if empty then .absent else value

/-- Observation performs field selection and presence before canonical emission.
It preserves partial child values until their parent's omission policy runs. -/
def observeFuel (codecs : ScalarCodecs) (targets : Targets) (role : Role) :
    Nat → Identity → Value → Except BuildFailure Value
  | 0, _, _ => .error .invalidPlan
  | fuel + 1, identity, value => do
    let some declaration := findTarget targets identity | .error .invalidPlan
    match declaration.target, value with
    | _, .absent => return .absent
    | .scalar _ kind _, .scalar scalar =>
      if scalar.kind = kind then return value else .error .unrepresentable
    | .nullable _, .null => return .null
    | .nullable child, value | .alias child, value => observeFuel codecs targets role fuel child value
    | .nonNull child, value =>
      if valueIsNull value then .error .unrepresentable
      else observeFuel codecs targets role fuel child value
    | .array _ _, .nilArray =>
      if contractRole role then return .array [] else .error .incomplete
    | .map _ _ _ _, .nilMap =>
      if contractRole role then return .map [] else .error .incomplete
    | .array child _, .array values => do
      return .array (← values.mapM (observeFuel codecs targets role fuel child))
    | .map kind _ child _, .map entries => do
      let observed ← entries.mapM fun entry => do
        let some key := observeKey codecs.numbers kind entry.1 | .error .unrepresentable
        let value ← observeFuel codecs targets role fuel child entry.2
        return (key, value)
      return .map observed
    | .object members preserveAdditional, .object fields extra => do
      let observed ← members.mapM fun member => do
        let value ← observeFuel codecs targets role fuel member.child (memberValue fields member.identity)
        let visible ← observePresence targets member value
        return (member.identity, visible)
      let additional ← if preserveAdditional then extra.mapM (fun entry => do
          let wire ← materializeBuilt codecs entry.2
          return (entry.1, .jsonSnapshot wire)) else .ok []
      return .object observed additional
    | .union occurrence _ alternatives, .union actual branch payload => do
      if actual != occurrence then .error .unrepresentable
      else
        let some alternative := alternatives.find? (·.identity == branch) | .error .unrepresentable
        let observed ← observeFuel codecs targets role fuel alternative.child payload
        return .union occurrence branch observed
    | .select field child, .object fields _ =>
      observeFuel codecs targets role fuel child (memberValue fields field)
    | .any, .any payload => return .jsonSnapshot (← materializeBuilt codecs payload)
    | .custom _, _ => .error .unsupported
    | _, _ => .error .unrepresentable

def observeValue (codecs : ScalarCodecs) (targets : Targets) (role : Role)
    (identity : Identity) (value : Value) : Except BuildFailure Value :=
  if validateTargets targets then observeFuel codecs targets role (valueBudget targets value) identity value
  else .error .invalidPlan

end ValueContract.Candidate
