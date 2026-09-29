import ValueContract.ProjectionProofs
import ValueContract.ObservationExecutionProofs
import ValueContract.CanonicalConstructionProofs

namespace ValueContract.Candidate

private theorem bind_ok {input : Except ε α} {next : α → Except ε β} {result : β} :
    (input >>= next) = .ok result ↔ ∃ value, input = .ok value ∧ next value = .ok result := by
  cases input <;> simp [Bind.bind, Except.bind]

/-- Construction of a present representation implements the independent
observation followed by canonical spelling. The source and target may differ
through legitimate visibility, selected-body and field-presence projection. -/
theorem build_some_iff {codecs targets role identity value observed wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    build codecs targets role identity value = .ok (observed, some wire) ↔
      Observe codecs targets role identity value observed ∧
      Canonical codecs targets identity observed wire := by
  unfold build
  constructor
  · intro done
    obtain ⟨actual, observedDone, rest⟩ := bind_ok.mp done
    obtain ⟨result, constructed, same⟩ := bind_ok.mp rest
    cases same
    exact ⟨(observeValue_iff wellFormed found).mp observedDone,
      (construct_some_iff wellFormed found).mp constructed⟩
  · rintro ⟨observedRelation, canonical⟩
    exact bind_ok.mpr ⟨observed, (observeValue_iff wellFormed found).mpr observedRelation,
      bind_ok.mpr ⟨some wire, (construct_some_iff wellFormed found).mpr canonical, rfl⟩⟩

/-- Universal candidate correctness for a particular emitted wire: observation,
canonical spelling, schema validity and actual runtime preservation all concern
that same wire. Documentation occurrences have no runtime decoder assertion. -/
theorem project_emitted_iff {codecs checks targets use role identity resolved wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration) :
    project codecs checks targets use role identity resolved = .emitted wire ↔
      ∃ observed, Observe codecs targets role identity resolved.value observed ∧
        Canonical codecs targets identity observed wire ∧
        SchemaAccepts codecs checks targets identity wire ∧
        RuntimeObligation use codecs checks targets identity wire observed := by
  constructor
  · intro emitted
    obtain ⟨observed, built, valid, runtime⟩ := project_emitted_checks wellFormed emitted
    obtain ⟨observation, canonical⟩ := (build_some_iff wellFormed found).mp built
    exact ⟨observed, observation, canonical, valid, runtime⟩
  · rintro ⟨observed, observation, canonical, valid, runtime⟩
    exact project_checks_complete wellFormed
      ((build_some_iff wellFormed found).mpr ⟨observation, canonical⟩) valid runtime

/-- Soundness uses independently specified representability, never projector
success as a representability premise. -/
theorem project_sound {codecs checks targets use role identity resolved wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (emitted : project codecs checks targets use role identity resolved = .emitted wire) :
    Representable use codecs checks targets role identity resolved.value := by
  obtain ⟨observed, observation, canonical, valid, runtime⟩ :=
    (project_emitted_iff wellFormed found).mp emitted
  exact ⟨observed, wire, observation, canonical, valid, runtime⟩

/-- Progress covers every independently representable finite value, including
partial service objects whose missing fields are excluded by the target. -/
theorem project_progress {codecs checks targets use role identity resolved declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (representable : Representable use codecs checks targets role identity resolved.value) :
    ∃ wire, project codecs checks targets use role identity resolved = .emitted wire := by
  obtain ⟨observed, wire, observation, canonical, valid, runtime⟩ := representable
  exact ⟨wire, (project_emitted_iff wellFormed found).mpr
    ⟨observed, observation, canonical, valid, runtime⟩⟩

/-- Runtime-backed emission preserves the independently observed service value,
including visible union identities; schema uniqueness alone never suffices. -/
theorem project_runtime_preservation {codecs checks targets role identity resolved wire declaration}
    (wellFormed : WellFormedTargets targets)
    (found : findTarget targets identity = some declaration)
    (emitted : project codecs checks targets .runtime role identity resolved = .emitted wire) :
    ∃ observed decoded, Observe codecs targets role identity resolved.value observed ∧
      SchemaAccepts codecs checks targets identity wire ∧
      RuntimeDecodes codecs checks targets identity wire decoded ∧
      StrictEquivalent codecs.numbers observed decoded := by
  obtain ⟨observed, observation, _, valid, decoded, decodes, preserved⟩ :=
    (project_emitted_iff wellFormed found).mp emitted
  exact ⟨observed, decoded, observation, valid, decodes, preserved⟩

end ValueContract.Candidate
