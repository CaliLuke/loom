/-
Candidate vocabulary. Declaration and target graphs are finite tables, while
values are inductive trees of arbitrary finite depth. This file does not define
typing, representation, decoding, or success in terms of the candidate projector.
The milestone-1 vocabulary and counterexamples remain in Model/Legacy unchanged.
-/
import ValueContract.SourceBytes

namespace ValueContract.Candidate

/-- Raw floating-point representation is distinct from mathematical meaning.
Exact values are used for schema arithmetic; binary formats retain the width
that controls literal and JSON spelling. -/
inductive NumericFormat where
  | exact | binary32 | binary64
  deriving DecidableEq, Repr

/-- Integer runtime parsing owns both signedness and effective machine width.
Mathematical targets are explicit abstract controls, never a default Go width. -/
inductive IntegerFormat where
  | mathematical
  | signed (bits : Nat)
  | unsigned (bits : Nat)
  deriving DecidableEq, Repr

/-- Candidate scalars retain arbitrary-precision mathematical values together
with raw representation evidence. Signed zero affects encoding, not numeric
comparison. A literal origin distinguishes source-specific numeric formatting
methods; zero denotes ordinary builtin literal formatting. It does not affect
JSON spelling or enum numeric equality. The milestone-1 vocabulary is unchanged. -/
inductive Scalar where
  | boolean (value : Bool)
  | integer (value : Int) (literalOrigin : Nat := 0)
  | decimal (coefficient exponent : Int) (format : NumericFormat := .exact)
      (negativeZero : Bool := false) (literalOrigin : Nat := 0)
  | string (value : String)
  | bytes (value : List UInt8)
  deriving DecidableEq, Repr

def Scalar.kind : Scalar → ScalarKind
  | .boolean _ => .boolean
  | .integer _ _ => .integer
  | .decimal _ _ _ _ _ => .decimal
  | .string _ => .string
  | .bytes _ => .bytes

/-- Concrete source admission evidence is retained until declared normalization.
It is deliberately absent from canonical values and target decoder scalars.
The normalized constructor vocabulary is for abstract controls only; production
inputs supply the concrete primitive captured from the original Go value. -/
structure SourceScalar where
  value : Scalar
  primitive : RawPrimitive := .normalized value.kind
  deriving DecidableEq, Repr

namespace SourceScalar

/-- Explicit abstract scalar vocabulary, never production admission evidence. -/
def boolean (value : Bool) : SourceScalar := ⟨.boolean value, .normalized .boolean⟩
def integer (value : Int) (literalOrigin : Nat := 0) : SourceScalar :=
  ⟨.integer value literalOrigin, .normalized .integer⟩
def decimal (coefficient exponent : Int) (format : NumericFormat := .exact)
    (negativeZero : Bool := false) (literalOrigin : Nat := 0) : SourceScalar :=
  ⟨.decimal coefficient exponent format negativeZero literalOrigin, .normalized .decimal⟩
def string (value : String) : SourceScalar := ⟨.string value, .normalized .string⟩
def bytes (value : List UInt8) : SourceScalar := ⟨.bytes value, .normalized .bytes⟩

end SourceScalar

instance : Coe Scalar SourceScalar where
  coe value := ⟨value, .normalized value.kind⟩

/-- Protobuf byte values and oneofs are structured target values. JSON bytes
use the same text constructor as strings, never a hidden base64 tag. -/
inductive Wire where
  | null
  | boolean (value : Bool)
  /-- JSON numbers share their final lexical domain before branch matching. -/
  | number (text : String)
  /-- Native protobuf numeric categories; never used for JSON uniqueness. -/
  | integer (value : Int)
  | decimal (coefficient exponent : Int)
  | text (value : String)
  | bytes (value : List UInt8)
  | array (items : List Wire)
  | object (fields : List (String × Wire))
  | oneof (name : String) (value : Wire)
  deriving Repr

/-- A missing slot is not a supplied null, nil container, or empty container. -/
inductive Value where
  | absent | null | nilArray | nilMap | nilBytes
  | scalar (value : Scalar)
  | object (fields : List (Identity × Value)) (additional : List (String × Value))
  | array (items : List Value)
  | map (entries : List (Scalar × Value))
  | union (occurrence branch : Identity) (payload : Value)
  /-- Codec-owned Any retains raw built-in meaning until materialization. -/
  | any (payload : Value)
  /-- Adapter-owned DeepEqual class of an immutable finite built-in snapshot.
  This evidence affects only raw Any equality; materialization ignores it. -/
  | host (identity : Nat) (payload : Value)
  /-- Target observation only: owned JSON after Any codec materialization.
  There is no corresponding Input constructor, so authors cannot forge it. -/
  | jsonSnapshot (wire : Wire)
  deriving Repr

