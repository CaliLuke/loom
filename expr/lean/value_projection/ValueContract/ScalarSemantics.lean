import ValueContract.CandidateModel

namespace ValueContract.Candidate

/-- Narrow numerical serialization/parsing boundary. JSON lexical values are
shared by integer and decimal targets. Schema parsing and runtime integer/
decimal decoding are separate functions: neither may inspect a branch list.
Actual Go precision/formatting/parser correspondence remains a test obligation. -/
structure NumericCodec where
  encodeInteger : Int → String
  encodeDecimal : Int → Int → NumericFormat → Bool → String
  schemaNumber : String → Option Decimal
  decodeInteger : IntegerFormat → String → Option Int
  decodeDecimal : NumericFormat → String → Option ParsedDecimal
  /-- JSON object-key parsers may accept spellings that JSON number-token parsers
  reject. Their precision and acceptance are separately tested adapter boundaries. -/
  decodeKeyInteger : IntegerFormat → String → Option Int := decodeInteger
  decodeKeyDecimal : NumericFormat → String → Option ParsedDecimal := decodeDecimal
  /-- Authored literal formatting and precision parsing are independent of JSON
  encoding. They receive neither declarations nor candidate branch outcomes. -/
  literalInteger : Int → Nat → String := fun value _ => encodeInteger value
  literalDecimal : Int → Int → NumericFormat → Bool → Nat → String :=
    fun coefficient exponent format negativeZero _ => encodeDecimal coefficient exponent format negativeZero
  parseLiteral : NumericFormat → String → Option ParsedDecimal := fun _ _ => none

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
  | .integer value _ => some ⟨value, 0⟩
  | .decimal coefficient exponent _ _ _ => some ⟨coefficient, exponent⟩
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
def coerceBuiltinScalar (kind : ScalarKind) (value : Scalar) : Option Scalar :=
  if value.kind = kind then some value
  else match kind, value with
    | .bytes, .string text => some (.bytes text.toUTF8.data.toList)
    | .decimal, .integer number _ => some (.decimal number 0)
    | _, _ => none

/-- Declarative scalar coercion is separate from the executable definition. -/
inductive BuiltinCoerces : ScalarKind → Scalar → Scalar → Prop where
  | same (typed : value.kind = kind) : BuiltinCoerces kind value value
  | textBytes (text : String) :
      BuiltinCoerces .bytes (.string text) (.bytes text.toUTF8.data.toList)
  | integerDecimal (number : Int) (origin : Nat) : BuiltinCoerces .decimal (.integer number origin) (.decimal number 0)

theorem builtinCoercion_sound {kind input output}
    (success : coerceBuiltinScalar kind input = some output) : BuiltinCoerces kind input output := by
  unfold coerceBuiltinScalar at success
  split at success
  next typed =>
    cases Option.some.inj success
    exact .same typed
  next notTyped =>
    cases kind <;> cases input <;> simp_all
    all_goals cases success
    · exact .integerDecimal _ _
    · exact .textBytes _

theorem builtinCoercion_progress {kind input output} (valid : BuiltinCoerces kind input output) :
    coerceBuiltinScalar kind input = some output := by
  cases valid with
  | same typed => simp [coerceBuiltinScalar, typed]
  | textBytes text => simp [coerceBuiltinScalar, Scalar.kind]
  | integerDecimal number origin => simp [coerceBuiltinScalar, Scalar.kind]

/-- Literal spelling is distinct from JSON spelling and never inspects a
contract or selected branch. Only numeric source scalars enter this boundary. -/
def numericLiteral (codec : NumericCodec) : Scalar → Option String
  | .integer value origin => some (codec.literalInteger value origin)
  | .decimal coefficient exponent format negativeZero origin =>
      some (codec.literalDecimal coefficient exponent format negativeZero origin)
  | _ => none

inductive NumericLiteral (codec : NumericCodec) : Scalar → String → Prop where
  | integer : NumericLiteral codec (.integer value origin) (codec.literalInteger value origin)
  | decimal : NumericLiteral codec (.decimal coefficient exponent format negativeZero origin)
      (codec.literalDecimal coefficient exponent format negativeZero origin)

theorem numericLiteral_iff (codec : NumericCodec) (value : Scalar) (text : String) :
    numericLiteral codec value = some text ↔ NumericLiteral codec value text := by
  constructor
  · intro found
    cases value <;> simp only [numericLiteral, Option.some.injEq, reduceCtorEq] at found
    all_goals subst text; constructor
  · intro spelled
    cases spelled <;> rfl

