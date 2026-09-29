import ValueContract.CandidateModel

namespace ValueContract.Candidate

/-- Narrow numerical serialization/parsing boundary. JSON lexical values are
shared by integer and decimal targets. Schema parsing and runtime integer/
decimal decoding are separate functions: neither may inspect a branch list.
Actual Go precision/formatting/parser correspondence remains a test obligation. -/
structure NumericCodec where
  encodeInteger : Int → String
  encodeDecimal : Int → Int → String
  schemaNumber : String → Option Decimal
  decodeInteger : String → Option Int
  decodeDecimal : String → Option Decimal
  /-- JSON object-key parsers may accept spellings that JSON number-token parsers
  reject. Their precision and acceptance are separately tested adapter boundaries. -/
  decodeKeyInteger : String → Option Int := decodeInteger
  decodeKeyDecimal : String → Option Decimal := decodeDecimal

structure ScalarCodecs where
  bytes : ByteCodec
  numbers : NumericCodec

def mapKeyCompatible (kind : MapKeyKind) (value : Scalar) : Bool :=
  match value with
  | .bytes _ => false
  | _ => match kind with
    | .scalar expected => value.kind == expected
    | .builtin => true

def Decimal.fraction (value : Decimal) : Int × Nat :=
  if value.exponent < 0 then
    (value.coefficient, 10 ^ value.exponent.natAbs)
  else
    (value.coefficient * Int.ofNat (10 ^ value.exponent.toNat), 1)

def Decimal.le (left right : Decimal) : Bool :=
  let l := left.fraction
  let r := right.fraction
  l.1 * Int.ofNat r.2 ≤ r.1 * Int.ofNat l.2

def Decimal.lt (left right : Decimal) : Bool :=
  let l := left.fraction
  let r := right.fraction
  l.1 * Int.ofNat r.2 < r.1 * Int.ofNat l.2

def Decimal.asInteger (value : Decimal) : Option Int :=
  let pair := value.fraction
  if pair.1 % Int.ofNat pair.2 = 0 then some (pair.1 / Int.ofNat pair.2) else none

def lengthAllowed (bounds : LengthBounds) (length : Nat) : Bool :=
  bounds.minimum.all (fun lower => lower ≤ Int.ofNat length) &&
    bounds.maximum.all (fun upper => Int.ofNat length ≤ upper)

def numberAllowed (bounds : NumericBounds) (number : Decimal) : Bool :=
  bounds.minimum.all (fun lower => if bounds.exclusiveMinimum then lower.lt number
    else lower.le number) &&
    bounds.maximum.all (fun upper => if bounds.exclusiveMaximum then number.lt upper
      else number.le upper)

def scalarNumber : Scalar → Option Decimal
  | .integer value => some ⟨value, 0⟩
  | .decimal coefficient exponent => some ⟨coefficient, exponent⟩
  | _ => none

def scalarLength : Scalar → Option Nat
  | .string value => some value.length
  | .bytes value => some value.length
  | _ => none

/-- Enum numeric comparison uses exact mathematical decimal meaning; destination
floating-point precision conversion remains an explicitly tested adapter step. -/
def scalarEqual (left right : Scalar) : Bool :=
  match scalarNumber left, scalarNumber right with
  | some l, some r => l.le r && r.le l
  | _, _ => left == right

/-- Format/pattern execution is a named external boundary. It does not receive a
target plan, candidate result, schema, or branch list and cannot select a branch. -/
abbrev ExternalScalarChecks := Nat → Scalar → Bool

def scalarAllowed (checks : ExternalScalarChecks) (rules : ScalarRules)
    (value : Scalar) : Bool :=
  rules.enumeration.all (fun values => values.any (scalarEqual value)) &&
    (scalarLength value).all (lengthAllowed rules.length) &&
    (scalarNumber value).all (numberAllowed rules.numeric) &&
    rules.externalChecks.all (fun key => checks key value)

/-- A supplied text for Bytes denotes its literal UTF-8 octets, including text
that already looks like base64. Other kind changes are not invented here. -/
def coerceScalar (kind : ScalarKind) (value : Scalar) : Option Scalar :=
  if value.kind = kind then some value
  else match kind, value with
    | .bytes, .string text => some (.bytes text.toUTF8.data.toList)
    | .decimal, .integer number => some (.decimal number 0)
    | _, _ => none

/-- Declarative scalar coercion is separate from the executable definition. -/
inductive Coerces : ScalarKind → Scalar → Scalar → Prop where
  | same (typed : value.kind = kind) : Coerces kind value value
  | textBytes (text : String) :
      Coerces .bytes (.string text) (.bytes text.toUTF8.data.toList)
  | integerDecimal (number : Int) : Coerces .decimal (.integer number) (.decimal number 0)

theorem coercion_sound {kind input output}
    (success : coerceScalar kind input = some output) : Coerces kind input output := by
  unfold coerceScalar at success
  split at success
  next typed =>
    cases Option.some.inj success
    exact .same typed
  next notTyped =>
    cases kind <;> cases input <;> simp_all
    all_goals cases success
    · exact .integerDecimal _
    · exact .textBytes _

theorem coercion_progress {kind input output} (valid : Coerces kind input output) :
    coerceScalar kind input = some output := by
  cases valid with
  | same typed => simp [coerceScalar, typed]
  | textBytes text => simp [coerceScalar, Scalar.kind]
  | integerDecimal number => simp [coerceScalar, Scalar.kind]

/-- Syntactic base64 language accepted by the built-in JSON byte decoder.
Pad bits deliberately are not checked: noncanonical spellings are accepted.
The result is decoded octet length, not the length of the wire string. -/
def base64Length (text : String) : Option Nat := Id.run do
  let chars := text.toList
  let count := chars.length
  if count % 4 != 0 then return none
  let padding := if chars.getLast? == some '=' then
    if (chars.take (count - 1)).getLast? == some '=' then 2 else 1
    else 0
  let alphabet (char : Char) :=
    ('A' ≤ char && char ≤ 'Z') || ('a' ≤ char && char ≤ 'z') ||
      ('0' ≤ char && char ≤ '9') || char == '+' || char == '/'
  if !(chars.take (count - padding)).all alphabet then return none
  if !(chars.drop (count - padding)).all (· == '=') then return none
  return some (count / 4 * 3 - padding)

/-- Codec assumptions are limited to octet/string conversion. The syntax
function is fixed above and is independent of decoding and projection. No law
here asserts branch uniqueness, enum equivalence, or candidate success. -/
structure ByteCodecLaws (codec : ByteCodec) : Prop where
  roundTrip : ∀ bytes, codec.decode (codec.encode bytes) = some bytes
  language : ∀ text count, base64Length text = some count ↔
    ∃ bytes, codec.decode text = some bytes ∧ bytes.length = count

theorem base64_empty : base64Length "" = some 0 := by decide

theorem base64_alias_length : base64Length "aGl=" = some 2 := by decide

theorem base64_canonical_length : base64Length "aGk=" = some 2 := by decide

theorem base64_linefeed_rejected : base64Length "aGk=\n" = none := by decide

theorem base64_bad_padding_rejected : base64Length "a===" = none := by decide

theorem literal_base64_is_not_decoded :
    coerceScalar .bytes (.string "aGk=") = some (.bytes [97, 71, 107, 61]) := by decide

end ValueContract.Candidate
