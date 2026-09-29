import ValueContract.Typing

namespace ValueContract.Candidate

def encodeScalar (codecs : ScalarCodecs) (encoding : Encoding) : Scalar → Wire
  | .boolean value => .boolean value
  | .integer value _ => match encoding with
    | .json => .number (codecs.numbers.encodeInteger value)
    | .protobuf => .integer value
  | .decimal coefficient exponent format negativeZero _ => match encoding with
    | .json => .number (codecs.numbers.encodeDecimal coefficient exponent format negativeZero)
    | .protobuf => .decimal coefficient exponent
  | .string value => .text value
  | .bytes value => match encoding with
    | .json => .text (codecs.bytes.encode value)
    | .protobuf => .bytes value

/-- Independent canonical relation. Numeric kinds and Bytes/String erase to
shared JSON lexical constructors BEFORE schema or runtime branch matching. -/
inductive CanonicalScalar (codecs : ScalarCodecs) : Encoding → Scalar → Wire → Prop where
  | boolean : CanonicalScalar codecs encoding (.boolean value) (.boolean value)
  | jsonInteger : CanonicalScalar codecs .json (.integer value origin)
      (.number (codecs.numbers.encodeInteger value))
  | jsonDecimal : CanonicalScalar codecs .json (.decimal coefficient exponent format negativeZero origin)
      (.number (codecs.numbers.encodeDecimal coefficient exponent format negativeZero))
  | protobufInteger : CanonicalScalar codecs .protobuf (.integer value origin) (.integer value)
  | protobufDecimal : CanonicalScalar codecs .protobuf (.decimal coefficient exponent format negativeZero origin)
      (.decimal coefficient exponent)
  | string : CanonicalScalar codecs encoding (.string value) (.text value)
  | jsonBytes : CanonicalScalar codecs .json (.bytes value) (.text (codecs.bytes.encode value))
  | protobufBytes : CanonicalScalar codecs .protobuf (.bytes value) (.bytes value)

def decodeScalar (codecs : ScalarCodecs) (encoding : Encoding) (kind : ScalarKind)
    (format : NumericFormat) (integers : IntegerFormat) : Wire → Option Scalar
  | .boolean value => if kind = .boolean then some (.boolean value) else none
  | .number text => if encoding = .json then
      match kind with
      | .integer => (codecs.numbers.decodeInteger integers text).map Scalar.integer
      | .decimal => (codecs.numbers.decodeDecimal format text).map
          (fun value => .decimal value.value.coefficient value.value.exponent format value.negativeZero)
      | _ => none
    else none
  | .integer value => if encoding = .protobuf ∧ kind = .integer then
      some (.integer value) else none
  | .decimal coefficient exponent => if encoding = .protobuf ∧ kind = .decimal then
      some (.decimal coefficient exponent format) else none
  | .text value => if kind = .string then some (.string value)
      else if kind = .bytes ∧ encoding = .json then (codecs.bytes.decode value).map Scalar.bytes
      else none
  | .bytes value => if kind = .bytes ∧ encoding = .protobuf then some (.bytes value) else none
  | _ => none

/-- Runtime relation is independent of decodeScalar and of schema acceptance.
In particular, integer/decimal decoding sees the SAME numeric text, without
access to the source's kind or selected branch. -/
inductive ScalarDecodes (codecs : ScalarCodecs) (format : NumericFormat)
    (integers : IntegerFormat) : Encoding → ScalarKind → Wire → Scalar → Prop where
  | boolean : ScalarDecodes codecs format integers encoding .boolean (.boolean value) (.boolean value)
  | jsonInteger (decoded : codecs.numbers.decodeInteger integers text = some value) :
      ScalarDecodes codecs format integers .json .integer (.number text) (.integer value)
  | jsonDecimal (decoded : codecs.numbers.decodeDecimal format text = some value) :
      ScalarDecodes codecs format integers .json .decimal (.number text)
        (.decimal value.value.coefficient value.value.exponent format value.negativeZero)
  | protobufInteger : ScalarDecodes codecs format integers .protobuf .integer (.integer value) (.integer value)
  | protobufDecimal : ScalarDecodes codecs format integers .protobuf .decimal (.decimal coefficient exponent)
      (.decimal coefficient exponent format)
  | string : ScalarDecodes codecs format integers encoding .string (.text value) (.string value)
  | jsonBytes (decoded : codecs.bytes.decode text = some value) :
      ScalarDecodes codecs format integers .json .bytes (.text text) (.bytes value)
  | protobufBytes : ScalarDecodes codecs format integers .protobuf .bytes (.bytes value) (.bytes value)

