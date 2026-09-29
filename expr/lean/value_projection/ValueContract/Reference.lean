/-
JSON-lines executable over the proved candidate functions. This file translates
protocol data and records observations; it contains no replacement semantic
resolver, branch selector, schema checker or projector.
-/
import ValueContract.ReferenceCodecs

open Lean

namespace ValueContract.Reference
open Candidate

structure ExternalCheck where
  identity : Nat
  accepted : List Candidate.Scalar
  deriving ToJson, FromJson

structure ProjectionRequest where
  targets : Targets
  root : Identity
  use : TargetUse
  deriving ToJson, FromJson

structure EvaluationRequest where
  declarations : Candidate.Declarations
  root : Identity
  supplied : Supplied Input
  codecs : CodecInputs
  checks : List ExternalCheck
  projection : Option ProjectionRequest
  deriving ToJson, FromJson

structure DecoderRequest where
  targets : Targets
  root : Identity
  wire : Candidate.Wire
  codecs : CodecInputs
  checks : List ExternalCheck
  deriving ToJson, FromJson

inductive Command where
  | selectExample (reachable suppressGenerated : Bool)
      (sources : SourceSelection.ExampleSources Input)
  | selectContract (reachable : Bool) (supplied : Option (Supplied Input))
  | evaluate (request : EvaluationRequest)
  | decode (request : DecoderRequest)
  | codecs (bytes : List (List UInt8)) (texts : List String) (decimals : List Decimal)
  deriving ToJson, FromJson

structure Request where
  version : Nat
  command : Command
  deriving ToJson, FromJson

private def outcomeJson [ToJson ε] [ToJson α] : Except ε α → Json
  | .error error => Json.mkObj [("error", toJson error)]
  | .ok value => Json.mkObj [("ok", toJson value)]

private partial def wireTexts : Candidate.Wire → List String
  | .number text | .text text => [text]
  | .array items => items.flatMap wireTexts
  | .object fields => fields.flatMap fun (key, value) => key :: wireTexts value
  | .oneof _ value => wireTexts value
  | _ => []

private partial def valueTexts (codecs : ScalarCodecs) : Candidate.Value → List String
  | .scalar scalar => wireTexts (encodeScalar codecs .json scalar)
  | .nilBytes => [codecs.bytes.encode []]
  | .object fields additional => (fields.flatMap fun (_, value) => valueTexts codecs value) ++
      additional.flatMap (fun (key, value) => key :: valueTexts codecs value)
  | .array items => items.flatMap (valueTexts codecs)
  | .map entries => entries.flatMap fun (key, value) =>
      (encodeKey codecs.numbers key).toList ++ valueTexts codecs value
  | .union _ _ payload | .host _ payload | .any payload => valueTexts codecs payload
  | .jsonSnapshot wire => wireTexts wire
  | _ => []

private def targetTexts (codecs : ScalarCodecs) (target : TargetDeclaration) : List String :=
  target.enumeration.toList.flatten.flatMap (valueTexts codecs) ++
  target.schemaEnumeration.toList.flatten.flatMap wireTexts ++
  match target.target with
  | .object fields _ => fields.flatMap fun field => match field.presence with
      | .implicitDefault scalar => wireTexts (encodeScalar codecs .json scalar)
      | _ => []
  | _ => []

private def scalarDecimals : Candidate.Scalar → List DecimalIdentity
  | .integer value _ => [⟨⟨value, 0⟩, .exact, false⟩]
  | .decimal coefficient exponent format negativeZero _ =>
      [⟨⟨coefficient, exponent⟩, format, negativeZero⟩]
  | _ => []

/-- Preserve concrete source admission through codec-request collection. The
canonical scalar alone cannot tell whether numeric normalization may run. -/
private partial def inputSourceScalars : Input → List SourceScalar
  | .scalar scalar => [scalar]
  | .byteSequence sequence => sequence.items.map fun item =>
      ⟨.integer (Int.ofNat item.2.toNat), .builtinUInt8⟩
  | .object fields => fields.flatMap (fun (_, value) => inputSourceScalars value)
  | .array values => values.flatMap inputSourceScalars
  | .map entries => entries.flatMap (fun (key, value) => key :: inputSourceScalars value)
  | .selected _ _ value | .host _ value => inputSourceScalars value
  | _ => []

