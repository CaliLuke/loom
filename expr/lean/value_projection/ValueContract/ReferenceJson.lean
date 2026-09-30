/-
JSON adapters for the executable reference. These instances serialize the actual
model types; they do not define another resolver or projection policy. Entry
lists and tagged constructors preserve duplicate keys, identities and presence.
The adapter and Lean compiler are tested boundaries, not new proved claims.
-/
import Lean
import ValueContract.SourceSelection
import ValueContract.Resolve
import ValueContract.Projection
import ValueContract.AliasContracts

open Lean

namespace ValueContract.Reference

instance : ToJson UInt8 where
  toJson value := toJson value.toNat

instance : FromJson UInt8 where
  fromJson? json := do
    let value : Nat ← fromJson? json
    if value < 256 then return UInt8.ofNat value
    else throw "byte must be between 0 and 255"

deriving instance ToJson, FromJson for Identity
deriving instance ToJson, FromJson for ScalarKind
deriving instance ToJson, FromJson for Scalar
deriving instance ToJson, FromJson for Role
deriving instance ToJson, FromJson for Source
deriving instance ToJson, FromJson for Supplied
deriving instance ToJson, FromJson for UnionStyle
deriving instance ToJson, FromJson for SourceSelection.ExampleSources
deriving instance ToJson, FromJson for SourceSelection.Choice
deriving instance ToJson, FromJson for Candidate.IntegerFormat
deriving instance ToJson, FromJson for Candidate.NumericFormat
deriving instance ToJson, FromJson for Candidate.Scalar
deriving instance ToJson, FromJson for Candidate.RawPrimitive
deriving instance ToJson, FromJson for Candidate.SourcePrimitive
deriving instance ToJson, FromJson for Candidate.SourceScalar
deriving instance ToJson, FromJson for Candidate.Wire
deriving instance ToJson, FromJson for Candidate.Value
deriving instance ToJson, FromJson for Candidate.NativeByteSequence
deriving instance ToJson, FromJson for Candidate.Input
deriving instance ToJson, FromJson for Candidate.Decimal
deriving instance ToJson, FromJson for Candidate.AuthoredNumericBounds
deriving instance ToJson, FromJson for Candidate.PredicateKind
deriving instance ToJson, FromJson for Candidate.PredicateIdentity
deriving instance ToJson, FromJson for Candidate.AuthoredPredicateClause
deriving instance ToJson, FromJson for Candidate.AuthoredContractValue
deriving instance ToJson, FromJson for Candidate.RequiredField
deriving instance ToJson, FromJson for Candidate.AliasContractLayer
deriving instance ToJson, FromJson for Candidate.EffectiveAliasContract
deriving instance ToJson, FromJson for Candidate.AliasContractError
deriving instance ToJson, FromJson for Candidate.ParsedDecimal
deriving instance ToJson, FromJson for Candidate.LengthBounds
deriving instance ToJson, FromJson for Candidate.NumericBounds
deriving instance ToJson, FromJson for Candidate.ScalarRules
deriving instance ToJson, FromJson for Candidate.Member
deriving instance ToJson, FromJson for Candidate.Alternative
deriving instance ToJson, FromJson for Candidate.MapKeyKind
deriving instance ToJson, FromJson for Candidate.Contract
deriving instance ToJson, FromJson for Candidate.Declaration
deriving instance ToJson, FromJson for Candidate.Presence
deriving instance ToJson, FromJson for Candidate.TargetMember
deriving instance ToJson, FromJson for Candidate.TargetAlternative
deriving instance ToJson, FromJson for Candidate.Encoding
deriving instance ToJson, FromJson for Candidate.Target
deriving instance ToJson, FromJson for Candidate.TargetDeclaration
deriving instance ToJson, FromJson for Candidate.TargetUse
deriving instance ToJson, FromJson for Candidate.Resolution
deriving instance ToJson, FromJson for Candidate.Failure
deriving instance ToJson, FromJson for Candidate.ProjectionResult
deriving instance ToJson, FromJson for Candidate.BuildFailure

end ValueContract.Reference
