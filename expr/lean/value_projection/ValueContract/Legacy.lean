import ValueContract.Model

namespace ValueContract.Legacy

def unionID : Identity := ⟨1, 1⟩
def firstBranch : Identity := ⟨1, 2⟩
def secondBranch : Identity := ⟨1, 3⟩
def bytes : List UInt8 := [104, 105]
def payload : Value := .scalar (.bytes bytes)

/-- The old source returns the chosen payload without its branch identity. -/
def eraseSelection : Value → Value
  | .union _ _ value => value
  | value => value

/-- A later shape-based pass can choose the first identical branch. This models
the loss/reselection mechanism, not all of CanonicalizeExample's matching rules. -/
def recoverFirst (value : Value) : Value := .union unionID firstBranch value

theorem erasedBranchesIndistinguishable :
    eraseSelection (.union unionID firstBranch payload) =
      eraseSelection (.union unionID secondBranch payload) := rfl

theorem legacyBranchLoss :
    recoverFirst (eraseSelection (.union unionID secondBranch payload)) ≠
      .union unionID secondBranch payload := by
  intro equal
  have branchEqual := Value.union.inj equal |>.2.1
  cases branchEqual

/-- Legacy byte-to-text conversion on the concrete valid UTF-8 input "hi". -/
def legacyBytesWire : Wire := .text "hi"

/-- A finite lexical specimen for the witnesses, not a complete base64 codec.
The real JSON codec is exercised outside Lean; universal codec laws are pending. -/
def specimenCodec : ByteCodec where
  encode value := if value = bytes then "aGk=" else ""
  decode value := if value = "aGk=" then some bytes else if value = "" then some [] else none

theorem legacyByteReinterpretation :
    ¬ DecodeScalar specimenCodec .bytes legacyBytesWire (.bytes bytes) := by
  intro decoded
  cases decoded with
  | bytes decoded => contradiction

theorem byteStringDecodes :
    DecodeScalar specimenCodec .bytes (.text "aGk=") (.bytes bytes) :=
  .bytes rfl

/-- The identical wire value matches both scalar interpretations. The model
cannot establish unique untagged matching by attaching a hidden byte tag. -/
theorem byteTextWireCollision :
    DecodeScalar specimenCodec .bytes (.text "aGk=") (.bytes bytes) ∧
    DecodeScalar specimenCodec .string (.text "aGk=") (.string "aGk=") :=
  ⟨byteStringDecodes, .string⟩

theorem byteEncodingWitness : specimenCodec.encode bytes = "aGk=" := by
  simp [specimenCodec]

/-- A second finite specimen additionally models one decoded byte and a
noncanonical pad-bit alias. This is not a general base64 implementation. -/
def boundaryCodec : ByteCodec where
  encode value := if value = [104] then "aA==" else specimenCodec.encode value
  decode value := if value = "aGl=" then some bytes
    else if value = "aA==" then some [104] else specimenCodec.decode value

/-- Bounds on a count, without equating byte counts and wire-string lengths. -/
def lengthWithin (minimum : Nat) (maximum : Option Nat) (count : Nat) : Prop :=
  minimum ≤ count ∧ match maximum with
    | some upper => count ≤ upper
    | none => True

/-- Copying a two-byte bound to JSON string length rejects the valid "hi"
encoding. This is a concrete old-schema counterexample, not the repaired policy. -/
theorem legacyLengthRejectsValidBytes :
    DecodeScalar boundaryCodec .bytes (.text "aGk=") (.bytes bytes) ∧
    lengthWithin 2 (some 2) bytes.length ∧
    ¬ lengthWithin 2 (some 2) "aGk=".length := by
  exact ⟨.bytes rfl, by unfold lengthWithin; decide, by unfold lengthWithin; decide⟩

/-- The opposite mismatch: a four-character encoding contains only one byte,
so copying a minimum of two to wire length accepts a semantically short value. -/
theorem legacyLengthAcceptsShortBytes :
    DecodeScalar boundaryCodec .bytes (.text "aA==") (.bytes [104]) ∧
    ¬ lengthWithin 2 none ([104] : List UInt8).length ∧
    lengthWithin 2 none "aA==".length := by
  exact ⟨.bytes rfl, by unfold lengthWithin; decide, by unfold lengthWithin; decide⟩

/-- Canonical enum emission compares wire strings, independently of decoding. -/
def schemaByteEnum (codec : ByteCodec) (allowed : List (List UInt8))
    (text : String) : Prop :=
  ∃ value ∈ allowed, text = codec.encode value

/-- Runtime byte enum checking compares the decoded byte value. -/
def decoderByteEnum (codec : ByteCodec) (allowed : List (List UInt8))
    (text : String) : Prop :=
  ∃ value ∈ allowed, codec.decode text = some value

/-- The canonical spelling is admitted by both independently stated predicates. -/
theorem canonicalByteEnumAgreement :
    schemaByteEnum boundaryCodec [bytes] "aGk=" ∧
    decoderByteEnum boundaryCodec [bytes] "aGk=" := by
  constructor
  · exact ⟨bytes, by simp, rfl⟩
  · exact ⟨bytes, by simp, rfl⟩

