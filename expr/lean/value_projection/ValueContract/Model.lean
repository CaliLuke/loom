/-
Finite value-contract vocabulary and independent judgments. There is no candidate
resolver or projector in milestone 1. In particular, none of these judgments is
defined by successful execution of a candidate function.
-/
import Std

namespace ValueContract

/-- Elementwise predicates, kept in core Lean without a proof-library dependency. -/
def All (predicate : α → Prop) (items : List α) : Prop :=
  ∀ item ∈ items, predicate item

/-- Ordered correspondence preserves multiplicity, unlike a map conversion. -/
def All₂ (relation : α → β → Prop) (left : List α) (right : List β) : Prop :=
  left.length = right.length ∧ ∀ pair ∈ left.zip right, relation pair.1 pair.2

/-- Stable evaluated identities, not field spellings or payload hashes. -/
structure Identity where
  occurrence : Nat
  declaration : Nat
  deriving DecidableEq, Repr

inductive ScalarKind where
  | boolean | integer | decimal | string | bytes
  deriving DecidableEq, Repr

/-- Decimal is an exact coefficient/exponent pair, not a claim about Go floats.
Declared precision, numeric range and runtime rounding are separate obligations. -/
inductive Scalar where
  | boolean (value : Bool)
  | integer (value : Int)
  | decimal (coefficient exponent : Int)
  | string (value : String)
  | bytes (value : List UInt8)
  deriving DecidableEq, Repr

def Scalar.kind : Scalar → ScalarKind
  | .boolean _ => .boolean
  | .integer _ => .integer
  | .decimal _ _ => .decimal
  | .string _ => .string
  | .bytes _ => .bytes

/-- Entry lists deliberately retain duplicate keys until collision checking. -/
inductive RawValue where
  | absent | null | nilArray | nilMap
  | scalar (value : Scalar)
  | object (fields : List (String × RawValue))
  | array (items : List RawValue)
  | map (entries : List (Scalar × RawValue))
  | selected (union branch : Identity) (payload : RawValue)
  | cycle (identity : Nat)
  | opaque (identity : Nat)
  deriving Repr

/-- Finite semantic trees retain branch identity even for identical payloads.
Absent and null are separate constructors; nil containers are not empty lists. -/
inductive Value where
  | absent | null | nilArray | nilMap
  | scalar (value : Scalar)
  | object (fields : List (Identity × Value))
  | array (items : List Value)
  | map (entries : List (Scalar × Value))
  | union (identity branch : Identity) (payload : Value)
  deriving Repr

inductive Role where
  | authoredExample | synthesizedExample | enumMember | defaultValue
  deriving DecidableEq, Repr

structure Source where
  occurrence : Identity
  origin : Nat
  role : Role
  deriving DecidableEq, Repr

structure Supplied (α : Type) where
  source : Source
  value : α
  deriving Repr

inductive Outcome (α : Type) where
  | resolved (value : α)
  | incomplete | ambiguous | invalid | unsupported | suppressed
  deriving Repr

/-- References permit recursive declarations while values remain finite trees. -/
inductive Shape where
  | scalar (kind : ScalarKind)
  | nullable (inner : Shape)
  | array (element : Shape)
  | map (key : ScalarKind) (element : Shape)
  | object (fields : List (Identity × Shape))
  | union (identity : Identity) (branches : List (Identity × Shape))
  | reference (identity : Identity)

abbrev Declarations := Identity → Option Shape

