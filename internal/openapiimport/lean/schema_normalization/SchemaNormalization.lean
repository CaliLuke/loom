import Std

namespace OpenAPIImport.SchemaNormalization

/-- OAS 3.0 uses its Schema Object nullability rule. OAS 3.1 and 3.2 use
JSON Schema 2020-12 semantics for schema assertions. -/
inductive Version where
  | oas30
  | oas31
  | oas32
  deriving DecidableEq, Repr

/-- Representative JSON instances: two strings distinguish enum membership;
objects distinguish absent, valid, and invalid `name` values. -/
inductive Value where
  | null
  | stringX
  | stringOther
  | objectWithoutName
  | objectNameX
  | objectNameInvalid
  | booleanTrue
  | booleanFalse
  deriving DecidableEq, Repr

/-- Finite source-schema shapes for the four importer bugs and controls. -/
inductive Source where
  | booleanTrue
  | booleanFalse
  | emptyObject
  | typelessStringProperty
  | explicitObjectStringProperty
  | stringType
  | enumStringType
  | legacyNullableString
  | constrainedNullAnyOf
  | bareNullableAnyOf
  | modernStringNullUnion
  | nullOnly
  deriving DecidableEq, Repr

/-- Normal forms the proposed importer can emit. There is deliberately no
bottom/false form; unsupported source schemas must fail closed. -/
inductive Normalized where
  | any
  | string
  | nullableString
  | enumString
  | objectStringProperty
  deriving DecidableEq, Repr

def modernVersion : Version → Bool
  | .oas30 => false
  | .oas31 => true
  | .oas32 => true

def inputValid : Version → Source → Bool
  | .oas30, .booleanTrue => false
  | .oas30, .booleanFalse => false
  | .oas30, .constrainedNullAnyOf => false
  | .oas30, .bareNullableAnyOf => false
  | .oas30, .modernStringNullUnion => false
  | .oas30, .nullOnly => false
  | _, _ => true

def isString : Value → Bool
  | .stringX => true
  | .stringOther => true
  | _ => false

def sourceAccepts (version : Version) : Source → Value → Bool
  | .booleanTrue, _ => true
  | .booleanFalse, _ => false
  | .emptyObject, _ => true
  | .typelessStringProperty, .objectWithoutName => true
  | .typelessStringProperty, .objectNameX => true
  | .typelessStringProperty, .objectNameInvalid => false
  | .typelessStringProperty, _ => true
  | .explicitObjectStringProperty, .objectWithoutName => true
  | .explicitObjectStringProperty, .objectNameX => true
  | .explicitObjectStringProperty, .objectNameInvalid => false
  | .explicitObjectStringProperty, _ => false
  | .stringType, value => isString value
  | .enumStringType, value => value == .stringX
  | .legacyNullableString, value => isString value || (version == .oas30 && value == .null)
  | .constrainedNullAnyOf, value => isString value
  | .bareNullableAnyOf, value =>
      modernVersion version && (isString value || value == .null)
  | .modernStringNullUnion, value =>
      modernVersion version && (isString value || value == .null)
  | .nullOnly, value => modernVersion version && value == .null

def normalizedAccepts : Normalized → Value → Bool
  | .any, _ => true
  | .string, value => isString value
  | .nullableString, value => isString value || value == .null
  | .enumString, value => value == .stringX
  | .objectStringProperty, .objectWithoutName => true
  | .objectStringProperty, .objectNameX => true
  | .objectStringProperty, .objectNameInvalid => false
  | .objectStringProperty, _ => false

/-- Abstracted pre-fix behavior at the known lossy decision points. -/
def legacyNormalize (version : Version) : Source → Normalized
  | .booleanTrue => .any
  | .booleanFalse => .any
  | .emptyObject => .any
  | .typelessStringProperty => .objectStringProperty
  | .explicitObjectStringProperty => .objectStringProperty
  | .stringType => .string
  | .enumStringType => .enumString
  | .legacyNullableString => .nullableString
  | .constrainedNullAnyOf => .nullableString
  | .bareNullableAnyOf => .nullableString
  | .modernStringNullUnion => if modernVersion version then .nullableString else .string
  | .nullOnly => .any

