import ValueContract.DecoderSemantics

namespace ValueContract.Candidate

/-- Runtime-backed targets must decode the very same canonical wire admitted by
the schema and preserve the visible observation. Documentation-only targets have
no decoder assertion; the usage flag is supplied by occurrence ownership. -/
def RuntimeObligation (use : TargetUse) (codecs : ScalarCodecs)
    (checks : ExternalScalarChecks) (targets : Targets) (identity : Identity)
    (wire : Wire) (observed : Value) : Prop :=
  match use with
  | .documentation => True
  | .runtime => ∃ decoded, RuntimeDecodes codecs checks targets identity wire decoded ∧
      StrictEquivalent observed decoded

/-- Representability describes canonical wire existence using independent
observation, canonical representation, schema and runtime judgments. It neither
calls project nor requires every service field to survive target visibility.
A partially supplied service object can therefore be complete for a selected
body, while an ambiguity obstruction never enters as a successful Resolution. -/
def Representable (use : TargetUse) (codecs : ScalarCodecs) (checks : ExternalScalarChecks)
    (targets : Targets) (role : Role) (identity : Identity) (value : Value) : Prop :=
  ∃ observed wire, Observe codecs targets role identity value observed ∧
    Canonical codecs targets identity observed wire ∧
    SchemaAccepts codecs checks targets identity wire ∧
    RuntimeObligation use codecs checks targets identity wire observed

end ValueContract.Candidate
