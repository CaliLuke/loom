import ValueContract.WireEquality
import ValueContract.ValueDepth

namespace ValueContract.Candidate

/-- Structural depth for strict observation equality. JSON snapshots contribute
their actual wire height; host evidence and Any each contribute one transparent
layer. This executable measure has no configured input-depth limit. -/
def strictValueDepth : Value → Nat
  | .jsonSnapshot wire => wireHeight wire + 1
  | .array items => 1 + (items.map strictValueDepth).foldl max 0
  | .object fields additional => 1 + max
      ((fields.map (fun field => strictValueDepth field.2)).foldl max 0)
      ((additional.map (fun field => strictValueDepth field.2)).foldl max 0)
  | .map entries => 1 + (entries.map (fun entry => strictValueDepth entry.2)).foldl max 0
  | .union _ _ payload | .any payload | .host _ payload => 1 + strictValueDepth payload
  | _ => 1
termination_by value => sizeOf value
decreasing_by
  all_goals simp_wf
  all_goals first
    | exact pairChildSize_lt ‹_ ∈ _›
    | exact Nat.lt_trans (List.sizeOf_lt_of_mem ‹_ ∈ _›) (by omega)
    | have smaller := pairChildSize_lt ‹_ ∈ _›; omega
    | omega

end ValueContract.Candidate