/-- Open-object members have names but no invented declared-field identity.
Raw input entry lists retain duplicates for validation before any lookup. -/
inductive Input where
  | absent | null | nilArray | nilMap | nilBytes
  | scalar (value : SourceScalar)
  | byteSequence (sequence : NativeByteSequence)
  | object (fields : List (String × Input))
  | array (items : List Input)
  | map (entries : List (SourceScalar × Input))
  | selected (occurrence branch : Identity) (payload : Input)
  | cycle (identity : Nat)
  | opaque (identity : Nat)
  /-- Retains concrete host-type equality evidence before declared coercion. -/
  | host (identity : Nat) (payload : Input)
  deriving Repr

/-- Native byte elements keep their original host class when a declared Array
consumes them. Their concrete scalar type and literal origin are fixed by the
native-byte constructor, independently of the selected declaration or target. -/
def NativeByteSequence.inputs (sequence : NativeByteSequence) : List Input :=
  sequence.items.map fun item =>
    .host item.1 (.scalar ⟨.integer (Int.ofNat item.2.toNat), .builtinUInt8⟩)

/-- Raw Any keeps the codec meaning after source dispatch. Source-only array,
slice and container naming evidence remains on Input, not on target values. -/
def NativeByteSequence.rawValue (sequence : NativeByteSequence) : Value :=
  if sequence.isNil then .nilBytes else .scalar (.bytes sequence.octets)

/-- Remove only outer host evidence; child nodes retain their own evidence. -/
def stripHostValue : Value → Value
  | .host _ payload => stripHostValue payload
  | value => value

def stripHostInput : Input → Input
  | .host _ payload => stripHostInput payload
  | input => input

/-- Null remains null through host evidence and the raw Any wrapper. -/
def valueIsNull : Value → Bool
  | .null => true
  | .host _ payload | .any payload => valueIsNull payload
  | _ => false

/-- Exact decimal bounds use a coefficient and a signed power of ten.
This does not model machine floating-point parsing or rounding. -/
structure Decimal where
  coefficient : Int
  exponent : Int
  deriving DecidableEq, Repr

/-- Exact numeric result of one machine parser, including signed zero. The
requested precision is supplied separately and cannot be chosen by the parser. -/
structure ParsedDecimal where
  value : Decimal
  negativeZero : Bool := false
  deriving DecidableEq, Repr

/-- Lower and upper bounds may be negative in the evaluated DSL. Their meaning
is comparison with the actual nonnegative length, not a Nat coercion of bounds. -/
structure LengthBounds where
  minimum : Option Int := none
  maximum : Option Int := none
  deriving DecidableEq, Repr

structure NumericBounds where
  minimum : Option Decimal := none
  maximum : Option Decimal := none
  exclusiveMinimum : Bool := false
  exclusiveMaximum : Bool := false
  deriving DecidableEq, Repr

/-- A named external check identifies format/pattern semantics supplied by the
adapter. Its result is an explicit boundary; it cannot establish core typing or
branch preservation. None and some [] distinguish no enum from an empty enum. -/
structure ScalarRules where
  /-- Concrete source policy is checked before normalization. None names the
  explicit abstract scalar-kind policy used by normalized model controls.
  Runtime target decoding does not consult this source-only field. -/
  sourcePrimitive : Option SourcePrimitive := none
  /-- Effective declared or target precision, independently owned per occurrence. -/
  numericFormat : NumericFormat := .exact
  /-- Runtime integer policy; independent of authored/schema numeric bounds.
  Source integer matching preserves existing acceptance and does not use it. -/
  integerFormat : IntegerFormat := .mathematical
  enumeration : Option (List Scalar) := none
  length : LengthBounds := {}
  numeric : NumericBounds := {}
  externalChecks : List Nat := []
  deriving DecidableEq, Repr

structure Member where
  identity : Identity
  sourceName : String
  wireAlias : String
  required : Bool
  child : Identity
  deriving DecidableEq, Repr

structure Alternative where
  identity : Identity
  name : String
  child : Identity
  deriving DecidableEq, Repr