/-- Structural typing at a derivation depth. Depth is explicit rather than a
fixed bound: the public judgment existentially quantifies over all natural depths.
References may unfold recursively, but a successful derivation is finite.
Requiredness, enums and numeric ranges are additional milestone-2 obligations. -/
def hasTypeAt : Nat → Declarations → Shape → Value → Prop
  | 0, _, _, _ => False
  | depth + 1, declarations, shape, value =>
    match shape, value with
    | .scalar kind, .scalar scalar => scalar.kind = kind
    | .nullable _, .null => True
    | .nullable inner, value => hasTypeAt depth declarations inner value
    | .array element, .array values => All (hasTypeAt depth declarations element) values
    | .map key element, .map entries => All (fun entry =>
        entry.1.kind = key ∧ hasTypeAt depth declarations element entry.2) entries
    | .object fields, .object values => All₂ (fun field item =>
        field.1 = item.1 ∧ hasTypeAt depth declarations field.2 item.2) fields values
    | .union identity branches, .union actual branch payload => identity = actual ∧
        ∃ shape, (branch, shape) ∈ branches ∧ hasTypeAt depth declarations shape payload
    | .reference identity, value => ∃ shape,
        declarations identity = some shape ∧ hasTypeAt depth declarations shape value
    | _, _ => False

def HasType (declarations : Declarations) (shape : Shape) (value : Value) : Prop :=
  ∃ depth, hasTypeAt depth declarations shape value

inductive UnionStyle where
  | tagged (tagKey valueKey : String)
  | untagged
  | protobuf
  deriving DecidableEq, Repr

structure Field (α : Type) where
  identity : Identity
  wireName : String
  required : Bool
  omitEmpty : Bool
  child : α

structure Branch (α : Type) where
  identity : Identity
  wireName : String
  child : α

/-- Target plans are explicit inputs. Producing them from the DSL, validating
name uniqueness and protobuf allocation are not proved by this foundation. -/
inductive Plan where
  | scalar (kind : ScalarKind)
  | nullable (inner : Plan)
  | array (element : Plan)
  | map (key : ScalarKind) (element : Plan)
  | object (fields : List (Field Plan))
  | union (identity : Identity) (style : UnionStyle) (branches : List (Branch Plan))
  | select (field : Identity) (child : Plan)

def fieldValue (fields : List (Identity × Value)) (identity : Identity) : Value :=
  match fields.find? (fun entry => entry.1 == identity) with
  | some entry => entry.2
  | none => .absent

/-- Only a field's explicit omission rule collapses empty to absent. Null is
never collapsed here; protobuf default/presence rules remain milestone 2. -/
def fieldObservation (omitEmpty : Bool) (value : Value) : Value :=
  if omitEmpty then
    match value with
    | .array [] | .map [] | .scalar (.bytes []) | .scalar (.string "") => .absent
    | _ => value
  else value

/-- Independent target observation, with an unbounded derivation depth.
Object plans select visibility; selected Body plans remove the enclosing service
object. A visible union retains its occurrence and branch identities. -/
def observeAt : Nat → Plan → Value → Value → Prop
  | 0, _, _, _ => False
  | depth + 1, plan, value, observed =>
    match plan, value, observed with
    | _, .absent, .absent => True
    | .scalar kind, .scalar scalar, .scalar result => scalar.kind = kind ∧ result = scalar
    | .nullable _, .null, .null => True
    | .nullable inner, value, observed => observeAt depth inner value observed
    | .array element, .array values, .array results =>
        All₂ (observeAt depth element) values results
    | .map key element, .map entries, .map results => All₂ (fun entry result =>
        entry.1 = result.1 ∧ entry.1.kind = key ∧
        observeAt depth element entry.2 result.2) entries results
    | .object fields, .object values, .object results => All₂ (fun field result =>
        result.1 = field.identity ∧ ∃ observation,
        observeAt depth field.child (fieldValue values field.identity) observation ∧
        result.2 = fieldObservation field.omitEmpty observation) fields results
    | .union identity _ branches, .union actual branch payload,
        .union observedIdentity observedBranch result =>
        actual = identity ∧ observedIdentity = identity ∧ observedBranch = branch ∧
        ∃ rule ∈ branches, rule.identity = branch ∧ observeAt depth rule.child payload result
    | .select field child, .object values, observed =>
        observeAt depth child (fieldValue values field) observed
    | _, _, _ => False