/-- JSON byte schemas use lexical grammar and canonical enum spellings, not
whether a decoder maps a noncanonical spelling to a semantic enum member. -/
def jsonByteSchema (codec : ByteCodec) (checks : ExternalScalarChecks)
    (rules : ScalarRules) (text : String) : Bool :=
  (base64Length text).any (lengthAllowed rules.length) &&
    rules.enumeration.all (fun values => values.any fun value => match value with
      | .bytes bytes => text == codec.encode bytes
      | _ => false) &&
    rules.externalChecks.all (fun key => checks key (.string text))

/-- JSON numeric schema meaning has one lexical parser. Mathematical integrality
is distinct from the runtime integer decoder's lexical acceptance. -/
def jsonNumberSchema (codec : NumericCodec) (checks : ExternalScalarChecks)
    (kind : ScalarKind) (rules : ScalarRules) (text : String) : Bool :=
  (codec.schemaNumber text).any fun number => match kind with
    | .integer => number.asInteger.any (fun value => scalarAllowed checks rules (.integer value))
    | .decimal => scalarAllowed checks rules (.decimal number.coefficient number.exponent)
    | _ => false

def scalarSchema (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (encoding : Encoding) (kind : ScalarKind) (rules : ScalarRules) (wire : Wire) : Bool :=
  match encoding, kind, wire with
  | .json, .bytes, .text text => jsonByteSchema codecs.bytes checks rules text
  | .json, kind, .number text => jsonNumberSchema codecs.numbers checks kind rules text
  | _, .boolean, .boolean value => scalarAllowed checks rules (.boolean value)
  | .protobuf, .integer, .integer value => scalarAllowed checks rules (.integer value)
  | .protobuf, .decimal, .decimal coefficient exponent =>
      scalarAllowed checks rules (.decimal coefficient exponent)
  | _, .string, .text value => scalarAllowed checks rules (.string value)
  | .protobuf, .bytes, .bytes value => scalarAllowed checks rules (.bytes value)
  | _, _, _ => false

/-- Numerical preservation compares exact meaning, not an incidental decimal
coefficient/exponent spelling. Scalar kind is preserved independently. -/
def SameScalar (left right : Scalar) : Prop :=
  left.kind = right.kind ∧ scalarEqual left right = true

def ScalarRepresentable (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (encoding : Encoding) (kind : ScalarKind) (rules : ScalarRules) (value : Scalar) : Prop :=
  value.kind = kind ∧ scalarAllowed checks rules value = true ∧
    ∃ wire decoded, CanonicalScalar codecs encoding value wire ∧
      scalarSchema codecs checks encoding kind rules wire = true ∧
      ScalarDecodes codecs rules.numericFormat rules.integerFormat encoding kind wire decoded ∧ SameScalar value decoded ∧
      scalarAllowed checks rules decoded = true

def scalarPreserved (checks : ExternalScalarChecks) (rules : ScalarRules)
    (value decoded : Scalar) : Bool :=
  value.kind == decoded.kind && scalarEqual value decoded && scalarAllowed checks rules decoded

/-- Canonical construction followed by independent schema and runtime checks.
There is no source replacement or semantic-kind tag in JSON numeric matching. -/
def projectScalar (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (encoding : Encoding) (kind : ScalarKind) (rules : ScalarRules) (value : Scalar) :
    Option Wire :=
  let wire := encodeScalar codecs encoding value
  if value.kind = kind ∧ scalarAllowed checks rules value = true ∧
      scalarSchema codecs checks encoding kind rules wire = true ∧
      (decodeScalar codecs encoding kind rules.numericFormat rules.integerFormat wire).any (scalarPreserved checks rules value) = true then
    some wire
  else none

theorem canonicalScalar_realized {codecs encoding value wire}
    (canonical : CanonicalScalar codecs encoding value wire) :
    encodeScalar codecs encoding value = wire := by
  cases canonical <;> rfl

theorem canonicalScalar_exists (codecs : ScalarCodecs) (encoding : Encoding) (value : Scalar) :
    CanonicalScalar codecs encoding value (encodeScalar codecs encoding value) := by
  cases value with
  | boolean => exact .boolean
  | integer => cases encoding with
    | json => exact .jsonInteger
    | protobuf => exact .protobufInteger
  | decimal => cases encoding with
    | json => exact .jsonDecimal
    | protobuf => exact .protobufDecimal
  | string => exact .string
  | bytes => cases encoding with
    | json => exact .jsonBytes
    | protobuf => exact .protobufBytes

theorem canonicalScalar_unique {codecs encoding value left right}
    (a : CanonicalScalar codecs encoding value left)
    (b : CanonicalScalar codecs encoding value right) : left = right := by
  rw [← canonicalScalar_realized a, ← canonicalScalar_realized b]

theorem scalarDecoder_progress {codecs encoding kind format integers wire value}
    (decoded : ScalarDecodes codecs format integers encoding kind wire value) :
    decodeScalar codecs encoding kind format integers wire = some value := by
  cases decoded <;> simp_all [decodeScalar]

theorem scalarDecoder_sound {codecs encoding kind format integers wire value}
    (decoded : decodeScalar codecs encoding kind format integers wire = some value) :
    ScalarDecodes codecs format integers encoding kind wire value := by
  cases wire <;> cases kind <;> cases encoding <;> simp_all [decodeScalar]
  all_goals try { subst value; constructor }
  all_goals
    obtain ⟨actual, found, equal⟩ := decoded
    subst value
    constructor
    exact found

theorem scalarPreserved_iff {checks rules value decoded} :
    scalarPreserved checks rules value decoded = true ↔
      SameScalar value decoded ∧ scalarAllowed checks rules decoded = true := by
  simp [scalarPreserved, SameScalar, and_assoc]

/-- Success proves schema validity and actual modeled decoding preservation. -/
theorem scalarProjection_sound {codecs checks encoding kind rules value wire}
    (success : projectScalar codecs checks encoding kind rules value = some wire) :
    value.kind = kind ∧ scalarAllowed checks rules value = true ∧
      CanonicalScalar codecs encoding value wire ∧
      scalarSchema codecs checks encoding kind rules wire = true ∧
      ∃ decoded, ScalarDecodes codecs rules.numericFormat rules.integerFormat encoding kind wire decoded ∧ SameScalar value decoded ∧
        scalarAllowed checks rules decoded = true := by
  unfold projectScalar at success
  dsimp only at success
  split at success
  next valid =>
    cases Option.some.inj success
    have existsDecoded : ∃ decoded,
        decodeScalar codecs encoding kind rules.numericFormat rules.integerFormat (encodeScalar codecs encoding value) = some decoded ∧
        scalarPreserved checks rules value decoded = true := by
      cases found : decodeScalar codecs encoding kind rules.numericFormat rules.integerFormat (encodeScalar codecs encoding value) with
      | none => simp [found] at valid
      | some decoded => exact ⟨decoded, rfl, by simpa [found] using valid.2.2.2⟩
    obtain ⟨decoded, decodedAt, preserved⟩ := existsDecoded
    exact ⟨valid.1, valid.2.1, canonicalScalar_exists _ _ _, valid.2.2.1,
      decoded, scalarDecoder_sound decodedAt, (scalarPreserved_iff.mp preserved).1,
      (scalarPreserved_iff.mp preserved).2⟩
  next => contradiction

/-- Independent representability includes the canonical output, schema and actual
runtime meaning. It is not defined by projectScalar or a preferred branch. -/
theorem scalarProjection_progress {codecs checks encoding kind rules value}
    (representable : ScalarRepresentable codecs checks encoding kind rules value) :
    ∃ wire, projectScalar codecs checks encoding kind rules value = some wire := by
  obtain ⟨kindMatches, allowed, wire, decoded, canonical, schema, runtime, same, runtimeAllowed⟩ :=
    representable
  have emitted := canonicalScalar_realized canonical
  have decodedAt := scalarDecoder_progress runtime
  have preserved := scalarPreserved_iff.mpr ⟨same, runtimeAllowed⟩
  exact ⟨wire, by simp [projectScalar, emitted, kindMatches, allowed, schema, decodedAt, preserved]⟩

end ValueContract.Candidate
