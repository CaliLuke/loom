import ValueContract.ResolutionControls
import ValueContract.ScalarProjection

namespace ValueContract.Candidate.NumericRepresentationControls

/-- Exact binary value of float32(0.1), also exactly representable by float64. -/
def binaryTenth (format : NumericFormat) : Scalar :=
  .decimal 100000001490116119384765625 (-27) format false

def numbers : NumericCodec := {
  ResolutionControls.keys with
  encodeDecimal := fun coefficient _ format negativeZero =>
    if coefficient = 0 then (if negativeZero then "-0" else "0")
    else if format = .binary32 then "0.1" else "0.10000000149011612"
}

def codecs : ScalarCodecs := ⟨⟨fun _ => "", fun _ => none⟩, numbers⟩

/-- Numeric enum equality is independent of raw width and signed-zero evidence. -/
theorem representationPreservesNumericEquality (coefficient exponent : Int)
    (left right : NumericFormat) (leftSign rightSign : Bool) :
    scalarEqual (.decimal coefficient exponent left leftSign)
      (.decimal coefficient exponent right rightSign) = true := by
  simp [scalarEqual, scalarNumber, Decimal.le]

theorem equalMeaningDistinctWire :
    scalarEqual (binaryTenth .binary32) (binaryTenth .binary64) = true ∧
    encodeScalar codecs .json (binaryTenth .binary32) = .number "0.1" ∧
    encodeScalar codecs .json (binaryTenth .binary64) = .number "0.10000000149011612" := by
  cbv

theorem equalMeaningDistinctMemberNames :
    nameKeys numbers [binaryTenth .binary32, binaryTenth .binary64] =
      .ok ["0.1", "0.10000000149011612"] := by cbv

theorem signedZeroSpellingRetained :
    scalarEqual (.decimal 0 0 .binary64 true) (.decimal 0 0 .binary64 false) = true ∧
    encodeScalar codecs .json (.decimal 0 0 .binary64 true) = .number "-0" ∧
    encodeScalar codecs .json (.decimal 0 0 .binary64 false) = .number "0" := by cbv

/-- Retained counterexample for the old pairwise numeric-equality uniqueness
rule. Distinct canonical member names above are valid, yet that rule rejects
this list. Source key identity must be separated from enum numeric equality. -/
theorem numericEqualityIsNotKeyUniqueness :
    ¬ ([binaryTenth .binary32, binaryTenth .binary64].Pairwise
      (fun left right => scalarEqual left right = false)) := by decide

end ValueContract.Candidate.NumericRepresentationControls
