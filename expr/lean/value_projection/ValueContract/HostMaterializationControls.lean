import ValueContract.CandidateControls
import ValueContract.Materialization

namespace ValueContract.Candidate

/-- Host equality evidence has no influence on the canonical JSON wire. -/
theorem materialize_host_erases (codecs : ScalarCodecs) (identity : Nat) (payload : Value) :
    materialize codecs (.host identity payload) = materialize codecs payload := by
  simp only [materialize]

namespace HostControls

open Controls

theorem nestedHostMaterializes :
    materialize codecs (.host 1 (.object [] [
      ("data", .host 2 (.array [.host 3 (.scalar (.integer 1))]))])) =
      .ok (.object [("data", .array [.number "1"])]) := by cbv

theorem hostCannotBypassKeyCollision :
    materialize codecs (.host 7 (.map [
      (.integer 1, .host 7 (.scalar (.string "a"))),
      (.string "1", .host 7 (.scalar (.string "b")))])) = .error .invalid := by cbv

theorem hostCannotForgeBranchEvidence :
    materialize codecs (.host 7 (.union ⟨1, 0⟩ ⟨1, 1⟩ (.scalar (.string "a")))) =
      .error .unsupported := by cbv

end HostControls
end ValueContract.Candidate
