import ValueContract.Projection
import ValueContract.CandidateControls

namespace ValueContract.Candidate.ProjectionControls

open Controls

def rootID : Identity := ⟨70, 0⟩
def objectA : Identity := ⟨70, 1⟩
def objectB : Identity := ⟨70, 2⟩
def textID : Identity := ⟨70, 3⟩
def memberID : Identity := ⟨70, 4⟩
def branchA : Identity := ⟨70, 5⟩
def branchB : Identity := ⟨70, 6⟩

def unknownTargets (reject : Bool) : Targets := [
  { identity := rootID, expansionRank := 1,
    target := .union rootID .untagged [⟨branchA, "a", objectA⟩, ⟨branchB, "b", objectB⟩] },
  { identity := objectA, expansionRank := 0,
    target := .object [⟨memberID, "x", true, .explicit, textID⟩] false },
  { identity := objectB, expansionRank := 0, target := .object [] false,
    schemaAllowsUnknown := false, decoderRejectsUnknown := reject },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

def objectWire : Wire := .object [("x", .text "hello")]
def selectedObject : Resolution :=
  ⟨.union rootID branchA (.object [(memberID, .scalar (.string "hello"))] []), []⟩

/-- The open decoder accepts the second branch even though its closed schema
rejects the same wire. Schema-only uniqueness must therefore be insufficient. -/
theorem unknownPolicySchemaUnique :
    schema codecs noExternal (unknownTargets false) rootID objectWire = .ok true := by
  cbv

theorem unknownPolicyRuntimeAmbiguous :
    decode codecs noExternal (unknownTargets false) rootID objectWire = .ok none := by
  cbv

theorem unknownPolicyProjectionRejected :
    project codecs noExternal (unknownTargets false) .runtime .authoredExample rootID selectedObject =
      .unrepresentable := by
  cbv

theorem unknownPolicyDocumentationAccepted :
    project codecs noExternal (unknownTargets false) .documentation .authoredExample rootID selectedObject =
      .emitted objectWire := by
  cbv

theorem unknownPolicyStrictDecoderAccepted :
    project codecs noExternal (unknownTargets true) .runtime .authoredExample rootID selectedObject =
      .emitted objectWire := by
  cbv

def bodyTargets : Targets := [
  { identity := rootID, expansionRank := 1, target := .select memberID textID },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem partialSourceCompleteBody :
    project codecs noExternal bodyTargets .runtime .authoredExample rootID
      ⟨.object [(memberID, .scalar (.string "body")), (branchB, .absent)] [], [[branchB]]⟩ =
      .emitted (.text "body") := by cbv

theorem retainedMissingBodyIncomplete :
    project codecs noExternal bodyTargets .runtime .authoredExample rootID
      ⟨.object [(memberID, .absent)] [], [[memberID]]⟩ = .incomplete := by cbv

def scalarUnionTargets (style : UnionStyle) : Targets := [
  { identity := rootID, expansionRank := 1,
    target := .union rootID style [⟨branchA, "first", textID⟩, ⟨branchB, "second", textID⟩] },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem secondEqualPayloadBranchRetained :
    project codecs noExternal (scalarUnionTargets (.tagged "kind" "data"))
      .runtime .authoredExample rootID ⟨.union rootID branchB (.scalar (.string "same")), []⟩ =
      .emitted (.object [("data", .text "same"), ("kind", .text "second")]) := by cbv

theorem untaggedEqualPayloadBranchesRejected :
    project codecs noExternal (scalarUnionTargets .untagged)
      .runtime .authoredExample rootID ⟨.union rootID branchB (.scalar (.string "same")), []⟩ =
      .unrepresentable := by cbv

theorem protobufSecondBranchRetained :
    project codecs noExternal (scalarUnionTargets .protobuf)
      .runtime .authoredExample rootID ⟨.union rootID branchB (.scalar (.string "same")), []⟩ =
      .emitted (.oneof "second" (.text "same")) := by cbv

def collectionTargets : Targets := [
  { identity := rootID, expansionRank := 0, target := .array rootID {} }]

theorem finiteRecursiveArraysEmit :
    project codecs noExternal collectionTargets .runtime .authoredExample rootID
      ⟨.array [.array [.array [], .array []]], []⟩ =
      .emitted (.array [.array [.array [], .array []]]) := by cbv

theorem nilExampleIncomplete :
    project codecs noExternal collectionTargets .runtime .authoredExample rootID
      ⟨.nilArray, []⟩ = .incomplete := by cbv

theorem nilContractEmitsEmpty :
    project codecs noExternal collectionTargets .runtime .enumMember rootID
      ⟨.nilArray, []⟩ = .emitted (.array []) := by cbv

def mapTargets : Targets := [
  { identity := rootID, expansionRank := 0, target := .map .builtin {} textID {} },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem heterogeneousMapObservesNames :
    project codecs noExternal mapTargets .runtime .authoredExample rootID
      ⟨.map [(.integer 1, .scalar (.string "one")), (.boolean true, .scalar (.string "yes"))], []⟩ =
      .emitted (.object [("1", .text "one"), ("true", .text "yes")]) := by cbv

theorem mapNameCollisionCannotBeOverwritten :
    project codecs noExternal mapTargets .runtime .authoredExample rootID
      ⟨.map [(.integer 1, .scalar (.string "one")), (.string "1", .scalar (.string "other"))], []⟩ =
      .unrepresentable := by cbv

def visibilityTargets : Targets := [
  { identity := rootID, expansionRank := 0,
    target := .object [⟨memberID, "child", false, .omitEmpty, objectA⟩] false },
  { identity := objectA, expansionRank := 0, target := .object [] false }]

theorem objectEmptiedByVisibilityOmitted :
    project codecs noExternal visibilityTargets .runtime .authoredExample rootID
      ⟨.object [(memberID, .object [(branchA, .scalar (.string "hidden"))] [])] [], []⟩ =
      .emitted (.object []) := by cbv

def byteAliasTargets : Targets := [
  { identity := rootID, expansionRank := 1,
    target := .union rootID .untagged [⟨branchA, "text", textID⟩, ⟨branchB, "bytes", objectB⟩] },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} },
  { identity := objectB, expansionRank := 0, target := .scalar .json .bytes byteEnum }]

theorem byteAliasSchemaRuntimeSplit :
    schema codecs noExternal byteAliasTargets rootID (.text "aGl=") = .ok true ∧
    decode codecs noExternal byteAliasTargets rootID (.text "aGl=") = .ok none := by cbv

theorem byteAliasProjectionCannotUseSchemaUniqueness :
    project codecs noExternal byteAliasTargets .runtime .authoredExample rootID
      ⟨.union rootID branchA (.scalar (.string "aGl=")), []⟩ = .unrepresentable := by cbv

def numericTargets : Targets := [
  { identity := rootID, expansionRank := 1,
    target := .union rootID .untagged [⟨branchA, "int", objectA⟩, ⟨branchB, "float", objectB⟩] },
  { identity := objectA, expansionRank := 0, target := .scalar .json .integer {} },
  { identity := objectB, expansionRank := 0, target := .scalar .json .decimal {} }]

theorem numericSourceKindsDoNotDisambiguateWire :
    project codecs noExternal numericTargets .runtime .authoredExample rootID
      ⟨.union rootID branchB (.scalar (.decimal 1 0)), []⟩ = .unrepresentable := by cbv

def anyTargets : Targets := [{ identity := rootID, expansionRank := 0, target := .any }]

theorem anyNilBytesUseCodecMeaning :
    project codecs noExternal anyTargets .runtime .authoredExample rootID
      ⟨.any .nilBytes, []⟩ = .emitted (.text "") := by cbv

def nullableTargets : Targets := [
  { identity := rootID, expansionRank := 1, target := .nullable textID },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem explicitNullEmitsAndAbsenceDoesNot :
    project codecs noExternal nullableTargets .runtime .authoredExample rootID
      ⟨.null, []⟩ = .emitted .null ∧
    project codecs noExternal nullableTargets .runtime .authoredExample rootID
      ⟨.absent, []⟩ = .incomplete := by cbv

/-- Reproduces the rejected initial evaluator layering: a branch-local return
can bypass the following enum gate in an enclosing Lean do block. -/
def legacyHoistedSchemaEnum (target : Target) : Except Failure Bool := do
  let accepted ← match target with
    | .array _ _ => do
      return true
    | _ => .ok true
  return accepted && false

theorem legacyHoistedEnumCounterexample :
    legacyHoistedSchemaEnum (.array textID {}) = .ok true := by cbv

def emptyEnumTargets (target : Target) : Targets := [
  { identity := rootID, expansionRank := 1, target := target,
    schemaEnumeration := some [], enumeration := some [] },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem wholeNodeArrayEnumsChecked :
    schema codecs noExternal (emptyEnumTargets (.array textID {})) rootID (.array []) = .ok false ∧
    decode codecs noExternal (emptyEnumTargets (.array textID {})) rootID (.array []) = .ok none := by cbv

theorem wholeNodeObjectEnumsChecked :
    schema codecs noExternal (emptyEnumTargets (.object [] false)) rootID (.object []) = .ok false ∧
    decode codecs noExternal (emptyEnumTargets (.object [] false)) rootID (.object []) = .ok none := by cbv

theorem wholeNodeMapEnumsChecked :
    schema codecs noExternal (emptyEnumTargets (.map (.scalar .string) {} textID {}))
      rootID (.object []) = .ok false ∧
    decode codecs noExternal (emptyEnumTargets (.map (.scalar .string) {} textID {}))
      rootID (.object []) = .ok none := by cbv

theorem wholeNodeUnionEnumsChecked :
    schema codecs noExternal (emptyEnumTargets (.union rootID .untagged [⟨branchA, "one", textID⟩]))
      rootID (.text "hello") = .ok false ∧
    decode codecs noExternal (emptyEnumTargets (.union rootID .untagged [⟨branchA, "one", textID⟩]))
      rootID (.text "hello") = .ok none := by cbv

theorem wholeNodeAnyEnumsChecked :
    schema codecs noExternal (emptyEnumTargets .any) rootID (.text "hello") = .ok false ∧
    decode codecs noExternal (emptyEnumTargets .any) rootID (.text "hello") = .ok none := by cbv

end ValueContract.Candidate.ProjectionControls
