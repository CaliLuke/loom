import ValueContract.ProjectionControls
import ValueContract.ObservationProofs
import ValueContract.SchemaProofs

namespace ValueContract.Candidate.PresenceControls

open ProjectionControls Controls

def requiredTargets (presence : Presence) : Targets := [
  { identity := rootID, expansionRank := 0,
    target := .object [⟨memberID, "x", true, presence, textID⟩] false },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

/-- The rejected canonical fragment rule omitted the requiredness check. -/
def legacyOmittedFragment (presence : Presence) (value : Value) : Prop :=
  value = .absent ∨ implicitDefaultOmitted presence value = true

theorem legacyRequiredOmissionCounterexample :
    legacyOmittedFragment .explicit .absent ∧
    legacyOmittedFragment (.implicitDefault (.string "")) (.scalar (.string "")) := by
  exact ⟨Or.inl rfl, Or.inr rfl⟩

/-- Requiredness constrains canonical representation even when a different
untagged branch could accept the same empty wire. -/
theorem requiredSingletonCannotBeEmpty (presence : Presence) (value : Value) :
    ¬ Canonical codecs (requiredTargets presence) rootID
      (.object [(memberID, value)] []) (.object []) := by
  rintro ⟨depth, relation⟩
  cases depth with
  | zero => exact relation
  | succ depth =>
    obtain ⟨declaration, member, same, relation⟩ := relation
    simp only [requiredTargets, List.mem_cons, List.not_mem_nil, or_false] at member
    rcases member with rfl | rfl
    · obtain ⟨fragments, additional, related, extra, unique, wire⟩ := relation
      have lengths := related.1
      cases fragments with
      | nil => simp at lengths
      | cons fragment rest =>
        have empty : rest = [] := by simpa using lengths.symm
        subst rest
        have one := related.2 (⟨memberID, "x", true, presence, textID⟩, fragment) (by simp)
        have noExtra : additional = [] := extra
        subst additional
        rcases fragment with ⟨name, output⟩
        cases output with
        | none => exact Bool.noConfusion one.2.1
        | some child => simp [retainedMembers, orderedMembers] at wire
    · have impossible : textID = rootID := same
      exact False.elim (by cases impossible)

theorem requiredImplicitDefaultIncomplete :
    project codecs noExternal (requiredTargets (.implicitDefault (.string "")))
      .documentation .defaultValue rootID ⟨.object [(memberID, .scalar (.string ""))] [], []⟩ =
      .incomplete := by cbv

theorem requiredMissingIncomplete :
    project codecs noExternal (unknownTargets false) .documentation .authoredExample rootID
      ⟨.union rootID branchA (.object [(memberID, .absent)] []), []⟩ = .incomplete := by cbv

theorem otherBranchSchemaStillAccepts :
    schema codecs noExternal (unknownTargets false) rootID (.object []) = .ok true := by cbv

def optionalTargets (presence : Presence) : Targets := [
  { identity := rootID, expansionRank := 0,
    target := .object [⟨memberID, "x", false, presence, textID⟩] false },
  { identity := textID, expansionRank := 0, target := .scalar .json .string {} }]

theorem optionalAbsenceStillEmits :
    project codecs noExternal (optionalTargets .explicit) .runtime .authoredExample rootID
      ⟨.object [(memberID, .absent)] [], []⟩ = .emitted (.object []) := by cbv

theorem optionalImplicitDefaultStillEmits :
    project codecs noExternal (optionalTargets (.implicitDefault (.string "")))
      .runtime .defaultValue rootID ⟨.object [(memberID, .scalar (.string ""))] [], []⟩ =
      .emitted (.object []) := by cbv

end ValueContract.Candidate.PresenceControls