private def inputScalars (input : Input) : List Candidate.Scalar :=
  (inputSourceScalars input).map (·.value)

private partial def valueScalars : Candidate.Value → List Candidate.Scalar
  | .scalar scalar => [scalar]
  | .object fields additional => fields.flatMap (fun (_, value) => valueScalars value) ++
      additional.flatMap (fun (_, value) => valueScalars value)
  | .array values => values.flatMap valueScalars
  | .map entries => entries.flatMap (fun (key, value) => key :: valueScalars value)
  | .union _ _ value | .host _ value | .any value => valueScalars value
  | _ => []

private def valueDecimals (value : Candidate.Value) : List DecimalIdentity :=
  (valueScalars value).flatMap scalarDecimals

/-- Conservatively collect declaration/input coercion pairs before resolving.
Rejected concrete primitives need no literal callback; explicit abstract controls
retain their own admission rule. Prepared enums are already canonical values and
never enter this raw-source operation. -/
private def sourceCoercions (declaration : Candidate.Declaration)
    (inputs : List SourceScalar) : List (NumericFormat × Candidate.Scalar) :=
  let rules := match declaration.contract with
    | .scalar .decimal rules | .map (.scalar .decimal) rules _ _ => some rules
    | _ => none
  match rules with
  | none => []
  | some rules =>
    if rules.numericFormat == .exact then [] else
    inputs.filterMap fun input =>
      if sourceAdmits (rules.sourcePrimitive.getD (.abstract .decimal)) input.primitive &&
          (scalarNumber input.value).isSome then
        some (rules.numericFormat, input.value)
      else none

private def targetDecimals (target : TargetDeclaration) : List DecimalIdentity :=
  target.enumeration.toList.flatten.flatMap valueDecimals ++
  match target.target with
  | .object fields _ => fields.flatMap fun field => match field.presence with
      | .implicitDefault scalar => scalarDecimals scalar
      | _ => []
  | _ => []

private def targetIntegerFormats (target : TargetDeclaration) : List IntegerFormat :=
  match target.target with
  | .scalar .json .integer rules | .map (.scalar .integer) rules _ _ => [rules.integerFormat]
  | _ => []

private def targetDecimalFormats (target : TargetDeclaration) : List NumericFormat :=
  match target.target with
  | .scalar .json .decimal rules | .map (.scalar .decimal) rules _ _ => [rules.numericFormat]
  | _ => []

private def targetKeyDecimalFormats (target : TargetDeclaration) : List NumericFormat :=
  match target.target with
  | .map (.scalar .decimal) rules _ _ => [rules.numericFormat]
  | _ => []

/-- Enumerate the lexical operations independently of decoder outcomes. A
rejected competing branch still requires all of its precision-specific rows. -/
private def targetCodecRequests (inputs : CodecInputs) (targets : Targets)
    (texts : List String) : List (String × Json) := Id.run do
  let missing := texts.filter fun text => !(inputs.numberReadings.any (·.text == text))
  let integerFormats := (targets.flatMap targetIntegerFormats).eraseDups
  let decimalFormats := (targets.flatMap targetDecimalFormats).eraseDups
  let missingIntegers := (integerFormats.flatMap fun format => texts.map (format, ·)).filter
    fun (format, text) => !(inputs.integerReadings.any fun row =>
      row.format == format && row.text == text)
  let missingDecimals := (decimalFormats.flatMap fun format => texts.map (format, ·)).filter
    fun (format, text) => !(inputs.decimalReadings.any fun row =>
      row.format == format && row.text == text)
  -- Target enum and observation equality inspect decoded map keys, whose
  -- rounded value can differ from every source value. Close the encoding
  -- boundary over these parser outputs before any semantic conclusion.
  let keyFormats := (targets.flatMap targetKeyDecimalFormats).eraseDups
  let decodedKeys := inputs.decimalReadings.filterMap fun row =>
    if keyFormats.contains row.format && texts.contains row.text then
      row.key.map fun value => (⟨value.value, row.format, value.negativeZero⟩ : DecimalIdentity)
    else none
  let missingSpellings := decodedKeys.eraseDups.filter fun number =>
    !(inputs.decimalSpellings.any (·.number == number))
  let mut requests := []
  if !missing.isEmpty then requests := requests ++ [("numericRequests", toJson missing)]
  if !missingIntegers.isEmpty then
    requests := requests ++ [("integerReadingRequests", toJson missingIntegers)]
  if !missingDecimals.isEmpty then
    requests := requests ++ [("decimalReadingRequests", toJson missingDecimals)]
  if !missingSpellings.isEmpty then
    requests := requests ++ [("decimalRequests", toJson missingSpellings)]
  return requests