/-- Candidate admission preserves all constraints represented in this model.
Unsupported forms return `none` instead of becoming an unconstrained schema. -/
def candidateNormalize (version : Version) : Source → Option Normalized
  | .booleanTrue => if modernVersion version then some .any else none
  | .booleanFalse => none
  | .emptyObject => some .any
  | .typelessStringProperty => none
  | .explicitObjectStringProperty => some .objectStringProperty
  | .stringType => some .string
  | .enumStringType => some .enumString
  | .legacyNullableString =>
      if version == .oas30 then some .nullableString else none
  | .constrainedNullAnyOf => none
  | .bareNullableAnyOf =>
      if modernVersion version then some .nullableString else none
  | .modernStringNullUnion =>
      if modernVersion version then some .nullableString else none
  | .nullOnly => none

/-- Every admitted normalization has the same acceptance on the representative
instance domain. This theorem covers all versions and every source constructor. -/
theorem candidate_preserves_acceptance
    (version : Version) (source : Source) (normalized : Normalized)
    (valid : inputValid version source = true)
    (admitted : candidateNormalize version source = some normalized) :
    ∀ value, sourceAccepts version source value = normalizedAccepts normalized value := by
  intro value
  cases version <;> cases source <;> cases normalized <;> cases value <;>
    simp_all [inputValid, candidateNormalize, sourceAccepts, normalizedAccepts,
      modernVersion, isString]

/-- `true` and `{}` are equivalent unconstrained schemas and normalize to Any. -/
theorem true_and_empty_are_any :
    candidateNormalize .oas31 .booleanTrue = some .any ∧
      candidateNormalize .oas32 .booleanTrue = some .any ∧
      candidateNormalize .oas31 .emptyObject = some .any := by
  decide

/-- A bare null branch beside a string branch is an exact nullable simplification. -/
theorem bare_null_nullable_is_preserved :
    candidateNormalize .oas31 .bareNullableAnyOf = some .nullableString ∧
      candidateNormalize .oas32 .bareNullableAnyOf = some .nullableString := by
  decide

/-- Modern type unions are supported only by the OAS 3.1/3.2 dialects. -/
theorem modern_union_control :
    candidateNormalize .oas31 .modernStringNullUnion = some .nullableString ∧
      candidateNormalize .oas32 .modernStringNullUnion = some .nullableString ∧
      candidateNormalize .oas30 .modernStringNullUnion = none := by
  decide

/-- OAS 3.0 `nullable: true` adds null to an explicit type. -/
theorem oas30_nullable_control :
    sourceAccepts .oas30 .legacyNullableString .null = true ∧
      sourceAccepts .oas31 .legacyNullableString .null = false ∧
      sourceAccepts .oas32 .legacyNullableString .null = false := by
  decide

/-- The constrained null composition is rejected until the importer IR has a
lossless representation for the full composition. -/
theorem constrained_null_composition_is_rejected :
    candidateNormalize .oas31 .constrainedNullAnyOf = none ∧
      candidateNormalize .oas32 .constrainedNullAnyOf = none := by
  decide

/-- The old false-to-Any rule admits values rejected by the source. -/
theorem counterexample_false_schema_becomes_any :
    sourceAccepts .oas31 .booleanFalse .null = false ∧
      normalizedAccepts (legacyNormalize .oas31 .booleanFalse) .null = true := by
  simp [sourceAccepts, normalizedAccepts, legacyNormalize]

/-- Typeless `properties` leaves scalars unconstrained; inferring object narrows
that schema's accepted values. -/
theorem counterexample_typeless_properties_infers_object :
    sourceAccepts .oas31 .typelessStringProperty .booleanTrue = true ∧
      normalizedAccepts (legacyNormalize .oas31 .typelessStringProperty) .booleanTrue = false := by
  simp [sourceAccepts, normalizedAccepts, legacyNormalize]