/-- Schema enum rejection cannot establish runtime branch disjointness:
"aGl=" is a String and also decodes to the allowed bytes "hi", although the
canonical byte enum contains only "aGk=". This scalar witness does not claim
support for any new union shape or implement a candidate union matcher. -/
theorem aliasSchemaDecoderMismatch :
    DecodeScalar boundaryCodec .string (.text "aGl=") (.string "aGl=") ∧
    ¬ schemaByteEnum boundaryCodec [bytes] "aGl=" ∧
    decoderByteEnum boundaryCodec [bytes] "aGl=" := by
  refine ⟨.string, ?_, bytes, by simp, rfl⟩
  simp [schemaByteEnum, boundaryCodec, specimenCodec, bytes]

def authoredSource : Source := ⟨⟨2, 1⟩, 7, .authoredExample⟩
def syntheticSource : Source := ⟨⟨2, 1⟩, 8, .synthesizedExample⟩
def authored : Supplied Value := ⟨authoredSource, .scalar (.string "authored")⟩
def generated : Supplied Value := ⟨syntheticSource, .scalar (.string "random")⟩

/-- A generator independently synthesizing its message ignores the selected
authored value; carrying source metadata afterward cannot restore that value. -/
def legacyChoose (_selected synthesized : Supplied Value) : Supplied Value := synthesized

theorem legacyAuthoredReplacement :
    (legacyChoose authored generated).value ≠ authored.value := by
  intro equal
  have textEqual : ("random" : String) = "authored" := Scalar.string.inj (Value.scalar.inj equal)
  exact (by decide : ("random" : String) ≠ "authored") textEqual

theorem legacySourceReplacement :
    (legacyChoose authored generated).source ≠ authored.source := by
  decide

def bodyID : Identity := ⟨3, 1⟩
def headerID : Identity := ⟨3, 2⟩
def bodyPlan : Plan := .select bodyID (.scalar .bytes)
def serviceValue : Value :=
  .object [(bodyID, payload), (headerID, .scalar (.string "header"))]

/-- Legitimate header removal is an observation, not a contract failure. -/
theorem selectedBodyObservation : Observe bodyPlan serviceValue payload := by
  exact ⟨2, by simp [observeAt, bodyPlan, serviceValue, fieldValue, bodyID,
    headerID, payload, Scalar.kind]⟩

/-- Field loss remains in the representability/progress domain. -/
theorem selectedBodyRepresentable : Representable specimenCodec bodyPlan serviceValue := by
  refine ⟨.text "aGk=", payload, ?_, ?_, selectedBodyObservation⟩
  · exact ⟨2, by simp [wireTypedAt, bodyPlan, specimenCodec]⟩
  · exact ⟨2, DecodeScalar.bytes rfl⟩

theorem observationIsNotWholeService : payload ≠ serviceValue := by
  intro equal
  cases equal

def visibleUnionPlan : Plan := .union unionID (.tagged "kind" "value")
  [⟨firstBranch, "first", .scalar .bytes⟩, ⟨secondBranch, "second", .scalar .bytes⟩]

theorem visibleBranchRetained : Observe visibleUnionPlan
    (.union unionID secondBranch payload) (.union unionID secondBranch payload) := by
  refine ⟨2, ?_⟩
  simp only [observeAt, visibleUnionPlan, true_and]
  refine ⟨⟨secondBranch, "second", .scalar .bytes⟩, by simp, rfl, ?_⟩
  simp [payload, Scalar.kind]

theorem nullDistinctFromAbsent : Value.null ≠ Value.absent := by
  intro equal
  cases equal

theorem nullObservation : Observe (.nullable (.scalar .string)) .null .null := ⟨1, True.intro⟩

theorem absentObservation : Observe (.nullable (.scalar .string)) .absent .absent :=
  ⟨1, True.intro⟩

theorem omissionIsFieldLocal :
    fieldObservation true (.array []) = .absent ∧
    fieldObservation false (.array []) = .array [] ∧
    fieldObservation true .null = .null := ⟨rfl, rfl, rfl⟩

/-- Duplicate authored map keys survive in the input vocabulary. -/
theorem duplicateEntriesRetained :
    ([(Scalar.integer 1, RawValue.null), (.string "1", .null)] :
      List (Scalar × RawValue)).length = 2 := rfl

def recursiveID : Identity := ⟨4, 1⟩
def nextID : Identity := ⟨4, 2⟩
def declarations : Declarations := fun identity =>
  if identity = recursiveID then
    some (.object [(nextID, .nullable (.reference recursiveID))]) else none

/-- A recursive declaration admits a finite null-terminated value. No global
depth bound is imposed on the public structural-typing judgment. -/
theorem finiteRecursiveValue : HasType declarations (.reference recursiveID)
    (.object [(nextID, .null)]) := by
  refine ⟨3, ?_⟩
  simp [hasTypeAt, declarations, All₂]
  intro a b c d ha hb hc hd
  subst a; subst b; subst c; subst d
  exact ⟨rfl, True.intro⟩

end ValueContract.Legacy
