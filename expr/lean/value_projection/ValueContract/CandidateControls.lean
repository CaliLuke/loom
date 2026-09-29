import ValueContract.Legacy
import ValueContract.ScalarProjection
import ValueContract.Observation
import ValueContract.GraphValidation

namespace ValueContract.Candidate.Controls

/-- Explicit tiny numeric specimen, not a universal numerical codec. The actual
Go JSON-v2 regression independently marshals int(1) and float64(1) to `1` and
decodes that same text into both destinations. -/
def numbers : NumericCodec where
  encodeInteger value := if value = 1 then "1" else "outside-specimen"
  encodeDecimal coefficient exponent :=
    if coefficient = 1 ∧ exponent = 0 then "1" else "outside-specimen"
  schemaNumber text := if text = "1" ∨ text = "1.0" ∨ text = "1e0" then some ⟨1, 0⟩ else none
  decodeInteger text := if text = "1" then some 1 else none
  decodeDecimal text := if text = "1" ∨ text = "1.0" ∨ text = "1e0" then some ⟨1, 0⟩ else none

def codecs : ScalarCodecs := ⟨Legacy.boundaryCodec, numbers⟩

def noExternal : ExternalScalarChecks := fun _ _ => true

theorem numericSharedWire :
    encodeScalar codecs .json (.integer 1) = .number "1" ∧
    encodeScalar codecs .json (.decimal 1 0) = .number "1" := by
  exact ⟨rfl, rfl⟩

theorem numericSharedSchema :
    scalarSchema codecs noExternal .json .integer {} (.number "1") = true ∧
    scalarSchema codecs noExternal .json .decimal {} (.number "1") = true := by decide

theorem numericSharedDecoder :
    ScalarDecodes codecs .json .integer (.number "1") (.integer 1) ∧
    ScalarDecodes codecs .json .decimal (.number "1") (.decimal 1 0) := by
  exact ⟨.jsonInteger rfl, .jsonDecimal (value := ⟨1, 0⟩) rfl⟩

theorem numericLexicalAliasDiffers :
    scalarSchema codecs noExternal .json .integer {} (.number "1.0") = true ∧
    decodeScalar codecs .json .integer (.number "1.0") = none ∧
    ScalarDecodes codecs .json .decimal (.number "1.0") (.decimal 1 0) := by
  exact ⟨by decide, rfl, .jsonDecimal (value := ⟨1, 0⟩) rfl⟩

def byteEnum : ScalarRules := { enumeration := some [.bytes [104, 105]] }

theorem aliasSchemaRuntimeDisagree :
    scalarSchema codecs noExternal .json .bytes byteEnum (.text "aGl=") = false ∧
    ScalarDecodes codecs .json .bytes (.text "aGl=") (.bytes [104, 105]) ∧
    scalarAllowed noExternal byteEnum (.bytes [104, 105]) = true := by
  exact ⟨by decide, .jsonBytes rfl, by decide⟩

theorem candidateByteTextCollision :
    encodeScalar codecs .json (.bytes [104, 105]) = .text "aGk=" ∧
    encodeScalar codecs .json (.string "aGk=") = .text "aGk=" := by
  exact ⟨rfl, rfl⟩

theorem integerProjectionEmits :
    projectScalar codecs noExternal .json .integer {} (.integer 1) = some (.number "1") := by
  rfl

