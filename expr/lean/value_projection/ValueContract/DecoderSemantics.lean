import ValueContract.SchemaSemantics
import ValueContract.TargetEquality
import ValueContract.KeyDecoding

namespace ValueContract.Candidate

/-- List collection retains all outcomes; no rejected branch is removed before
uniqueness is decided, and absence of an outcome is not a negative result. -/
def collectDecoded (values : List (Option Value)) : Option (List Value) := values.mapM id

def missingField (member : TargetMember) : Option Value :=
  if member.required then none else match member.presence with
    | .implicitDefault value => some (.scalar value)
    | _ => some .absent

def unionDecoded (occurrence : Identity) (alternatives : List TargetAlternative)
    (outcomes : List (Option Value)) : Option Value :=
  match (alternatives.zip outcomes).filterMap
      (fun pair => pair.2.map (fun payload => Value.union occurrence pair.1.identity payload)) with
  | [value] => some value
  | _ => none

/-- Key decoder acceptance is independent of canonical encoder membership. -/
def KeyOutcome (numbers : NumericCodec) (kind : MapKeyKind) (name : String)
    (result : Option Scalar) : Prop := match result with
  | some key => KeyDecodes numbers kind name key
  | none => ∀ key, ¬ KeyDecodes numbers kind name key

def mapDecoded (checks : ExternalScalarChecks) (rules : ScalarRules) (bounds : LengthBounds)
    (keys : List (Option Scalar)) (values : List (Option Value)) : Option Value := do
  let keys ← keys.mapM (fun key => key.filter (scalarAllowed checks rules))
  let values ← collectDecoded values
  if lengthAllowed bounds values.length &&
      keys.all (fun key => (keys.filter (scalarEqual key)).length == 1) then
    some (.map (keys.zip values))
  else none

def objectDecoded (rejectUnknown preserveAdditional : Bool) (numbers : NumericCodec)
    (members : List TargetMember) (items : List (String × Wire))
    (outcomes : List (Option Value)) : Option Value := do
  let values ← collectDecoded outcomes
  let extra := items.filter (fun entry => !(members.any (·.wireName == entry.1)))
  if !(decide (items.map Prod.fst).Nodup) || (rejectUnknown && !extra.isEmpty) then none
  else if preserveAdditional && !(extra.all (fun entry => jsonValid numbers entry.2)) then none
  else some (.object ((members.map TargetMember.identity).zip values)
    (if preserveAdditional then extra.map (fun entry => (entry.1, .jsonSnapshot entry.2)) else []))

mutual

/-- Independent actual decoder outcomes, including failed alternatives. The
schema is never consulted by this relation; byte and key aliases remain visible. -/
inductive RuntimeEvaluation (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) : Identity → Wire → Option Value → Prop where
  | rejected (declaration : TargetDeclaration) (member : declaration ∈ targets)
      (same : declaration.identity = identity)
      (body : RuntimeNode codecs checks targets declaration.decoderRejectsUnknown
        declaration.target wire none) :
      RuntimeEvaluation codecs checks targets identity wire none
  | accepted (declaration : TargetDeclaration) (member : declaration ∈ targets)
      (same : declaration.identity = identity)
      (body : RuntimeNode codecs checks targets declaration.decoderRejectsUnknown
        declaration.target wire (some value))
      (enumeration : TargetEnumAllows codecs.numbers declaration.enumeration value) :
      RuntimeEvaluation codecs checks targets identity wire (some value)
  | enumRejected (declaration : TargetDeclaration) (member : declaration ∈ targets)
      (same : declaration.identity = identity)
      (body : RuntimeNode codecs checks targets declaration.decoderRejectsUnknown
        declaration.target wire (some value))
      (enumeration : ¬ TargetEnumAllows codecs.numbers declaration.enumeration value) :
      RuntimeEvaluation codecs checks targets identity wire none

