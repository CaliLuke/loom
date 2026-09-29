import ValueContract.Materialization
import ValueContract.KeyDecoding

namespace ValueContract.Candidate

def contractRole : Role → Bool
  | .enumMember | .defaultValue => true
  | _ => false

/-- Absence is produced only after duplicate identities have been rejected.
Unknown raw object names are not silently converted into declared identities. -/
def memberValue (fields : List (Identity × Value)) (identity : Identity) : Value :=
  match fields.find? (fun entry => entry.1 == identity) with
  | some entry => entry.2
  | none => .absent

def emptyValue : Value → Bool
  | .array [] | .map [] | .scalar (.bytes []) | .scalar (.string "") => true
  | _ => false

/-- This is a field-local observation rule, not a global value equivalence.
Nil containers and explicit null are deliberately not treated as empty. -/
def presenceValue (presence : Presence) (value : Value) : Value :=
  match presence, value with
  | .omitEmpty, value => if emptyValue value then .absent else value
  | .implicitDefault scalar, .absent => .scalar scalar
  | _, value => value

def implicitDefaultOmitted (presence : Presence) (value : Value) : Bool :=
  match presence, value with
  | .implicitDefault expected, .scalar actual =>
    expected.kind == actual.kind && scalarEqual expected actual
  | _, _ => false

/-- Empty JSON containers and text are omitted only under an explicit field
presence policy. Null and numeric defaults are not empty under this policy. -/
def snapshotEmpty : Wire → Bool
  | .text "" | .bytes [] | .array [] | .object [] => true
  | _ => false

/-- Whether an already-observed child has an empty target representation.
This is independent of wire construction. An object can become empty after
visibility removes fields or field-local omission removes its remaining data.
An implicit protobuf default may still be observed while absent from the wire.
Untagged unions inherit payload emptiness; tagged/oneof envelopes do not. -/
def emptyObservedAt : Nat → Targets → Identity → Value → Prop
  | 0, _, _, _ => False
  | depth + 1, targets, identity, value =>
    ∃ declaration ∈ targets, declaration.identity = identity ∧
    match declaration.target, value with
    | _, .null | _, .nilArray | _, .nilMap | _, .nilBytes => False
    | .scalar _ .string _, .scalar (.string "") => True
    | .scalar _ .bytes _, .scalar (.bytes []) => True
    | .array _ _, .array [] | .map _ _ _ _, .map [] => True
    | .any, .jsonSnapshot wire => snapshotEmpty wire = true
    | .nullable child, value | .nonNull child, value | .alias child, value |
        .select _ child, value =>
      emptyObservedAt depth targets child value
    | .object members _, .object fields extra => extra = [] ∧
      ∀ member ∈ members, memberValue fields member.identity = .absent ∨
        implicitDefaultOmitted member.presence (memberValue fields member.identity) = true
    | .union occurrence .untagged alternatives, .union actual branch payload =>
      occurrence = actual ∧ ∃ alternative ∈ alternatives, alternative.identity = branch ∧
        emptyObservedAt depth targets alternative.child payload
    | _, _ => False

def EmptyObserved (targets : Targets) (identity : Identity) (value : Value) : Prop :=
  ∃ depth, emptyObservedAt depth targets identity value

/-- Presence is applied AFTER the child observation. Its emptiness predicate is
unbounded and independent, so choosing a small observation derivation depth
cannot manufacture a false "not empty" result. -/
def FieldPresence (targets : Targets) (child : Identity) (presence : Presence)
    (value result : Value) : Prop :=
  match presence with
  | .explicit => result = value
  | .implicitDefault scalar => result = presenceValue (.implicitDefault scalar) value
  | .omitEmpty => (EmptyObserved targets child value ∧ result = .absent) ∨
      (¬ EmptyObserved targets child value ∧ result = value)

/-- Retained untyped extra members are codec-owned JSON snapshots, like Any.
Their nil-container meaning is not borrowed from declared collection examples. -/
def observeFreeAt (depth : Nat) (codecs : ScalarCodecs) (value observed : Value) : Prop :=
  depth > 0 ∧ ∃ wire, observed = .jsonSnapshot wire ∧
    Materializes codecs value wire ∧ JSONValid codecs.numbers wire

/-- Independent target observation selects visible semantic identities and
retains visible union identities. It never uses schema acceptance, decoding,
canonical wire construction, or projector success to decide what was observed.
Missing fields remain observable as absent; canonical emission separately checks
requiredness. A selected Body drops all other service fields. -/
def observeAt : Nat → ScalarCodecs → Targets → Role → Identity → Value → Value → Prop
  | 0, _, _, _, _, _, _ => False
  | depth + 1, codecs, targets, role, identity, value, observed =>
    ∃ declaration ∈ targets, declaration.identity = identity ∧
    match declaration.target, value, observed with
    | _, .absent, .absent => True
    | .scalar _ kind _, .scalar scalar, .scalar result =>
      scalar.kind = kind ∧ result = scalar
    | .nullable _, .null, .null => True
    | .nullable child, value, observed | .alias child, value, observed =>
      observeAt depth codecs targets role child value observed
    | .nonNull child, value, observed => valueIsNull value = false ∧
      observeAt depth codecs targets role child value observed
    | .array _ _, .nilArray, .array [] | .map _ _ _ _, .nilMap, .map [] =>
      contractRole role = true
    | .array child _, .array values, .array results =>
      All₂ (observeAt depth codecs targets role child) values results
    | .map key _ child _, .map entries, .map results =>
      All₂ (fun entry result => ObservedKey codecs.numbers key entry.1 result.1 ∧
        observeAt depth codecs targets role child entry.2 result.2) entries results
    | .object members preserveAdditional, .object fields extra, .object results observedExtra =>
      All₂ (fun member result => member.identity = result.1 ∧
        ∃ childObserved, observeAt depth codecs targets role member.child
          (memberValue fields member.identity) childObserved ∧
          FieldPresence targets member.child member.presence childObserved result.2)
        members results ∧
      if preserveAdditional then All₂ (fun entry result => entry.1 = result.1 ∧
        observeFreeAt depth codecs entry.2 result.2) extra observedExtra
      else observedExtra = []
    | .union occurrence _ alternatives, .union actual branch payload,
        .union resultOccurrence resultBranch resultPayload =>
      actual = occurrence ∧ resultOccurrence = occurrence ∧ resultBranch = branch ∧
      ∃ alternative ∈ alternatives, alternative.identity = branch ∧
        observeAt depth codecs targets role alternative.child payload resultPayload
    | .any, .any payload, .jsonSnapshot wire =>
      Materializes codecs payload wire ∧ JSONValid codecs.numbers wire
    | .select field child, .object fields _, observed =>
      observeAt depth codecs targets role child (memberValue fields field) observed
    | _, _, _ => False

def Observe (codecs : ScalarCodecs) (targets : Targets) (role : Role) (identity : Identity)
    (value observed : Value) : Prop :=
  ∃ depth, observeAt depth codecs targets role identity value observed

theorem null_not_empty : emptyValue .null = false := rfl

theorem nil_not_empty : emptyValue .nilArray = false ∧ emptyValue .nilMap = false := by
  exact ⟨rfl, rfl⟩

theorem explicit_null_presence (presence : Presence) : presenceValue presence .null = .null := by
  cases presence <;> rfl

theorem field_empty_omitted : presenceValue .omitEmpty (.array []) = .absent := rfl

theorem field_empty_explicit : presenceValue .explicit (.array []) = .array [] := rfl

end ValueContract.Candidate