theorem stringProjectionEmits (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (text : String) :
    projectScalar codecs checks .json .string {} (.string text) = some (.text text) := by
  simp [projectScalar, encodeScalar, scalarSchema, decodeScalar, scalarPreserved,
    scalarAllowed, scalarLength, scalarNumber, scalarEqual, lengthAllowed, Scalar.kind]

/-- Universal non-vacuity for every finite byte list under explicitly named
codec laws. These laws concern lexical bytes only, not candidate success or
branch uniqueness. The independently computed grammar must accept the output. -/
theorem bytesProjectionEmits (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (laws : ByteCodecLaws codecs.bytes) (bytes : List UInt8) :
    projectScalar codecs checks .json .bytes {} (.bytes bytes) =
      some (.text (codecs.bytes.encode bytes)) := by
  have syntaxValid := (laws.language (codecs.bytes.encode bytes) bytes.length).mpr
    ⟨bytes, laws.roundTrip bytes, rfl⟩
  simp [projectScalar, encodeScalar, scalarSchema, jsonByteSchema, decodeScalar,
    scalarPreserved, scalarAllowed, scalarLength, scalarNumber, scalarEqual,
    lengthAllowed, Scalar.kind, syntaxValid, laws.roundTrip]

def arrayID : Identity := ⟨10, 1⟩
def scalarID : Identity := ⟨10, 2⟩
def objectID : Identity := ⟨10, 3⟩
def fieldID : Identity := ⟨10, 4⟩
def bodyID : Identity := ⟨10, 5⟩

def objectDeclaration (presence : Presence) : TargetDeclaration :=
  { identity := objectID, expansionRank := 0,
    target := .object [⟨fieldID, "items", false, presence, arrayID⟩] false }

def presenceTargets (presence : Presence) : Targets := [
  { identity := arrayID, expansionRank := 0, target := .array scalarID {} },
  { identity := scalarID, expansionRank := 0, target := .scalar .json .string {} },
  objectDeclaration presence,
  { identity := bodyID, expansionRank := 1, target := .select fieldID arrayID }]

theorem emptyArrayTargetEmpty (presence : Presence) :
    EmptyObserved (presenceTargets presence) arrayID (.array []) := by
  refine ⟨1, ?_⟩
  simp [emptyObservedAt, presenceTargets, objectDeclaration, arrayID, scalarID, objectID, bodyID]

theorem nilContractObservesEmpty :
    Observe codecs (presenceTargets .explicit) .enumMember arrayID .nilArray (.array []) := by
  refine ⟨1, ?_⟩
  simp [observeAt, presenceTargets, objectDeclaration, arrayID, scalarID, objectID, bodyID, contractRole]

theorem nilExampleNotObserved (depth : Nat) :
    ¬ observeAt depth codecs (presenceTargets .explicit) .authoredExample arrayID .nilArray (.array []) := by
  cases depth <;> simp [observeAt, presenceTargets, objectDeclaration, arrayID, scalarID, objectID, bodyID, contractRole]

theorem nilContractFieldOmitted :
    Observe codecs (presenceTargets .omitEmpty) .enumMember objectID
      (.object [(fieldID, .nilArray)] []) (.object [(fieldID, .absent)] []) := by
  refine ⟨2, objectDeclaration .omitEmpty, by simp [presenceTargets], rfl, ?_⟩
  constructor
  · constructor
    · rfl
    · intro pair member
      have equal := List.mem_singleton.mp member
      subst pair
      refine ⟨rfl, .array [], ?_, Or.inl ⟨emptyArrayTargetEmpty _, rfl⟩⟩
      simp [observeAt, presenceTargets, objectDeclaration, memberValue, arrayID,
        scalarID, objectID, bodyID, contractRole]
  · rfl

theorem emptyFieldOmitted :
    Observe codecs (presenceTargets .omitEmpty) .authoredExample objectID
      (.object [(fieldID, .array [])] []) (.object [(fieldID, .absent)] []) := by
  refine ⟨2, objectDeclaration .omitEmpty, by simp [presenceTargets], rfl, ?_⟩
  constructor
  · constructor
    · rfl
    · intro pair member
      have equal := List.mem_singleton.mp member
      subst pair
      refine ⟨rfl, .array [], ?_, Or.inl ⟨emptyArrayTargetEmpty _, rfl⟩⟩
      simp [observeAt, presenceTargets, objectDeclaration, memberValue, arrayID,
        scalarID, objectID, bodyID, contractRole, All₂]
  · rfl

theorem selectedBodyRetainsEmpty :
    Observe codecs (presenceTargets .omitEmpty) .authoredExample bodyID
      (.object [(fieldID, .array [])] []) (.array []) := by
  refine ⟨2, ?_⟩
  simp [observeAt, presenceTargets, objectDeclaration, arrayID, scalarID, objectID, bodyID,
    All₂, memberValue, contractRole]

/-- These are finite codec-boundary specimens, not a universal base64 or
JSON serializer theorem. Both typed source kinds erase to the same wire text. -/
theorem anyBytesStringSharedWire :
    materialize codecs (.scalar (.bytes [104, 105])) = .ok (.text "aGk=") ∧
    materialize codecs (.scalar (.string "aGk=")) = .ok (.text "aGk=") := by
  simp [materialize, jsonValid, encodeScalar, codecs, Legacy.boundaryCodec, Legacy.specimenCodec, Legacy.bytes, numbers]

theorem anyNilContainersMaterialize :
    materialize codecs .nilBytes = .ok (.text "") ∧
    materialize codecs .nilArray = .ok (.array []) ∧
    materialize codecs .nilMap = .ok (.object []) := by
  simp [materialize, codecs, Legacy.boundaryCodec, Legacy.specimenCodec, Legacy.bytes]

theorem invalidNumericSnapshotRejected :
    jsonValid numbers (.number "not-json") = false ∧
    jsonValid numbers (.integer 1) = false := by
  simp [jsonValid, numbers]

theorem anyCollidingKeysRejected :
    materialize codecs (.map [(.integer 1, .null), (.string "1", .null)]) =
      .error .invalid := by
  simp only [materialize]
  change (Except.error Failure.invalid : Except Failure Wire) = _
  rfl

theorem anyDuplicateMembersRejected :
    materialize codecs (.object [] [("x", .null), ("x", .null)]) =
      .error .invalid := by simp [materialize]

theorem rawCannotForgeSnapshot (wire : Wire) :
    materialize codecs (.jsonSnapshot wire) = .error .unsupported := by simp [materialize]

end ValueContract.Candidate.Controls