/-- For `anyOf: [{type: string}, {type: null, enum: ["forbidden"]}]`, the
second branch accepts no values; legacy reduction nevertheless admits null. -/
theorem counterexample_constrained_null_composition_adds_null :
    sourceAccepts .oas31 .constrainedNullAnyOf .stringOther = true ∧
      sourceAccepts .oas31 .constrainedNullAnyOf .null = false ∧
      normalizedAccepts (legacyNormalize .oas31 .constrainedNullAnyOf) .null = true := by
  simp [sourceAccepts, normalizedAccepts, legacyNormalize, isString]

/-- Applying OAS 3.0 `nullable` semantics in 3.1/3.2 broadens a string schema. -/
theorem counterexample_modern_legacy_nullable_keyword :
    sourceAccepts .oas31 .legacyNullableString .null = false ∧
      normalizedAccepts (legacyNormalize .oas31 .legacyNullableString) .null = true ∧
      sourceAccepts .oas32 .legacyNullableString .null = false ∧
      normalizedAccepts (legacyNormalize .oas32 .legacyNullableString) .null = true := by
  simp [sourceAccepts, normalizedAccepts, legacyNormalize, isString]

/-- False, typeless property schemas, and null-only schemas are rejected when the
normalized shape has no exact representation for them. -/
theorem unsupported_shapes_fail_closed :
    candidateNormalize .oas31 .booleanFalse = none ∧
      candidateNormalize .oas31 .typelessStringProperty = none ∧
      candidateNormalize .oas31 .nullOnly = none ∧
      candidateNormalize .oas31 .legacyNullableString = none ∧
      candidateNormalize .oas32 .legacyNullableString = none := by
  decide

end OpenAPIImport.SchemaNormalization

namespace OpenAPIImport.SchemaNormalization.ValueDomain

/-- Membership assertions on an already admitted string/null type. Const is a
singleton; unsupported conjunctions remain outside the modeled subset. -/
inductive Membership where
  | absent | stringX | stringXNull | nullOnly | wrongTypeOnly | mixed | empty | constX | constNull
  deriving DecidableEq, Repr

/-- The DSL separates non-null values and nullability, with no bottom form. -/
inductive Domain where
  | strings (nullable : Bool)
  | enumX (nullable : Bool)
  deriving DecidableEq, Repr

def member : Membership → Value → Bool
  | .absent, _ => true
  | .stringX, v => v == .stringX
  | .stringXNull, v => v == .stringX || v == .null
  | .nullOnly, v => v == .null
  | .wrongTypeOnly, v => v == .booleanTrue
  | .mixed, v => v == .stringX || v == .null || v == .booleanTrue
  | .empty, _ => false
  | .constX, v => v == .stringX
  | .constNull, v => v == .null

def sourceAccepts (nullable : Bool) (membership : Membership) (value : Value) : Bool :=
  (isString value || (nullable && value == .null)) && member membership value

def accepts : Domain → Value → Bool
  | .strings nullable, value => isString value || (nullable && value == .null)
  | .enumX nullable, value => value == .stringX || (nullable && value == .null)

/-- Intersect membership before separating Enum and Nullable. Preserve empty
and null-only intersections as rejections, never as absent validation. -/
def normalize (nullable : Bool) : Membership → Option Domain
  | .absent => some (.strings nullable)
  | .stringX | .constX => some (.enumX false)
  | .stringXNull | .mixed => some (.enumX nullable)
  | .nullOnly | .wrongTypeOnly | .empty | .constNull => none

/-- Current renderer copies nullable independently of enum membership. -/
def legacyWithoutNull (nullable : Bool) : Domain := .enumX nullable

/-- Incomplete repair strips null but loses the empty finite-domain marker. -/
def stripNullOnly (nullable : Bool) : Domain := .strings nullable

/-- Every admitted intersection preserves acceptance for all representatives. -/
theorem normalized_domain_preserves_acceptance
    (nullable : Bool) (membership : Membership) (domain : Domain)
    (admitted : normalize nullable membership = some domain) :
    ∀ value, sourceAccepts nullable membership value = accepts domain value := by
  intro value
  cases nullable <;> cases membership <;> cases domain <;> cases value <;>
    simp_all [normalize, sourceAccepts, member, accepts, isString] <;> decide

/-- Copied nullability accepts a value the finite source rejects. -/
theorem current_nullable_enum_counterexample :
    sourceAccepts true .stringX .null = false ∧
      accepts (legacyWithoutNull true) .null = true := by decide

