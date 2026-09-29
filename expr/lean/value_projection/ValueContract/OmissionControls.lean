import ValueContract.ProjectionControls
import ValueContract.ProjectionProofs
import ValueContract.LegacyProjectionBuild
open ValueContract (All₂)
open ValueContract.Candidate ValueContract.Candidate.Controls
open ValueContract.Candidate.ProjectionControls

namespace ValueContract.Candidate.OmissionControls

def omittedTargets : Targets := [
  { identity := rootID, expansionRank := 0,
    target := .object [⟨memberID, "child", false, .omitEmpty, objectA⟩] false },
  { identity := objectA, expansionRank := 0,
    target := .object [⟨branchA, "x", true, .explicit, textID⟩] false },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]
def incompleteChild : Value := .object [(branchA, .absent)] []
def supplied : Value := .object [(memberID, incompleteChild)] []
def visible : Value := .object [(memberID, .absent)] []

theorem observed : Observe codecs omittedTargets .authoredExample rootID supplied visible := by
  refine ⟨4, omittedTargets[0], by simp [omittedTargets], rfl, ?_, rfl⟩
  refine ⟨rfl, ?_⟩
  intro pair member
  simp at member
  subst pair
  refine ⟨rfl, incompleteChild, ?_, ?_⟩
  · refine ⟨omittedTargets[1], by simp [omittedTargets], rfl, ?_, rfl⟩
    refine ⟨rfl, ?_⟩
    intro pair member
    simp at member
    subst pair
    refine ⟨rfl, .absent, ?_, rfl⟩
    exact ⟨omittedTargets[2], by simp [omittedTargets], rfl, trivial⟩
  · apply Or.inl
    refine ⟨⟨1, omittedTargets[1], by simp [omittedTargets], rfl, rfl, ?_⟩, rfl⟩
    intro member inside
    simp at inside
    subst member
    exact Or.inl rfl

theorem canonical : Canonical codecs omittedTargets rootID visible (.object []) := by
  refine ⟨2, omittedTargets[0], by simp [omittedTargets], rfl, [("child", none)], [], ?_⟩
  simp [omittedTargets, All₂, memberValue, retainedMembers, orderedMembers]

theorem valid : SchemaAccepts codecs noExternal omittedTargets rootID (.object []) := by
  apply schemaFuel_sound (fuel := 10)
  cbv

theorem preserved : RuntimeObligation .runtime codecs noExternal omittedTargets rootID (.object []) visible := by
  refine ⟨visible, ?_, 4, ?_⟩
  · apply decodeFuel_sound (fuel := 10)
    cbv
  · exact (valueEqualAt_strict_iff ..).mp (by cbv)

theorem representable : Representable .runtime codecs noExternal omittedTargets .authoredExample rootID supplied :=
  ⟨visible, .object [], observed, canonical, valid, preserved⟩

theorem candidateEmitsAfterOmission : project codecs noExternal omittedTargets .runtime .authoredExample rootID
    ⟨supplied, []⟩ = .emitted (.object []) := by cbv

/-- The former executable violated progress despite the full independent runtime
representability witness. This preserves the rejected algorithm, not just output. -/
theorem legacyChildFirstProgressFailure :
    Representable .runtime codecs noExternal omittedTargets .authoredExample rootID supplied ∧
    LegacyBuild.build codecs omittedTargets .authoredExample rootID supplied = .error .incomplete := by
  exact ⟨representable, by cbv⟩

def retainedTargets : Targets := [
  { identity := rootID, expansionRank := 0,
    target := .object [⟨memberID, "child", false, .explicit, objectA⟩] false },
  { identity := objectA, expansionRank := 0,
    target := .object [⟨branchA, "x", true, .explicit, textID⟩] false },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem retainedChildStillIncomplete :
    project codecs noExternal retainedTargets .runtime .authoredExample rootID ⟨supplied, []⟩ =
      .incomplete := by cbv

end ValueContract.Candidate.OmissionControls
