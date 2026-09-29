import ValueContract.Canonical
import ValueContract.WireEquality

namespace ValueContract.Candidate

def wireMember (members : List (String × Wire)) (name : String) : Option Wire :=
  (members.find? (fun entry => entry.1 == name)).map Prod.snd

def wireIsNull : Wire → Bool
  | .null => true
  | _ => false

def schemaEnumAllowed (numbers : NumericCodec) (enumeration : Option (List Wire))
    (wire : Wire) : Bool :=
  enumeration.all (fun members => members.any (wireEqual numbers wire))

def structuralMismatch : Target → Wire → Bool
  | .scalar _ _ _, _ | .nullable _, _ | .nonNull _, _ | .alias _, _ |
      .select _ _, _ | .any, _ | .custom _, _ | .union _ .untagged _, _ => false
  | .array _ _, .array _ | .map _ _ _ _, .object _ | .object _ _, .object _ |
      .union _ (.tagged _ _) _, .object _ | .union _ .protobuf _, .oneof _ _ => false
  | _, _ => true

mutual

/-- Finite derivations include all rejection outcomes needed for uniqueness.
This is unbounded recursive semantics: object/collection child derivations may
be arbitrarily deep. Graph ranks constrain only nonconsuming edges. -/
inductive SchemaEvaluation (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) : Identity → Wire → Bool → Prop where
  | node (declaration : TargetDeclaration) (member : declaration ∈ targets)
      (same : declaration.identity = identity)
      (body : SchemaNode codecs checks targets
        declaration.schemaAllowsUnknown declaration.target wire accepted) :
      SchemaEvaluation codecs checks targets identity wire
        (accepted && schemaEnumAllowed codecs.numbers declaration.schemaEnumeration wire)

/-- One independent schema step. All branch outcomes are supplied explicitly;
absence of a derivation is never interpreted as a failed branch. -/
inductive SchemaNode (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) : Bool → Target → Wire → Bool → Prop where
  | scalar : SchemaNode codecs checks targets allowsUnknown (.scalar encoding kind rules)
      wire (scalarSchema codecs checks encoding kind rules wire)
  | nullableNull : SchemaNode codecs checks targets allowsUnknown (.nullable identity) .null true
  | nullableValue (nonnull : wireIsNull wire = false) (result : SchemaEvaluation codecs checks targets identity wire accepted) :
      SchemaNode codecs checks targets allowsUnknown (.nullable identity) wire accepted
  | alias (result : SchemaEvaluation codecs checks targets identity wire accepted) :
      SchemaNode codecs checks targets allowsUnknown (.alias identity) wire accepted
  | select (result : SchemaEvaluation codecs checks targets identity wire accepted) :
      SchemaNode codecs checks targets allowsUnknown (.select field identity) wire accepted
  | nonNullRejected : SchemaNode codecs checks targets allowsUnknown (.nonNull identity) .null false
  | nonNullValue (nonnull : wireIsNull wire = false) (result : SchemaEvaluation codecs checks targets identity wire accepted) :
      SchemaNode codecs checks targets allowsUnknown (.nonNull identity) wire accepted
  | array (lengths : items.length = outcomes.length)
      (results : ∀ pair ∈ items.zip outcomes,
      SchemaEvaluation codecs checks targets identity pair.1 pair.2) :
      SchemaNode codecs checks targets allowsUnknown (.array identity bounds) (.array items)
        (lengthAllowed bounds items.length && outcomes.all id)
  | map (lengths : items.length = outcomes.length)
      (results : ∀ pair ∈ items.zip outcomes,
      SchemaEvaluation codecs checks targets identity pair.1.2 pair.2) :
      SchemaNode codecs checks targets allowsUnknown (.map key keyRules identity bounds) (.object items)
        (decide (items.map Prod.fst).Nodup && lengthAllowed bounds items.length && outcomes.all id)
  | object (lengths : members.length = outcomes.length)
      (missing : ∀ pair ∈ members.zip outcomes,
        wireMember items pair.1.wireName = none → pair.2 = !pair.1.required)
      (present : ∀ pair ∈ members.zip outcomes, ∀ value,
        wireMember items pair.1.wireName = some value →
          SchemaEvaluation codecs checks targets pair.1.child value pair.2) :
      SchemaNode codecs checks targets allowsUnknown (.object members preserveAdditional) (.object items)
        (decide (items.map Prod.fst).Nodup && outcomes.all id &&
          (allowsUnknown || items.all (fun entry => members.any (·.wireName == entry.1))))
  | untagged (lengths : alternatives.length = outcomes.length)
      (results : ∀ pair ∈ alternatives.zip outcomes,
        SchemaEvaluation codecs checks targets pair.1.child wire pair.2) :
      SchemaNode codecs checks targets allowsUnknown (.union occurrence .untagged alternatives) wire
        ((outcomes.filter id).length == 1)
  | tagged (tag : wireMember items tagKey = some (.text name))
      (payload : wireMember items valueKey = some value)
      (chosen : alternatives.find? (fun alternative => alternative.wireName == name) = some alternative)
      (result : SchemaEvaluation codecs checks targets alternative.child value accepted) :
      SchemaNode codecs checks targets allowsUnknown (.union occurrence (.tagged tagKey valueKey) alternatives)
        (.object items) (accepted && decide (items.map Prod.fst).Nodup &&
          (allowsUnknown || items.all (fun entry => entry.1 == tagKey || entry.1 == valueKey)))
  | taggedMalformed (malformed : ¬ ∃ name value alternative,
      wireMember items tagKey = some (.text name) ∧ wireMember items valueKey = some value ∧
      alternatives.find? (fun alternative => alternative.wireName == name) = some alternative) :
      SchemaNode codecs checks targets allowsUnknown (.union occurrence (.tagged tagKey valueKey) alternatives)
        (.object items) false
  | protobuf (chosen : alternatives.find? (fun alternative => alternative.wireName == name) = some alternative)
      (result : SchemaEvaluation codecs checks targets alternative.child value accepted) :
      SchemaNode codecs checks targets allowsUnknown (.union occurrence .protobuf alternatives)
        (.oneof name value) accepted
  | protobufMalformed (missing : alternatives.find? (fun alternative => alternative.wireName == name) = none) :
      SchemaNode codecs checks targets allowsUnknown (.union occurrence .protobuf alternatives)
        (.oneof name value) false
  | anyValid (valid : JSONValid codecs.numbers wire) :
      SchemaNode codecs checks targets allowsUnknown .any wire true
  | anyInvalid (invalid : ¬ JSONValid codecs.numbers wire) :
      SchemaNode codecs checks targets allowsUnknown .any wire false
  | mismatch (mismatch : structuralMismatch target wire = true) :
      SchemaNode codecs checks targets allowsUnknown target wire false

end

def SchemaAccepts (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (identity : Identity) (wire : Wire) : Prop :=
  SchemaEvaluation codecs checks targets identity wire true

end ValueContract.Candidate
