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