/-- Stripping a null-only enum admits strings if its emptiness is forgotten. -/
theorem strip_only_counterexample :
    sourceAccepts true .nullOnly .stringOther = false ∧
      accepts (stripNullOnly true) .stringOther = true := by decide

/-- Enum members never widen the source type. -/
theorem nonnullable_enum_null_is_excluded :
    normalize false .stringXNull = some (.enumX false) ∧
      sourceAccepts false .stringXNull .null = false := by decide

/-- Supported forms remain admitted; reject-all is not an adequate repair. -/
theorem valid_domains_remain_representable :
    normalize true .stringXNull = some (.enumX true) ∧
      normalize true .stringX = some (.enumX false) ∧
      normalize true .absent = some (.strings true) ∧
      normalize true .constX = some (.enumX false) := by decide

/-- Empty and null-only domains retain admission failures. -/
theorem null_only_and_empty_domains_rejected :
    normalize true .nullOnly = none ∧ normalize false .nullOnly = none ∧
      normalize true .constNull = none ∧ normalize true .empty = none ∧
      normalize true .wrongTypeOnly = none := by decide

end OpenAPIImport.SchemaNormalization.ValueDomain

namespace OpenAPIImport.SchemaNormalization.ObjectRepresentation

/-- Supported object domains after dialect/type admission. -/
inductive Shape where
  | openOmitted | openTrue | openSchema | closedEmpty | record
  deriving DecidableEq, Repr

/-- Empty field callbacks do not establish objects; collection constructors
need a concrete type. Named declarations and array promotion already exist. -/
inductive Position where
  | declaration | field | mapValue | arrayItem
  deriving DecidableEq, Repr

inductive Representation where
  | freeMap | closedObject | recordObject | accidentalString | missingArgument
  deriving DecidableEq, Repr

def accepts (nullable : Bool) (representation : Representation) (value : Value) : Bool :=
  if value == .null then nullable else
  match representation, value with
  | .freeMap, .objectWithoutName | .freeMap, .objectNameX | .freeMap, .objectNameInvalid => true
  | .closedObject, .objectWithoutName => true
  | .recordObject, .objectNameX => true
  | .accidentalString, .stringX | .accidentalString, .stringOther => true
  | _, _ => false

def canonical : Shape → Representation
  | .openOmitted | .openTrue | .openSchema => .freeMap
  | .closedEmpty => .closedObject
  | .record => .recordObject

/-- Parent acceptance/constructor abstraction at the known failure seams. -/
def parent : Shape → Position → Representation
  | .openOmitted, .field | .closedEmpty, .field => .accidentalString
  | .openOmitted, .mapValue | .closedEmpty, .mapValue | .record, .mapValue => .missingArgument
  | shape, _ => canonical shape

/-- Classify before choosing syntax; existing named types supply the concrete
constructor when a callback cannot establish the object. -/
def candidate (shape : Shape) (_position : Position) : Representation := canonical shape

theorem parent_empty_field_counterexample :
    accepts false (canonical .closedEmpty) .objectWithoutName = true ∧
      accepts false (parent .closedEmpty .field) .objectWithoutName = false := by decide

theorem parent_open_spelling_counterexample :
    parent .openOmitted .field ≠ parent .openTrue .field := by decide

theorem parent_map_constructor_counterexample :
    parent .closedEmpty .mapValue = .missingArgument ∧
      parent .record .mapValue = .missingArgument := by decide

theorem candidate_preserves_acceptance (shape : Shape) (position : Position)
    (nullable : Bool) (value : Value) :
    accepts nullable (candidate shape position) value =
      accepts nullable (canonical shape) value := by rfl

theorem candidate_constructors_are_complete (shape : Shape) (position : Position) :
    candidate shape position ≠ .missingArgument ∧
      candidate shape position ≠ .accidentalString := by
  cases shape <;> cases position <;> decide

theorem named_controls_unchanged :
    candidate .closedEmpty .declaration = parent .closedEmpty .declaration ∧
      candidate .record .declaration = parent .record .declaration ∧
      candidate .openTrue .declaration = parent .openTrue .declaration := by decide

