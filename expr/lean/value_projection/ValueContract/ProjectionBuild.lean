import ValueContract.CanonicalConstruction

namespace ValueContract.Candidate

/-- Visibility and field-local presence finish before required-wire checks.
The final projector separately checks this one canonical wire against the schema
and, for runtime-backed targets, the actual decoder. -/
def build (codecs : ScalarCodecs) (targets : Targets) (role : Role)
    (identity : Identity) (value : Value) : Except BuildFailure BuiltValue := do
  let observed ← observeValue codecs targets role identity value
  let wire ← construct codecs targets identity observed
  return (observed, wire)

end ValueContract.Candidate