/-- Any-key maps may contain heterogeneous built-in scalar keys. Bytes, null,
opaque and custom keys are not admitted by the built-in JSON member codec. -/
inductive MapKeyKind where
  | scalar (kind : ScalarKind)
  | builtin
  deriving DecidableEq, Repr

/-- References are graph edges, including nullable/alias edges that consume no
input node. An object/array/map edge consumes an input child; recursive graphs
are therefore not restricted to a fixed inhabitant depth. -/
inductive Contract where
  | scalar (kind : ScalarKind) (rules : ScalarRules)
  | nullable (child : Identity)
  /-- Effective occurrence override, for example ArrayOfRequired(Any). -/
  | nonNull (child : Identity)
  | array (child : Identity) (length : LengthBounds)
  | map (key : MapKeyKind) (keyRules : ScalarRules) (child : Identity)
      (length : LengthBounds)
  | object (members : List Member) (isOpen : Bool)
  | union (occurrence : Identity) (alternatives : List Alternative)
  | alias (child : Identity)
  | any
  | custom (codec : Nat)
  deriving DecidableEq, Repr

/-- The rank witnesses termination of same-input reference expansion only.
It is reset after consuming an input child, and is not a value-depth bound. -/
structure Declaration where
  identity : Identity
  expansionRank : Nat
  contract : Contract
  enumeration : Option (List Value) := none
  deriving Repr

abbrev Declarations := List Declaration

/-- Target-local presence equivalence. A protobuf implicit scalar field may
omit its default while preserving that default under observation/decoding.
Explicit presence and null never silently inherit that equivalence. -/
inductive Presence where
  | explicit
  | omitEmpty
  | implicitDefault (value : Scalar)
  deriving DecidableEq, Repr

structure TargetMember where
  identity : Identity
  wireName : String
  required : Bool
  presence : Presence
  child : Identity
  deriving DecidableEq, Repr

structure TargetAlternative where
  identity : Identity
  wireName : String
  child : Identity
  deriving DecidableEq, Repr

inductive Encoding where
  | json
  | protobuf
  deriving DecidableEq, Repr

/-- A complete representation plan, not a renderer flag on a canonical type.
The target graph has its own identities/ranks; its shape need not retain every
service field. Projecting an object may legitimately erase service members. -/
inductive Target where
  | scalar (encoding : Encoding) (kind : ScalarKind) (rules : ScalarRules)
  | nullable (child : Identity)
  | nonNull (child : Identity)
  | array (child : Identity) (length : LengthBounds)
  | map (key : MapKeyKind) (keyRules : ScalarRules) (child : Identity)
      (length : LengthBounds)
  | object (members : List TargetMember) (preserveAdditional : Bool)
  | union (occurrence : Identity) (style : UnionStyle)
      (alternatives : List TargetAlternative)
  | select (field : Identity) (child : Identity)
  | alias (child : Identity)
  | any
  | custom (codec : Nat)
  deriving DecidableEq, Repr

structure TargetDeclaration where
  identity : Identity
  expansionRank : Nat
  target : Target
  /-- Runtime semantic enum members, including collection values. -/
  enumeration : Option (List Value) := none
  /-- Actual emitted schema enum alternatives. These are checked independently
  of runtime enum membership, including noncanonical byte decoder aliases. -/
  schemaEnumeration : Option (List Wire) := none
  /-- Schema acceptance of undeclared object/envelope members is independent
  of whether the runtime decoder retains or ignores them. -/
  schemaAllowsUnknown : Bool := true
  /-- Runtime rejection policy for undeclared object/envelope members.
  False means ignore unless the target explicitly preserves additional data. -/
  decoderRejectsUnknown : Bool := false
  deriving Repr

abbrev Targets := List TargetDeclaration

/-- Missing paths and ambiguity obstructions are retained with the resolved
tree. Incompleteness is assessed again after target visibility selection;
ambiguity must never be erased by selecting a different target field. -/
structure Resolution where
  value : Value
  missing : List (List Identity)
  deriving Repr

inductive Failure where
  | invalid | ambiguous | unsupported | cyclic | malformedDeclaration
  deriving DecidableEq, Repr

abbrev ResolveResult := Except Failure Resolution

/-- The opaque/custom boundary has no built-in proof merely because a callback
returned a value. Correspondence tests must discharge its separate contract. -/
inductive ProjectionResult where
  | emitted (value : Wire)
  | incomplete | unrepresentable | unsupported | invalidPlan
  deriving Repr

end ValueContract.Candidate
