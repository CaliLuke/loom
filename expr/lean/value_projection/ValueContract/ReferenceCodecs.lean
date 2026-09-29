/-
Executable lexical adapters for reference comparisons. No theorem claims these
implement Go's codecs: the conformance harness tests them against the real
standard-library codecs. Numeric overrides expose machine rounding/precision
as an explicit input boundary of the proved, exact-decimal model.
-/
import ValueContract.ReferenceJson

namespace ValueContract.Reference
open Lean Candidate

private def alphabet : Array Char :=
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".toList.toArray

private def digit (value : Nat) : Char := alphabet[value % 64]!

private def encodeGroups : List UInt8 → List Char
  | [] => []
  | [a] => [digit (a.toNat / 4), digit (a.toNat % 4 * 16), '=', '=']
  | [a, b] => [digit (a.toNat / 4), digit (a.toNat % 4 * 16 + b.toNat / 16),
      digit (b.toNat % 16 * 4), '=']
  | a :: b :: c :: rest =>
      digit (a.toNat / 4) :: digit (a.toNat % 4 * 16 + b.toNat / 16) ::
      digit (b.toNat % 16 * 4 + c.toNat / 64) :: digit (c.toNat % 64) ::
      encodeGroups rest

def encodeBytes (value : List UInt8) : String := String.ofList (encodeGroups value)

private def undigit (value : Char) : Option Nat := alphabet.toList.idxOf? value

private def decodeGroups : List Char → Option (List UInt8)
  | [] => some []
  | [a, b, '=', '='] => do
      let x ← undigit a
      let y ← undigit b
      return [UInt8.ofNat (x * 4 + y / 16)]
  | [a, b, c, '='] => do
      let x ← undigit a
      let y ← undigit b
      let z ← undigit c
      return [UInt8.ofNat (x * 4 + y / 16), UInt8.ofNat (y % 16 * 16 + z / 4)]
  | a :: b :: c :: d :: rest => do
      let x ← undigit a
      let y ← undigit b
      let z ← undigit c
      let w ← undigit d
      let tail ← decodeGroups rest
      return UInt8.ofNat (x * 4 + y / 16) :: UInt8.ofNat (y % 16 * 16 + z / 4) ::
        UInt8.ofNat (z % 4 * 64 + w) :: tail
  | _ => none

/-- Go's non-strict standard alphabet decoder ignores CR/LF and accepts unused
pad-bit aliases. Schema grammar checks remain a separate model operation. -/
def decodeBytes (text : String) : Option (List UInt8) :=
  decodeGroups (text.toList.filter fun ch => ch != '\r' && ch != '\n')

private def zeros (count : Nat) : String := String.ofList (List.replicate count '0')

/-- Exact decimal spelling with JSON's ordinary/scientific magnitude thresholds.
IEEE rounding is supplied separately, never guessed by the semantic model. -/
def encodeDecimal (coefficient exponent : Int) : String := Id.run do
  if coefficient = 0 then return "0"
  let sign := if coefficient < 0 then "-" else ""
  let mut digits := (toString coefficient.natAbs).toList
  let mut exp := exponent
  for _ in [:digits.length] do
    if digits.getLast? = some '0' then
      digits := digits.dropLast
      exp := exp + 1
  let position := (digits.length : Int) + exp
  let text := String.ofList digits
  if position > 21 || position ≤ -6 then
    let head := String.ofList (digits.take 1)
    let tail := String.ofList (digits.drop 1)
    let mantissa := if tail.isEmpty then head else head ++ "." ++ tail
    let power := position - 1
    return sign ++ mantissa ++ "e" ++ (if power ≥ 0 then "+" else "") ++ toString power
  if position ≤ 0 then return sign ++ "0." ++ zeros position.natAbs ++ text
  if position.toNat ≥ digits.length then
    return sign ++ text ++ zeros (position.toNat - digits.length)
  return sign ++ String.ofList (digits.take position.toNat) ++ "." ++
    String.ofList (digits.drop position.toNat)