private def externalChecks (checks : List ExternalCheck) : ExternalScalarChecks :=
  fun id scalar => checks.any fun check => check.identity == id &&
    check.accepted.any (fun accepted => accepted == scalar)

private def evaluate (request : EvaluationRequest) : Json := Id.run do
  -- Map-key collisions already use the encoder during source resolution.
  -- Require its machine-precision boundary before invoking that operation.
  -- Integer pairs also cover the resolver's integer-to-decimal coercion.
  let coercions := (request.declarations.flatMap fun declaration =>
    sourceCoercions declaration (inputSourceScalars request.supplied.value)).eraseDups
  let missingLiterals := (coercions.map Prod.snd).eraseDups.filter fun input =>
    !(request.codecs.literalSpellings.any (·.input == input))
  if !missingLiterals.isEmpty then
    return Json.mkObj [("literalSpellingRequests", toJson missingLiterals)]
  let readings := (coercions.filterMap fun (format, input) =>
    (request.codecs.literalSpellings.find? (·.input == input)).map fun row =>
      (format, row.text)).eraseDups
  let missingReadings := readings.filter fun (format, text) =>
    !(request.codecs.literalReadings.any fun row => row.format == format && row.text == text)
  if !missingReadings.isEmpty then
    return Json.mkObj [("literalReadingRequests", toJson missingReadings)]
  let normalized := request.codecs.literalReadings.flatMap fun row =>
    if readings.contains (row.format, row.text) then row.result.toList.map fun result =>
      (⟨result.value, row.format, result.negativeZero⟩ : DecimalIdentity)
    else []
  let decimals := ((inputScalars request.supplied.value).flatMap scalarDecimals ++ normalized ++
    request.declarations.flatMap (fun declaration =>
      declaration.enumeration.toList.flatten.flatMap valueDecimals) ++
    request.projection.toList.flatMap (fun projection =>
      projection.targets.flatMap targetDecimals)).eraseDups
  let missingDecimals := decimals.filter fun value =>
    !(request.codecs.decimalSpellings.any (·.number == value))
  if !missingDecimals.isEmpty then
    return Json.mkObj [("decimalRequests", toJson missingDecimals)]
  let codecs := scalarCodecs request.codecs
  let checks := externalChecks request.checks
  let resolved := resolve request.declarations checks codecs.numbers
    request.supplied.source.role request.root request.supplied.value
  let mut result := [
    ("source", toJson request.supplied.source),
    ("declarationsValid", toJson (validateDeclarations request.declarations)),
    ("resolved", outcomeJson resolved)]
  if let some projection := request.projection then
    result := result ++ [("targetsValid", toJson (validateTargets projection.targets))]
    if let .ok value := resolved then
      -- Request actual codec readings for every lexical input the projector
      -- can inspect. Missing rows must not silently use exact-decimal fallback
      -- as evidence for a machine-number decoder.
      let built := build codecs projection.targets request.supplied.source.role
        projection.root value.value
      let builtTexts := match built with
        | .ok (_, some wire) => wireTexts wire
        | _ => []
      let texts := (valueTexts codecs value.value ++ builtTexts ++
        projection.targets.flatMap (targetTexts codecs)).eraseDups
      let missing := targetCodecRequests request.codecs projection.targets texts
      if !missing.isEmpty then return Json.mkObj (result ++ missing)
      let role := request.supplied.source.role
      result := result ++ [
        ("observed", outcomeJson (observeValue codecs projection.targets role
          projection.root value.value)),
        ("projected", toJson (project codecs checks projection.targets projection.use
          role projection.root value))]
  return Json.mkObj result