def Observe (plan : Plan) (value observed : Value) : Prop :=
  ∃ depth, observeAt depth plan value observed

/-- Structured wire values, before rendering. Bytes and String share the same
text constructor: no hidden byte tag can make identical JSON strings disjoint. -/
inductive Wire where
  | null
  | boolean (value : Bool)
  | integer (value : Int)
  | decimal (coefficient exponent : Int)
  | text (value : String)
  | array (items : List Wire)
  | object (fields : List (String × Wire))
  deriving Repr

/-- An explicit external lexical codec parameter. M1 does not prove that any
implementation of these functions is base64. Candidate theorems must name the
codec laws they require, and actual decoder tests remain mandatory. -/
structure ByteCodec where
  encode : List UInt8 → String
  decode : String → Option (List UInt8)

/-- A named premise, not a global axiom or a fact about a production codec. -/
def ByteCodec.RoundTrip (codec : ByteCodec) : Prop :=
  ∀ bytes, codec.decode (codec.encode bytes) = some bytes

/-- Independent scalar decoding. A JSON string may decode under both Bytes and
String; untagged matching must detect that collision rather than reselect. -/
inductive DecodeScalar (codec : ByteCodec) : ScalarKind → Wire → Scalar → Prop where
  | boolean : DecodeScalar codec .boolean (.boolean b) (.boolean b)
  | integer : DecodeScalar codec .integer (.integer n) (.integer n)
  | decimal : DecodeScalar codec .decimal (.decimal c e) (.decimal c e)
  | string : DecodeScalar codec .string (.text s) (.string s)
  | bytes : codec.decode text = some bs → DecodeScalar codec .bytes (.text text) (.bytes bs)

/-- Initial independent wire typing, not a call to a decoder or projector.
The complete numeric/enum/uniqueness constraints remain milestone 2. -/
def wireTypedAt (codec : ByteCodec) : Nat → Plan → Wire → Prop
  | 0, _, _ => False
  | depth + 1, plan, wire =>
    match plan, wire with
    | .scalar .boolean, .boolean _ | .scalar .integer, .integer _ |
      .scalar .decimal, .decimal _ _ | .scalar .string, .text _ => True
    | .scalar .bytes, .text text => ∃ bytes, codec.decode text = some bytes
    | .nullable _, .null => True
    | .nullable inner, wire => wireTypedAt codec depth inner wire
    | .array element, .array items => All (wireTypedAt codec depth element) items
    | .select _ child, wire => wireTypedAt codec depth child wire
    | _, _ => False

def WireTyped (codec : ByteCodec) (plan : Plan) (wire : Wire) : Prop :=
  ∃ depth, wireTypedAt codec depth plan wire

/-- Independent scalar/array/selected-body decoding. Full object/map/union
codec judgments must be added before milestone 2's progress theorem is accepted. -/
def decodeAt (codec : ByteCodec) : Nat → Plan → Wire → Value → Prop
  | 0, _, _, _ => False
  | depth + 1, plan, wire, value =>
    match plan, wire, value with
    | .scalar kind, wire, .scalar scalar => DecodeScalar codec kind wire scalar
    | .nullable _, .null, .null => True
    | .nullable inner, wire, value => decodeAt codec depth inner wire value
    | .array element, .array wires, .array values => All₂ (decodeAt codec depth element) wires values
    | .select _ child, wire, value => decodeAt codec depth child wire value
    | _, _, _ => False

def Decode (codec : ByteCodec) (plan : Plan) (wire : Wire) (value : Value) : Prop :=
  ∃ depth, decodeAt codec depth plan wire value

/-- A specification relation independent of a candidate projector. The M1
fragment includes lossy selected bodies. Full object/map/union representability,
including unique untagged wire matches, is a milestone-2 requirement. -/
def Representable (codec : ByteCodec) (plan : Plan) (value : Value) : Prop :=
  ∃ wire observed, WireTyped codec plan wire ∧ Decode codec plan wire observed ∧ Observe plan value observed

end ValueContract
