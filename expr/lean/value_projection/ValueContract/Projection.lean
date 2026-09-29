import ValueContract.ProjectionBuild
import ValueContract.Representation
import ValueContract.StrictValueDepth

namespace ValueContract.Candidate

def buildFailureResult : BuildFailure → ProjectionResult
  | .incomplete => .incomplete
  | .unsupported => .unsupported
  | .invalidPlan => .invalidPlan
  | .unrepresentable => .unrepresentable

def evaluationFailureResult : Failure → ProjectionResult
  | .malformedDeclaration | .cyclic => .invalidPlan
  | .unsupported => .unsupported
  | .invalid | .ambiguous => .unrepresentable

/-- The candidate constructs one canonical representation, checks its schema,
then independently decodes that SAME wire for runtime-backed targets. Branches
are never selected from target visibility or schema-only oneOf membership. -/
def project (codecs : ScalarCodecs) (checks : ExternalScalarChecks) (targets : Targets)
    (use : TargetUse) (role : Role) (identity : Identity) (resolved : Resolution) : ProjectionResult :=
  match build codecs targets role identity resolved.value with
  | .error failure => buildFailureResult failure
  | .ok (_, none) => .incomplete
  | .ok (observed, some wire) =>
    match schema codecs checks targets identity wire with
    | .error failure => evaluationFailureResult failure
    | .ok false => .unrepresentable
    | .ok true => match use with
      | .documentation => .emitted wire
      | .runtime => match decode codecs checks targets identity wire with
        | .error failure => evaluationFailureResult failure
        | .ok none => .unrepresentable
        | .ok (some decoded) =>
          if valueEqualAt codecs.numbers (strictValueDepth observed + strictValueDepth decoded + 1) false observed decoded then .emitted wire
          else .unrepresentable

end ValueContract.Candidate