private def decodeRequest (request : DecoderRequest) : Json := Id.run do
  let missingDecimals := (request.targets.flatMap targetDecimals).eraseDups.filter fun value =>
    !(request.codecs.decimalSpellings.any (·.number == value))
  if !missingDecimals.isEmpty then
    return Json.mkObj [("decimalRequests", toJson missingDecimals)]
  let codecs := scalarCodecs request.codecs
  let texts := (wireTexts request.wire ++ request.targets.flatMap (targetTexts codecs)).eraseDups
  let missing := targetCodecRequests request.codecs request.targets texts
  if !missing.isEmpty then return Json.mkObj missing
  let checks := externalChecks request.checks
  return Json.mkObj [
    ("targetsValid", toJson (validateTargets request.targets)),
    ("schema", outcomeJson (schema codecs checks request.targets request.root request.wire)),
    ("decoded", outcomeJson (decode codecs checks request.targets request.root request.wire))]

private def integerFormatSupported : IntegerFormat → Bool
  | .mathematical => true
  | .signed bits | .unsigned bits => bits == 32 || bits == 64

private def validateCodecs (inputs : CodecInputs) (targets : Targets) : Except String Unit := do
  let literals := inputs.literalSpellings.map (·.input)
  let literalReads := inputs.literalReadings.map (fun row => (row.format, row.text))
  let integers := inputs.integerReadings.map (fun row => (row.format, row.text))
  let decimals := inputs.decimalReadings.map (fun row => (row.format, row.text))
  if literals.eraseDups.length != literals.length then throw "duplicate numeric literal spelling"
  if literalReads.eraseDups.length != literalReads.length then throw "duplicate numeric literal reading"
  if literals.any (fun value => (scalarNumber value).isNone) then throw "literal codec input is not numeric"
  if integers.eraseDups.length != integers.length then throw "duplicate integer codec reading"
  if decimals.eraseDups.length != decimals.length then throw "duplicate decimal codec reading"
  if !((targets.flatMap targetIntegerFormats ++ inputs.integerReadings.map (·.format)).all
      integerFormatSupported) then throw "unsupported runtime integer width"
  let values := inputs.decimalSpellings.map (·.number)
  let texts := inputs.numberReadings.map (·.text)
  if values.eraseDups.length != values.length then throw "duplicate decimal codec spelling"
  if texts.eraseDups.length != texts.length then throw "duplicate numeric codec reading"

def execute (request : Request) : Except String Json := do
  if request.version != 1 then throw "unsupported reference protocol version"
  match request.command with
  | .evaluate evaluation =>
      validateCodecs evaluation.codecs (evaluation.projection.toList.flatMap (·.targets))
  | .decode decoding => validateCodecs decoding.codecs decoding.targets
  | _ => pure ()
  let result := match request.command with
    | .selectExample reachable suppressed sources =>
        toJson (SourceSelection.selectExample reachable suppressed sources)
    | .selectContract reachable supplied => toJson (SourceSelection.selectContract reachable supplied)
    | .evaluate request => evaluate request
    | .decode request => decodeRequest request
    | .codecs bytes texts decimals => Json.mkObj [
        ("encodedBytes", toJson (bytes.map encodeBytes)),
        ("decodedBytes", toJson (texts.map decodeBytes)),
        ("schemaNumbers", toJson (texts.map parseNumber)),
        ("integerNumbers", toJson (texts.map parseInteger)),
        ("encodedDecimals", toJson (decimals.map fun value =>
          encodeDecimal value.coefficient value.exponent))]
  return Json.mkObj [("version", toJson request.version), ("result", result)]

end ValueContract.Reference

/-- One request and response per line. Protocol errors are fatal and never count
as a semantic rejection or a successful conformance case. -/
def main : IO UInt32 := do
  let stdin ← IO.getStdin
  let stdout ← IO.getStdout
  let stderr ← IO.getStderr
  repeat
    let line ← stdin.getLine
    if line.isEmpty then return 0
    let result := do
      let json ← Lean.Json.parse line
      let request : ValueContract.Reference.Request ← Lean.fromJson? json
      ValueContract.Reference.execute request
    match result with
    | .error message =>
        stderr.putStrLn ("value contract reference: " ++ message)
        return 1
    | .ok response =>
        stdout.putStrLn response.compress
        stdout.flush
  return 0