inductive RuntimeNode (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) : Bool → Target → Wire → Option Value → Prop where
  | scalar (decoded : ScalarDecodes codecs encoding kind wire value)
      (valid : scalarAllowed checks rules value = true) :
      RuntimeNode codecs checks targets rejectUnknown (.scalar encoding kind rules) wire
        (some (.scalar value))
  | scalarRejected (rejected : ∀ value, ScalarDecodes codecs encoding kind wire value →
      scalarAllowed checks rules value = false) :
      RuntimeNode codecs checks targets rejectUnknown (.scalar encoding kind rules) wire none
  | nullableNull : RuntimeNode codecs checks targets rejectUnknown (.nullable identity) .null (some .null)
  | nullableValue (nonnull : wireIsNull wire = false)
      (result : RuntimeEvaluation codecs checks targets identity wire value) :
      RuntimeNode codecs checks targets rejectUnknown (.nullable identity) wire value
  | alias (result : RuntimeEvaluation codecs checks targets identity wire value) :
      RuntimeNode codecs checks targets rejectUnknown (.alias identity) wire value
  | select (result : RuntimeEvaluation codecs checks targets identity wire value) :
      RuntimeNode codecs checks targets rejectUnknown (.select field identity) wire value
  | nonNullRejected : RuntimeNode codecs checks targets rejectUnknown (.nonNull identity) .null none
  | nonNullValue (nonnull : wireIsNull wire = false)
      (result : RuntimeEvaluation codecs checks targets identity wire value) :
      RuntimeNode codecs checks targets rejectUnknown (.nonNull identity) wire value
  | array (lengths : items.length = outcomes.length)
      (results : ∀ pair ∈ items.zip outcomes,
        RuntimeEvaluation codecs checks targets identity pair.1 pair.2) :
      RuntimeNode codecs checks targets rejectUnknown (.array identity bounds) (.array items)
        (if lengthAllowed bounds items.length then (collectDecoded outcomes).map Value.array else none)
  | map (keyLengths : items.length = keys.length) (valueLengths : items.length = outcomes.length)
      (keyResults : ∀ pair ∈ items.zip keys, KeyOutcome codecs.numbers kind pair.1.1 pair.2)
      (results : ∀ pair ∈ items.zip outcomes,
        RuntimeEvaluation codecs checks targets identity pair.1.2 pair.2) :
      RuntimeNode codecs checks targets rejectUnknown (.map kind rules identity bounds) (.object items)
        (if (items.map Prod.fst).Nodup then mapDecoded checks rules bounds keys outcomes else none)
  | object (lengths : members.length = outcomes.length)
      (missing : ∀ pair ∈ members.zip outcomes,
        wireMember items pair.1.wireName = none → pair.2 = missingField pair.1)
      (present : ∀ pair ∈ members.zip outcomes, ∀ value,
        wireMember items pair.1.wireName = some value →
          RuntimeEvaluation codecs checks targets pair.1.child value pair.2) :
      RuntimeNode codecs checks targets rejectUnknown (.object members preserveAdditional) (.object items)
        (objectDecoded rejectUnknown preserveAdditional codecs.numbers members items outcomes)
  | untagged (lengths : alternatives.length = outcomes.length)
      (results : ∀ pair ∈ alternatives.zip outcomes,
        RuntimeEvaluation codecs checks targets pair.1.child wire pair.2) :
      RuntimeNode codecs checks targets rejectUnknown (.union occurrence .untagged alternatives) wire
        (unionDecoded occurrence alternatives outcomes)
  | tagged (tag : wireMember items tagKey = some (.text name))
      (payload : wireMember items valueKey = some value)
      (chosen : alternatives.find? (fun alternative => alternative.wireName == name) = some alternative)
      (result : RuntimeEvaluation codecs checks targets alternative.child value outcome) :
      RuntimeNode codecs checks targets rejectUnknown (.union occurrence (.tagged tagKey valueKey) alternatives)
        (.object items) (if (items.map Prod.fst).Nodup &&
          (!rejectUnknown || items.all (fun entry => entry.1 == tagKey || entry.1 == valueKey)) then
            outcome.map (Value.union occurrence alternative.identity) else none)
  | taggedMalformed (malformed : ¬ ∃ name value alternative,
      wireMember items tagKey = some (.text name) ∧ wireMember items valueKey = some value ∧
      alternatives.find? (fun alternative => alternative.wireName == name) = some alternative) :
      RuntimeNode codecs checks targets rejectUnknown (.union occurrence (.tagged tagKey valueKey) alternatives)
        (.object items) none
  | protobuf (chosen : alternatives.find? (fun alternative => alternative.wireName == name) = some alternative)
      (result : RuntimeEvaluation codecs checks targets alternative.child value outcome) :
      RuntimeNode codecs checks targets rejectUnknown (.union occurrence .protobuf alternatives)
        (.oneof name value) (outcome.map (Value.union occurrence alternative.identity))
  | protobufMalformed (missing : alternatives.find? (fun alternative => alternative.wireName == name) = none) :
      RuntimeNode codecs checks targets rejectUnknown (.union occurrence .protobuf alternatives)
        (.oneof name value) none
  | anyValid (valid : JSONValid codecs.numbers wire) :
      RuntimeNode codecs checks targets rejectUnknown .any wire (some (.jsonSnapshot wire))
  | anyInvalid (invalid : ¬ JSONValid codecs.numbers wire) :
      RuntimeNode codecs checks targets rejectUnknown .any wire none
  | mismatch (mismatch : structuralMismatch target wire = true) :
      RuntimeNode codecs checks targets rejectUnknown target wire none

end

def RuntimeDecodes (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (identity : Identity) (wire : Wire) (value : Value) : Prop :=
  RuntimeEvaluation codecs checks targets identity wire (some value)

end ValueContract.Candidate