def parseNumber (text : String) : Option Decimal := do
  if text.toList.any Char.isWhitespace then none else do
    let json ← (Json.parse text).toOption
    let number ← json.getNum?.toOption
    return ⟨number.mantissa, -(number.exponent : Int)⟩

def parseInteger (text : String) : Option Int := do
  let number ← parseNumber text
  if text.toList.any (fun ch => ch = '.' || ch = 'e' || ch = 'E') then none
  else number.asInteger

/-- Each reading is obtained independently of the production resolver using
the actual target codec. A null result denotes rejection, not a missing row. -/
structure NumberReading where
  text : String
  schema : Option Decimal
  deriving ToJson, FromJson

/-- Scalar and member-name parsers share target width/signedness but retain
separate lexical acceptance. Neither result may be inferred from schema bounds. -/
structure IntegerReading where
  format : IntegerFormat
  text : String
  scalar : Option Int
  key : Option Int
  deriving ToJson, FromJson

structure DecimalReading where
  format : NumericFormat
  text : String
  scalar : Option ParsedDecimal
  key : Option ParsedDecimal
  deriving ToJson, FromJson

/-- Lookup identity retains exact meaning, source width and signed zero. Equal
mathematical values must not collapse distinct codec operations. -/
structure DecimalIdentity where
  value : Decimal
  format : NumericFormat
  negativeZero : Bool
  deriving DecidableEq, ToJson, FromJson

structure DecimalSpelling where
  number : DecimalIdentity
  text : String
  deriving ToJson, FromJson

structure LiteralSpelling where
  input : Candidate.Scalar
  text : String
  deriving ToJson, FromJson

structure LiteralReading where
  format : NumericFormat
  text : String
  result : Option ParsedDecimal
  deriving ToJson, FromJson

structure CodecInputs where
  numberReadings : List NumberReading
  integerReadings : List IntegerReading
  decimalReadings : List DecimalReading
  decimalSpellings : List DecimalSpelling
  literalSpellings : List LiteralSpelling
  literalReadings : List LiteralReading
  deriving ToJson, FromJson

def scalarCodecs (inputs : CodecInputs) : ScalarCodecs where
  bytes := ⟨encodeBytes, decodeBytes⟩
  numbers := {
    literalInteger := fun value origin =>
      match inputs.literalSpellings.find? (·.input == Candidate.Scalar.integer value origin) with
      | some row => row.text
      | none => toString value
    literalDecimal := fun coefficient exponent format negativeZero origin =>
      match inputs.literalSpellings.find? (·.input ==
          Candidate.Scalar.decimal coefficient exponent format negativeZero origin) with
      | some row => row.text
      | none => ""
    parseLiteral := fun format text =>
      (inputs.literalReadings.find? fun row => row.format == format && row.text == text).bind
        (·.result)
    encodeInteger := toString
    encodeDecimal := fun coefficient exponent format negativeZero =>
      match inputs.decimalSpellings.find? (fun row =>
        row.number = ⟨⟨coefficient, exponent⟩, format, negativeZero⟩) with
      | some row => row.text
      | none => encodeDecimal coefficient exponent
    schemaNumber := fun text => match inputs.numberReadings.find? (·.text = text) with
      | some row => row.schema
      | none => parseNumber text
    decodeInteger := fun format text =>
      (inputs.integerReadings.find? fun row => row.format == format && row.text == text).bind
        (·.scalar)
    decodeKeyInteger := fun format text =>
      (inputs.integerReadings.find? fun row => row.format == format && row.text == text).bind
        (·.key)
    decodeDecimal := fun format text =>
      (inputs.decimalReadings.find? fun row => row.format == format && row.text == text).bind
        (·.scalar)
    decodeKeyDecimal := fun format text =>
      (inputs.decimalReadings.find? fun row => row.format == format && row.text == text).bind
        (·.key)
  }

end ValueContract.Reference