/-- Every declared floating-point role formats the supplied literal and parses
it at the occurrence's precision before constraints, enums or branch ranking.
Any values do not call this operation and retain their original representation. -/
def coerceScalar (codec : NumericCodec) (format : NumericFormat)
    (kind : ScalarKind) (value : Scalar) : Option Scalar :=
  if kind = .decimal ∧ format ≠ .exact then do
    let text ← numericLiteral codec value
    let parsed ← codec.parseLiteral format text
    return .decimal parsed.value.coefficient parsed.value.exponent format parsed.negativeZero
  else coerceBuiltinScalar kind value

/-- Independent declared coercion composes a numeric-only literal relation and
one precision-specific parse. It does not assume successful resolution. -/
inductive Coerces (codec : NumericCodec) (format : NumericFormat) :
    ScalarKind → Scalar → Scalar → Prop where
  | builtin (unrounded : ¬ (kind = .decimal ∧ format ≠ .exact))
      (coercion : BuiltinCoerces kind input output) : Coerces codec format kind input output
  | floating (precision : format ≠ .exact) (spelling : NumericLiteral codec input text)
      (parsed : codec.parseLiteral format text = some number) :
      Coerces codec format .decimal input
        (.decimal number.value.coefficient number.value.exponent format number.negativeZero)

theorem coercion_sound {codec format kind input output}
    (success : coerceScalar codec format kind input = some output) :
    Coerces codec format kind input output := by
  unfold coerceScalar at success
  split at success
  next rounded =>
    simp only [Bind.bind, Option.bind_eq_some_iff, pure, Option.some.injEq] at success
    obtain ⟨text, spelled, number, parsed, same⟩ := success
    subst output
    rcases rounded with ⟨rfl, precision⟩
    exact .floating precision ((numericLiteral_iff ..).mp spelled) parsed
  next unrounded => exact .builtin unrounded (builtinCoercion_sound success)

theorem coercion_progress {codec format kind input output}
    (valid : Coerces codec format kind input output) :
    coerceScalar codec format kind input = some output := by
  cases valid with
  | builtin unrounded coercion =>
      simpa only [coerceScalar, unrounded, ↓reduceIte] using builtinCoercion_progress coercion
  | floating precision spelling parsed =>
      simp [coerceScalar, precision, (numericLiteral_iff ..).mpr spelling, parsed]

/-- Source eligibility is a concrete host-type check before normalization.
The abstract policy is explicit and used only by normalized model controls. -/
def coerceSourceScalar (codec : NumericCodec) (rules : ScalarRules)
    (kind : ScalarKind) (source : SourceScalar) : Option Scalar :=
  if sourceAdmits (rules.sourcePrimitive.getD (.abstract kind)) source.primitive then
    coerceScalar codec rules.numericFormat kind source.value
  else none

/-- Independent source coercion requires both original-host admission and
normalization; a successful numeric parse alone cannot establish eligibility. -/
def SourceCoerces (codec : NumericCodec) (rules : ScalarRules)
    (kind : ScalarKind) (source : SourceScalar) (value : Scalar) : Prop :=
  SourceAdmits (rules.sourcePrimitive.getD (.abstract kind)) source.primitive ∧
    Coerces codec rules.numericFormat kind source.value value

theorem sourceCoercion_iff (codec : NumericCodec) (rules : ScalarRules)
    (kind : ScalarKind) (source : SourceScalar) (value : Scalar) :
    coerceSourceScalar codec rules kind source = some value ↔
      SourceCoerces codec rules kind source value := by
  unfold coerceSourceScalar SourceCoerces
  by_cases admitted : sourceAdmits (rules.sourcePrimitive.getD (.abstract kind)) source.primitive = true
  · simp only [admitted, ↓reduceIte, (sourceAdmits_iff ..).mp admitted, true_and]
    exact ⟨coercion_sound, coercion_progress⟩
  · have rejected := mt (sourceAdmits_iff ..).mpr admitted
    simp [admitted, rejected]

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

theorem literal_base64_is_not_decoded (codec : NumericCodec) (format : NumericFormat) :
    coerceScalar codec format .bytes (.string "aGk=") = some (.bytes [97, 71, 107, 61]) := by
  simp [coerceScalar, coerceBuiltinScalar, Scalar.kind]
  decide

end ValueContract.Candidate