end OpenAPIImport.SchemaNormalization.ObjectRepresentation

namespace OpenAPIImport.SchemaNormalization.EncodedBytes

/-- Representative spellings include two strings decoding to identical bytes;
the malformed case checks that String still needs format validation. -/
inductive Wire where
  | empty | one | two | padAlias | seven | malformed
  deriving DecidableEq, Repr

inductive Rule where
  | unconstrained | minFour | maxEight | enumTwo | spellingPattern
  deriving DecidableEq, Repr

def encodedLength : Wire → Nat
  | .empty => 0
  | .one | .two | .padAlias => 4
  | .seven => 12
  | .malformed => 3

def decodedLength : Wire → Nat
  | .empty => 0
  | .one => 1
  | .two | .padAlias => 2
  | .seven => 7
  | .malformed => 0

def decodedIdentity : Wire → Nat
  | .empty => 0
  | .one => 1
  | .two | .padAlias => 2
  | .seven => 3
  | .malformed => 4

def validBase64 (wire : Wire) : Bool := wire != .malformed

def stringRule : Rule → Wire → Bool
  | .unconstrained, _ => true
  | .minFour, wire => encodedLength wire >= 4
  | .maxEight, wire => encodedLength wire <= 8
  | .enumTwo, wire | .spellingPattern, wire => wire == .two

def source (rule : Rule) (wire : Wire) : Bool := validBase64 wire && stringRule rule wire

/-- Parent copies encoded lengths to decoded storage and interprets source
literal ASCII as bytes. No representative decodes to the ASCII bytes aGk=. -/
def parent (rule : Rule) (wire : Wire) : Bool := validBase64 wire && match rule with
  | .unconstrained => true
  | .minFour => decodedLength wire >= 4
  | .maxEight => decodedLength wire <= 8
  | .enumTwo | .spellingPattern => false

/-- Inverse lengths and decoded enums still collapse source enum identity.
Arbitrary encoded patterns remain unavailable in the decoded carrier. -/
def decodedAlternative (rule : Rule) (wire : Wire) : Bool := validBase64 wire && match rule with
  | .unconstrained => true
  | .minFour => decodedLength wire >= 1
  | .maxEight => decodedLength wire <= 6
  | .enumTwo => decodedIdentity wire == decodedIdentity .two
  | .spellingPattern => false

/-- Validate format without replacing the source spelling; ordinary string
rules then preserve the authored value domain. -/
def candidate (rule : Rule) (wire : Wire) : Bool := validBase64 wire && stringRule rule wire

theorem parent_minimum_counterexample :
    source .minFour .two = true ∧ parent .minFour .two = false := by decide

theorem parent_maximum_counterexample :
    source .maxEight .seven = false ∧ parent .maxEight .seven = true := by decide

theorem parent_literal_counterexample :
    source .enumTwo .two = true ∧ parent .enumTwo .two = false := by decide

theorem inverse_lengths_do_not_preserve_enum :
    source .enumTwo .padAlias = false ∧ decodedAlternative .enumTwo .padAlias = true := by decide

theorem string_without_format_is_insufficient :
    stringRule .unconstrained .malformed = true ∧ source .unconstrained .malformed = false := by decide

theorem candidate_preserves_acceptance (rule : Rule) (wire : Wire) :
    candidate rule wire = source rule wire := by rfl

/-- A non-injective decoder cannot retain every wire-domain assertion. This
statement is independent of the finite representatives and any codec library. -/
theorem noninjective_decoder_loses_distinctions {W D : Type} (decode : W → D)
    (predicate : W → Bool) (x y : W) (collision : decode x = decode y)
    (distinction : predicate x != predicate y) :
    ¬ ∃ decodedPredicate : D → Bool,
      decodedPredicate (decode x) = predicate x ∧ decodedPredicate (decode y) = predicate y := by
  rintro ⟨decodedPredicate, hx, hy⟩
  have same : predicate x = predicate y := hx.symm.trans ((congrArg decodedPredicate collision).trans hy)
  simp [same] at distinction

end OpenAPIImport.SchemaNormalization.EncodedBytes
